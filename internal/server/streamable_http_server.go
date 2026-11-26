package server

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"encoding/json"
	"errors"
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

const (
	minAPIKeyLength     = 16
	rateLimitMaxEntries = 10000
)

// authAttempt tracks authentication attempts for rate limiting.
type authAttempt struct {
	count       int
	firstTry    time.Time
	lastTry     time.Time
	lockedUntil time.Time
}

// authRateLimiter manages rate limiting for failed authentication attempts per IP.
type authRateLimiter struct {
	attempts    map[string]*authAttempt
	mu          sync.Mutex
	maxAttempts int
	window      time.Duration
	lockoutTime time.Duration
	maxEntries  int
}

func newAuthRateLimiter() *authRateLimiter {
	return &authRateLimiter{
		attempts:    make(map[string]*authAttempt),
		maxAttempts: 5,
		window:      1 * time.Minute,
		lockoutTime: 5 * time.Minute,
		maxEntries:  rateLimitMaxEntries,
	}
}

// isLocked reports whether the IP is currently locked out.
func (rl *authRateLimiter) isLocked(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	attempt, exists := rl.attempts[ip]
	if !exists {
		return false
	}

	return time.Now().Before(attempt.lockedUntil)
}

// recordFailure records a failed authentication attempt and may lock the IP.
func (rl *authRateLimiter) recordFailure(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	attempt, exists := rl.attempts[ip]
	if !exists {
		if len(rl.attempts) >= rl.maxEntries {
			rl.cleanupLocked(now)
			if len(rl.attempts) >= rl.maxEntries {
				rl.dropOldestLocked()
			}
		}

		attempt = &authAttempt{
			count:    0,
			firstTry: now,
		}
		rl.attempts[ip] = attempt
	}

	if now.Sub(attempt.firstTry) > rl.window {
		attempt.count = 0
		attempt.firstTry = now
	}

	attempt.count++
	attempt.lastTry = now

	if attempt.count > rl.maxAttempts {
		attempt.lockedUntil = now.Add(rl.lockoutTime)
	}
}

// reset clears the entry for an IP after a successful authentication.
func (rl *authRateLimiter) reset(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.attempts, ip)
}

// cleanupLocked removes stale entries; caller must hold mu.
func (rl *authRateLimiter) cleanupLocked(now time.Time) {
	for ip, attempt := range rl.attempts {
		if now.Sub(attempt.lastTry) > rl.lockoutTime {
			delete(rl.attempts, ip)
		}
	}
}

// dropOldestLocked removes the entry with the oldest lastTry; caller must hold mu.
func (rl *authRateLimiter) dropOldestLocked() {
	var oldestIP string

	var oldest time.Time

	first := true
	for ip, attempt := range rl.attempts {
		if first || attempt.lastTry.Before(oldest) {
			oldestIP = ip
			oldest = attempt.lastTry
			first = false
		}
	}

	if !first {
		delete(rl.attempts, oldestIP)
	}
}

// cleanup removes old entries.
func (rl *authRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.cleanupLocked(time.Now())
}

// StreamableHTTPServer implements MCP server using Streamable HTTP transport.
type StreamableHTTPServer struct {
	mcpServer      *server.MCPServer
	streamableHTTP http.Handler
	httpServer     *http.Server
	config         *configuration.Configuration
	logger         *logger.ModuleLogger
	db             *database.DB
	rateLimiter    *authRateLimiter
	address        string
	shutdownCh     chan struct{}
	shutdownOnce   sync.Once
}

// NewStreamableHTTPServer creates a new Streamable HTTP-based MCP server.
func NewStreamableHTTPServer(
	mcpServer *server.MCPServer,
	db *database.DB,
	config *configuration.Configuration,
	baseLogger *logger.Logger,
) *StreamableHTTPServer {
	moduleLogger := logger.NewModuleLogger(baseLogger, logger.ModuleServer)

	return &StreamableHTTPServer{
		mcpServer:   mcpServer,
		config:      config,
		logger:      moduleLogger,
		db:          db,
		rateLimiter: newAuthRateLimiter(),
		address:     config.GetServerAddress(),
		shutdownCh:  make(chan struct{}),
	}
}

