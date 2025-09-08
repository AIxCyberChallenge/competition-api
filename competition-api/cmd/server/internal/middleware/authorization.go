package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"reflect"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

// Checks that all `needed` permissions are present on `has`
func hasPermission(
	ctx context.Context,
	needed *models.Permissions,
	has *models.Permissions,
	l *slog.Logger,
) bool {
	ctx, span := tracer.Start(ctx, "hasPermission")
	defer span.End()

	logger.Logger.DebugContext(ctx, "comparing permissions", "needed", *needed, "has", *has)

	// Reflection feels kinda wrong but I dont know how to codegen in golang
	// and I do not want a static check where we forget to add a new field
	//
	// Other option is weaker typesafety using maps
	valNeeded := reflect.Indirect(reflect.ValueOf(needed))
	valHas := reflect.Indirect(reflect.ValueOf(has))

	typNeeded := valNeeded.Type()
	typHas := valHas.Type()

	if typNeeded != typHas {
		span.RecordError(nil)
		span.SetStatus(codes.Ok, "non matching types")
		return false
	}

	for i := range valNeeded.NumField() {
		fieldNeeded := valNeeded.Field(i)
		fieldHas := valHas.Field(i)

		if fieldNeeded.Kind() != reflect.Bool || fieldHas.Kind() != reflect.Bool {
			l.WarnContext(ctx, "non boolean fields on permissions skipping")
			continue
		}

		// if we need it but dont have
		if fieldNeeded.Bool() && !fieldHas.Bool() {
			l.DebugContext(ctx, "missing permission")
			span.RecordError(nil)
			span.SetStatus(codes.Ok, "missing permission")
			return false
		}
	}

	l.DebugContext(ctx, "granting access")
	span.RecordError(nil)
	span.SetStatus(codes.Ok, "granting access")
	return true
}

// Auth on the correct must contain all of the permissions set to true on the provided `permissions`
func HasPermissions(authKey string, permissions *models.Permissions) echo.MiddlewareFunc {
	l := logger.Logger.With("authKey", authKey, "permissions", permissions)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx, span := tracer.Start(c.Request().Context(), "HasPermissions", trace.WithAttributes(
				attribute.String("authKey", authKey),
			))
			defer span.End()

			l.DebugContext(ctx, "getting auth object")
			auth, ok := c.Get(authKey).(*models.Auth)
			if !ok {
				l.WarnContext(ctx, "failed to get auth object")
				span.RecordError(nil)
				span.SetStatus(codes.Error, "failed to get auth object")
				return echo.NewHTTPError(http.StatusUnauthorized, types.StringError("Unauthorized"))
			}

			comparison := hasPermission(ctx, permissions, &auth.Permissions, l)
			if !comparison {
				span.RecordError(nil)
				span.SetStatus(codes.Ok, "unauthorized")
				return echo.NewHTTPError(http.StatusUnauthorized, types.StringError("Unauthorized"))
			}

			span.RecordError(nil)
			span.SetStatus(codes.Ok, "checked permissions")
			return next(c)
		}
	}
}
