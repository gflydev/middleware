# gFly Middlewares

## CORS
Support 6 access controls `Access-Control-Allow-Origin`, `Access-Control-Allow-Headers`, `Access-Control-Allow-Methods`, `Access-Control-Allow-Credentials`, `Access-Control-Expose-Headers`, `Access-Control-Max-Age`. 

### Usage
Install
```bash
go get -u github.com/gflydev/middleware/cors@v1.0.0
```

Quick usage `main.go`
```go
// Add global middlewares
app.Use(cors.New(cors.Data{
    core.HeaderAccessControlAllowOrigin: cors.AllowedOrigin,
}))
```
### Access controls:
- Access-Control-Allow-Origin: Accept all domains `*` (default)
- Access-Control-Allow-Headers: Accept all header parameters `Authorization, Content-Type, x-requested-with, origin, true-client-ip, X-Correlation-ID` (default)
- Access-Control-Allow-Methods: List supported methods `PUT`, `POST`, `GET`, `DELETE`, `OPTIONS`, `PATCH` (default)
- Access-Control-Allow-Credentials: N/A
- Access-Control-Expose-Headers: N/A
- Access-Control-Max-Age: N/A

## Rate Limit
Protect your application from abuse by limiting the number of requests per IP address within a configurable time window. Uses cache to track request counts with automatic TTL expiration.

### Usage
Install
```bash
go get -u github.com/gflydev/middleware/ratelimit@v1.0.0
```

### Global Middleware (via `.env` configuration)
Set environment variables in your `.env` file:
```env
RATE_LIMIT_MAX_REQUESTS=5
RATE_LIMIT_WINDOW_MINUTES=15
```

Quick usage `main.go`
```go
import middleware "github.com/gflydev/middleware/ratelimit"

// Apply rate limiting to a group with excluded paths
g.Group(prefixAPI, func(apiRouter *core.Group) {
    apiRouter.Use(middleware.RateLimit(prefixAPI + "/health"))
    apiRouter.GET("/info", NewDefaultApi())
})
```

Or apply globally to all routes:
```go
app.Use(middleware.RateLimit())
```

### Global Middleware (with custom parameters)
Use `RateLimitParam` to set limits in code instead of `.env`:
```go
// Allow 10 requests per 30 minutes, exclude the health endpoint
app.Use(middleware.RateLimitParam(10, 30, "/api/v1/health"))
```

### Per-Route Middleware
Apply rate limiting to individual routes for fine-grained control:
```go
// Using .env configuration
groupUsers.POST("/users", f.Middleware(middleware.RateLimitRoute)(user.NewCreateUserApi()))

// Using custom parameters (3 requests per 15 minutes for auth)
groupAuth.POST("/signin", f.Middleware(middleware.RateLimitParamsRoute(3, 15))(auth.NewSignInApi()))
```

### How it works
- Tracks requests by `clientIP:path` combination stored in cache
- On first request, creates a cache entry with count `1` and TTL equal to the time window
- On subsequent requests, increments the counter
- When the limit is exceeded, returns HTTP `429 Too Many Requests`
- Cache entries automatically expire after the configured time window
- If cache is unavailable, requests are allowed through (fail-open)
