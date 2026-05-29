# Rate Limit Controller Usage Examples

This document demonstrates how to use the new controller-specific rate limiting middleware functions.

## Available Functions

### 1. RateLimitRoute
A simple middleware that uses environment variables for configuration.

**Usage:**
```go
// Apply to individual routes
groupUsers.POST("/users", f.Middleware(middleware.RateLimitRoute)(user.NewCreateUserApi()))
groupAuth.POST("/signin", f.Middleware(middleware.RateLimitRoute)(auth.NewSignInApi()))
```

**Configuration:**
Uses environment variables:
- `RATE_LIMIT_MAX_REQUESTS` (default: 5)
- `RATE_LIMIT_WINDOW_MINUTES` (default: 15)

### 2. RateLimitParamsRoute
A parameterized middleware that allows custom rate limits per controller.

**Usage:**
```go
// Different rate limits for different controllers
groupUsers.POST("/users", f.Middleware(middleware.RateLimitParamsRoute(10, 5))(user.NewCreateUserApi()))
groupAuth.POST("/signin", f.Middleware(middleware.RateLimitParamsRoute(3, 15))(auth.NewSignInApi()))
```

**Parameters:**
- `maxRequests`: Maximum number of requests allowed per time window
- `windowMinutes`: Time window in minutes

## Comparison with Group-Level Rate Limiting

### Group-Level (existing)
```go
authGroup.Use(appMiddleware.RateLimit())
```
Applies to all routes in the group with the same rate limit.

### Controller-Level (new)
```go
authGroup.POST("/signin", f.Middleware(middleware.RateLimitRoute)(auth.NewSignInApi()))
authGroup.POST("/signup", f.Middleware(middleware.RateLimitParamsRoute(10, 5))(auth.NewSignUpApi()))
```
Allows different rate limits for different controllers within the same group.

## Benefits

1. **Granular Control**: Different rate limits for different endpoints
2. **Flexible Configuration**: Use environment variables or custom parameters
3. **Easy Integration**: Same pattern as other controller-specific middleware
4. **Backward Compatibility**: Existing group-level middleware continues to work
