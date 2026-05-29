# CSRF Middleware Documentation

## Overview

The CSRF (Cross-Site Request Forgery) middleware provides protection against CSRF attacks by implementing token-based validation. It generates unique tokens for each session and validates them on state-changing HTTP requests.

## Features

- **Secure Token Generation**: Uses cryptographically secure random token generation
- **Double-Submit Cookie Pattern**: Implements both session and cookie-based token storage
- **Configurable**: Supports custom configuration for different environments
- **Path Exclusion**: Allows excluding specific paths from CSRF protection
- **Multiple Token Sources**: Supports tokens from headers, form fields, and cookies
- **Constant-Time Comparison**: Prevents timing attacks during token validation

## Installation

The CSRF middleware is located in `internal/csrf/middleware.go`, utilities are in `internal/csrf/utils.go`, and the controller is in `internal/csrf/controller.go`.

## Basic Usage

### 1. Apply CSRF Middleware to Routes

```go
import (
    "gfly/internal/csrf"
    "github.com/gflydev/core"
)

func setupRoutes(app *core.App) {
    // Apply CSRF protection to all routes except login/register
    app.Use(csrf.CSRFMiddleware(
        "/api/v1/auth/login",
        "/api/v1/auth/register",
        "/api/v1/csrf/token", // Allow getting CSRF tokens
    ))

    // Your protected routes here
    app.POST("/api/v1/users", userController.Create)
    app.PUT("/api/v1/users/:id", userController.Update)
    app.DELETE("/api/v1/users/:id", userController.Delete)
}
```

### 2. Get CSRF Token in Controllers

```go
import (
    "gfly/internal/csrf"
    "github.com/gflydev/core"
)

func (ctrl *UserController) GetProfile(c *core.Ctx) error {
    // Get CSRF token for the current session
    csrfToken := csrf.GetCSRFToken(c)

    return c.JSON(map[string]interface{}{
        "user": user,
        "csrf_token": csrfToken,
    })
}

// Or use the utility function for standardized response
func (ctrl *UserController) GetCSRFInfo(c *core.Ctx) error {
    token := csrf.GetCSRFToken(c)
    // CSRFTokenResponse the response object
    response := csrf.CSRFTokenResponse(token)
    return c.JSON(response)
}
```

### 3. Frontend Integration

#### JavaScript/AJAX Requests

```javascript
// Get CSRF token
fetch('/api/v1/csrf/token')
    .then(response => response.json())
    .then(data => {
        const csrfToken = data.csrf_token;

        // Use token in subsequent requests
        fetch('/api/v1/users', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-CSRF-Token': csrfToken
            },
            body: JSON.stringify(userData)
        });
    });
```

#### HTML Forms

```html
<!-- Include CSRF token as hidden field -->
<form method="POST" action="/api/v1/users">
    <input type="hidden" name="_token" value="{{ .csrf_token }}">
    <input type="text" name="name" required>
    <button type="submit">Create User</button>
</form>
```

## Configuration

### Default Configuration

```go
type CSRFConfig struct {
    TokenLength   int           // 32 bytes
    SessionKey    string        // "csrf_token"
    HeaderName    string        // "X-CSRF-Token"
    FormFieldName string        // "_token"
    CookieName    string        // "csrf_token"
    Expiry        time.Duration // 86400s (24h), configurable via CSRF_EXPIRY env
    SecureOnly    bool          // true in production
    SameSite      string        // "Strict"
}
```

### Custom Configuration

```go
import "gfly/internal/csrf"

// Create custom configuration
config := csrf.CSRFConfig{
    TokenLength:   64,
    SessionKey:    "my_csrf_token",
    HeaderName:    "X-My-CSRF-Token",
    FormFieldName: "_csrf_token",
    CookieName:    "my_csrf_token",
    Expiry:        12 * time.Hour,  // default: 86400s (24h), configurable via CSRF_EXPIRY env
    SecureOnly:    true,
    SameSite:      "Lax",
}

// Apply middleware with custom config
app.Use(csrf.CSRFMiddlewareWithConfig(config, "/api/v1/auth/login"))
```

## Token Sources

The middleware checks for CSRF tokens in the following order:

