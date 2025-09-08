package webhooks

import (
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/challenges"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/github"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/taskrunner"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/config"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/upload"
)

type Handler struct {
	archiver         upload.Uploader
	db               *gorm.DB
	taskRunnerClient *taskrunner.Client
	challengesClient *challenges.Client
	githubClient     *github.Client
	ignoredRepos     *[]string
	roundID          string
	teams            []config.Team
}

func CreateHandler(
	db *gorm.DB,
	taskRunnerClient *taskrunner.Client,
	challengesClient *challenges.Client,
	roundID string,
	githubClient *github.Client,
	teams []config.Team,
	ignoredRepos *[]string,
	archiver upload.Uploader,
) *Handler {
	return &Handler{
		db:               db,
		taskRunnerClient: taskRunnerClient,
		challengesClient: challengesClient,
		roundID:          roundID,
		githubClient:     githubClient,
		teams:            teams,
		ignoredRepos:     ignoredRepos,
		archiver:         archiver,
	}
}

func (h *Handler) AddRoutes(e *echo.Echo) {
	e.POST("/webhook/github/", h.HandleGithubWebhook)
}
