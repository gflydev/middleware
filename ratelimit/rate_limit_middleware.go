package middleware

import (
	"fmt"
	"github.com/gflydev/cache"
	"github.com/gflydev/core"
	"github.com/gflydev/core/log"
	"github.com/gflydev/core/utils"
	"github.com/gflydev/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// keyedMutex serializes the read-modify-write of the counter for a given key.
//
// The cache interface only exposes Get/Set/Del (no atomic increment), so
// concurrent requests sharing a key could otherwise all read the same count and
// slip past the limit. This mutex closes that window within a single process.
// Distributed deployments still need an atomic counter in the shared cache to be
// fully correct.
var keyedMutex sync.Map // map[string]*sync.Mutex

func lockKey(key string) func() {
	m, _ := keyedMutex.LoadOrStore(key, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// rateLimitResult captures the outcome of a rate-limit check so callers can set
// the appropriate response headers.
type rateLimitResult struct {
	limited    bool // whether the request exceeds the limit
	limit      int  // configured maximum requests per window
	remaining  int  // requests still allowed in the current window (never negative)
	retryAfter int  // seconds until the window resets
}

// RateLimit creates rate-limiting middleware using configuration from .env file
// It reads RATE_LIMIT_MAX_REQUESTS and RATE_LIMIT_WINDOW_MINUTES from environment variables
//
// Parameters:
//   - excludes (...string): Optional paths to exclude from rate limiting
//
// Returns:
//   - core.MiddlewareHandler: A middleware handler function
func RateLimit(excludes ...string) core.MiddlewareHandler {
	// Read configuration from environment variables
	maxRequests := utils.Getenv("RATE_LIMIT_MAX_REQUESTS", 5)
	windowMinutes := utils.Getenv("RATE_LIMIT_WINDOW_MINUTES", 15)

	return RateLimitParam(maxRequests, windowMinutes, excludes...)
}

// RateLimitParam implements rate limiting to prevent abuse and race conditions
// It limits the number of requests per IP address within a time window
//
// Parameters:
//   - maxRequests (int): Maximum number of requests allowed per time window
//   - windowMinutes (int): Time window in minutes
//   - excludes (...string): Optional paths to exclude from rate limiting
//
// Returns:
//   - core.MiddlewareHandler: A middleware handler function
func RateLimitParam(maxRequests, windowMinutes int, excludes ...string) core.MiddlewareHandler {
	log.Tracef("Initial rate limiting configured: %d requests per %d minutes", maxRequests, windowMinutes)

	return func(c *core.Ctx) error {
		path := c.Path()

		// Skip rate limiting for excluded paths
		for _, exclude := range excludes {
			if exclude == path {
				log.Debugf("Skip rate limiting for %v", path)
				return nil
			}
		}

		return enforce(c, maxRequests, windowMinutes)
	}
}

// RateLimitRoute applies rate limiting to a specific controller/route
// This middleware can be applied to individual routes for controller-specific rate limiting
//
// Use:
//
//	groupUsers.POST("/users", f.Middleware(middleware.RateLimitRoute)(user.NewCreateUserApi()))
//	groupAuth.POST("/signin", f.Middleware(middleware.RateLimitRoute)(auth.NewSignInApi()))
func RateLimitRoute(c *core.Ctx) error {
	// Read configuration from environment variables
	maxRequests := utils.Getenv("RATE_LIMIT_MAX_REQUESTS", 5)
	windowMinutes := utils.Getenv("RATE_LIMIT_WINDOW_MINUTES", 15)

	return enforce(c, maxRequests, windowMinutes)
}

// RateLimitParamsRoute applies rate limiting to a specific controller with custom parameters
// This allows for different rate limits per controller
//
// Use:
//
//	groupUsers.POST("/users", f.Middleware(middleware.RateLimitParamsRoute(10, 5))(user.NewCreateUserApi()))
//	groupAuth.POST("/signin", f.Middleware(middleware.RateLimitParamsRoute(3, 15))(auth.NewSignInApi()))
func RateLimitParamsRoute(maxRequests, windowMinutes int) func(*core.Ctx) error {
	return func(c *core.Ctx) error {
		return enforce(c, maxRequests, windowMinutes)
	}
}

// enforce contains the common rate limiting logic: it derives the cache key from
// the client IP and path, checks the limit, sets the standard rate-limit headers,
// and returns a 429 error when the limit is exceeded.
func enforce(c *core.Ctx, maxRequests, windowMinutes int) error {
	// Get client IP address
	clientIP := c.ClientIP()
	if clientIP == "" {
		clientIP = "unknown"
	}

	// Create a unique key for this IP and path combination
	path := c.Path()
	key := fmt.Sprintf("%s:%s", clientIP, path)

	log.Tracef("Check rate limiting key: %s with limit %d requests per %d minutes", key, maxRequests, windowMinutes)

	res := checkRateLimit(key, maxRequests, windowMinutes)

	// Expose standard rate-limit headers so clients can back off gracefully.
	c.SetHeader("X-RateLimit-Limit", strconv.Itoa(res.limit))
	c.SetHeader("X-RateLimit-Remaining", strconv.Itoa(res.remaining))

	if res.limited {
		c.SetHeader("Retry-After", strconv.Itoa(res.retryAfter))
		log.Warnf("Rate limit exceeded for IP %s on path %s", clientIP, path)

		return c.Error(http.Error{
			Message: "Too many requests. Please try again later.",
		}, core.StatusTooManyRequests)
	}

	return nil
}

// checkRateLimit checks and updates the request counter for a key using cache.
//
// The counter is stored as "count:expiryUnixNano" so the window has a fixed end:
// incrementing preserves the original expiry instead of resetting the TTL on
// every request (which would let a steady stream of requests extend the window
// indefinitely). It fails open — if the cache is unavailable the request is
// allowed — to avoid turning a cache outage into an outage of the whole service.
func checkRateLimit(key string, maxRequests, windowMinutes int) rateLimitResult {
	unlock := lockKey(key)
	defer unlock()

	cacheKey := fmt.Sprintf("rate_limit:%s", key)
	window := time.Duration(windowMinutes) * time.Minute
	now := time.Now()

	// startWindow begins a fresh window with this request counted as the first.
	startWindow := func() rateLimitResult {
		expiry := now.Add(window)
		if err := cache.Set(cacheKey, encodeEntry(1, expiry), window); err != nil {
			log.Errorf("Failed to set rate limit in cache: %v", err)
			return rateLimitResult{limit: maxRequests, remaining: maxRequests - 1}
		}
		return rateLimitResult{limit: maxRequests, remaining: maxRequests - 1, retryAfter: windowMinutes * 60}
	}

	raw, err := cache.Get(cacheKey)
	if err != nil {
		// No entry yet (or cache miss) - treat as the first request of a window.
		return startWindow()
	}

	count, expiry, ok := decodeEntry(raw)
	if !ok || !now.Before(expiry) {
		// Corrupt value or an expired entry the cache has not evicted yet.
		return startWindow()
	}

	ttl := time.Until(expiry)
	retryAfter := int(ttl.Seconds()) + 1

	// Limit already reached: block without resetting the window.
	if count >= maxRequests {
		return rateLimitResult{limited: true, limit: maxRequests, remaining: 0, retryAfter: retryAfter}
	}

	// Increment while preserving the original expiry.
	newCount := count + 1
	if err := cache.Set(cacheKey, encodeEntry(newCount, expiry), ttl); err != nil {
		log.Errorf("Failed to update rate limit in cache: %v", err)
		return rateLimitResult{limit: maxRequests, remaining: maxRequests - newCount, retryAfter: retryAfter}
	}

	remaining := maxRequests - newCount
	if remaining < 0 {
		remaining = 0
	}

	return rateLimitResult{
		limited:    newCount > maxRequests,
		limit:      maxRequests,
		remaining:  remaining,
		retryAfter: retryAfter,
	}
}

// encodeEntry serializes the counter and window expiry for cache storage.
func encodeEntry(count int, expiry time.Time) string {
	return fmt.Sprintf("%d:%d", count, expiry.UnixNano())
}

// decodeEntry parses a value written by encodeEntry. ok is false for any value
// that does not match the expected format (including legacy count-only values).
func decodeEntry(v interface{}) (count int, expiry time.Time, ok bool) {
	s, isStr := v.(string)
	if !isStr {
		return 0, time.Time{}, false
	}

	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, time.Time{}, false
	}

	c, err1 := strconv.Atoi(parts[0])
	nano, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil {
		return 0, time.Time{}, false
	}

	return c, time.Unix(0, nano), true
}
