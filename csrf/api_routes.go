package csrf

import (
	"github.com/gflydev/core"
)

// RegisterApi func for describe a group of API routes.
func RegisterApi(apiRouter *core.Group) {
	// API Routers
	apiRouter.GET("/csrf", NewCsrfTokenApi())
}
