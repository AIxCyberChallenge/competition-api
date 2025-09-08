package competition

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	srverr "github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/error"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/response"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/audit"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

func (h *Handler) OutOfBudget(c echo.Context) error {
	_, span := tracer.Start(c.Request().Context(), "OutOfBudget")
	defer span.End()

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

	span.AddEvent("received out of budget request")

	type requestData struct {
		CompetitorID string `json:"competitor_id" validate:"required,uuid_rfc4122"`
	}

	span.AddEvent("parsing request body")
	var rdata requestData

	err := c.Bind(&rdata)
	if err != nil {
		span.SetStatus(codes.Error, "failed to parse request data")
		span.RecordError(err)
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.StringError("failed to parse request data"),
		)
	}

	span.SetAttributes(attribute.String("team.id", rdata.CompetitorID))

	span.AddEvent("validating request body")
	err = c.Validate(rdata)
	if err != nil {
		span.SetStatus(codes.Error, "failed to validate request")
		span.RecordError(err)
		return echo.NewHTTPError(http.StatusBadRequest, types.ValidationError(err))
	}

	span.AddEvent("parsing competitor ID as uuid")
	competitorID, err := uuid.Parse(rdata.CompetitorID)
	if err != nil {
		span.SetStatus(codes.Error, "failed to parse competitor ID")
		span.RecordError(err)
		return response.InternalServerError
	}

	teamIDStr := competitorID.String()
	span.AddEvent("out_of_budget", trace.WithAttributes(attribute.String("team.id", teamIDStr)))
	audit.LogOutOfBudget(audit.Context{RoundID: h.RoundID, TeamID: &teamIDStr})

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "recorded out of budget event")
	return c.JSONBlob(http.StatusOK, []byte(`{"status": "ok"}`))
}
