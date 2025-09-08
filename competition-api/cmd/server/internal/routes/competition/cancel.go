package competition

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"

	srverr "github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/error"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/response"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

func (h *Handler) CancelTask(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "CancelTask")
	defer span.End()

	task, ok := c.Get("task").(*models.Task)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("task: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	span.SetAttributes(
		attribute.String("task.id", task.ID.String()),
	)

	span.AddEvent("starting cancel task job")
	_, err := h.jobClient.CreateCancelJob(
		ctx,
		fmt.Sprintf("/v1/task/%s/", task.ID.String()),
		h.Teams,
		h.RoundID,
		task.Deadline,
	)
	if err != nil {
		span.SetStatus(codes.Error, "error starting cancel task job")
		span.RecordError(err)
		return response.InternalServerError
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "cancelling task")
	return c.NoContent(http.StatusOK)
}

// NOTE: this cancels all tasks they are currently working on not just the ones for the round id that is active
// They do not know what a round id is.
func (h *Handler) CancelAllTasks(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "CancelAllTasks")
	defer span.End()

	db := h.db.WithContext(ctx)

	task := new(models.Task)

	// get the time the last task will expire
	err := db.Order("deadline DESC").First(task, "deadline > CURRENT_TIMESTAMP()").Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			span.SetStatus(codes.Error, "no tasks found")
			span.RecordError(err)
			return echo.NewHTTPError(http.StatusBadRequest, types.StringError("no tasks"))
		}

		span.SetStatus(codes.Error, "error getting tasks from DB")
		span.RecordError(err)
		return response.InternalServerError
	}

	span.AddEvent("starting cancel all tasks job")
	_, err = h.jobClient.CreateCancelJob(
		ctx,
		"/v1/task/",
		h.Teams,
		h.RoundID,
		task.Deadline,
	)
	if err != nil {
		span.SetStatus(codes.Error, "error starting cancel all tasks job")
		span.RecordError(err)
		return response.InternalServerError
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "cancelling all tasks")
	return c.NoContent(http.StatusOK)
}
