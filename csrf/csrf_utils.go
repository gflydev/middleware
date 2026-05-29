package csrf

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/gflydev/core/log"
	"time"
)

// TokenLength defines the default length for CSRF tokens
const TokenLength = 32

// GenerateCSRFToken creates a cryptographically secure random token
func GenerateCSRFToken(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based token (less secure but functional)
		return base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("fallback_%d", time.Now().UnixNano())))
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// ConstantTimeCompare performs constant-time string comparison to prevent timing attacks
func ConstantTimeCompare(a, b string) bool {
	log.Debugf("Comparing %s and %s", a, b)

	if len(a) != len(b) {
		return false
	}

	result := 0
	for i := 0; i < len(a); i++ {
		result |= int(a[i]) ^ int(b[i])
	}

	return result == 0
}