1. **HTTP Header**: `X-CSRF-Token` (or custom header name)
2. **Form Field**: `_token` (or custom field name)
3. **Cookie**: `csrf_token` (or custom cookie name)

## Security Features

### 1. Cryptographically Secure Token Generation

```go
// Tokens are generated using crypto/rand
token := csrf.GenerateCSRFToken(32)
```

### 2. Constant-Time Comparison

```go
// Prevents timing attacks
isValid := csrf.ConstantTimeCompare(sessionToken, requestToken)
```

### 3. Token Regeneration

Tokens are regenerated after successful validation for additional security.

### 4. Safe HTTP Methods

The middleware automatically skips validation for safe HTTP methods:
- GET
- HEAD
- OPTIONS
- TRACE

## Environment Variables

You can configure CSRF behavior using environment variables:

```env
# Application environment (affects SecureOnly cookie setting)
APP_ENV=production

# CSRF Settings:
#  CSRF token expiry in seconds (default: 86400 = 24 hours)
CSRF_EXPIRY=86400

# Session configuration (affects token storage)
SESSION_DRIVER=redis
SESSION_LIFETIME=1440
```

## API Endpoints

### Get CSRF Token

```
GET /api/v1/csrf/token
```

Response:
```json
{
    "csrf_token": "base64-encoded-token",
    "token_name": "_token",
    "header_name": "X-CSRF-Token",
    "expires_in": 86400
}
```

### Validate CSRF Token (Debug)

```
POST /api/v1/csrf/validate?token=your-token
```

Response:
```json
{
    "valid": true,
    "message": "Token is valid"
}
```

## Best Practices

### 1. Exclude Authentication Endpoints

Always exclude login, register, and token retrieval endpoints:

```go
csrf.CSRFMiddleware(
    "/api/v1/auth/login",
    "/api/v1/auth/register",
    "/api/v1/csrf/token",
)
```

### 2. Use HTTPS in Production

CSRF protection is most effective when combined with HTTPS:

```go
config := csrf.DefaultCSRFConfig()
config.SecureOnly = true // Only send cookies over HTTPS
```

### 3. Implement Proper Session Management

Ensure your application has proper session management:

```go
// Sessions should be properly configured
app.Use(sessionMiddleware.SessionManipulation)
```

### 4. Handle Token Expiration

Implement proper error handling for expired tokens:

```javascript
fetch('/api/v1/users', {
    method: 'POST',
    headers: {
        'X-CSRF-Token': csrfToken
    },
    body: JSON.stringify(data)
})
.then(response => {
    if (response.status === 403) {
        // Token expired or invalid, refresh token
        return refreshCSRFToken().then(newToken => {
            // Retry request with new token
            return fetch('/api/v1/users', {
                method: 'POST',
                headers: {
                    'X-CSRF-Token': newToken
                },
                body: JSON.stringify(data)
            });
        });
    }
    return response;
});
```

## Testing

Run the CSRF tests:

```bash
go test ./test/csrf_test.go -v
```

## Troubleshooting

### Common Issues

1. **Token Mismatch Errors**
   - Ensure the token is being sent in the correct header/field
   - Check that sessions are properly configured
   - Verify the token hasn't expired

2. **Tokens Not Generated**
   - Ensure session middleware is applied before CSRF middleware
   - Check that the session driver is properly configured

3. **AJAX Requests Failing**
   - Make sure to include the CSRF token in the request header
   - Verify the header name matches the configuration

### Debug Mode

Use the validation endpoint to debug token issues:

```bash
curl -X POST "http://localhost:8080/api/v1/csrf/validate?token=your-token"
```

## Integration Examples

### With Authentication Middleware

```go
// Apply middlewares in correct order
app.Use(sessionMiddleware.SessionManipulation)
app.Use(csrf.CSRFMiddleware("/api/v1/auth/login"))
app.Use(authMiddleware.JWTAuth("/api/v1/auth/login"))
```

### With Rate Limiting

```go
// CSRF protection with rate limiting
app.Use(middleware.RateLimitMiddleware())
app.Use(csrf.CSRFMiddleware("/api/v1/auth/login"))
```

This documentation provides comprehensive guidance for implementing and using CSRF protection in your ThietNgon-Go application.
