package competition

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/jobs"
	servermiddleware "github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/middleware"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/config"
)

const name = "github.com/aixcyberchallenge/competition-api/competition-api/server/routes/competition"

var tracer = otel.Tracer(name)

type Handler struct {
	jobClient *jobs.KubernetesClient
	db        *gorm.DB
	// TODO: maybe just save pointer to the whole config object?
	RoundID string
	Teams   []config.Team
}

func Create(c *config.Config, jobClient *jobs.KubernetesClient, db *gorm.DB) *Handler {
	return &Handler{
		RoundID:   *c.RoundID,
		Teams:     c.Teams,
		jobClient: jobClient,
		db:        db,
	}
}

func (h *Handler) AddRoutes(e *echo.Echo, middlewareHandler *servermiddleware.Handler) {
	competitionGroup := e.Group("/competition",
		middleware.BasicAuth(middlewareHandler.BasicAuthValidator),
		servermiddleware.HasPermissions("auth", &models.Permissions{CompetitionManagement: true}),
	)
	competitionGroup.POST("/out-of-budget/", h.OutOfBudget)
	competitionGroup.DELETE("/cancel-task/", h.CancelAllTasks)
	competitionGroup.DELETE(
		"/cancel-task/:task_id/",
		h.CancelTask,
		servermiddleware.PopulateFromIDParam[models.Task](middlewareHandler, "task_id", "task"),
	)
}
