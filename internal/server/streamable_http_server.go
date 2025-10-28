package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	server "github.com/mark3labs/mcp-go/server"
	"github.com/matthieu/mcp-server-prtg/internal/database"
	"github.com/matthieu/mcp-server-prtg/internal/services/configuration"
	"github.com/matthieu/mcp-server-prtg/internal/services/logger"
	"github.com/matthieu/mcp-server-prtg/internal/version"
)

// authAttempt tracks authentication attempts for rate limiting
type authAttempt struct {
	count       int
	firstTry    time.Time
	lastTry     time.Time
	lockedUntil time.Time
}

// authRateLimiter manages rate limiting for authentication attempts per IP
type authRateLimiter struct {
	attempts map[string]*authAttempt
	mu       sync.RWMutex

	// Configuration
	maxAttempts int           // Max attempts before lockout
	window      time.Duration // Time window for counting attempts
	lockoutTime time.Duration // How long to lock out after max attempts
}

// newAuthRateLimiter creates a new rate limiter with default settings
func newAuthRateLimiter() *authRateLimiter {
	return &authRateLimiter{
		attempts:    make(map[string]*authAttempt),
		maxAttempts: 5,               // 5 attempts max
		window:      1 * time.Minute, // per minute
		lockoutTime: 5 * time.Minute, // locked for 5 minutes after max attempts
	}
}

// checkAndRecord checks if an IP is rate limited and records the attempt
// Returns true if the request should be allowed, false if rate limited
func (rl *authRateLimiter) checkAndRecord(ip string, success bool) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Get or create attempt record
	attempt, exists := rl.attempts[ip]
	if !exists {
		attempt = &authAttempt{
			count:    0,
			firstTry: now,
		}
		rl.attempts[ip] = attempt
	}

	// Check if IP is currently locked out
	if now.Before(attempt.lockedUntil) {
		return false // Still locked out
	}

	// If successful auth, reset the counter
	if success {
		delete(rl.attempts, ip)
		return true
	}

	// Reset counter if window has expired
	if now.Sub(attempt.firstTry) > rl.window {
		attempt.count = 0
		attempt.firstTry = now
	}

	// Increment counter
	attempt.count++
	attempt.lastTry = now

	// Check if max attempts exceeded
	if attempt.count > rl.maxAttempts {
		attempt.lockedUntil = now.Add(rl.lockoutTime)
		return false // Lock out this IP
	}

	return true // Allow attempt
}

// cleanup removes old entries (optional, called periodically)
func (rl *authRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, attempt := range rl.attempts {
		// Remove entries older than lockout time
		if now.Sub(attempt.lastTry) > rl.lockoutTime {
			delete(rl.attempts, ip)
		}
	}
}

// StreamableHTTPServer implements MCP server using Streamable HTTP transport
type StreamableHTTPServer struct {
	mcpServer      *server.MCPServer
	streamableHTTP http.Handler
	httpServer     *http.Server
	config         *configuration.Configuration
	logger         *logger.ModuleLogger
	db             *database.DB
	rateLimiter    *authRateLimiter
	address        string
	shutdownCh     chan struct{} // Channel for graceful shutdown of background tasks
}

// NewStreamableHTTPServer creates a new Streamable HTTP-based MCP server
func NewStreamableHTTPServer(mcpServer *server.MCPServer, db *database.DB, config *configuration.Configuration, baseLogger *logger.Logger) *StreamableHTTPServer {
	logger := logger.NewModuleLogger(baseLogger, logger.ModuleServer)

	// Get server address for binding
	address := config.GetServerAddress()

	return &StreamableHTTPServer{
		mcpServer:   mcpServer,
		config:      config,
		logger:      logger,
		db:          db,
		rateLimiter: newAuthRateLimiter(),
		address:     address,
		shutdownCh:  make(chan struct{}),
	}
}

