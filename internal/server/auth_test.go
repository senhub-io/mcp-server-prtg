package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/senhub-io/mcp-server-prtg/internal/services/logger"
)

const testAPIKey = "0123456789abcdef0123456789abcdef"

func newTestHandler(next http.Handler, key string) http.Handler {
	log := logger.NewModuleLogger(logger.NewSilentLogger(), logger.ModuleServer)

	return createAuthMiddleware(next, func() string { return key }, newAuthRateLimiter(), log)
}

func doRequest(handler http.Handler, remoteAddr, authHeader, rawQuery string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/mcp"+rawQuery, nil)
	req.RemoteAddr = remoteAddr
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return rec
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestAuthMiddlewareValidBearer(t *testing.T) {
	handler := newTestHandler(okHandler(), testAPIKey)

	rec := doRequest(handler, "1.2.3.4:5678", "Bearer "+testAPIKey, "")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthMiddlewareCaseInsensitiveBearer(t *testing.T) {
	handler := newTestHandler(okHandler(), testAPIKey)

	rec := doRequest(handler, "1.2.3.4:5678", "bEaReR "+testAPIKey, "")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthMiddlewareWrongToken(t *testing.T) {
	handler := newTestHandler(okHandler(), testAPIKey)

	rec := doRequest(handler, "1.2.3.4:5678", "Bearer wrong-token-wrong", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddlewareMissingHeader(t *testing.T) {
	handler := newTestHandler(okHandler(), testAPIKey)

	rec := doRequest(handler, "1.2.3.4:5678", "", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddlewareQueryTokenRejected(t *testing.T) {
	handler := newTestHandler(okHandler(), testAPIKey)

	rec := doRequest(handler, "1.2.3.4:5678", "", "?token="+testAPIKey)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddlewareEmptyExpectedKey(t *testing.T) {
	handler := newTestHandler(okHandler(), "")

	rec := doRequest(handler, "1.2.3.4:5678", "Bearer ", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	rec = doRequest(handler, "1.2.3.4:5678", "Bearer "+testAPIKey, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddlewareRateLimitLockout(t *testing.T) {
	log := logger.NewModuleLogger(logger.NewSilentLogger(), logger.ModuleServer)
	rl := newAuthRateLimiter()

	handler := createAuthMiddleware(okHandler(), func() string { return testAPIKey }, rl, log)

	const remoteAddr = "9.9.9.9:1234"

	for i := 0; i < 6; i++ {
		rec := doRequest(handler, remoteAddr, "Bearer wrong-token-wrong", "")
		want := http.StatusUnauthorized
		if i == 5 {
			want = http.StatusTooManyRequests
		}

		assert.Equal(t, want, rec.Code, "attempt %d", i+1)
	}

	rec := doRequest(handler, remoteAddr, "Bearer "+testAPIKey, "")
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestAuthMiddlewareSpoofedHeadersIgnored(t *testing.T) {
	log := logger.NewModuleLogger(logger.NewSilentLogger(), logger.ModuleServer)
	rl := newAuthRateLimiter()

	handler := createAuthMiddleware(okHandler(), func() string { return testAPIKey }, rl, log)

	const remoteAddr = "9.9.9.9:1234"

	for i := 0; i < 6; i++ {
		req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
		req.RemoteAddr = remoteAddr
		req.Header.Set("X-Forwarded-For", "1.1.1.1")
		req.Header.Set("X-Real-IP", "2.2.2.2")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		want := http.StatusUnauthorized
		if i == 5 {
			want = http.StatusTooManyRequests
		}

		assert.Equal(t, want, rec.Code, "attempt %d", i+1)
	}

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.RemoteAddr = remoteAddr
	req.Header.Set("X-Forwarded-For", "3.3.3.3")
	req.Header.Set("Authorization", "Bearer "+testAPIKey)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	req = httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.RemoteAddr = "8.8.8.8:9999"
	req.Header.Set("X-Forwarded-For", "9.9.9.9")
	req.Header.Set("Authorization", "Bearer "+testAPIKey)

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthMiddlewareKeyRotation(t *testing.T) {
	log := logger.NewModuleLogger(logger.NewSilentLogger(), logger.ModuleServer)
	rl := newAuthRateLimiter()

	key := testAPIKey
	handler := createAuthMiddleware(okHandler(), func() string { return key }, rl, log)

	rec := doRequest(handler, "7.7.7.7:1", "Bearer "+testAPIKey, "")
	assert.Equal(t, http.StatusOK, rec.Code)

	newKey := "rotated-secret-key-0000"
	key = newKey

	rec = doRequest(handler, "7.7.7.7:1", "Bearer "+testAPIKey, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	rec = doRequest(handler, "7.7.7.7:1", "Bearer "+newKey, "")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthRateLimiterMapCapped(t *testing.T) {
	rl := newAuthRateLimiter()
	rl.maxEntries = 10
	rl.window = time.Hour

	for i := 0; i < 50; i++ {
		ip := "10.0." + strconvItoa(i/256) + "." + strconvItoa(i%256)
		rl.recordFailure(ip)
	}

	rl.mu.Lock()
	size := len(rl.attempts)
	rl.mu.Unlock()

	require.LessOrEqual(t, size, 10)
}

func strconvItoa(n int) string {
	if n == 0 {
		return "0"
	}

	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}

	return digits
}

func TestLockedEntrySurvivesMapSaturation(t *testing.T) {
	rl := &authRateLimiter{
		attempts:    make(map[string]*authAttempt),
		maxAttempts: 5,
		window:      time.Minute,
		lockoutTime: 5 * time.Minute,
		maxEntries:  3,
	}

	target := "10.0.0.1"
	for i := 0; i < rl.maxAttempts+1; i++ {
		rl.recordFailure(target)
	}
	require.True(t, rl.isLocked(target))

	for i := 0; i < 10; i++ {
		locked := rl.recordFailure(fmt.Sprintf("192.0.2.%d", i+1))
		assert.False(t, locked)
	}

	assert.True(t, rl.isLocked(target))
}

func TestIPv6SamePrefixSharesKey(t *testing.T) {
	handler := newTestHandler(okHandler(), testAPIKey)

	for i := 0; i < 6; i++ {
		doRequest(handler, "[2001:db8:1:2::1]:1234", "", "")
	}

	rec := doRequest(handler, "[2001:db8:1:2::ffff]:5678", "", "")
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	rec = doRequest(handler, "[2001:db8:1:3::1]:4321", "Bearer "+testAPIKey, "")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestLimiterKey(t *testing.T) {
	assert.Equal(t, "192.0.2.10", limiterKey("192.0.2.10"))
	assert.Equal(t, "2001:db8:1:2::/64", limiterKey("2001:db8:1:2::dead:beef"))
	assert.Equal(t, "not-an-ip", limiterKey("not-an-ip"))
}
