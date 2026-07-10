package csrf

import (
	"encoding/base64"
	"testing"
)

func TestGenerateCSRFTokenLength(t *testing.T) {
	token := GenerateCSRFToken(TokenLength)
	if token == "" {
		t.Fatal("GenerateCSRFToken returned an empty token")
	}

	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("token is not valid base64: %v", err)
	}
	if len(decoded) != TokenLength {
		t.Errorf("decoded token length = %d, want %d", len(decoded), TokenLength)
	}
}

func TestGenerateCSRFTokenUniqueness(t *testing.T) {
	seen := make(map[string]struct{})
	for i := 0; i < 1000; i++ {
		token := GenerateCSRFToken(TokenLength)
		if _, ok := seen[token]; ok {
			t.Fatalf("duplicate token generated: %q", token)
		}
		seen[token] = struct{}{}
	}
}

func TestConstantTimeCompare(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{"equal", "abc123", "abc123", true},
		{"different value same length", "abc123", "abc124", false},
		{"different length", "abc", "abcd", false},
		{"both empty", "", "", true},
		{"one empty", "abc", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ConstantTimeCompare(tc.a, tc.b); got != tc.want {
				t.Errorf("ConstantTimeCompare(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestIsSafeMethod(t *testing.T) {
	safe := []string{"GET", "HEAD", "OPTIONS", "TRACE", "get", "head"}
	for _, m := range safe {
		if !isSafeMethod(m) {
			t.Errorf("isSafeMethod(%q) = false, want true", m)
		}
	}

	unsafe := []string{"POST", "PUT", "PATCH", "DELETE", "post"}
	for _, m := range unsafe {
		if isSafeMethod(m) {
			t.Errorf("isSafeMethod(%q) = true, want false", m)
		}
	}
}

func TestPathMatches(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{"exact", "/api/v1/users", "/api/v1/users", true},
		{"different", "/api/v1/users", "/api/v1/posts", false},
		{"param match", "/api/v1/users/{id}", "/api/v1/users/42", true},
		{"length mismatch", "/api/v1/users", "/api/v1/users/42", false},
		{"param with different static", "/api/{v}/users", "/api/v2/posts", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := pathMatches(tc.pattern, tc.path); got != tc.want {
				t.Errorf("pathMatches(%q, %q) = %v, want %v", tc.pattern, tc.path, got, tc.want)
			}
		})
	}
}
