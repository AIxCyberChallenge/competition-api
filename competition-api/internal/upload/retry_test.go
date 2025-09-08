package upload_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/sethvargo/go-retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/upload"
	mockuploader "github.com/aixcyberchallenge/competition-api/competition-api/internal/upload/mock"
)

func TestStoreIdentifier(t *testing.T) {
	t.Run("NoError", func(t *testing.T) {
		ctx := context.Background()
		expected := "identifier"

		ctrl := gomock.NewController(t)
		u := mockuploader.NewMockUploader(ctrl)

		u.EXPECT().StoreIdentifier(gomock.Any()).Return(expected, nil).Times(1)

		retry := upload.NewRetryUploader(u)
		actual, err := retry.StoreIdentifier(ctx)

		require.NoError(t, err, "failed to get store identifier")

		assert.Equal(t, expected, actual, "not matching identifier")
	})

	t.Run("ErrorAfter1Try", func(t *testing.T) {
		ctx := context.Background()
		expected := "identifier"

		ctrl := gomock.NewController(t)
		u := mockuploader.NewMockUploader(ctrl)

		counter := new(int)
		u.EXPECT().
			StoreIdentifier(gomock.Any()).
			DoAndReturn(func(_ context.Context) (string, error) {
				*counter++
				if *counter == 2 {
					return expected, nil
				}

				return "", errors.New("expected error")
			}).
			Times(2)

		retry := upload.NewRetryUploader(u)
		actual, err := retry.StoreIdentifier(ctx)

		require.NoError(t, err, "failed to get store identifier")

		assert.Equal(t, expected, actual, "not matching identifier")
	})

	t.Run("Error", func(t *testing.T) {
		ctx := context.Background()

		ctrl := gomock.NewController(t)
		u := mockuploader.NewMockUploader(ctrl)

		u.EXPECT().StoreIdentifier(gomock.Any()).Return("", errors.New("expected error")).Times(4)

		retry := upload.NewRetryUploaderBackoff(u, func() retry.Backoff {
			b := retry.NewConstant(time.Millisecond * 10)
			b = retry.WithMaxRetries(3, b)
			return b
		})
		_, err := retry.StoreIdentifier(ctx)

		require.Error(t, err, "somehow did not get error")
	})
}

func TestUpload(t *testing.T) {
	t.Run("NoError", func(t *testing.T) {
		ctx := context.Background()

		ctrl := gomock.NewController(t)
		u := mockuploader.NewMockUploader(ctrl)

		reader := strings.NewReader("hello there")
		url := "url"

		u.EXPECT().
			Upload(gomock.Any(), gomock.Any(), gomock.Eq(int64(reader.Len())), gomock.Eq(url)).
			Return(nil).
			Times(1)

		retry := upload.NewRetryUploader(u)
		err := retry.Upload(ctx, reader, int64(reader.Len()), url)

		require.NoError(t, err, "failed to get store identifier")
	})

	t.Run("ErrorAfter1Try", func(t *testing.T) {
		ctx := context.Background()

		ctrl := gomock.NewController(t)
		u := mockuploader.NewMockUploader(ctrl)

		reader := strings.NewReader("hello there")
		url := "url"

		counter := new(int)
		u.EXPECT().
			Upload(gomock.Any(), gomock.Any(), gomock.Eq(int64(reader.Len())), gomock.Eq(url)).
			DoAndReturn(func(_ context.Context, _ io.Reader, _ int64, _ string) error {
				*counter++
				if *counter == 2 {
					return nil
				}

				return errors.New("expected error")
			}).
			Times(2)

		retry := upload.NewRetryUploader(u)
		err := retry.Upload(ctx, reader, int64(reader.Len()), url)

		require.NoError(t, err, "failed to upload")
	})

	t.Run("Error", func(t *testing.T) {
		ctx := context.Background()

		ctrl := gomock.NewController(t)
		u := mockuploader.NewMockUploader(ctrl)

		reader := strings.NewReader("hello there")
		url := "url"

		u.EXPECT().
			Upload(gomock.Any(), gomock.Any(), gomock.Eq(int64(reader.Len())), gomock.Eq(url)).
			Return(errors.New("expected error")).
			Times(4)

		retry := upload.NewRetryUploaderBackoff(u, func() retry.Backoff {
			b := retry.NewConstant(time.Millisecond * 10)
			b = retry.WithMaxRetries(3, b)
			return b
		})
		err := retry.Upload(ctx, reader, int64(reader.Len()), url)

		require.Error(t, err, "somehow uploaded")
	})
}

func TestExists(t *testing.T) {
	t.Run("NoErrorExists", func(t *testing.T) {
		ctx := context.Background()

		ctrl := gomock.NewController(t)
		u := mockuploader.NewMockUploader(ctrl)

		url := "url"
		expected := true

		u.EXPECT().Exists(gomock.Any(), gomock.Eq(url)).Return(expected, nil).Times(1)

		retry := upload.NewRetryUploader(u)
		actual, err := retry.Exists(ctx, url)

		require.NoError(t, err, "failed to get exists")

		assert.Equal(t, expected, actual, "did not get expected")
	})

	t.Run("NoErrorNotExists", func(t *testing.T) {
		ctx := context.Background()

		ctrl := gomock.NewController(t)
		u := mockuploader.NewMockUploader(ctrl)

		url := "url"
		expected := false

		u.EXPECT().Exists(gomock.Any(), gomock.Eq(url)).Return(expected, nil).Times(1)

		retry := upload.NewRetryUploader(u)
		actual, err := retry.Exists(ctx, url)
		require.NoError(t, err, "failed to get exists")

		assert.Equal(t, expected, actual, "did not get expected")
	})

	t.Run("ErrorAfter1Try", func(t *testing.T) {
		ctx := context.Background()

		ctrl := gomock.NewController(t)
		u := mockuploader.NewMockUploader(ctrl)

		expected := true
		url := "url"

		counter := new(int)
		u.EXPECT().
			Exists(gomock.Any(), gomock.Eq(url)).
			DoAndReturn(func(_ context.Context, _ string) (bool, error) {
				*counter++
				if *counter == 2 {
					return expected, nil
				}

				return false, errors.New("expected error")
			}).
			Times(2)

		retry := upload.NewRetryUploader(u)
		actual, err := retry.Exists(ctx, url)
		require.NoError(t, err, "failed to get exists")

		assert.Equal(t, expected, actual, "did not get expected")
	})

	t.Run("Error", func(t *testing.T) {
		ctx := context.Background()

		ctrl := gomock.NewController(t)
		u := mockuploader.NewMockUploader(ctrl)

		url := "url"

		u.EXPECT().
			Exists(gomock.Any(), gomock.Eq(url)).
			Return(false, errors.New("expected error")).
			Times(4)

		retry := upload.NewRetryUploaderBackoff(u, func() retry.Backoff {
			b := retry.NewConstant(time.Millisecond * 10)
			b = retry.WithMaxRetries(3, b)
			return b
		})
		_, err := retry.Exists(ctx, url)

		require.Error(t, err, "somehow exists")
	})
}
