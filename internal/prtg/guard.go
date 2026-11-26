package prtg

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	defaultMaxConcurrent   = 2
	defaultMaxCacheEntries = 500
	defaultFlightTimeout   = 60 * time.Second
)

type cacheEntry struct {
	body    []byte
	expires time.Time
}

type requestGuard struct {
	maxConcurrent   int
	maxCacheEntries int
	cacheDisabled   bool
	flightTimeout   time.Duration

	sem   chan struct{}
	sf    singleflight.Group
	mu    sync.Mutex
	cache map[string]cacheEntry

	// injectable so tests can move time forward
	now func() time.Time

	upstreamRequests atomic.Int64
	cacheHits        atomic.Int64
	coalesced        atomic.Int64
}

func newRequestGuard(maxConcurrent int, cacheDisabled bool, requestTimeout time.Duration) *requestGuard {
	if maxConcurrent <= 0 {
		maxConcurrent = defaultMaxConcurrent
	}
	flightTimeout := 2 * requestTimeout
	if flightTimeout <= 0 {
		flightTimeout = defaultFlightTimeout
	}

	return &requestGuard{
		maxConcurrent:   maxConcurrent,
		flightTimeout:   flightTimeout,
		maxCacheEntries: defaultMaxCacheEntries,
		cacheDisabled:   cacheDisabled,
		sem:             make(chan struct{}, maxConcurrent),
		cache:           make(map[string]cacheEntry),
		now:             time.Now,
	}
}

func (g *requestGuard) do(ctx context.Context, key string, ttl time.Duration, fetch func(context.Context) ([]byte, error)) ([]byte, error) {
	if g.cacheDisabled {
		ttl = 0
	}

	if ttl > 0 {
		g.mu.Lock()
		entry, ok := g.cache[key]
		g.mu.Unlock()
		if ok && g.now().Before(entry.expires) {
			g.cacheHits.Add(1)
			return entry.body, nil
		}
	}

	ran := false
	ch := g.sf.DoChan(key, func() (interface{}, error) {
		ran = true

		// detached from the first caller so one cancellation cannot fail the others
		dctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), g.flightTimeout)
		defer cancel()

		select {
		case g.sem <- struct{}{}:
		case <-dctx.Done():
			return nil, dctx.Err()
		}
		defer func() { <-g.sem }()

		g.upstreamRequests.Add(1)

		body, err := fetch(dctx)
		if err != nil {
			return nil, err
		}
		if ttl > 0 {
			g.store(key, body, ttl)
		}
		return body, nil
	})

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("request guard: %w", ctx.Err())
	case r := <-ch:
		if r.Err != nil {
			return nil, r.Err
		}
		if !ran {
			g.coalesced.Add(1)
		}
		body, ok := r.Val.([]byte)
		if !ok {
			return nil, fmt.Errorf("request guard: unexpected result type %T", r.Val)
		}

		return body, nil
	}
}

func (g *requestGuard) store(key string, body []byte, ttl time.Duration) {
	now := g.now()

	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.cache) >= g.maxCacheEntries {
		g.evictLocked(now)
	}
	g.cache[key] = cacheEntry{body: body, expires: now.Add(ttl)}
}

func (g *requestGuard) evictLocked(now time.Time) {
	for k, e := range g.cache {
		if !now.Before(e.expires) {
			delete(g.cache, k)
		}
	}
	if len(g.cache) >= g.maxCacheEntries {
		soonest := ""
		var min time.Time
		for k, e := range g.cache {
			if soonest == "" || e.expires.Before(min) {
				soonest = k
				min = e.expires
			}
		}
		if soonest != "" {
			delete(g.cache, soonest)
		}
	}
}

// GuardStats reports request-guard counters.
type GuardStats struct {
	UpstreamRequests int64
	CacheHits        int64
	Coalesced        int64
}

func (g *requestGuard) stats() GuardStats {
	return GuardStats{
		UpstreamRequests: g.upstreamRequests.Load(),
		CacheHits:        g.cacheHits.Load(),
		Coalesced:        g.coalesced.Load(),
	}
}

// Stats returns the request-guard counters for this client.
func (c *Client) Stats() GuardStats {
	return c.guard.stats()
}

func cacheTTL(endpoint string) time.Duration {
	path := endpoint
	if i := strings.IndexAny(path, "?#"); i >= 0 {
		path = path[:i]
	}

	switch {
	case strings.Contains(path, "/timeseries/"):
		if strings.HasSuffix(path, "/live") {
			return 30 * time.Second
		}
		return 5 * time.Minute
	case strings.Contains(path, "/experimental/channels"):
		return time.Minute
	case strings.Contains(path, "/experimental/sensors"),
		strings.Contains(path, "/sensors/"),
		strings.Contains(path, "/sensor-status-summary"),
		strings.Contains(path, "/objects/count"):
		return 30 * time.Second
	case strings.Contains(path, "/devices/"):
		return time.Minute
	case strings.Contains(path, "/experimental/groups"),
		strings.Contains(path, "/experimental/objects"):
		return 5 * time.Minute
	default:
		return 0
	}
}
