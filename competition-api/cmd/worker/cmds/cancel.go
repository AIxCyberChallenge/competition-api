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

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/behavior"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/config"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	workererrors "github.com/aixcyberchallenge/competition-api/competition-api/internal/worker_errors"
)

var (
	cancelRoute          string
	cancelCRSCredentials string
	cancelDeadline       int64
)

var cancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel a CRS task",
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx, span := tracer.Start(cmd.Context(), "cancelCmd")
		defer span.End()

		span.SetAttributes(
			attribute.String("route", cancelRoute),
			attribute.Int64("deadline", cancelDeadline),
		)

		logger.Logger.InfoContext(ctx, "Starting cancel", "route", cancelRoute)

		if cancelCRSCredentials == "" {
			err := workererrors.ExitErrorWrap(
				types.ExitErrored,
				errors.New("error env CRS_API_CREDENTIALS required"),
			)
			span.RecordError(err)
			span.SetStatus(codes.Error, "error env CRS_API_CREDENTIALS required")
			return err
		}

		crsCredentialsRaw, err := base64.StdEncoding.DecodeString(cancelCRSCredentials)
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

		logger.Logger.Debug("decoded crs credentials", "count", len(crsCredentials))

		deliveryTargets := behavior.DeliveryTargetsFromTeams(ctx, crsCredentials)
		finisher := func(_ *behavior.DeliveryTarget, _ int, _ error) {}
		err = behavior.Deliver(
			ctx,
			http.MethodDelete,
			cancelRoute,
			nil,
			time.Unix(cancelDeadline, 0),
			finisher,
			deliveryTargets...)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to cancel all crs")
			// we do not reschedule here because the tasks have expired
			return workererrors.ExitErrorWrap(types.ExitErrored, err)
		}

		span.RecordError(nil)
		span.SetStatus(codes.Ok, "cancelled all crs")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cancelCmd)

	cancelCmd.PersistentFlags().StringVarP(&cancelRoute, "route", "r", "", "Route")
	cancelCmd.PersistentFlags().
		Int64VarP(&cancelDeadline, "until", "u", math.MaxInt64, "Deadline (Unix seconds timestamp)")

	for _, flag := range []string{"route", "until"} {
		err := cancelCmd.MarkPersistentFlagRequired(flag)
		if err != nil {
			logger.Logger.Error("error setting flag required", "flag", flag, "error", err)
			os.Exit(1)
		}
	}

	cancelCRSCredentials = os.Getenv("CRS_API_CREDENTIALS")
}
