package v1

import (
	"context"
	"fmt"
	"net/http"
	"time"

	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/challenges"
	srverr "github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/error"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/response"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/config"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

func (h *Handler) RequestChallenge(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "RequestChallenge")
	defer span.End()
	span.AddEvent("received request for challenge")

	auth, ok := c.Get("auth").(*models.Auth)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("auth: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	span.SetAttributes(
		attribute.String("auth.note", auth.Note),
		attribute.String("auth.id", auth.ID.String()),
		attribute.String("round.id", *h.config.Generate.RoundID),
	)

	var hourSecs int64 = 3600

	type requestData struct {
		types.RequestSubmission
		ChallengeName string `param:"challenge_name" validate:"required"`
	}
	rdata := requestData{
		RequestSubmission: types.RequestSubmission{DurationSecs: &hourSecs},
	}

	span.AddEvent("parsing request body")
	err := c.Bind(&rdata)
	if err != nil {
		span.SetStatus(codes.Ok, "failed to parse request data")
		span.RecordError(err)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.StringError("failed to parse request data"),
		)
	}

	span.AddEvent("validating request body")
	err = c.Validate(rdata)
	if err != nil {
		span.SetStatus(codes.Ok, "failed to validate request data")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, types.ValidationError(err))
	}

	span.SetAttributes(attribute.String("challenge.name", rdata.ChallengeName))
	span.AddEvent("finding team ID in config")
	teams := make([]config.Team, 0, 1)
	for _, team := range h.config.Teams {
		if team.ID != auth.ID.String() {
			continue
		}

		teams = append(teams, team)
		break
	}

	if len(teams) == 0 {
		span.SetStatus(codes.Error, "did not find team ID in config")
		span.RecordError(nil)
		return response.InternalServerError
	}

	var challenge *config.GenerateChallengeConfig

	span.AddEvent("finding requested challenge")
	for _, chal := range h.config.Generate.Challenges {
		if *chal.Name == rdata.ChallengeName {
			challenge = &chal
		}
	}

	if challenge == nil {
		span.SetStatus(codes.Error, "specified challenge does not exist")
		span.RecordError(nil)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.Message{Message: "specified challenge does not exist"},
		)
	}

	span.AddEvent("starting run delta for only one team")
	h.taskrunnerClient.Run(ctx, func(ctx context.Context) {
		//nolint:govet // shadow: intentionally shadow ctx and span to avoid using the incorrect one.
		ctx, span := tracer.Start(ctx, "RequestChallngeTaskingFunction")
		defer span.End()

		span.AddEvent("get token for installation ID")
		token, err := h.githubClient.CreateInstallationToken(
			ctx,
			*h.config.Generate.InstallationID,
		)
		if err != nil {
			span.SetStatus(codes.Error, "failed to get installation token")
			span.RecordError(err)
			return
		}

		challengeConfig := challenges.ChallengeConfig{
			Name:         fmt.Sprintf("%s - testing", auth.Note),
			BaseRef:      challenge.Config.BaseRef,
			HeadRef:      *challenge.Config.HeadRef,
			RepoURL:      *challenge.Config.RepoURL,
			TaskDuration: time.Second * time.Duration(*rdata.DurationSecs),
			AuthMethod: githttp.BasicAuth{
				Username: "token",
				Password: token.GetToken(),
			},
		}

		if challenge.Config.BaseRef == nil {
			span.SetAttributes(
				attribute.String("challenge.head_ref", *challenge.Config.HeadRef),
			)
			span.AddEvent("running full scan")

			err := h.challengesClient.RunFullScan(
				ctx,
				challengeConfig,
				teams,
				*h.config.Generate.RoundID,
			)
			if err != nil {
				span.SetStatus(codes.Error, "failed to run full scan")
				span.RecordError(err)
				return
			}
		} else {
			span.SetAttributes(
				attribute.String("challenge.base_ref", *challenge.Config.BaseRef),
				attribute.String("challenge.head_ref", *challenge.Config.HeadRef),
			)
			span.AddEvent("running delta scan")

			err := h.challengesClient.RunDeltaScan(
				ctx,
				challengeConfig,
				teams,
				*h.config.Generate.RoundID,
			)
			if err != nil {
				span.SetStatus(codes.Error, "failed to run delta scan")
				span.RecordError(err)
				return
			}
		}

		span.RecordError(nil)
		span.SetStatus(codes.Ok, "created challenge task")
	})

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "")

	return c.JSON(http.StatusOK, types.Message{Message: "received request for task"})
}

func (h *Handler) RequestList(c echo.Context) error {
	_, span := tracer.Start(c.Request().Context(), "RequestList")
	defer span.End()
	span.AddEvent("received request for challenge list")

	challengeList := []string{}

	for _, challenge := range h.config.Generate.Challenges {
		challengeList = append(challengeList, *challenge.Name)
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "")
	return c.JSON(http.StatusOK, types.RequestListResponse{Challenges: challengeList})
}
