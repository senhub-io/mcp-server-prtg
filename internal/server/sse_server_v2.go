package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/matthieu/mcp-server-prtg/internal/database"
	"github.com/matthieu/mcp-server-prtg/internal/services/configuration"
	"github.com/matthieu/mcp-server-prtg/internal/services/logger"
	server "github.com/mark3labs/mcp-go/server"
)

// SSEServerV2 wraps the MCP SSE server with authentication and TLS.
type SSEServerV2 struct {
	mcpServer     *server.MCPServer
	sseServer     *server.SSEServer
	proxyServer   *http.Server
	config        *configuration.Configuration
	logger        *logger.ModuleLogger
	db            *database.DB
	baseURL       string
	internalAddr  string
	externalAddr  string
}

// NewSSEServerV2 creates a new SSE-based MCP server with authentication proxy.
func NewSSEServerV2(mcpServer *server.MCPServer, db *database.DB, config *configuration.Configuration, baseLogger *logger.Logger) *SSEServerV2 {
	logger := logger.NewModuleLogger(baseLogger, logger.ModuleServer)

	// Determine base URL for SSE
	protocol := "http"
	if config.IsTLSEnabled() {
		protocol = "https"
	}

	externalAddr := config.GetServerAddress()
	baseURL := fmt.Sprintf("%s://%s", protocol, externalAddr)

	// Internal SSE server runs on a different port (no auth, localhost only)
	internalAddr := "127.0.0.1:18443"

	return &SSEServerV2{
		mcpServer:    mcpServer,
		config:       config,
		logger:       logger,
		db:           db,
		baseURL:      baseURL,
		internalAddr: internalAddr,
		externalAddr: externalAddr,
	}
}

// Start starts the SSE server with authentication proxy.
func (s *SSEServerV2) Start(ctx context.Context) error {
	s.logger.Info().
		Str("external_address", s.externalAddr).
		Str("internal_address", s.internalAddr).
		Bool("tls", s.config.IsTLSEnabled()).
		Msg("Starting MCP Server with SSE transport (v2)")

	// Create internal SSE server (no auth, localhost only)
	s.sseServer = server.NewSSEServer(s.mcpServer, fmt.Sprintf("http://%s", s.internalAddr))

	// Start internal SSE server in background
	go func() {
		s.logger.Info().Str("address", s.internalAddr).Msg("Starting internal SSE server")
		if err := s.sseServer.Start(s.internalAddr); err != nil {
			s.logger.Error().Err(err).Msg("Internal SSE server error")
		}
	}()

	// Wait a bit for internal server to start
	time.Sleep(500 * time.Millisecond)

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

	// Create HTTP server
	s.proxyServer = &http.Server{
		Addr:         s.externalAddr,
		Handler:      mux,
		ReadTimeout:  s.config.GetReadTimeout(),
		WriteTimeout: s.config.GetWriteTimeout(),
		IdleTimeout:  60 * time.Second,
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

// createAuthMiddleware creates authentication middleware using Bearer token.
func (s *SSEServerV2) createAuthMiddleware() func(http.HandlerFunc) http.HandlerFunc {
	expectedToken := s.config.GetAPIKey()

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
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
				s.logger.Warn().
					Str("remote_addr", r.RemoteAddr).
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Bool("has_auth_header", authHeader != "").
					Msg("Unauthorized access attempt")

				w.Header().Set("WWW-Authenticate", "Bearer realm=\"MCP Server PRTG\"")
				http.Error(w, "Unauthorized: Missing or invalid Bearer token", http.StatusUnauthorized)
				return
			}

			// Token valid, continue
			s.logger.Debug().
				Str("remote_addr", r.RemoteAddr).
				Str("path", r.URL.Path).
				Str("method", r.Method).
				Msg("Authenticated request")

			next(w, r)
		}
	}
}

// handleHealth handles health check requests.
func (s *SSEServerV2) handleHealth(w http.ResponseWriter, r *http.Request) {
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
		"version":"v2.0.0-alpha",
		"transport":"sse",
		"tls_enabled":%t,
		"base_url":"%s",
		"mcp_tools":6,
		"database":{
			"status":"%s",
			"error":"%s"
		},
		"timestamp":"%s"
	}`, s.config.IsTLSEnabled(), s.baseURL, dbStatus, dbError, time.Now().Format(time.RFC3339))

	fmt.Fprint(w, status)
}

// Shutdown gracefully shuts down the server.
func (s *SSEServerV2) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("Shutting down SSE server")

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