// Start starts the Streamable HTTP server
func (s *StreamableHTTPServer) Start(_ context.Context) error {
	s.logger.Info().
		Str("address", s.address).
		Bool("tls", s.config.IsTLSEnabled()).
		Msg("Starting MCP Server with Streamable HTTP transport")

	// Create Streamable HTTP server with heartbeat support
	// Default heartbeat interval is 30 seconds, can be configured
	heartbeatInterval := 30 * time.Second
	heartbeatOption := server.WithHeartbeatInterval(heartbeatInterval)
	s.streamableHTTP = server.NewStreamableHTTPServer(s.mcpServer, heartbeatOption)

	// Start rate limiter cleanup goroutine
	go s.cleanupRateLimiterPeriodically()

	// Create HTTP server with endpoints
	if err := s.startHTTPServer(); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	// Log startup information
	s.logStartupInfo()

	return nil
}

// startHTTPServer starts the HTTP server with all endpoints
func (s *StreamableHTTPServer) startHTTPServer() error {
	// Create mux with all endpoints
	mux := http.NewServeMux()

	// MCP endpoint with authentication middleware
	mux.Handle("/mcp", s.createAuthMiddleware(s.streamableHTTP))

	// Health check endpoint (no auth)
	mux.HandleFunc("/health", s.handleHealth)

	// Status endpoint (auth required)
	statusHandler := s.createAuthMiddleware(http.HandlerFunc(s.handleStatus))
	mux.Handle("/status", statusHandler)

	// Create HTTP server with optimized timeouts
	s.httpServer = &http.Server{
		Addr:              s.address,
		Handler:           mux,
		ReadTimeout:       0,                // No read timeout for streaming connections
		WriteTimeout:      0,                // No write timeout for streaming connections
		IdleTimeout:       60 * time.Minute, // Close inactive connections after 1 hour
		ReadHeaderTimeout: 10 * time.Second, // Protection against slow-loris attacks
		MaxHeaderBytes:    1 << 20,          // 1MB max header size
	}

	// Configure TLS if enabled
	if s.config.IsTLSEnabled() {
		certFile := s.config.GetTLSCertFile()
		keyFile := s.config.GetTLSKeyFile()

		tlsConfig := &tls.Config{
			MinVersion:               tls.VersionTLS12,
			PreferServerCipherSuites: true,
		}

		s.httpServer.TLSConfig = tlsConfig

		s.logger.Info().
			Str("cert", certFile).
			Str("key", keyFile).
			Msg("Starting HTTPS server")

		// Start server in background
		go func() {
			if err := s.httpServer.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				s.logger.Error().Err(err).Msg("HTTPS server error")
			}
		}()
	} else {
		s.logger.Warn().Msg("Starting HTTP server (TLS disabled - not recommended for production)")

		// Start server in background
		go func() {
			if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				s.logger.Error().Err(err).Msg("HTTP server error")
			}
		}()
	}

	return nil
}

// getClientIP extracts the real client IP from the request
// Handles X-Forwarded-For and X-Real-IP headers for proxy situations
func getClientIP(r *http.Request) string {
	// Try X-Real-IP first (single IP from trusted proxy)
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	// Try X-Forwarded-For (can be a list)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// Take the first IP (client IP)
		if idx := strings.Index(forwarded, ","); idx != -1 {
			return strings.TrimSpace(forwarded[:idx])
		}
		return strings.TrimSpace(forwarded)
	}

	// Fall back to RemoteAddr
	// RemoteAddr is in format "IP:port", extract just the IP
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}

	return r.RemoteAddr
}

