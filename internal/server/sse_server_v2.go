package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
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

// SSEServerV2 wraps the MCP SSE server with authentication and TLS.
type SSEServerV2 struct {
	mcpServer    *server.MCPServer
	sseServer    *server.SSEServer
	proxyServer  *http.Server
	config       *configuration.Configuration
	logger       *logger.ModuleLogger
	db           *database.DB
	rateLimiter  *authRateLimiter
	baseURL      string
	internalAddr string
	externalAddr string
	shutdownCh   chan struct{} // Channel for graceful shutdown of background tasks
}

// NewSSEServerV2 creates a new SSE-based MCP server with authentication proxy.
func NewSSEServerV2(mcpServer *server.MCPServer, db *database.DB, config *configuration.Configuration, baseLogger *logger.Logger) *SSEServerV2 {
	logger := logger.NewModuleLogger(baseLogger, logger.ModuleServer)

	// Get public URL for SSE endpoint (uses public_url from config if set)
	baseURL := config.GetPublicURL()

	// Internal SSE server runs on a different port (no auth, localhost only)
	internalAddr := "127.0.0.1:18443"

	// External address for binding
	externalAddr := config.GetServerAddress()

	return &SSEServerV2{
		mcpServer:    mcpServer,
		config:       config,
		logger:       logger,
		db:           db,
		rateLimiter:  newAuthRateLimiter(),
		baseURL:      baseURL,
		internalAddr: internalAddr,
		externalAddr: externalAddr,
		shutdownCh:   make(chan struct{}),
	}
}

// Start starts the SSE server with authentication proxy.
func (s *SSEServerV2) Start(_ context.Context) error {
	s.logger.Info().
		Str("external_address", s.externalAddr).
		Str("internal_address", s.internalAddr).
		Bool("tls", s.config.IsTLSEnabled()).
		Msg("Starting MCP Server with SSE transport (v2)")

	// Create internal SSE server with public base URL
	// The SSE server will append /message to this URL automatically
	s.sseServer = server.NewSSEServer(s.mcpServer, s.baseURL)

	// Start internal SSE server in background
	go func() {
		s.logger.Info().Str("address", s.internalAddr).Msg("Starting internal SSE server")
		if err := s.sseServer.Start(s.internalAddr); err != nil {
			s.logger.Error().Err(err).Msg("Internal SSE server error")
		}
	}()

	// Wait a bit for internal server to start
	time.Sleep(500 * time.Millisecond)

	// Start rate limiter cleanup goroutine
	go s.cleanupRateLimiterPeriodically()

	// Create authentication proxy
	if err := s.startAuthProxy(); err != nil {
		return fmt.Errorf("failed to start auth proxy: %w", err)
	}

	// Log startup information
	s.logStartupInfo()

	return nil
}

// startAuthProxy starts the external authentication proxy.
func (s *SSEServerV2) startAuthProxy() error {
	// Parse internal URL
	internalURL, err := url.Parse(fmt.Sprintf("http://%s", s.internalAddr))
	if err != nil {
		return fmt.Errorf("failed to parse internal URL: %w", err)
	}

	// Create reverse proxy to internal SSE server
	proxy := httputil.NewSingleHostReverseProxy(internalURL)

	// Custom director to preserve headers
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = internalURL.Host
	}

	// Create authenticated handler
	authHandler := s.createAuthMiddleware()(proxy.ServeHTTP)

	// Create mux with additional endpoints
	mux := http.NewServeMux()

	// SSE endpoints (proxied with auth)
	mux.HandleFunc("/sse", authHandler)
	mux.HandleFunc("/message", authHandler)

	// Health check endpoint (no auth)
	mux.HandleFunc("/health", s.handleHealth)

	// Status endpoint (auth required)
	mux.HandleFunc("/status", s.createAuthMiddleware()(s.handleStatus))

	// Create HTTP server with optimized timeouts for SSE
	// SSE connections are long-lived, but we still need protection against resource exhaustion
	s.proxyServer = &http.Server{
		Addr:              s.externalAddr,
		Handler:           mux,
		ReadTimeout:       0,                // No read timeout for SSE (connections stay open)
		WriteTimeout:      0,                // No write timeout for SSE (long-lived streaming)
		IdleTimeout:       5 * time.Minute,  // Close inactive connections after 5 minutes
		ReadHeaderTimeout: 10 * time.Second, // Protection against slow-loris attacks
		MaxHeaderBytes:    1 << 20,          // 1MB max header size (prevent memory exhaustion)
	}

	// Start server (with or without TLS)
	go s.startProxyServer()

	return nil
}

