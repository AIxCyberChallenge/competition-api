package engine_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sethvargo/go-retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/engine"
	mockengine "github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/engine/mock"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	workererrors "github.com/aixcyberchallenge/competition-api/competition-api/internal/worker_errors"
)

var params = &engine.Params{}

func TestBuild(t *testing.T) {
	t.Run("Passed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockEngine := mockengine.NewMockEngine(ctrl)

		mockEngine.EXPECT().Build(gomock.Any(), gomock.Eq(params)).Return(nil).Times(1)

		retryEngine := engine.NewBuildRetryEngineWithFactory(mockEngine, func() retry.Backoff {
			b := retry.NewConstant(time.Millisecond * 100)
			b = retry.WithMaxRetries(3, b)
			return b
		})

		err := retryEngine.Build(context.TODO(), params)
		assert.NoError(t, err, "got unexpected error")
	})

	t.Run("AllFailed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockEngine := mockengine.NewMockEngine(ctrl)

		mockEngine.EXPECT().
			Build(gomock.Any(), gomock.Eq(params)).
			Return(workererrors.StatusErrorWrap(types.SubmissionStatusFailed, false, engine.ErrBuildingFailed)).
			Times(4)

		retryEngine := engine.NewBuildRetryEngineWithFactory(mockEngine, func() retry.Backoff {
			b := retry.NewConstant(time.Millisecond * 100)
			b = retry.WithMaxRetries(3, b)
			return b
		})

		err := retryEngine.Build(context.TODO(), params)
		require.Error(t, err, "missing expected error")

		var se workererrors.StatusError
		require.ErrorAs(t, err, &se, "expected a status error when all are failed")

		assert.Equal(
			t,
			types.SubmissionStatusFailed,
			se.Status,
			"expected failed when all are failed",
		)
	})

	t.Run("AllErrored", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockEngine := mockengine.NewMockEngine(ctrl)

		mockEngine.EXPECT().
			Build(gomock.Any(), gomock.Eq(params)).
			Return(errors.New("random error")).
			Times(4)

		retryEngine := engine.NewBuildRetryEngineWithFactory(mockEngine, func() retry.Backoff {
			b := retry.NewConstant(time.Millisecond * 100)
			b = retry.WithMaxRetries(3, b)
			return b
		})

		err := retryEngine.Build(context.TODO(), params)
		require.Error(t, err, "missing expected error")

		var se workererrors.StatusError
		require.NotErrorAs(
			t,
			err,
			&se,
			"do not expect a status error if there were any random errors",
		)
	})

	t.Run("MixedFailedAndErrored", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockEngine := mockengine.NewMockEngine(ctrl)

		count := new(int)
		mockEngine.EXPECT().
			Build(gomock.Any(), gomock.Eq(params)).
			DoAndReturn(func(_ any, _ any) error {
				*count++
				if *count < 2 {
					return workererrors.StatusErrorWrap(
						types.SubmissionStatusFailed,
						false,
						engine.ErrBuildingFailed,
					)
				}

				return errors.New("random error")
			}).
			Times(4)

		retryEngine := engine.NewBuildRetryEngineWithFactory(mockEngine, func() retry.Backoff {
			b := retry.NewConstant(time.Millisecond * 100)
			b = retry.WithMaxRetries(3, b)
			return b
		})

		err := retryEngine.Build(context.Background(), params)
		require.Error(t, err, "missing expected error")

		var se workererrors.StatusError
		require.NotErrorAs(
			t,
			err,
			&se,
			"do not expect a status error if there were any random errors",
		)
	})

	t.Run("FailedErroredPassed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockEngine := mockengine.NewMockEngine(ctrl)

		count := new(int)
		mockEngine.EXPECT().
			Build(gomock.Any(), gomock.Eq(params)).
			DoAndReturn(func(_ any, _ any) error {
				*count++

				switch *count {
				case 1:
					return workererrors.StatusErrorWrap(
						types.SubmissionStatusFailed,
						false,
						engine.ErrBuildingFailed,
					)
				case 2:
					return errors.New("random err")
				default:
					return nil
				}
			}).
			Times(3)

		retryEngine := engine.NewBuildRetryEngineWithFactory(mockEngine, func() retry.Backoff {
			b := retry.NewConstant(time.Millisecond * 100)
			b = retry.WithMaxRetries(3, b)
			return b
		})

		err := retryEngine.Build(context.TODO(), params)
		assert.NoError(t, err, "got unexpected error")
	})
}
