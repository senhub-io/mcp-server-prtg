package admin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/senhub-io/mcp-server-prtg/internal/services/configuration"
)

type fakeStore struct {
	settings configuration.Settings
	applyErr error
	apiKey   string
}

func (f *fakeStore) Settings() configuration.Settings { return f.settings }

func (f *fakeStore) ApplySettings(s configuration.Settings) error {
	if f.applyErr != nil {
		return f.applyErr
	}
	f.settings = s
	return nil
}

func (f *fakeStore) GetAPIKey() string { return f.apiKey }

func testDeps(store Store) Deps {
	logger := zerolog.New(io.Discard)
	return Deps{
		Store: store,
		Status: func(ctx context.Context) any {
			return map[string]any{"version": "test", "uptime": "1h"}
		},
		TestDatabase: func(ctx context.Context, s configuration.DatabaseSettings) TestResult {
			return TestResult{OK: true, Message: "db ok", Details: []string{"connected"}, DurationMs: 7}
		},
		TestPRTG: func(ctx context.Context, s configuration.PRTGSettings) TestResult {
			return TestResult{OK: true, Message: "prtg ok", DurationMs: 3}
		},
		ClientConfig: func(host string) (string, string, error) {
			if host == "" {
				host = "localhost"
			}
			return "config text for " + host, host, nil
		},
		Restart: func() {},
		Logger:  &logger,
	}
}

