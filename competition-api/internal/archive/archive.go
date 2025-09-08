package archive

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/audit"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/upload"
)

var tracer = otel.Tracer("github.com/aixcyberchallenge/competition-api/competition-api/internal/archive")

type FileMetadata struct {
	LocalFilePath *string
	Buffer        *[]byte
	ArchivedFile  types.ArchivedFile
	Entity        audit.FileArchivedEntity
	EntityID      string
}

// TODO: use interfaces for file metadata instead of static dispatch
//
//revive:disable-next-line
func ArchiveFile(
	ctx context.Context,
	auditContext audit.Context,
	u upload.Uploader,
	metadata *FileMetadata,
) error { //revive:disable-line:exported
	ctx, span := tracer.Start(ctx, "ArchiveFile")
	defer span.End()

	var buffer io.ReadSeeker
	var size int64

	if metadata.LocalFilePath == nil && metadata.Buffer == nil {
		err := errors.New("tried to archive a file without a buffer or file path")
		span.SetStatus(codes.Error, "can't archive a file without a buffer or file path")
		span.RecordError(err)
		return err
	}

	if metadata.LocalFilePath != nil {
		span.AddEvent("archiving from local file")
		span.SetAttributes(attribute.String("path", *metadata.LocalFilePath))

		fstat, err := os.Stat(*metadata.LocalFilePath)
		if err != nil {
			span.SetStatus(codes.Error, "error statting file")
			span.RecordError(err)
			return err
		}

		span.AddEvent("opening file")
		f, err := os.Open(*metadata.LocalFilePath)
		if err != nil {
			span.SetStatus(codes.Error, "failed to open file for upload")
			span.RecordError(err)
			return err
		}
		defer f.Close()

		size = fstat.Size()
		buffer = f
	} else if metadata.Buffer != nil {
		span.AddEvent("archiving from in-memory buffer")
		buffer = bytes.NewReader(*metadata.Buffer)
		size = int64(len(*metadata.Buffer))
	}

	objectName, err := upload.Hashed(ctx, u, buffer, size)
	if err != nil {
		span.SetStatus(codes.Error, "failed to upload file")
		span.RecordError(err)
		return err
	}

	identifier, err := u.StoreIdentifier(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get identifier")
		return err
	}

	span.AddEvent("generating audit log message")
	audit.LogFileArchived(
		auditContext,
		identifier,
		objectName,
		metadata.ArchivedFile,
		metadata.Entity,
		metadata.EntityID,
	)

	return nil
}
