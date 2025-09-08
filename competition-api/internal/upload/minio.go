package upload

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Ensure MinioUploader implements Uploader interface.
var _ Uploader = (*MinioUploader)(nil)

// Minio (S3) backed uploader
type MinioUploader struct {
	client *minio.Client
	bucket string
}

func NewMinioUploader(
	endpoint, id, secret string,
	ssl bool,
	bucket string,
) (*MinioUploader, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(id, secret, ""),
		Secure: ssl,
	})
	if err != nil {
		return nil, err
	}

	return &MinioUploader{
		client: client,
		bucket: bucket,
	}, nil
}

func NewMinioUploaderFromClient(client *minio.Client, bucket string) *MinioUploader {
	return &MinioUploader{
		client: client,
		bucket: bucket,
	}
}

func (u *MinioUploader) Upload(
	ctx context.Context,
	reader io.ReadSeeker,
	length int64,
	url string,
) error {
	ctx, span := tracer.Start(ctx, "MinioUploader.Upload", trace.WithAttributes(
		attribute.String("url", url),
		attribute.Int64("length", length),
	))
	defer span.End()

	_, err := u.client.PutObject(ctx, u.bucket, url, reader, length, minio.PutObjectOptions{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to put object")
		return err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "put object")
	return nil
}

func (u *MinioUploader) Exists(ctx context.Context, url string) (bool, error) {
	ctx, span := tracer.Start(ctx, "MinioUploader.Exists", trace.WithAttributes(
		attribute.String("url", url),
	))
	defer span.End()

	_, err := u.client.StatObject(ctx, u.bucket, url, minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			span.RecordError(nil)
			span.SetStatus(codes.Ok, "did not find object")
			return false, nil
		}

		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to stat object")
		return false, err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "statted object")
	return true, nil
}

func (u *MinioUploader) StoreIdentifier(_ context.Context) (string, error) {
	return u.bucket, nil
}

func (u *MinioUploader) PresignedReadURL(
	ctx context.Context,
	url string,
	duration time.Duration,
) (string, error) {
	ctx, span := tracer.Start(ctx, "MinioUploader.PresignedReadURL", trace.WithAttributes(
		attribute.String("url", url),
		attribute.String("duration", duration.String()),
	))
	defer span.End()

	presigned, err := u.client.PresignedGetObject(ctx, u.bucket, url, duration, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get presigned url")
		return "", err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "got presigned url")
	return presigned.String(), nil
}
