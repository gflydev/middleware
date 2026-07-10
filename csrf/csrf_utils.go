package csrf

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
)

// TokenLength defines the default length for CSRF tokens
const TokenLength = 32

// GenerateCSRFToken creates a cryptographically secure random token.
// It returns an empty string if the system's secure random source fails,
// so callers must never treat an empty token as valid.
func GenerateCSRFToken(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Never fall back to a predictable token: a guessable CSRF token
		// defeats the protection entirely. Signal failure with an empty string.
		return ""
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// ConstantTimeCompare performs constant-time string comparison to prevent timing attacks.
//
// The token values are never logged: doing so would leak valid CSRF tokens to
// anyone with read access to the logs.
func ConstantTimeCompare(a, b string) bool {
	// subtle.ConstantTimeCompare returns 0 when the lengths differ, so the
	// length check is handled without leaking timing information.
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
