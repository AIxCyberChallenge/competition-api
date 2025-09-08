package routes

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	slogecho "github.com/samber/slog-echo"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"

	servermiddleware "github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/middleware"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/validator"
)

func BuildEcho(logger *slog.Logger) (*echo.Echo, error) {
	e := echo.New()

	validate := validator.Create()
	e.Validator = &validate

	e.Pre(middleware.AddTrailingSlash())

	e.Use(
		otelecho.Middleware("competition-api"),
		slogecho.NewWithConfig(logger, slogecho.Config{}),
		servermiddleware.Time("time"),
	)

	e.GET("/health/", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	return e, nil
}
