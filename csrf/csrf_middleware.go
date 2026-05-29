package csrf

import (
	"fmt"
	"github.com/gflydev/core"
	"github.com/gflydev/core/log"
	"github.com/gflydev/core/utils"
	"github.com/gflydev/http"
	"strings"
	"time"
)

const (
	// CSRF token configuration

	SessionKey    = "csrf_token"
	HeaderName    = "X-CSRF-Token"
	FormFieldName = "_token"
	CookieName    = "csrf_token"
)

// Config holds configuration for CSRF protection
type Config struct {
	TokenLength   int
	SessionKey    string
	HeaderName    string
	FormFieldName string
	CookieName    string
	Expiry        time.Duration
	SecureOnly    bool
	SameSite      string
}

// DefaultCSRFConfig returns default CSRF configuration
func DefaultCSRFConfig() Config {
	expirySeconds := utils.Getenv("CSRF_EXPIRY", 86400) // default: 24 hours

	return Config{
		TokenLength:   TokenLength,
		SessionKey:    SessionKey,
		HeaderName:    HeaderName,
		FormFieldName: FormFieldName,
		CookieName:    CookieName,
		Expiry:        time.Duration(expirySeconds) * time.Second,
		SecureOnly:    utils.Getenv("APP_ENV", "dev") == "prod",
		SameSite:      "Strict",
	}
}

// Middleware creates CSRF protection middleware
//
// Parameters:
//   - excludes (...string): Optional paths to exclude from CSRF protection
//
// Returns:
//   - core.MiddlewareHandler: A middleware handler function
//
// The middleware protects against Cross-Site Request Forgery attacks by:
// 1. Generating unique tokens for each session
// 2. Validating tokens on state-changing requests (POST, PUT, PATCH, DELETE)
// 3. Storing tokens in both session and cookie for double-submit cookie pattern
func Middleware(excludes ...string) core.MiddlewareHandler {
	config := DefaultCSRFConfig()
	log.Tracef("CSRF protection configured: %d byte tokens, expiry: %v", config.TokenLength, config.Expiry)

	return MiddlewareWithConfig(config, excludes...)
}

// MiddlewareWithConfig creates CSRF protection middleware with custom configuration
func MiddlewareWithConfig(config Config, excludes ...string) core.MiddlewareHandler {
	return func(c *core.Ctx) error {
		path := c.Path()
		method := string(c.Root().Request.Header.Method())

		// Skip CSRF protection for excluded paths
		for _, exclude := range excludes {
			if pathMatches(exclude, path) {
				log.Tracef("Skip CSRF protection for %v", path)
				return nil
			}
		}

		// Skip CSRF protection for safe HTTP methods
		if isSafeMethod(method) {
			// For safe methods, just ensure token exists and set cookie
			token := getOrCreateCSRFToken(c, config)
			setCSRFCookie(c, token, config)
			return nil
		}

		// For unsafe methods, validate CSRF token
		if !validateCSRFToken(c, config) {
			log.Warnf("CSRF token validation failed for %s %s", method, path)
			return c.Error(http.Error{
				Message: "CSRF token mismatch",
			}, core.StatusForbidden)
		}

		// Regenerate token after successful validation for additional security
		token := GenerateCSRFToken(config.TokenLength)
		c.SetSession(config.SessionKey, token)
		setCSRFCookie(c, token, config)

		return nil
	}
}

// GetCSRFToken retrieves the current CSRF token for the session
// This function can be used in templates or API responses to provide the token to clients
func GetCSRFToken(c *core.Ctx) string {
	config := DefaultCSRFConfig()
	return getOrCreateCSRFToken(c, config)
}

// getOrCreateCSRFToken retrieves existing token or creates a new one
func getOrCreateCSRFToken(c *core.Ctx, config Config) string {
	// Try to get existing token from session
	if sessionToken := c.GetSession(config.SessionKey); sessionToken != nil {
		if token, ok := sessionToken.(string); ok && token != "" {
			return token
		}
	}

	// Generate new token if none exists
	token := GenerateCSRFToken(config.TokenLength)
	c.SetSession(config.SessionKey, token)
	return token
}

// Validate validates the CSRF token from request against session token
func Validate(c *core.Ctx) bool {
	config := DefaultCSRFConfig()

	return validateCSRFToken(c, config)
}

// validateCSRFToken validates the CSRF token from request against session token
func validateCSRFToken(c *core.Ctx, config Config) bool {
	// Get token from session
	sessionToken := c.GetSession(config.SessionKey)
	if sessionToken == nil {
		log.Debugf("No CSRF token in session")
		return false
	}

	sessionTokenStr, ok := sessionToken.(string)
	if !ok || sessionTokenStr == "" {
		log.Debugf("Invalid CSRF token in session")
		return false
	}

	// Get token from request (header, form, or cookie)
	requestToken := getCSRFTokenFromRequest(c, config)
	if requestToken == "" {
		log.Debugf("No CSRF token in request")
		return false
	}

	// Compare tokens using constant-time comparison to prevent timing attacks
	return ConstantTimeCompare(sessionTokenStr, requestToken)
}

// getCSRFTokenFromRequest extracts CSRF token from various request sources
func getCSRFTokenFromRequest(c *core.Ctx, config Config) string {
	// 1. Check header first (for AJAX requests)
	if token := c.GetHeader(config.HeaderName); token != "" {
		return token
	}

	// 2. Check form field (for form submissions)
	if token := c.FormStr(config.FormFieldName); token != "" {
		return token
	}

	// 3. Check cookie (for double-submit cookie pattern)
	if token := c.GetCookie(config.CookieName); token != "" {
		return token
	}

	return ""
}

// setCSRFCookie sets the CSRF token as a cookie
func setCSRFCookie(c *core.Ctx, token string, config Config) {
	cookie := fmt.Sprintf("%s=%s; Path=/; Max-Age=%d",
		config.CookieName,
		token,
		int(config.Expiry.Seconds()))

	if config.SecureOnly {
		cookie += "; Secure"
	}

	if config.SameSite != "" {
		cookie += fmt.Sprintf("; SameSite=%s", config.SameSite)
	}

	c.Root().Response.Header.Set("Set-Cookie", cookie)
}

// isSafeMethod checks if HTTP method is considered safe (doesn't modify state)
func isSafeMethod(method string) bool {
	safeMethods := []string{"GET", "HEAD", "OPTIONS", "TRACE"}
	method = strings.ToUpper(method)
	for _, safe := range safeMethods {
		if method == safe {
			return true
		}
	}
	return false
}

// pathMatches checks if the given path matches the pattern
// Reusing the same logic from jwt_middleware.go for consistency
func pathMatches(pattern, path string) bool {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(patternParts) != len(pathParts) {
		return false
	}

	for i, part := range patternParts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			continue // It's a parameter, so it matches
		}
		if part != pathParts[i] {
			return false
		}
	}

	return true
}
