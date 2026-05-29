package middleware

import (
	"fmt"
	"github.com/gflydev/cache"
	"github.com/gflydev/core"
	"github.com/gflydev/core/log"
	"github.com/gflydev/core/utils"
	"github.com/gflydev/http"
	"strconv"
	"time"
)

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

		// Get client IP address
		clientIP := c.ClientIP()
		if clientIP == "" {
			clientIP = "unknown"
		}

		// Create a unique key for this IP and path combination
		key := fmt.Sprintf("%s:%s", clientIP, path)

		log.Tracef("Check rate limiting key: %s with limit %d requests per %d minutes", key, maxRequests, windowMinutes)

		// Check and update rate limit
		if isRateLimited(key, maxRequests, windowMinutes) {
			log.Warnf("Rate limit exceeded for IP %s on path %s", clientIP, path)
			log.Warnf("Rate limit key %s", key)

			return c.Error(http.Error{
				Message: "Too many requests. Please try again later.",
			}, core.StatusTooManyRequests)
		}

		return nil
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

	return applyRateLimit(c, maxRequests, windowMinutes)
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
		return applyRateLimit(c, maxRequests, windowMinutes)
	}
}

// applyRateLimit is a helper function that contains the common rate limiting logic
// It handles getting client IP, creating cache key, checking rate limits, and returning appropriate errors
func applyRateLimit(c *core.Ctx, maxRequests, windowMinutes int) error {
	// Get client IP address
	clientIP := c.ClientIP()
	if clientIP == "" {
		clientIP = "unknown"
	}

	// Create a unique key for this IP and path combination
	path := c.Path()
	key := fmt.Sprintf("%s:%s", clientIP, path)

	// Check and update rate limit
	if isRateLimited(key, maxRequests, windowMinutes) {
		log.Warnf("Rate limit exceeded for IP %s on path %s", clientIP, path)
		log.Warnf("Rate limit key %s", key)

		return c.Error(http.Error{
			Message: "Too many requests. Please try again later.",
		}, core.StatusTooManyRequests)
	}

	return nil
}

// there isRateLimited checks if the request should be rate limited using cache
func isRateLimited(key string, maxRequests, windowMinutes int) bool {
	// Create cache key with rate limit prefix
	cacheKey := fmt.Sprintf("rate_limit:%s", key)

	// Get current count from cache
	countStr, err := cache.Get(cacheKey)
	if err != nil {
		// First request from this key - set count to 1 with TTL
		duration := time.Duration(windowMinutes) * time.Minute
		if err := cache.Set(cacheKey, "1", duration); err != nil {
			log.Errorf("Failed to set rate limit in cache: %v", err)
			return false // Allow request if cache fails
		}
		return false
	}

	// Convert count to integer
	countString, ok := countStr.(string)
	if !ok {
		log.Errorf("Failed to cast rate limit count to string")
		return false // Allow request if casting fails
	}

	count, err := strconv.Atoi(countString)
	if err != nil {
		log.Errorf("Failed to parse rate limit count: %v", err)
		return false // Allow request if parsing fails
	}

	// Check if limit is already exceeded
	if count >= maxRequests {
		return true
	}

	// Increment the counter
	newCount := count + 1
	duration := time.Duration(windowMinutes) * time.Minute
	if err := cache.Set(cacheKey, strconv.Itoa(newCount), duration); err != nil {
		log.Errorf("Failed to update rate limit in cache: %v", err)
		return false // Allow request if cache update fails
	}

	// Check if limit is exceeded after increment
	return newCount > maxRequests
}
