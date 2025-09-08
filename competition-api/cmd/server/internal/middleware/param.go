package middleware

import (
	"errors"
	"reflect"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/response"
)

// Retrieves object from the db based on the id in the `paramName`
func PopulateFromIDParam[T models.CompetitionAPIModel](
	h *Handler,
	paramName string,
	contextName string,
) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx, span := tracer.Start(c.Request().Context(), "PopulateFromIDParam")
			defer span.End()

			span.SetAttributes(
				attribute.String("paramName", paramName),
				attribute.String("contextName", contextName),
				attribute.String("type", reflect.TypeOf((*T)(nil)).Elem().String()),
			)

			db := h.DB.WithContext(ctx)

			rawID := c.Param(paramName)

			span.SetAttributes(
				attribute.String("id.raw", rawID),
			)

			span.AddEvent("parsing rawID into uuid")
			id, err := uuid.Parse(rawID)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "failed to parse rawID into a UUID")
				return response.NotFoundError
			}

			span.SetAttributes(
				attribute.String("id.parsed", id.String()),
			)

			span.AddEvent("fetching object by id")
			data, err := models.ByID[T](ctx, db, id)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "failed to fetch object from db by id")

				if errors.Is(err, gorm.ErrRecordNotFound) {
					return response.NotFoundError
				}

				return response.InternalServerError
			}

			c.Set(contextName, data)

			span.RecordError(nil)
			span.SetStatus(codes.Ok, "fetched object by id")
			return next(c)
		}
	}
}
