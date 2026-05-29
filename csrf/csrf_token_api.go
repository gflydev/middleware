package csrf

import (
	"github.com/gflydev/core"
)

// CsrfResponse represents the JSON response for the CSRF token endpoint.
type CsrfResponse struct {
	CSRFToken  string `json:"csrf_token"`
	HeaderName string `json:"header_name"`
}

type CsrfTokenApi struct {
	core.Api
}

func NewCsrfTokenApi() *CsrfTokenApi {
	return &CsrfTokenApi{}
}

// Handle returns a CSRF token for the current session.
// @Summary	Get CSRF token
// @Description Returns a CSRF token to include in subsequent state-changing requests via X-CSRF-Token header
// @Tags Security
// @Produce json
// @Success 200 {object} CsrfResponse
// @Router /csrf [get]
func (h *CsrfTokenApi) Handle(c *core.Ctx) error {
	return c.Success(CsrfResponse{
		CSRFToken:  GetCSRFToken(c),
		HeaderName: HeaderName,
	})
}
