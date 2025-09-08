package middleware

import (
	"time"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Sets a fixed time as the authoritative time for a request being received
func Time(key string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			_, span := tracer.Start(c.Request().Context(), "Time", trace.WithAttributes(
				attribute.String("key", key),
			))
			defer span.End()

			t := time.Now()
			c.Set(key, t)

			span.AddEvent("set_time", trace.WithAttributes(
				attribute.String("time", t.String()),
			))

			span.RecordError(nil)
			span.SetStatus(codes.Ok, "set time")
			return next(c)
		}
	}
}
