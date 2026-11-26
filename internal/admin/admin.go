package admin

import (
	"context"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/senhub-io/mcp-server-prtg/internal/services/configuration"
)

//go:embed web/index.html web/admin.js web/console.css
var webFS embed.FS

const (
	maxBodyBytes  = 64 * 1024
	sessionCookie = "mcp_admin_session"
	csrfHeader    = "X-Requested-With"
	csrfValue     = "mcp-admin"
	testTimeout   = 20 * time.Second
)

var (
	indexHTML  = mustRead("web/index.html")
	adminJS    = mustRead("web/admin.js")
	consoleCSS = mustRead("web/console.css")
)

func mustRead(name string) []byte {
	b, err := webFS.ReadFile(name)
	if err != nil {
		return []byte{}
	}
	return b
}

type handler struct {
	d        Deps
	sessions *sessionStore
	limiter  *loginLimiter
}

func NewHandler(d Deps) http.Handler {
	h := &handler{
		d:        d,
		sessions: newSessionStore(),
		limiter:  newLoginLimiter(),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /admin", h.index)
	mux.HandleFunc("GET /admin/", h.index)
	mux.HandleFunc("GET /admin/static/", h.static)
	mux.HandleFunc("GET /admin/api/session", h.getSession)
	mux.HandleFunc("POST /admin/api/login", h.login)
	mux.HandleFunc("POST /admin/api/logout", h.logout)
	mux.HandleFunc("GET /admin/api/status", h.status)
	mux.HandleFunc("GET /admin/api/settings", h.getSettings)
	mux.HandleFunc("PUT /admin/api/settings", h.putSettings)
	mux.HandleFunc("POST /admin/api/test/database", h.testDatabase)
	mux.HandleFunc("POST /admin/api/test/prtg", h.testPRTG)
	mux.HandleFunc("GET /admin/api/client-config", h.clientConfig)
	mux.HandleFunc("POST /admin/api/restart", h.restart)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.setSecurityHeaders(w, r)
		if !isLoopback(r.RemoteAddr) || isProxied(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

// isProxied rejects requests relayed by a local proxy: the loopback check
// would otherwise accept a remote client forwarded through it.
func isProxied(r *http.Request) bool {
	for _, h := range []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Real-Ip"} {
		if r.Header.Get(h) != "" {
			return true
		}
	}

	return false
}

func isLoopback(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (h *handler) setSecurityHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	if strings.HasPrefix(r.URL.Path, "/admin/api/") {
		w.Header().Set("Cache-Control", "no-store")
	}
}

func (h *handler) index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(indexHTML)
}

func (h *handler) static(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/admin/static/")
	var body []byte
	var ctype string
	switch name {
	case "admin.js":
		body, ctype = adminJS, "application/javascript; charset=utf-8"
	case "console.css":
		body, ctype = consoleCSS, "text/css; charset=utf-8"
	default:
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", ctype)
	_, _ = w.Write(body)
}

func (h *handler) requireSession(w http.ResponseWriter, r *http.Request) bool {
	c, err := r.Cookie(sessionCookie)
	if err != nil || !h.sessions.validate(c.Value) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return false
	}
	return true
}

func checkCSRF(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get(csrfHeader) != csrfValue {
		writeError(w, http.StatusForbidden, "missing X-Requested-With header")
		return false
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, http.StatusForbidden, "Content-Type must be application/json")
		return false
	}
	return true
}

func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

type loginRequest struct {
	APIKey string `json:"api_key"`
}

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	if !checkCSRF(w, r) {
		return
	}
	var req loginRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if !h.limiter.allow() {
		h.warn(r, "login rate limit exceeded", nil)
		writeError(w, http.StatusTooManyRequests, "too many failed login attempts")
		return
	}
	key := h.d.Store.GetAPIKey()
	if subtle.ConstantTimeCompare([]byte(req.APIKey), []byte(key)) != 1 {
		h.limiter.recordFailure()
		h.warn(r, "failed login attempt", nil)
		writeError(w, http.StatusUnauthorized, "invalid api key")
		return
	}
	h.limiter.reset()
	id, err := h.sessions.create()
	if err != nil {
		h.warn(r, "failed to create session", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    id,
		Path:     "/admin",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   r.TLS != nil,
		Expires:  time.Now().Add(sessionAbsoluteTTL),
	})
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": true})
}

func (h *handler) getSession(w http.ResponseWriter, r *http.Request) {
	if !h.requireSession(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": true})
}

func (h *handler) logout(w http.ResponseWriter, r *http.Request) {
	if !checkCSRF(w, r) {
		return
	}
	c, err := r.Cookie(sessionCookie)
	if err != nil || !h.sessions.validate(c.Value) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	h.sessions.delete(c.Value)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/admin",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   r.TLS != nil,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) status(w http.ResponseWriter, r *http.Request) {
	if !h.requireSession(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, h.d.Status(r.Context()))
}

func sanitizeSettings(s configuration.Settings) configuration.Settings {
	s.Database.Password = nil
	s.PRTG.Token = nil

	return s
}

func (h *handler) getSettings(w http.ResponseWriter, r *http.Request) {
	if !h.requireSession(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, sanitizeSettings(h.d.Store.Settings()))
}

func (h *handler) putSettings(w http.ResponseWriter, r *http.Request) {
	if !checkCSRF(w, r) {
		return
	}
	if !h.requireSession(w, r) {
		return
	}
	var s configuration.Settings
	if !decodeBody(w, r, &s) {
		return
	}
	if err := h.d.Store.ApplySettings(s); err != nil {
		if errors.Is(err, configuration.ErrInvalidSettings) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.warn(r, "failed to apply settings", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"restart_required": true})
}

func (h *handler) testDatabase(w http.ResponseWriter, r *http.Request) {
	if !checkCSRF(w, r) {
		return
	}
	if !h.requireSession(w, r) {
		return
	}
	var s configuration.DatabaseSettings
	if !decodeBody(w, r, &s) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), testTimeout)
	defer cancel()
	start := time.Now()
	res := h.d.TestDatabase(ctx, s)
	if res.DurationMs == 0 {
		res.DurationMs = time.Since(start).Milliseconds()
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *handler) testPRTG(w http.ResponseWriter, r *http.Request) {
	if !checkCSRF(w, r) {
		return
	}
	if !h.requireSession(w, r) {
		return
	}
	var s configuration.PRTGSettings
	if !decodeBody(w, r, &s) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), testTimeout)
	defer cancel()
	start := time.Now()
	res := h.d.TestPRTG(ctx, s)
	if res.DurationMs == 0 {
		res.DurationMs = time.Since(start).Milliseconds()
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *handler) clientConfig(w http.ResponseWriter, r *http.Request) {
	if !h.requireSession(w, r) {
		return
	}
	host := r.URL.Query().Get("host")
	text, resolved, err := h.d.ClientConfig(host)
	if err != nil {
		h.warn(r, "failed to build client config", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"text": text, "host": resolved})
}

func (h *handler) restart(w http.ResponseWriter, r *http.Request) {
	if !checkCSRF(w, r) {
		return
	}
	if !h.requireSession(w, r) {
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"restarting": true})
	time.AfterFunc(time.Second, h.d.Restart)
}

func (h *handler) warn(r *http.Request, msg string, err error) {
	if h.d.Logger == nil {
		return
	}
	e := h.d.Logger.Warn().Str("path", r.URL.Path).Str("remote", r.RemoteAddr)
	if err != nil {
		e = e.Err(err)
	}
	e.Msg(msg)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
