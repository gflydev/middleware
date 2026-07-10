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
- On first request, creates a cache entry with count `1` and a fixed window expiry
- On subsequent requests, increments the counter **without** resetting the window, so a steady stream of requests cannot extend the window indefinitely
- Concurrent requests for the same key are serialized with an in-process lock, so the limit is not exceeded under load (distributed deployments still need an atomic counter in the shared cache)
- Every response includes `X-RateLimit-Limit` and `X-RateLimit-Remaining` headers; blocked responses also include `Retry-After` (seconds)
- When the limit is exceeded, returns HTTP `429 Too Many Requests`
- Cache entries automatically expire after the configured time window
- If cache is unavailable, requests are allowed through (fail-open)

## CSRF
Protect your application from Cross-Site Request Forgery (CSRF) attacks using the double-submit cookie pattern. Generates cryptographically secure tokens per session and validates them on state-changing requests (POST, PUT, PATCH, DELETE). Safe methods (GET, HEAD, OPTIONS, TRACE) are automatically excluded.

### Usage
Install
```bash
go get -u github.com/gflydev/middleware/csrf@v1.0.0
```

Set token expiry in your `.env` file (in seconds, default: 86400 = 24h):
```env
# CSRF Settings:
#   CSRF token expiry in seconds (default: 86400 = 24 hours)
CSRF_EXPIRY=86400
```

Quick usage `main.go`
```go
import "github.com/gflydev/middleware/csrf"

// Apply CSRF protection globally, excluding public endpoints
app.Use(csrf.Middleware(
    "/api/v1/auth/login",
    "/api/v1/auth/register",
))
```

### Token API

The CSRF package includes a built-in API endpoint that returns the current CSRF token and header name - useful for SPAs that need to fetch the token programmatically.

Register the router in `main.go`:
```go
import "github.com/gflydev/middleware/csrf"

// Register CSRF token API
csrfRouter := csrf.NewCsrfTokenApi()
router.GET("/csrf", csrfRouter)

// OR
csrf.RegisterApi(router)
```

Response format (JSON):
```json
{
    "csrf_token":  "a1b2c3d4e5f6...",
    "header_name": "X-CSRF-Token"
}
```

### Frontend Integration

**AJAX Requests** — include the token in the `X-CSRF-Token` header:
```javascript
fetch('/api/v1/users', {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken
    },
    body: JSON.stringify(data)
});
```

**HTML Forms** — include the token as a hidden `_token` field:
```html
<form method="POST" action="/api/v1/users">
    <input type="hidden" name="_token" value="{{ .csrf_token }}">
    <input type="text" name="name" required>
    <button type="submit">Create User</button>
</form>
```

Expose the token to your frontend via `GetCSRFToken()`:
```go
token := csrf.GetCSRFToken(c)
```

### How it works
- Token is generated and stored in both the session and a cookie
- Safe methods (GET, HEAD, OPTIONS, TRACE) are excluded — token is simply set/refreshed
- On unsafe methods (POST, PUT, PATCH, DELETE), the request token (from header → form field → cookie) is validated against the session token using constant-time comparison to prevent timing attacks
- After successful validation, the token is regenerated for additional security
- The token cookie is set via the framework's cookie API, so it does not clobber other response cookies (e.g. the session cookie)
- Token values are never written to logs, and comparison uses `crypto/subtle` constant-time equality
- Returns HTTP `403 Forbidden` with `"CSRF token mismatch"` on validation failure

## Development

Each middleware is an independent Go module. Run the checks from within a module directory:
```bash
cd ratelimit   # or cors, csrf
go vet ./...
go test -race ./...
```
CI runs `gofmt`, `go vet`, and the race-enabled test suite for all three modules on every push and pull request.

## License

Released under the [MIT License](LICENSE).