// createAuthMiddleware creates authentication middleware using Bearer token with rate limiting
func (s *StreamableHTTPServer) createAuthMiddleware(next http.Handler) http.Handler {
	expectedToken := s.config.GetAPIKey()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract client IP for rate limiting
		clientIP := getClientIP(r)

		// Check rate limit BEFORE validating token (prevent brute-force)
		if !s.rateLimiter.checkAndRecord(clientIP, false) {
			s.logger.Warn().
				Str("client_ip", clientIP).
				Str("path", r.URL.Path).
				Msg("Rate limit exceeded - IP temporarily blocked")

			w.Header().Set("Retry-After", "300") // 5 minutes
			http.Error(w, "Too many authentication attempts. Please try again later.", http.StatusTooManyRequests)
			return
		}

		// Extract Bearer token from Authorization header
		authHeader := r.Header.Get("Authorization")
		var providedToken string

		if authHeader != "" {
			// Check for "Bearer <token>" format
			const bearerPrefix = "Bearer "
			if len(authHeader) > len(bearerPrefix) && authHeader[:len(bearerPrefix)] == bearerPrefix {
				providedToken = authHeader[len(bearerPrefix):]
			}
		}

		// Fallback: check query parameter (for compatibility)
		if providedToken == "" {
			providedToken = r.URL.Query().Get("token")
		}

		// Validate token
		if providedToken != expectedToken {
			s.logger.Warn().
				Str("client_ip", clientIP).
				Str("remote_addr", r.RemoteAddr).
				Str("path", r.URL.Path).
				Str("method", r.Method).
				Bool("has_auth_header", authHeader != "").
				Msg("Unauthorized access attempt")

			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Authentication successful - record it
		s.rateLimiter.checkAndRecord(clientIP, true)

		// Log successful authentication
		s.logger.Debug().
			Str("client_ip", clientIP).
			Str("path", r.URL.Path).
			Str("method", r.Method).
			Msg("Authenticated request")

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// cleanupRateLimiterPeriodically runs periodic cleanup of rate limiter entries
func (s *StreamableHTTPServer) cleanupRateLimiterPeriodically() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.rateLimiter.cleanup()
			s.logger.Debug().Msg("Cleaned up rate limiter entries")
		case <-s.shutdownCh:
			s.logger.Debug().Msg("Stopping rate limiter cleanup goroutine")
			return
		}
	}
}

// handleHealth handles health check requests
func (s *StreamableHTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// handleStatus handles status requests (requires authentication)
func (s *StreamableHTTPServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"version":   version.Get(),
		"transport": "streamable-http",
		"protocol":  "2025-03-26",
		"uptime":    time.Since(startTime).String(),
	}

	// Check database connection
	if s.db != nil {
		if err := s.db.Health(r.Context()); err != nil {
			status["database"] = "error"
			status["database_error"] = err.Error()
		} else {
			status["database"] = "connected"
		}
	} else {
		status["database"] = "not_configured"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Simple JSON encoding
	fmt.Fprintf(w, `{"version":"%s","transport":"streamable-http","protocol":"2025-03-26","uptime":"%s","database":"%s"}`,
		status["version"], status["uptime"], status["database"])
}

// logStartupInfo logs startup information
func (s *StreamableHTTPServer) logStartupInfo() {
	protocol := "http"
	if s.config.IsTLSEnabled() {
		protocol = "https"
	}

	s.logger.Info().
		Str("url", fmt.Sprintf("%s://%s/mcp", protocol, s.address)).
		Str("health_check", fmt.Sprintf("%s://%s/health", protocol, s.address)).
		Str("status", fmt.Sprintf("%s://%s/status", protocol, s.address)).
		Str("version", version.Get()).
		Str("protocol", "2025-03-26").
		Msg("MCP Server ready")

	s.logger.Info().Msg("Configure Claude Desktop with:")
	s.logger.Info().Msgf(`  "mcpServers": {`)
	s.logger.Info().Msgf(`    "prtg": {`)
	s.logger.Info().Msgf(`      "url": "%s://%s/mcp",`, protocol, s.address)
	s.logger.Info().Msgf(`      "headers": {`)
	s.logger.Info().Msgf(`        "Authorization": "Bearer YOUR_API_KEY"`)
	s.logger.Info().Msgf(`      }`)
	s.logger.Info().Msgf(`    }`)
	s.logger.Info().Msgf(`  }`)
}

// Shutdown gracefully shuts down the server
func (s *StreamableHTTPServer) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("Shutting down Streamable HTTP server")

	// Signal background tasks to stop
	close(s.shutdownCh)

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	s.logger.Info().Msg("Server shutdown complete")
	return nil
}

var startTime = time.Now()
