package middleware

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/codes"

	srverr "github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/error"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/response"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

// Ensures the current `validRoundIDs` overlap with the round id on the task
func RoundID(validRoundIDs map[string]bool, taskParam string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx, span := tracer.Start(c.Request().Context(), "RoundID")
			defer span.End()

			task, ok := c.Get(taskParam).(*models.Task)
			if !ok {
				span.RecordError(srverr.ErrTypeAssertMismatch)
				span.SetStatus(codes.Error, fmt.Sprintf("task: %s", srverr.ErrTypeAssertMismatch))
				return response.InternalServerError
			}

			logger.Logger.DebugContext(
				ctx,
				"checking roundID",
				"task",
				task.RoundID,
				"roundId",
				validRoundIDs,
			)
			if !validRoundIDs[task.RoundID] {
				span.RecordError(nil)
				span.SetStatus(codes.Ok, "invalid round id")
				return echo.NewHTTPError(
					http.StatusBadRequest,
					types.StringError("this task is for an invalid round"),
				)
			}

			span.RecordError(nil)
			span.SetStatus(codes.Ok, "validated round id")
			return next(c)
		}
	}
}
