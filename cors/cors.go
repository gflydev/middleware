/*
Package middleware provides set of middleware functions that can be used to authenticate and authorize
requests in HTTP server.It also supports handling CORS, propagating headers, integrating with New Relic APM, and enabling
distributed tracing using OpenTelemetry.
*/

package cors

import "github.com/gflydev/core"

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
	return func(c *core.Ctx) error {
		corsHeadersConfig := getValidCORSHeaders(envHeaders)
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

	val := validCORSHeadersAndValues[core.HeaderAccessControlAllowHeaders]

	if val != AllowedHeaders {
		validCORSHeadersAndValues[core.HeaderAccessControlAllowHeaders] = AllowedHeaders + ", " + val
	}

	return validCORSHeadersAndValues
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
