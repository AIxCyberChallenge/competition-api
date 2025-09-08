package cmds

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/behavior"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/audit"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/config"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	workererrors "github.com/aixcyberchallenge/competition-api/competition-api/internal/worker_errors"
)

var (
	broadcastRoute          string
	broadcastRoundID        string
	broadcastTaskID         string
	broadcastCRSCredentials string
	broadcastPayload        string
	broadcastDeadline       int64
)

// 1 is unrecoverable error 2 is ran out of tries submitting. we consider both a failure but that behavior could change later
var broadcastCmd = &cobra.Command{
	Use:   "broadcast",
	Short: "Broadcast to a CRS",
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx, span := tracer.Start(cmd.Context(), "broadcastCmd")
		defer span.End()

		span.SetAttributes(
			attribute.String("route", broadcastRoute),
			attribute.String("round.id", broadcastRoundID),
			attribute.String("task.id", broadcastTaskID),
			attribute.Int64("deadline", broadcastDeadline),
		)

		logger.Logger.InfoContext(ctx,
			"Starting broadcast",
			"route",
			broadcastRoute,
			"payload",
			broadcastPayload,
		)

		if broadcastCRSCredentials == "" {
			err := workererrors.ExitErrorWrap(
				types.ExitErrored,
				errors.New("error env CRS_API_CREDENTIALS required"),
			)
			span.RecordError(err)
			span.SetStatus(codes.Error, "error env CRS_API_CREDENTIALS required")
			return err
		}

		payload, err := base64.StdEncoding.DecodeString(broadcastPayload)
		if err != nil {
			err = workererrors.ExitErrorWrap(types.ExitErrored, err)
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to decode payload")
			return err
		}

		logger.Logger.DebugContext(ctx, "payload decoded", "payload", string(payload))
		span.AddEvent("decoded_payload", trace.WithAttributes(
			attribute.String("payload", string(payload)),
		))

		crsCredentialsRaw, err := base64.StdEncoding.DecodeString(broadcastCRSCredentials)
		if err != nil {
			err = workererrors.ExitErrorWrap(types.ExitErrored, err)
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to decode credentials")
			return err
		}

		var crsCredentials []config.Team
		err = json.Unmarshal(crsCredentialsRaw, &crsCredentials)
		if err != nil {
			err = workererrors.ExitErrorWrap(types.ExitErrored, err)
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to decode credentials")
			return err
		}

		logger.Logger.DebugContext(ctx, "decoded crs credentials", "count", len(crsCredentials))
		span.AddEvent("decoded_crs_credentials", trace.WithAttributes(
			attribute.Int("count", len(crsCredentials)),
		))

		deliveryTargets := behavior.DeliveryTargetsFromTeams(ctx, crsCredentials)
		payloadString := string(payload)
		finisher := func(tgt *behavior.DeliveryTarget, retries int, err error) {
			c := audit.Context{
				RoundID: broadcastRoundID,
				TaskID:  &broadcastTaskID,
				TeamID:  &tgt.TeamID,
			}
			if err != nil {
				audit.LogBroadcastFailed(c, payloadString, retries)
				return
			}

			audit.LogBroadcastSucceeded(c, retries)
			logger.Logger.WarnContext(ctx, "broadcast_job_sent_successfully")
		}
		err = behavior.Deliver(
			ctx,
			http.MethodPost,
			broadcastRoute,
			&payloadString,
			time.Unix(broadcastDeadline, 0),
			finisher,
			deliveryTargets...)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to broadcast to all crs")
			// we do not reschedule here because the tasks have expired
			return workererrors.ExitErrorWrap(types.ExitErrored, err)
		}

		span.RecordError(nil)
		span.SetStatus(codes.Ok, "broadcast to all crs")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(broadcastCmd)

	broadcastCmd.PersistentFlags().StringVarP(&broadcastRoute, "route", "r", "", "Route")
	broadcastCmd.PersistentFlags().StringVarP(&broadcastRoundID, "round-id", "d", "", "Round ID")
	broadcastCmd.PersistentFlags().StringVarP(&broadcastTaskID, "task-id", "t", "", "Task ID")
	broadcastCmd.PersistentFlags().StringVarP(&broadcastPayload, "payload", "p", "", "Payload")
	broadcastCmd.PersistentFlags().
		Int64VarP(&broadcastDeadline, "until", "u", math.MaxInt64, "Deadline (Unix seconds timestamp)")

	for _, flag := range []string{"route", "round-id", "task-id", "payload", "until"} {
		err := broadcastCmd.MarkPersistentFlagRequired(flag)
		if err != nil {
			logger.Logger.Error("error setting flag required", "flag", flag, "error", err)
			os.Exit(1)
		}
	}

	broadcastCRSCredentials = os.Getenv("CRS_API_CREDENTIALS")
}
