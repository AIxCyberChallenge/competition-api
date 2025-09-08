package v1

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	srverr "github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/error"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/response"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

func (h *Handler) Ping(c echo.Context) error {
	_, span := tracer.Start(c.Request().Context(), "Ping")
	defer span.End()

	auth, ok := c.Get("auth").(*models.Auth)
	if !ok {
		span.RecordError(srverr.ErrTypeAssertMismatch)
		span.SetStatus(codes.Error, fmt.Sprintf("auth: %s", srverr.ErrTypeAssertMismatch))
		return response.InternalServerError
	}
	teamID := auth.ID.String()

	span.SetAttributes(
		attribute.String("auth.note", auth.Note),
		attribute.String("auth.id", auth.ID.String()),
		attribute.String("round.id", *h.config.RoundID),
		attribute.String("team.id", teamID),
	)

	span.AddEvent("received ping")

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "")
	return c.JSON(http.StatusOK, types.PingResponse{Status: "ready"})
}
