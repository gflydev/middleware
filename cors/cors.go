/*
Package cors provides middleware that sets Cross-Origin Resource Sharing (CORS)
response headers. It supports the six standard access-control headers:
Access-Control-Allow-Origin, Access-Control-Allow-Headers,
Access-Control-Allow-Methods, Access-Control-Allow-Credentials,
Access-Control-Expose-Headers, and Access-Control-Max-Age.
*/

package cors

import (
	"strings"

	"github.com/gflydev/core"
	"github.com/gflydev/core/log"
)

const (
	AllowedOrigin  = "*"
	AllowedHeaders = "Authorization, Content-Type, x-requested-with, origin, true-client-ip, X-Correlation-ID"
	AllowedMethods = "PUT, POST, GET, DELETE, OPTIONS, PATCH"
)

type Data map[string]string

// New an HTTP middleware that sets headers based on the provided envHeaders configuration
//
// Example: Add global middlewares in main file
//
//	app.Middleware(cors.New(cors.Data{
//		core.HeaderAccessControlAllowOrigin: cors.AllowedOrigin,
//	}))
func New(envHeaders Data) core.MiddlewareHandler {
	corsHeadersConfig := getValidCORSHeaders(envHeaders)

	// A wildcard origin combined with credentials is rejected by browsers and is
	// a security foot-gun, so warn about it once at construction time.
	if corsHeadersConfig[core.HeaderAccessControlAllowOrigin] == AllowedOrigin &&
		strings.EqualFold(corsHeadersConfig[core.HeaderAccessControlAllowCredentials], "true") {
		log.Warnf("CORS: Access-Control-Allow-Origin '*' with Access-Control-Allow-Credentials 'true' is rejected by browsers; set an explicit origin")
	}

	return func(c *core.Ctx) error {
		for k, v := range corsHeadersConfig {
			c.SetHeader(k, v)
		}

		return nil
	}
}

// getValidCORSHeaders returns a validated map of CORS headers.
// values specified in env are present in envHeaders
func getValidCORSHeaders(envHeaders Data) Data {
	validCORSHeadersAndValues := make(Data)

	for _, header := range allowedCORSHeader() {
		// If config is set, use that
		if val, ok := envHeaders[header]; ok && val != "" {
			validCORSHeadersAndValues[header] = val
			continue
		}

		// If config is not set - for the three headers, set default value.
		switch header {
		case core.HeaderAccessControlAllowOrigin:
			validCORSHeadersAndValues[header] = AllowedOrigin
		case core.HeaderAccessControlAllowHeaders:
			validCORSHeadersAndValues[header] = AllowedHeaders
		case core.HeaderAccessControlAllowMethods:
			validCORSHeadersAndValues[header] = AllowedMethods
		}
	}

	// Always keep the default headers allowed, then append any custom headers the
	// caller added, de-duplicating so a custom value that overlaps the defaults
	// does not produce repeated entries.
	validCORSHeadersAndValues[core.HeaderAccessControlAllowHeaders] =
		mergeHeaderList(AllowedHeaders, validCORSHeadersAndValues[core.HeaderAccessControlAllowHeaders])

	return validCORSHeadersAndValues
}

// mergeHeaderList combines two comma-separated header lists, preserving order and
// dropping case-insensitive duplicates.
func mergeHeaderList(base, extra string) string {
	seen := make(map[string]struct{})
	var merged []string

	for _, list := range []string{base, extra} {
		for _, part := range strings.Split(list, ",") {
			h := strings.TrimSpace(part)
			if h == "" {
				continue
			}
			key := strings.ToLower(h)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged = append(merged, h)
		}
	}

	return strings.Join(merged, ", ")
}

// allowedCORSHeader returns the HTTP headers used for CORS configuration in web applications.
func allowedCORSHeader() []string {
	return []string{
		core.HeaderAccessControlAllowOrigin,
		core.HeaderAccessControlAllowHeaders,
		core.HeaderAccessControlAllowMethods,
		core.HeaderAccessControlAllowCredentials,
		core.HeaderAccessControlExposeHeaders,
		core.HeaderAccessControlMaxAge,
	}
}