// Start starts the Streamable HTTP server.
func (s *StreamableHTTPServer) Start(_ context.Context) error {
	apiKey := s.config.GetAPIKey()
	if len(apiKey) < minAPIKeyLength {
		return fmt.Errorf("API key must be at least %d characters, refusing to start with a weak or empty key", minAPIKeyLength)
	}

	s.logger.Info().
		Str("address", s.address).
		Bool("tls", s.config.IsTLSEnabled()).
		Msg("Starting MCP Server with Streamable HTTP transport")

	heartbeatInterval := 30 * time.Second
	heartbeatOption := server.WithHeartbeatInterval(heartbeatInterval)
	s.streamableHTTP = server.NewStreamableHTTPServer(s.mcpServer, heartbeatOption)

	go s.cleanupRateLimiterPeriodically()

	if err := s.startHTTPServer(); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	s.logStartupInfo()

	return nil
}

// startHTTPServer binds the listener and starts serving.
func (s *StreamableHTTPServer) startHTTPServer() error {
	mux := http.NewServeMux()

	mux.Handle("/mcp", createAuthMiddleware(s.streamableHTTP, s.config.GetAPIKey, s.rateLimiter, s.logger))

	mux.HandleFunc("/health", s.handleHealth)

	mux.Handle("/status", createAuthMiddleware(http.HandlerFunc(s.handleStatus), s.config.GetAPIKey, s.rateLimiter, s.logger))

	s.httpServer = &http.Server{
		Addr:              s.address,
		Handler:           mux,
		ReadTimeout:       0,
		WriteTimeout:      0,
		IdleTimeout:       60 * time.Minute,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to bind %s: %w", s.address, err)
	}

	if s.config.IsTLSEnabled() {
		certFile := s.config.GetTLSCertFile()
		keyFile := s.config.GetTLSKeyFile()

		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			if closeErr := listener.Close(); closeErr != nil {
				s.logger.Warn().Err(closeErr).Msg("Failed to close listener after TLS setup error")
			}

			return fmt.Errorf("failed to load TLS key pair: %w", err)
		}

		s.httpServer.TLSConfig = &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		}

		tlsListener := tls.NewListener(listener, s.httpServer.TLSConfig)

		s.logger.Info().
			Str("cert", certFile).
			Str("key", keyFile).
			Msg("Starting HTTPS server")

		go s.serve(tlsListener)
	} else {
		s.logger.Warn().Msg("Starting HTTP server (TLS disabled - not recommended for production)")

		go s.serve(listener)
	}

	return nil
}

func (s *StreamableHTTPServer) serve(listener net.Listener) {
	if err := s.httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.logger.Error().Err(err).Msg("HTTP server error")
	}
}

// getClientIP extracts the client IP. Headers are intentionally ignored: they
// are client-controlled and would let an attacker bypass or poison the limiter.
func getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

// extractBearerToken returns the token from an Authorization: Bearer header.
func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

