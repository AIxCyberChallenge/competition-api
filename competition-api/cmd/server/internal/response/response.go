package response

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

var (
	InternalServerError = echo.NewHTTPError(
		http.StatusInternalServerError,
		types.StringError("something went wrong"),
	)
	NotFoundError = echo.NewHTTPError(http.StatusNotFound, types.StringError("not found"))
)