func doReq(t *testing.T, h http.Handler, method, path, remote, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	r.RemoteAddr = remote
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func login(t *testing.T, h http.Handler) *http.Cookie {
	t.Helper()
	w := doReq(t, h, http.MethodPost, "/admin/api/login", "127.0.0.1:12345",
		`{"api_key":"secret-key"}`,
		map[string]string{"X-Requested-With": "mcp-admin", "Content-Type": "application/json"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			return c
		}
	}
	t.Fatal("no session cookie set")
	return nil
}

func authHeaders(c *http.Cookie) map[string]string {
	return map[string]string{
		"X-Requested-With": "mcp-admin",
		"Content-Type":     "application/json",
		"Cookie":           sessionCookie + "=" + c.Value,
	}
}

func newTestHandler(t *testing.T) (http.Handler, *fakeStore) {
	t.Helper()
	store := &fakeStore{apiKey: "secret-key"}
	return NewHandler(testDeps(store)), store
}

func TestNonLoopbackForbidden(t *testing.T) {
	h, _ := newTestHandler(t)
	w := doReq(t, h, http.MethodGet, "/admin/api/status", "10.0.0.5:1", "", nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAPIRequiresSession(t *testing.T) {
	h, _ := newTestHandler(t)
	w := doReq(t, h, http.MethodGet, "/admin/api/status", "127.0.0.1:12345", "", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginFailuresAndRateLimit(t *testing.T) {
	h, _ := newTestHandler(t)
	headers := map[string]string{"X-Requested-With": "mcp-admin", "Content-Type": "application/json"}
	for i := 0; i < 5; i++ {
		w := doReq(t, h, http.MethodPost, "/admin/api/login", "127.0.0.1:12345", `{"api_key":"wrong"}`, headers)
		assert.Equal(t, http.StatusUnauthorized, w.Code, "attempt %d", i+1)
	}
	w := doReq(t, h, http.MethodPost, "/admin/api/login", "127.0.0.1:12345", `{"api_key":"wrong"}`, headers)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestLoginSetsSecureCookie(t *testing.T) {
	h, _ := newTestHandler(t)
	c := login(t, h)
	assert.True(t, c.HttpOnly)
	assert.Equal(t, http.SameSiteStrictMode, c.SameSite)
	assert.Equal(t, "/admin", c.Path)
}

func TestSettingsGetHidesSecrets(t *testing.T) {
	h, store := newTestHandler(t)
	pw := "s3cret-db-password"
	tok := "s3cret-prtg-token"
	store.settings.Database.Password = &pw
	store.settings.PRTG.Token = &tok
	store.settings.Database.PasswordSet = true
	store.settings.PRTG.TokenSet = true
	c := login(t, h)
	w := doReq(t, h, http.MethodGet, "/admin/api/settings", "127.0.0.1:12345", "", map[string]string{"Cookie": sessionCookie + "=" + c.Value})
	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.NotContains(t, body, "s3cret-db-password")
	assert.NotContains(t, body, "s3cret-prtg-token")
	assert.Contains(t, body, `"password_set":true`)
	assert.Contains(t, body, `"token_set":true`)
}

func TestPutSettingsRequiresCSRFHeader(t *testing.T) {
	h, _ := newTestHandler(t)
	c := login(t, h)
	headers := map[string]string{"Content-Type": "application/json", "Cookie": sessionCookie + "=" + c.Value}
	w := doReq(t, h, http.MethodPut, "/admin/api/settings", "127.0.0.1:12345", `{"server_port":8080}`, headers)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestPutSettingsInvalid(t *testing.T) {
	h, store := newTestHandler(t)
	store.applyErr = fmt.Errorf("%w: server_port must be > 0", configuration.ErrInvalidSettings)
	c := login(t, h)
	w := doReq(t, h, http.MethodPut, "/admin/api/settings", "127.0.0.1:12345", `{"server_port":0}`, authHeaders(c))
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "server_port must be")
}

func TestDatabaseReturnsCallbackResult(t *testing.T) {
	h, _ := newTestHandler(t)
	c := login(t, h)
	w := doReq(t, h, http.MethodPost, "/admin/api/test/database", "127.0.0.1:12345", `{"host":"db"}`, authHeaders(c))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"ok":true`)
	assert.Contains(t, w.Body.String(), "db ok")
	assert.Contains(t, w.Body.String(), `"duration_ms":7`)
}

func TestClientConfig(t *testing.T) {
	h, _ := newTestHandler(t)
	c := login(t, h)
	w := doReq(t, h, http.MethodGet, "/admin/api/client-config?host=example.com", "127.0.0.1:12345", "",
		map[string]string{"Cookie": sessionCookie + "=" + c.Value})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "config text for example.com")
	assert.Contains(t, w.Body.String(), `"host":"example.com"`)
}

func TestRestart(t *testing.T) {
	ch := make(chan struct{}, 1)
	store := &fakeStore{apiKey: "secret-key"}
	d := testDeps(store)
	d.Restart = func() { ch <- struct{}{} }
	h := NewHandler(d)
	c := login(t, h)
	w := doReq(t, h, http.MethodPost, "/admin/api/restart", "127.0.0.1:12345", `{}`, authHeaders(c))
	require.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), `"restarting":true`)
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("Restart was not called")
	}
}

func TestSecurityHeaders(t *testing.T) {
	h, _ := newTestHandler(t)
	w := doReq(t, h, http.MethodGet, "/admin/", "127.0.0.1:12345", "", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "default-src 'self'; img-src 'self' data:; frame-ancestors 'none'; base-uri 'none'; form-action 'self'", w.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "no-referrer", w.Header().Get("Referrer-Policy"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	w = doReq(t, h, http.MethodGet, "/admin/api/status", "127.0.0.1:12345", "", nil)
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
}

func TestLogoutInvalidatesSession(t *testing.T) {
	h, _ := newTestHandler(t)
	c := login(t, h)
	w := doReq(t, h, http.MethodPost, "/admin/api/logout", "127.0.0.1:12345", `{}`, authHeaders(c))
	require.Equal(t, http.StatusNoContent, w.Code)
	w = doReq(t, h, http.MethodGet, "/admin/api/status", "127.0.0.1:12345", "",
		map[string]string{"Cookie": sessionCookie + "=" + c.Value})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSettingsKeepSecretFlags(t *testing.T) {
	got := sanitizeSettings(configuration.Settings{
		Database: configuration.DatabaseSettings{PasswordSet: true},
		PRTG:     configuration.PRTGSettings{TokenSet: true},
	})
	assert.True(t, got.Database.PasswordSet)
	assert.True(t, got.PRTG.TokenSet)
	assert.Nil(t, got.Database.Password)
	assert.Nil(t, got.PRTG.Token)
}

func TestProxiedRequestForbidden(t *testing.T) {
	h, _ := newTestHandler(t)
	w := doReq(t, h, http.MethodGet, "/admin/", "127.0.0.1:12345", "", map[string]string{"X-Forwarded-For": "203.0.113.7"})
	assert.Equal(t, http.StatusForbidden, w.Code)
}