// createAuthMiddleware creates authentication middleware using a Bearer token
// with rate limiting on failed attempts. The expected key is read through
// keyFn at request time so rotation applies without a restart.
func createAuthMiddleware(next http.Handler, keyFn func() string, rl *authRateLimiter, log *logger.ModuleLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)

		if rl.isLocked(clientIP) {
			log.Warn().
				Str("client_ip", clientIP).
				Str("path", r.URL.Path).
				Msg("Rate limit exceeded - IP temporarily blocked")

			w.Header().Set("Retry-After", "300")
			http.Error(w, "Too many authentication attempts. Please try again later.", http.StatusTooManyRequests)

			return
		}

		providedToken := extractBearerToken(r)
		expectedToken := keyFn()

		authorized := expectedToken != "" &&
			len(expectedToken) >= minAPIKeyLength &&
			subtle.ConstantTimeCompare([]byte(providedToken), []byte(expectedToken)) == 1

		if !authorized {
			rl.recordFailure(clientIP)

			log.Warn().
				Str("client_ip", clientIP).
				Str("remote_addr", r.RemoteAddr).
				Str("path", r.URL.Path).
				Str("method", r.Method).
				Bool("has_auth_header", r.Header.Get("Authorization") != "").
				Msg("Unauthorized access attempt")

			http.Error(w, "Unauthorized", http.StatusUnauthorized)

			return
		}

		rl.reset(clientIP)

		log.Debug().
			Str("client_ip", clientIP).
			Str("path", r.URL.Path).
			Str("method", r.Method).
			Msg("Authenticated request")

		next.ServeHTTP(w, r)
	})
}

// cleanupRateLimiterPeriodically runs periodic cleanup of rate limiter entries.
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

// handleHealth handles health check requests.
func (s *StreamableHTTPServer) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
		s.logger.Error().Err(err).Msg("Failed to write health response")
	}
}

type statusResponse struct {
	Version   string `json:"version"`
	Transport string `json:"transport"`
	Protocol  string `json:"protocol"`
	Uptime    string `json:"uptime"`
	Database  string `json:"database"`
}

// handleStatus handles status requests (requires authentication).
func (s *StreamableHTTPServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	dbStatus := "not_configured"
	if s.db != nil {
		if err := s.db.Health(r.Context()); err != nil {
			dbStatus = "error"
			s.logger.Warn().Err(err).Msg("Database health check failed")
		} else {
			dbStatus = "connected"
		}
	}

	resp := statusResponse{
		Version:   version.Get(),
		Transport: "streamable-http",
		Protocol:  "2025-03-26",
		Uptime:    time.Since(startTime).String(),
		Database:  dbStatus,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(data); err != nil {
		s.logger.Error().Err(err).Msg("Failed to write status response")
	}
}

// logStartupInfo logs startup information.
func (s *StreamableHTTPServer) logStartupInfo() {
	protocol := "http"
	if s.config.IsTLSEnabled() {
		protocol = "https"
	}

	base := protocol + "://" + s.address

	s.logger.Info().
		Str("url", base+"/mcp").
		Str("health_check", base+"/health").
		Str("status", base+"/status").
		Str("version", version.Get()).
		Str("protocol", "2025-03-26").
		Msg("MCP Server ready")

	s.logger.Info().Msg("Configure Claude Desktop with:")
	s.logger.Info().Msg(`  "mcpServers": {`)
	s.logger.Info().Msg(`    "prtg": {`)
	s.logger.Info().Msg(`      "url": "` + base + `/mcp",`)
	s.logger.Info().Msg(`      "headers": {`)
	s.logger.Info().Msg(`        "Authorization": "Bearer YOUR_API_KEY"`)
	s.logger.Info().Msg(`      }`)
	s.logger.Info().Msg(`    }`)
	s.logger.Info().Msg(`  }`)
}

// Shutdown gracefully shuts down the server. It is safe to call when Start was
// never invoked and is idempotent.
func (s *StreamableHTTPServer) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("Shutting down Streamable HTTP server")

	s.shutdownOnce.Do(func() {
		close(s.shutdownCh)
	})

	if s.httpServer == nil {
		return nil
	}

	if err := s.httpServer.Shutdown(ctx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			if closeErr := s.httpServer.Close(); closeErr != nil {
				s.logger.Warn().Err(closeErr).Msg("Failed to force-close HTTP server")
			}
		}

		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	s.logger.Info().Msg("Server shutdown complete")

	return nil
}

//nolint:gochecknoglobals // Server start time is package-level constant set once at initialization.
var startTime = time.Now()
