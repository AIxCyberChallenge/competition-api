package upload

import (
	"context"
	"io"
	"time"

	"github.com/sethvargo/go-retry"
	"go.opentelemetry.io/otel/codes"
)

// Ensure RetryUploader implements Uploader interface.
var _ Uploader = (*RetryUploader)(nil)

// Meta uploader that wraps uploader operations in backoff loops
type RetryUploader struct {
	uploader Uploader
	backoff  func() retry.Backoff
}

func NewRetryUploaderBackoff(uploader Uploader, backoff func() retry.Backoff) *RetryUploader {
	return &RetryUploader{
		uploader: uploader,
		backoff:  backoff,
	}
}

// For non latency sensitive archiving
func NewRetryUploader(uploader Uploader) *RetryUploader {
	return &RetryUploader{
		uploader: uploader,
		backoff: func() retry.Backoff {
			b := retry.NewExponential(time.Second)
			b = retry.WithMaxDuration(time.Second*120, b)
			return b
		},
	}
}

func (r *RetryUploader) Exists(ctx context.Context, url string) (bool, error) {
	ctx, span := tracer.Start(ctx, "RetryUploader.Exists")
	defer span.End()

	var exists bool
	err := retry.Do(ctx, r.backoff(), func(rctx context.Context) error {
		//nolint:govet // shadow: intentionally shadow ctx and span to avoid using the incorrect one.
		ctx, span := tracer.Start(rctx, "RetryUploader.Exists.Retry")
		defer span.End()

		var err error
		exists, err = r.uploader.Exists(ctx, url)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to get exists")
			return retry.RetryableError(err)
		}

		span.RecordError(nil)
		span.SetStatus(codes.Ok, "successfully retried")
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get exists")
		return false, err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "got exists")
	return exists, nil
}

func (r *RetryUploader) StoreIdentifier(ctx context.Context) (string, error) {
	ctx, span := tracer.Start(ctx, "RetryUploader.StoreIdentifier")
	defer span.End()

	var ident string
	err := retry.Do(ctx, r.backoff(), func(ctx context.Context) error {
		//nolint:govet // shadow: intentionally shadow ctx and span to avoid using the incorrect one.
		ctx, span := tracer.Start(ctx, "RetryUploader.StoreIdentifier.Retry")
		defer span.End()

		var err error
		ident, err = r.uploader.StoreIdentifier(ctx)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to get store identifier")
			return retry.RetryableError(err)
		}

		span.RecordError(nil)
		span.SetStatus(codes.Ok, "successfully retried")
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get store identifier")
		return "", err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "got store identifier")
	return ident, nil
}

func (r *RetryUploader) Upload(
	ctx context.Context,
	reader io.ReadSeeker,
	length int64,
	url string,
) error {
	ctx, span := tracer.Start(ctx, "RetryUploader.Upload")
	defer span.End()

	err := retry.Do(ctx, r.backoff(), func(ctx context.Context) error {
		//nolint:govet // shadow: intentionally shadow ctx and span to avoid using the incorrect one.
		ctx, span := tracer.Start(ctx, "RetryUploader.Upload.Retry")
		defer span.End()

		if _, err := reader.Seek(0, io.SeekStart); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to seek to start of temp file")
			return err
		}

		if err := r.uploader.Upload(ctx, reader, length, url); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to upload")
			return retry.RetryableError(err)
		}

		span.RecordError(nil)
		span.SetStatus(codes.Ok, "successfully retried")
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to upload")
		return err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "got store identifier")
	return nil
}

func (r *RetryUploader) PresignedReadURL(
	ctx context.Context,
	url string,
	duration time.Duration,
) (string, error) {
	ctx, span := tracer.Start(ctx, "RetryUploader.PresignedReadURL")
	defer span.End()

	var presigned string
	//nolint:govet // shadow: intentionally shadow ctx and span to avoid using the incorrect one.
	err := retry.Do(ctx, r.backoff(), func(ctx context.Context) error {
		ctx, span := tracer.Start(ctx, "RetryUploader.PresignedReadURL.Retry")
		defer span.End()

		var err error
		presigned, err = r.uploader.PresignedReadURL(ctx, url, duration)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to get presigned")
			return retry.RetryableError(err)
		}

		span.RecordError(nil)
		span.SetStatus(codes.Ok, "successfully retried")
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get presigned")
		return "", err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "got presigned")
	return presigned, nil
}
