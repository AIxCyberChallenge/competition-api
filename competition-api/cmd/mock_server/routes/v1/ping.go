package v1

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

// Ping responds with a static value
//
//	@Summary		Test authentication creds and network connectivity
//	@Description	Test authentication creds and network connectivity
//	@Tags			ping
//	@Accept			json
//	@Product		json
//
//	@Security		BasicAuth
//
//	@Success		200	{object}	types.PingResponse
//
//	@Router			/v1/ping/ [get]
func Ping(c echo.Context) error {
	return c.JSON(http.StatusOK, types.PingResponse{Status: "ready"})
}
