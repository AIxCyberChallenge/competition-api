package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

// Status report the status of the CRS
//
//	@Summary		CRS Status
//	@Description	report the status of the CRS
//	@Tags			status
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Success		200	{object}	types.Status	"Status Obect"
//
//	@Router			/status/ [get]
func Status(c echo.Context) error {
	return c.JSON(http.StatusOK, types.Status{
		Ready:   true,
		Version: "1.0.0",
	})
}

// Reset status stats
//
//	@Summary		Reset status stats
//	@Description	Reset all stats in the status endpoint.
//	@Tags			status
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Success		200	{string}	string	"No Content"
//
//	@Router			/status/ [delete]
func ResetStatus(c echo.Context) error {
	return c.NoContent(http.StatusOK)
}
