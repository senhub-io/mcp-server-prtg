package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/matthieu/mcp-server-prtg/internal/services/configuration"
	"github.com/matthieu/mcp-server-prtg/internal/services/logger"
	server "github.com/mark3labs/mcp-go/server"
)

// SSEServer wraps the MCP SSE server with authentication and TLS.
type SSEServer struct {
	mcpServer  *server.MCPServer
	sseServer  *server.SSEServer
	httpServer *http.Server
	config     *configuration.Configuration
	logger     *logger.ModuleLogger
	baseURL    string
}

// NewSSEServer creates a new SSE-based MCP server.
func NewSSEServer(mcpServer *server.MCPServer, config *configuration.Configuration, baseLogger *logger.Logger) *SSEServer {
	logger := logger.NewModuleLogger(baseLogger, logger.ModuleServer)

	// Determine base URL for SSE
	protocol := "http"
	if config.IsTLSEnabled() {
		protocol = "https"
	}

	baseURL := fmt.Sprintf("%s://%s", protocol, config.GetServerAddress())

	return &SSEServer{
		mcpServer: mcpServer,
		config:    config,
		logger:    logger,
		baseURL:   baseURL,
	}
}

// Start starts the SSE server.
func (s *SSEServer) Start(ctx context.Context) error {
	s.logger.Info().
		Str("address", s.config.GetServerAddress()).
		Bool("tls", s.config.IsTLSEnabled()).
		Msg("Starting MCP Server with SSE transport")

	// Create SSE server from mcp-go
	s.sseServer = server.NewSSEServer(s.mcpServer, s.baseURL)

	// Create HTTP server with authentication middleware
	mux := http.NewServeMux()

	// Wrap SSE endpoints with authentication
	authMiddleware := s.createAuthMiddleware()

	// Register SSE endpoints
	// GET /sse - SSE stream endpoint
	mux.HandleFunc("/sse", authMiddleware(s.handleSSE))

	// POST /message - Message endpoint
	mux.HandleFunc("/message", authMiddleware(s.handleMessage))

	// Health check endpoint (no auth required)
	mux.HandleFunc("/health", s.handleHealth)

	// Status endpoint (auth required)
	mux.HandleFunc("/status", authMiddleware(s.handleStatus))

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         s.config.GetServerAddress(),
		Handler:      mux,
		ReadTimeout:  s.config.GetReadTimeout(),
		WriteTimeout: s.config.GetWriteTimeout(),
		IdleTimeout:  60 * time.Second,
	}

	// Start server (with or without TLS)
	go s.startServer()

	// Log startup information
	s.logStartupInfo()

	return nil
}

// startServer starts the HTTP server in a goroutine.
func (s *SSEServer) startServer() {
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

		s.httpServer.TLSConfig = tlsConfig

		s.logger.Info().
			Str("cert", certFile).
			Str("key", keyFile).
			Msg("Starting HTTPS server")

		err = s.httpServer.ListenAndServeTLS(certFile, keyFile)
	} else {
		s.logger.Warn().Msg("Starting HTTP server (TLS disabled - not recommended for production)")
		err = s.httpServer.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		s.logger.Error().Err(err).Msg("HTTP server error")
	}
}

// createAuthMiddleware creates authentication middleware.
func (s *SSEServer) createAuthMiddleware() func(http.HandlerFunc) http.HandlerFunc {
	apiKey := s.config.GetAPIKey()

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Extract API key from header
			providedKey := r.Header.Get("X-API-Key")

			// Validate API key
			if providedKey != apiKey {
				s.logger.Warn().
					Str("remote_addr", r.RemoteAddr).
					Str("path", r.URL.Path).
					Msg("Unauthorized access attempt")

				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// API key valid, continue
			next(w, r)
		}
	}
}

// handleSSE handles the SSE endpoint.
func (s *SSEServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	s.logger.Debug().
		Str("remote_addr", r.RemoteAddr).
		Msg("SSE connection established")

	// Delegate to mcp-go SSE server
	// This needs to be connected to the internal SSE handler
	// For now, we'll use a custom implementation
	s.serveSSE(w, r)
}

// handleMessage handles the message POST endpoint.
func (s *SSEServer) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.logger.Debug().
		Str("remote_addr", r.RemoteAddr).
		Msg("Received MCP message")

	// Delegate to mcp-go SSE server
	s.handleRPCMessage(w, r)
}

// handleHealth handles health check requests.
func (s *SSEServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

// handleStatus handles status requests (authenticated).
func (s *SSEServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	status := fmt.Sprintf(`{
		"status":"running",
		"version":"v1.0.0",
		"transport":"sse",
		"tls_enabled":%t,
		"base_url":"%s",
		"timestamp":"%s"
	}`, s.config.IsTLSEnabled(), s.baseURL, time.Now().Format(time.RFC3339))

	fmt.Fprint(w, status)
}

// Shutdown gracefully shuts down the server.
func (s *SSEServer) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("Shutting down SSE server")

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("HTTP server shutdown error: %w", err)
		}
	}

	if s.sseServer != nil {
		if err := s.sseServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("SSE server shutdown error: %w", err)
		}
	}

	s.logger.Info().Msg("SSE server shut down successfully")

	return nil
}

// logStartupInfo logs startup information.
func (s *SSEServer) logStartupInfo() {
	protocol := "HTTP"
	if s.config.IsTLSEnabled() {
		protocol = "HTTPS"
	}

	s.logger.Info().Msgf(`
╔════════════════════════════════════════════════════════════════╗
║            MCP Server PRTG - SSE Transport Started             ║
╠════════════════════════════════════════════════════════════════╣
║  Protocol:     %s                                           ║
║  Address:      %s                                    ║
║  Base URL:     %s                        ║
║  Endpoints:                                                    ║
║    - GET  /sse      (SSE stream - authenticated)              ║
║    - POST /message  (RPC messages - authenticated)            ║
║    - GET  /health   (Health check - public)                   ║
║    - GET  /status   (Server status - authenticated)           ║
╠════════════════════════════════════════════════════════════════╣
║  Authentication: X-API-Key header required                     ║
║  API Key: %s                                  ║
╚════════════════════════════════════════════════════════════════╝
	`, protocol, s.config.GetServerAddress(), s.baseURL, maskAPIKey(s.config.GetAPIKey()))
}

// maskAPIKey masks the API key for logging.
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}

	return key[:4] + "..." + key[len(key)-4:]
}

// serveSSE serves the SSE stream (custom implementation).
func (s *SSEServer) serveSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// For now, we'll use a simple implementation
	// In production, this should integrate with mcp-go's SSE implementation
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send initial endpoint event
	sessionID := generateSessionID()
	messageURL := fmt.Sprintf("%s/message?sessionId=%s", s.baseURL, sessionID)

	fmt.Fprintf(w, "event: endpoint\n")
	fmt.Fprintf(w, "data: %s\n\n", messageURL)
	flusher.Flush()

	// Keep connection alive
	<-r.Context().Done()

	s.logger.Debug().Str("session", sessionID).Msg("SSE connection closed")
}

// handleRPCMessage handles RPC messages (custom implementation).
func (s *SSEServer) handleRPCMessage(w http.ResponseWriter, r *http.Request) {
	// This needs to integrate with mcp-go's message handling
	// For now, return a simple response

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":{"status":"received"}}`)
}

// generateSessionID generates a unique session ID.
func generateSessionID() string {
	// Simple implementation - should use UUID in production
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}
