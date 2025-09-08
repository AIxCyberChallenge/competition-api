package jobrunner

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	srverr "github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/error"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/response"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

func (h *Handler) RunTest(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "RunTest")
	defer span.End()

	span.AddEvent("received job runner request")

	auth, ok := c.Get("auth").(*models.Auth)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("auth: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	requestTime, ok := c.Get("time").(time.Time)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("time: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	span.SetAttributes(
		attribute.String("auth.note", auth.Note),
		attribute.String("auth.id", auth.ID.String()),
		attribute.Int64("request.timestamp_ms", requestTime.UnixMilli()),
	)

	type requestData struct {
		types.JobArgs
	}

	var rdata requestData

	span.AddEvent("parsing request body")
	err := c.Bind(&rdata)
	if err != nil {
		span.SetStatus(codes.Error, "failed to parse request data")
		span.RecordError(err)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.StringError("failed to parse request data"),
		)
	}

	span.AddEvent("validating request body")
	err = c.Validate(rdata)
	if err != nil {
		span.SetStatus(codes.Error, "failed to validate request data")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, types.ValidationError(err))
	}

	res, err := h.doJobWork(ctx, &rdata.JobArgs)
	if err != nil {
		return err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "processed job")
	return c.JSON(http.StatusOK, res)
}

//gocyclo:ignore
func (h *Handler) RunBulkTests(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "RunBulkTests")
	defer span.End()

	span.AddEvent("received job runner request")

	auth, ok := c.Get("auth").(*models.Auth)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("auth: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	requestTime, ok := c.Get("time").(time.Time)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("time: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}

	span.SetAttributes(
		attribute.String("auth.note", auth.Note),
		attribute.String("auth.id", auth.ID.String()),
		attribute.Int64("request.timestamp_ms", requestTime.UnixMilli()),
	)

	type requestData struct {
		Jobs []types.JobArgs `json:"jobs" validate:"required"`
	}

	var rdata requestData

	span.AddEvent("parsing request body")
	err := c.Bind(&rdata)
	if err != nil {
		span.SetStatus(codes.Error, "failed to parse request data")
		span.RecordError(err)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.StringError("failed to parse request data"),
		)
	}

	span.AddEvent("validating request body")
	err = c.Validate(rdata)
	if err != nil {
		span.SetStatus(codes.Error, "failed to validate request data")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, types.ValidationError(err))
	}

	if len(rdata.Jobs) > 100 || len(rdata.Jobs) < 1 {
		span.SetStatus(codes.Error, "invalid job count")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, types.ValidationError(err))
	}

	jobResponses := make([]*types.JobResponse, 0, len(rdata.Jobs))
	for _, jobRequest := range rdata.Jobs {
		res, err := h.doJobWork(ctx, &jobRequest)
		if err != nil {
			return err
		}

		jobResponses = append(jobResponses, res)
	}
	span.RecordError(nil)
	span.SetStatus(codes.Ok, "processed job")
	return c.JSON(http.StatusOK, jobResponses)
}
