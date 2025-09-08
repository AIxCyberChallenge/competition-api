package upload

import (
	"context"
	"io"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/hash"
)

var tracer = otel.Tracer(
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/upload",
)

//go:generate mockgen -destination ./mock/mock.go -package mock . Uploader

// Generic file persistence interface
type Uploader interface {
	// Create / Overwrite file contents by `url` (blobName)
	Upload(ctx context.Context, reader io.ReadSeeker, length int64, url string) error
	// Check if a file exists (focused on preventing uploading the same file multiple times not authoritative existence)
	//
	// May always return false
	Exists(ctx context.Context, url string) (bool, error)
	// Provide an identifier for where files are being uploaded to. Useful for logging and auditing purposes.
	StoreIdentifier(ctx context.Context) (string, error)
	// Anonymous, readonly, internet accessible URL for downloading the file
	PresignedReadURL(ctx context.Context, url string, duration time.Duration) (string, error)
}

// Uploads a buffer where the `url` will be the hash of the contents of `reader` (CAS)
//
// Will:
// 1. seek to 0 so only pass in a buffer you want completely uploaded
// 2. not upload if a file with the same hash already exists
func Hashed(
	ctx context.Context,
	u Uploader,
	reader io.ReadSeeker,
	length int64,
) (string, error) {
	ctx, span := tracer.Start(ctx, "UploadHashed")
	defer span.End()

	_, err := reader.Seek(0, io.SeekStart)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to seek to start")
		return "", err
	}

	hashedContent, err := hash.Reader(ctx, reader)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to hash reader")
		return "", err
	}

	exists, err := u.Exists(ctx, hashedContent)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to check if file exists")
		return "", err
	}

	if exists {
		span.RecordError(nil)
		span.SetStatus(codes.Ok, "found existing file")
		return hashedContent, nil
	}

	_, err = reader.Seek(0, io.SeekStart)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to seek to start")
		return "", err
	}

	err = u.Upload(ctx, reader, length, hashedContent)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to seek to upload file")
		return "", err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "uploaded file by hash")
	return hashedContent, nil
}

// Uploads a file by path where the `url` will be the hash of the contents of `reader` (CAS)
func HashedFile(ctx context.Context, u Uploader, filePath string) (string, error) {
	ctx, span := tracer.Start(ctx, "UploadHashedFile", trace.WithAttributes(
		attribute.String("filePath", filePath),
	))
	defer span.End()

	f, err := os.Open(filePath)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to open file")
		return "", err
	}
	defer f.Close()

	stat, err := os.Stat(filePath)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to stat file")
		return "", err
	}

	content, err := Hashed(ctx, u, f, stat.Size())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to upload")
		return "", err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "uploaded file")
	return content, nil
}
