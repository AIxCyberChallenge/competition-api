package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoswagger "github.com/swaggo/echo-swagger"

	_ "github.com/aixcyberchallenge/competition-api/competition-api/cmd/mock_crs/docs"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/mock_crs/routes"
	routesv1 "github.com/aixcyberchallenge/competition-api/competition-api/cmd/mock_crs/routes/v1"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/validator"
)

func BasicAuthValidator(id, token string, _ echo.Context) (bool, error) {
	return id == "api_key_id" && token == "api_key_token", nil
}

// @title						Example CRS API
// @version					1.4.0
// @securityDefinitions.basic	BasicAuth
func main() {
	e := echo.New()

	validate := validator.Create()
	e.Validator = &validate

	e.Pre(
		middleware.AddTrailingSlashWithConfig(
			middleware.TrailingSlashConfig{Skipper: func(c echo.Context) bool {
				return strings.Contains(c.Request().URL.Path, "swagger")
			}},
		),
	)

	e.Use(middleware.Logger())
	e.Use(logRequestBody)

	e.GET("/swagger/*", echoswagger.WrapHandler)

	authenticator := middleware.BasicAuth(BasicAuthValidator)

	v1Group := e.Group("/v1", authenticator)

	e.GET("/status/", routes.Status, authenticator)
	e.DELETE("/status/", routes.ResetStatus, authenticator)

	v1Group.POST("/task/", routesv1.SubmitTask)
	v1Group.DELETE("/task/", routesv1.CancelTasks)
	v1Group.DELETE("/task/:task_id/", routesv1.CancelTask)

	v1Group.POST("/sarif/", routesv1.SubmitSARIFBroadcast)

	e.Logger.Fatal(e.Start(":1324"))
}

func logRequestBody(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Read original body
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}

		// Log request body
		fmt.Printf("Request body: %s\n", string(body)) // 【1】

		// Restore body for subsequent handlers
		c.Request().Body = io.NopCloser(bytes.NewBuffer(body))
		return next(c)
	}
}
