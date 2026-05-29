package csrf

import (
	"github.com/gflydev/core"
	_ "github.com/joho/godotenv/autoload"
)

// RegisterApi func for describe a group of API routes.
func RegisterApi(apiRouter *core.Group) {
	// API Routers
	apiRouter.GET("/csrf", NewCsrfTokenApi())
}
