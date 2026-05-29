package csrf

import (
	"fmt"
	"github.com/gflydev/core"
	"github.com/gflydev/core/utils"
)

// RegisterApi func for describe a group of API routes.
func RegisterApi(apiRouter *core.Group) {
	prefixAPI := fmt.Sprintf(
		"/%s/%s",
		utils.Getenv("API_PREFIX", "api"),
		utils.Getenv("API_VERSION", "v1"),
	)

	// API Routers
	apiRouter.Group(prefixAPI, func(apiRouter *core.Group) {
		// CSRF token endpoint
		apiRouter.GET("/csrf", NewCsrfTokenApi())
	})
}
