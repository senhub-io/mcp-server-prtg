package prtg

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newGuardTestClient(t *testing.T, srv *httptest.Server, mutate func(*ClientConfig)) *Client {
	t.Helper()
	cfg := ClientConfig{
		BaseURL:   srv.URL,
		Token:     "t",
		Timeout:   5 * time.Second,
		VerifySSL: true,
		Logger:    nil,
	}
	nop := zerolog.Nop()
	cfg.Logger = &nop
	if mutate != nil {
		mutate(&cfg)
	}
	client, err := NewClient(cfg)
	require.NoError(t, err)
	return client
}

func TestGuardCacheHitWithinTTL(t *testing.T) {
	var requests atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	client := newGuardTestClient(t, srv, nil)

	for range 2 {
		err := client.doRequest(context.Background(), "GET", "/api/v2/devices/1", nil, nil)
		require.NoError(t, err)
	}

	require.EqualValues(t, 1, requests.Load())
	stats := client.Stats()
	assert.EqualValues(t, 1, stats.UpstreamRequests)
	assert.EqualValues(t, 1, stats.CacheHits)
}

func TestGuardCacheExpires(t *testing.T) {
	var requests atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	client := newGuardTestClient(t, srv, nil)

	current := time.Unix(1000, 0)
	client.guard.now = func() time.Time { return current }

	require.NoError(t, client.doRequest(context.Background(), "GET", "/api/v2/sensors/1", nil, nil))
	current = current.Add(31 * time.Second)
	require.NoError(t, client.doRequest(context.Background(), "GET", "/api/v2/sensors/1", nil, nil))

	require.EqualValues(t, 2, requests.Load())
	stats := client.Stats()
	assert.EqualValues(t, 2, stats.UpstreamRequests)
	assert.EqualValues(t, 0, stats.CacheHits)
}

func TestGuardCoalescesConcurrentIdenticalRequests(t *testing.T) {
	var requests atomic.Int64
	var inFlight atomic.Int64
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		inFlight.Add(1)
		<-release
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	client := newGuardTestClient(t, srv, nil)

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := client.doRequest(context.Background(), "GET", "/api/v2/devices/1", nil, nil)
			assert.NoError(t, err)
		}()
	}

	require.Eventually(t, func() bool { return inFlight.Load() == 1 }, 2*time.Second, 5*time.Millisecond)
	close(release)
	wg.Wait()

	require.EqualValues(t, 1, requests.Load())
	stats := client.Stats()
	assert.EqualValues(t, 1, stats.UpstreamRequests)
	assert.EqualValues(t, 9, stats.Coalesced)
	assert.EqualValues(t, 0, stats.CacheHits)
}

func TestGuardMaxConcurrent(t *testing.T) {
	var inFlight atomic.Int64
	var maxInFlight atomic.Int64
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := inFlight.Add(1)
		for {
			m := maxInFlight.Load()
			if cur <= m || maxInFlight.CompareAndSwap(m, cur) {
				break
			}
		}
		<-release
		inFlight.Add(-1)
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	client := newGuardTestClient(t, srv, func(cfg *ClientConfig) { cfg.MaxConcurrent = 2 })

	var wg sync.WaitGroup
	for i := range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := client.doRequest(context.Background(), "GET", fmt.Sprintf("/api/v2/other/%d", i), nil, nil)
			assert.NoError(t, err)
		}()
	}

	require.Eventually(t, func() bool { return inFlight.Load() == 2 }, 2*time.Second, 5*time.Millisecond)
	close(release)
	wg.Wait()

	assert.EqualValues(t, 2, maxInFlight.Load())
}

func TestGuardDoesNotCacheErrors(t *testing.T) {
	var requests atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "boom")
			return
		}
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	client := newGuardTestClient(t, srv, nil)

	err := client.doRequest(context.Background(), "GET", "/api/v2/devices/1", nil, nil)
	require.Error(t, err)
	require.NoError(t, client.doRequest(context.Background(), "GET", "/api/v2/devices/1", nil, nil))

	require.EqualValues(t, 2, requests.Load())
	stats := client.Stats()
	assert.EqualValues(t, 2, stats.UpstreamRequests)
	assert.EqualValues(t, 0, stats.CacheHits)
}

func TestGuardCallerCancelDoesNotFailOthers(t *testing.T) {
	var inFlight atomic.Int64
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inFlight.Add(1)
		<-release
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	client := newGuardTestClient(t, srv, nil)

	ctxA, cancel := context.WithCancel(context.Background())
	errChA := make(chan error, 1)
	go func() {
		errChA <- client.doRequest(ctxA, "GET", "/api/v2/health", nil, nil)
	}()

	require.Eventually(t, func() bool { return inFlight.Load() == 1 }, 2*time.Second, 5*time.Millisecond)

	errChB := make(chan error, 1)
	go func() {
		errChB <- client.doRequest(context.Background(), "GET", "/api/v2/health", nil, nil)
	}()

	cancel()
	require.ErrorIs(t, <-errChA, context.Canceled)

	close(release)
	require.NoError(t, <-errChB)
}

func TestCacheTTL(t *testing.T) {
	cases := []struct {
		endpoint string
		want     time.Duration
	}{
		{"/api/v2/timeseries/sensor/1/live?range=1h", 30 * time.Second},
		{"/api/v2/timeseries/sensor/1", 5 * time.Minute},
		{"/api/v2/experimental/channels?sensor=1", time.Minute},
		{"/api/v2/experimental/sensors", 30 * time.Second},
		{"/api/v2/sensors/123", 30 * time.Second},
		{"/api/v2/sensor-status-summary", 30 * time.Second},
		{"/api/v2/objects/count", 30 * time.Second},
		{"/api/v2/devices/9", time.Minute},
		{"/api/v2/experimental/groups", 5 * time.Minute},
		{"/api/v2/experimental/objects", 5 * time.Minute},
		{"/api/v2/health", 0},
		{"/api/v2/unknown?x=1", 0},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.want, cacheTTL(tc.endpoint), tc.endpoint)
	}
}

func TestGuardCacheDisabled(t *testing.T) {
	var requests atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	client := newGuardTestClient(t, srv, func(cfg *ClientConfig) { cfg.CacheDisabled = true })

	for range 2 {
		require.NoError(t, client.doRequest(context.Background(), "GET", "/api/v2/devices/1", nil, nil))
	}

	require.EqualValues(t, 2, requests.Load())
	stats := client.Stats()
	assert.EqualValues(t, 2, stats.UpstreamRequests)
	assert.EqualValues(t, 0, stats.CacheHits)
}