// startProxyServer starts the proxy HTTP server.
func (s *SSEServerV2) startProxyServer() {
	var err error

	if s.config.IsTLSEnabled() {
		// Load TLS configuration
		certFile := s.config.GetTLSCertFile()
		keyFile := s.config.GetTLSKeyFile()

		// Configure TLS
		tlsConfig := &tls.Config{
			MinVersion:               tls.VersionTLS12,
			PreferServerCipherSuites: true,
		}

		s.proxyServer.TLSConfig = tlsConfig

		s.logger.Info().
			Str("cert", certFile).
			Str("key", keyFile).
			Msg("Starting HTTPS proxy server")

		err = s.proxyServer.ListenAndServeTLS(certFile, keyFile)
	} else {
		s.logger.Warn().Msg("Starting HTTP proxy server (TLS disabled - not recommended for production)")
		err = s.proxyServer.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		s.logger.Error().Err(err).Msg("HTTP proxy server error")
	}
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

// createAuthMiddleware creates authentication middleware using Bearer token with rate limiting.
func (s *SSEServerV2) createAuthMiddleware() func(http.HandlerFunc) http.HandlerFunc {
	expectedToken := s.config.GetAPIKey()

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
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

			// Fallback: check query parameter (for SSE compatibility)
			if providedToken == "" {
				providedToken = r.URL.Query().Get("token")
			}

			// Validate token
			if providedToken != expectedToken {
				// Record failed attempt (already counted in checkAndRecord above)
				s.logger.Warn().
					Str("client_ip", clientIP).
					Str("remote_addr", r.RemoteAddr).
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Bool("has_auth_header", authHeader != "").
					Msg("Unauthorized access attempt")

				w.Header().Set("WWW-Authenticate", "Bearer realm=\"MCP Server PRTG\"")
				http.Error(w, "Unauthorized: Missing or invalid Bearer token", http.StatusUnauthorized)
				return
			}

			// Token valid - record successful authentication and reset counter
			s.rateLimiter.checkAndRecord(clientIP, true)

			s.logger.Debug().
				Str("client_ip", clientIP).
				Str("remote_addr", r.RemoteAddr).
				Str("path", r.URL.Path).
				Str("method", r.Method).
				Msg("Authenticated request")

			next(w, r)
		}
	}
}

// handleHealth handles health check requests.
func (s *SSEServerV2) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

// handleStatus handles status requests (authenticated).
func (s *SSEServerV2) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check database health
	dbStatus := "connected"
	dbError := ""
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if s.db != nil {
		if err := s.db.Health(ctx); err != nil {
			dbStatus = "disconnected"
			dbError = err.Error()
			s.logger.Warn().Err(err).Msg("Database health check failed")
		}
	} else {
		dbStatus = "not_configured"
	}

	w.WriteHeader(http.StatusOK)

	status := fmt.Sprintf(`{
		"status":"running",
		"version":"%s",
		"transport":"sse",
		"tls_enabled":%t,
		"base_url":"%s",
		"mcp_tools":6,
		"database":{
			"status":"%s",
			"error":"%s"
		},
		"timestamp":"%s"
	}`, version.Get(), s.config.IsTLSEnabled(), s.baseURL, dbStatus, dbError, time.Now().Format(time.RFC3339))

	fmt.Fprint(w, status)
}

// cleanupRateLimiterPeriodically runs periodic cleanup of the rate limiter map to prevent memory leaks.
func (s *SSEServerV2) cleanupRateLimiterPeriodically() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	s.logger.Debug().Msg("Rate limiter cleanup goroutine started")

	for {
		select {
		case <-ticker.C:
			s.rateLimiter.cleanup()
			s.logger.Debug().Msg("Rate limiter cleanup completed")
		case <-s.shutdownCh:
			s.logger.Debug().Msg("Rate limiter cleanup goroutine shutting down")
			return
		}
	}
}

// Shutdown gracefully shuts down the server.
func (s *SSEServerV2) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("Shutting down SSE server")

	// Signal background goroutines to stop
	select {
	case <-s.shutdownCh:
		// Already closed
	default:
		close(s.shutdownCh)
	}

	var firstErr error

	// Shutdown proxy server
	if s.proxyServer != nil {
		if err := s.proxyServer.Shutdown(ctx); err != nil {
			s.logger.Error().Err(err).Msg("Error shutting down proxy server")
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	// Shutdown internal SSE server
	if s.sseServer != nil {
		if err := s.sseServer.Shutdown(ctx); err != nil {
			s.logger.Error().Err(err).Msg("Error shutting down SSE server")
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	if firstErr == nil {
		s.logger.Info().Msg("SSE server shut down successfully")
	}

	return firstErr
}

// logStartupInfo logs startup information.
func (s *SSEServerV2) logStartupInfo() {
	protocol := "HTTP"
	if s.config.IsTLSEnabled() {
		protocol = "HTTPS"
	}

	s.logger.Info().Msgf(`
╔════════════════════════════════════════════════════════════════╗
║         MCP Server PRTG - SSE Transport Started (v2)           ║
╠════════════════════════════════════════════════════════════════╣
║  Protocol:     %s                                           ║
║  External:     %s                                    ║
║  Internal:     %s                              ║
║  Base URL:     %s                        ║
║  MCP Tools:    6 tools registered                             ║
║  Endpoints:                                                    ║
║    - GET  /sse      (SSE stream - authenticated)              ║
║    - POST /message  (RPC messages - authenticated)            ║
║    - GET  /health   (Health check - public)                   ║
║    - GET  /status   (Server status - authenticated)           ║
╠════════════════════════════════════════════════════════════════╣
║  Authentication: Bearer Token (RFC 6750)                       ║
║    - Header: Authorization: Bearer <token>                     ║
║    - Query:  ?token=<token> (fallback for SSE)                 ║
║  Token: %s                                    ║
╚════════════════════════════════════════════════════════════════╝
	`, protocol, s.externalAddr, s.internalAddr, s.baseURL, maskAPIKey(s.config.GetAPIKey()))
}

// maskAPIKey masks the API key for logging.
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}

	return key[:4] + "..." + key[len(key)-4:]
}
