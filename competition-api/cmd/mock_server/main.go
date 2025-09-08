package main

import (
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoswagger "github.com/swaggo/echo-swagger"

	_ "github.com/aixcyberchallenge/competition-api/competition-api/cmd/mock_server/docs"
	routesv1 "github.com/aixcyberchallenge/competition-api/competition-api/cmd/mock_server/routes/v1"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/validator"
)

func BasicAuthValidator(id, token string, _ echo.Context) (bool, error) {
	return id == "api_key_id" && token == "api_key_token", nil
}

// @title						Example Competition API
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

	e.GET("/swagger/*", echoswagger.WrapHandler)

	v1Group := e.Group("/v1", middleware.BasicAuth(BasicAuthValidator))

	v1Group.GET("/ping/", routesv1.Ping)

	taskGroup := v1Group.Group("/task/:task_id")
	requestGroup := v1Group.Group("/request")

	broadcastSarifGroup := taskGroup.Group("/broadcast-sarif-assessment")
	submittedSarifGroup := taskGroup.Group("/submitted-sarif")
	vulnGroup := taskGroup.Group("/pov")
	patchGroup := taskGroup.Group("/patch")
	bundleGroup := taskGroup.Group("/bundle")
	freeformGroup := taskGroup.Group("/freeform")

	broadcastSarifGroup.POST("/:broadcast_sarif_id/", routesv1.SubmitSarifAssessment)

	submittedSarifGroup.POST("/", routesv1.SubmitSarif)

	vulnGroup.POST("/", routesv1.SubmitPOV)
	vulnGroup.GET("/:pov_id/", routesv1.POVStatus)

	patchGroup.POST("/", routesv1.SubmitPatch)
	patchGroup.GET("/:patch_id/", routesv1.PatchStatus)

	bundleGroup.POST("/", routesv1.SubmitBundle)
	bundleGroup.PATCH("/:bundle_id", routesv1.PatchBundle)
	bundleGroup.GET("/:bundle_id", routesv1.GetBundle)
	bundleGroup.DELETE("/:bundle_id/", routesv1.DeleteBundle)

	freeformGroup.POST("/", routesv1.SubmitFreeform)

	requestGroup.POST("/:challenge_name", routesv1.RequestChallenge)
	requestGroup.GET("/list/", routesv1.RequestList)

	e.Logger.Fatal(e.Start(":1323"))
}
