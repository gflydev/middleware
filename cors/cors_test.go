package cors

import (
	"strings"
	"testing"

	"github.com/gflydev/core"
)

func TestGetValidCORSHeadersDefaults(t *testing.T) {
	headers := getValidCORSHeaders(Data{})

	if got := headers[core.HeaderAccessControlAllowOrigin]; got != AllowedOrigin {
		t.Errorf("Allow-Origin = %q, want %q", got, AllowedOrigin)
	}
	if got := headers[core.HeaderAccessControlAllowHeaders]; got != AllowedHeaders {
		t.Errorf("Allow-Headers = %q, want %q", got, AllowedHeaders)
	}
	if got := headers[core.HeaderAccessControlAllowMethods]; got != AllowedMethods {
		t.Errorf("Allow-Methods = %q, want %q", got, AllowedMethods)
	}
}

func TestGetValidCORSHeadersCustomOrigin(t *testing.T) {
	headers := getValidCORSHeaders(Data{
		core.HeaderAccessControlAllowOrigin: "https://example.com",
	})

	if got := headers[core.HeaderAccessControlAllowOrigin]; got != "https://example.com" {
		t.Errorf("Allow-Origin = %q, want %q", got, "https://example.com")
	}
}

func TestGetValidCORSHeadersCustomHeadersAppended(t *testing.T) {
	headers := getValidCORSHeaders(Data{
		core.HeaderAccessControlAllowHeaders: "X-Custom-Header",
	})

	got := headers[core.HeaderAccessControlAllowHeaders]
	if !strings.Contains(got, "X-Custom-Header") {
		t.Errorf("Allow-Headers = %q, want it to contain custom header", got)
	}
	if !strings.Contains(got, "Authorization") {
		t.Errorf("Allow-Headers = %q, want it to retain default headers", got)
	}
}

func TestGetValidCORSHeadersNoDuplicateHeaders(t *testing.T) {
	// A custom value that overlaps the defaults must not be repeated.
	headers := getValidCORSHeaders(Data{
		core.HeaderAccessControlAllowHeaders: "Authorization, X-Custom-Header",
	})

	got := headers[core.HeaderAccessControlAllowHeaders]
	if n := strings.Count(strings.ToLower(got), "authorization"); n != 1 {
		t.Errorf("Allow-Headers = %q, want exactly one 'Authorization', got %d", got, n)
	}
}

func TestMergeHeaderList(t *testing.T) {
	tests := []struct {
		name  string
		base  string
		extra string
		want  string
	}{
		{"empty extra", "A, B", "", "A, B"},
		{"disjoint", "A, B", "C", "A, B, C"},
		{"case-insensitive dedup", "A, B", "a, C", "A, B, C"},
		{"trims spaces", "A ,  B", " C ", "A, B, C"},
		{"drops empty entries", "A", ",,C,", "A, C"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := mergeHeaderList(tc.base, tc.extra); got != tc.want {
				t.Errorf("mergeHeaderList(%q, %q) = %q, want %q", tc.base, tc.extra, got, tc.want)
			}
		})
	}
}
