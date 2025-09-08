package behavior

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sethvargo/go-retry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/config"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
)

type DeliveryTarget struct {
	TeamID   string
	BaseURL  *url.URL
	Username string
	Password string
}

func (d *DeliveryTarget) makeRequestObject(
	ctx context.Context,
	method string,
	route string,
	payload io.Reader,
) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, d.BaseURL.JoinPath(route).String(), payload)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(d.Username, d.Password)

	return req, nil
}

func DeliveryTargetsFromTeams(ctx context.Context, credentials []config.Team) []*DeliveryTarget {
	ctx, span := tracer.Start(ctx, "DeliveryTargetsFromTeams")
	defer span.End()
	deliveryTargets := make([]*DeliveryTarget, 0, len(credentials))
	for _, team := range credentials {
		func() {
			_, span := tracer.Start(ctx, "DeliveryTargetsFromTeams/SingleTeam")
			defer span.End()

			span.SetAttributes(
				attribute.String("id", team.ID),
				attribute.String("note", team.Note),
			)

			if team.CRS == nil || team.CRS.TaskMe == nil || !*team.CRS.TaskMe {
				logger.Logger.WarnContext(ctx, "skipping", "id", team.ID, "note", team.Note)
				span.RecordError(nil)
				span.SetStatus(codes.Ok, "skipped crs")
				return
			}

			baseURL, err := url.Parse(team.CRS.URL)
			if err != nil {
				logger.Logger.ErrorContext(
					ctx,
					"failed to parse",
					"url",
					team.CRS.URL,
					"error",
					err,
				)
				span.RecordError(err)
				span.SetStatus(codes.Error, "failed to parse URL for crs")
				return
			}

			deliveryTargets = append(deliveryTargets, &DeliveryTarget{
				TeamID:   team.ID,
				Username: team.CRS.APIKeyID,
				Password: team.CRS.APIKeyToken,
				BaseURL:  baseURL,
			})
		}()
	}

	return deliveryTargets
}

func Deliver(
	ctx context.Context,
	method string,
	route string,
	payload *string,
	deadline time.Time,
	finisher func(*DeliveryTarget, int, error),
	deliveryTargets ...*DeliveryTarget,
) error {
	ctx, span := tracer.Start(ctx, "Deliver", trace.WithAttributes(
		attribute.String("method", method),
		attribute.String("route", route),
		attribute.String("deadline", deadline.String()),
	))
	defer span.End()

	if payload != nil {
		logger.Logger.DebugContext(ctx, "sending payload", "payload", *payload)
	}

	errs := make([]error, len(deliveryTargets))
	wg := sync.WaitGroup{}
	wg.Add(len(deliveryTargets))
	for i, deliveryTarget := range deliveryTargets {
		go func() {
			defer wg.Done()
			errs[i] = deliverOne(
				ctx,
				method,
				route,
				payload,
				deadline,
				finisher,
				deliveryTarget,
			)
		}()
	}
	wg.Wait()
	err := errors.Join(errs...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to deliver to at least one target")
		return err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "delivered to each target")
	return nil
}

func deliverOne(
	ctx context.Context,
	method string,
	route string,
	payload *string,
	deadline time.Time,
	finisher func(*DeliveryTarget, int, error),
	deliveryTarget *DeliveryTarget,
) error {
	ctx, span := tracer.Start(ctx, "deliverOne", trace.WithAttributes(
		attribute.String("method", method),
		attribute.String("route", route),
		attribute.String("deadline", deadline.String()),
		attribute.String("target.baseURL", deliveryTarget.BaseURL.String()),
		attribute.String("target.username", deliveryTarget.Username),
	))
	defer span.End()

	b := retry.NewFibonacci(time.Second)
	b = retry.WithCappedDuration(time.Second*60, b)
	b = retry.WithMaxDuration(time.Until(deadline)+time.Second*60*2, b)

	retryCount := -1

	err := retry.Do(ctx, b, func(ctx context.Context) error {
		return retryDeliver(ctx, method, route, payload, &retryCount, deliveryTarget)
	})
	finisher(deliveryTarget, retryCount, err)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to deliver payload")
		return err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "delivered payload")
	return nil
}

func retryDeliver(
	ctx context.Context,
	method string,
	route string,
	payload *string,
	retryCount *int,
	deliveryTarget *DeliveryTarget,
) error {
	*retryCount++
	ctx, span := tracer.Start(ctx, "retryDeliver", trace.WithAttributes(
		attribute.String("method", method),
		attribute.String("route", route),
		attribute.Int("retryCount", *retryCount),
	))
	defer span.End()

	var reader io.Reader
	if payload != nil {
		reader = strings.NewReader(*payload)
	}
	req, err := deliveryTarget.makeRequestObject(ctx, method, route, reader)
	if err != nil {
		logger.Logger.ErrorContext(ctx, "failed to construct request", "error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to construct request")
		return retry.RetryableError(
			fmt.Errorf("failed to construct request: %w", err),
		)
	}

	logger.Logger.DebugContext(ctx, "sending request", "retryCount", retryCount)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Logger.ErrorContext(ctx,
			"failed to send request",
			"error",
			err,
			"retryCount",
			retryCount,
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to send request")
		return retry.RetryableError(fmt.Errorf("failed to send request: %w", err))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		body = []byte{}
		logger.Logger.WarnContext(ctx, "failed to read body", "error", err)
	}

	span.AddEvent("delivered_payload", trace.WithAttributes(
		attribute.Int("statusCode", resp.StatusCode),
		attribute.String("body", string(body)),
	))

	// technically api specified only one of these but for compatibility /shrug
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Logger.ErrorContext(ctx,
			"got invalid status code retrying",
			"code",
			resp.StatusCode,
			"retryCount",
			retryCount,
		)
		span.RecordError(fmt.Errorf("got invalid status code(%d)", resp.StatusCode))
		span.SetStatus(codes.Error, "got invalid status code")
		return retry.RetryableError(
			fmt.Errorf("got invalid status code(%d)", resp.StatusCode),
		)
	}
	span.RecordError(nil)
	span.SetStatus(codes.Ok, "delivered payload successfully")
	return nil
}
