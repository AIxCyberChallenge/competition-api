package main

import (
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	_ "github.com/aixcyberchallenge/competition-api/competition-api/cmd/mock_jobrunner/docs"
	jr "github.com/aixcyberchallenge/competition-api/competition-api/cmd/mock_jobrunner/routes/jobrunner"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/validator"
)

func BasicAuthValidator(id, token string, _ echo.Context) (bool, error) {
	return id == "api_key_id" && token == "api_key_token", nil
}

// @title						Jobrunner API
// @version					1.0.0
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

	jobrunnerGroup := e.Group("/jobrunner", middleware.BasicAuth(BasicAuthValidator))

	jobrunnerGroup.POST("/jobrunner/job/", jr.RunTest)
	jobrunnerGroup.GET("/jobrunner/job/:job_id/", jr.GetJobResults)

	e.Logger.Fatal(e.Start(":1323"))
}
