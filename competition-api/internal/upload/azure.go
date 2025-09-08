package upload

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Ensures AzureUploader implements Uploader interface.
var _ Uploader = (*AzureUploader)(nil)

// Azure Blob store backed uploader
type AzureUploader struct {
	client *azblob.Client
	// `container` in the storage account where files are saved
	container string
}

// `container` must be part of the storage account provided
func NewAzureUploader(
	accountName, accountKey, serviceURL, container string,
) (*AzureUploader, error) {
	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, err
	}

	client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
	if err != nil {
		return nil, err
	}

	if container == "" {
		return nil, errors.New("container is required")
	}

	return &AzureUploader{
		client:    client,
		container: container,
	}, nil
}

// `container` must be part of the storage account of `client`
func NewAzureUploaderFromClient(client *azblob.Client, container string) *AzureUploader {
	return &AzureUploader{
		client:    client,
		container: container,
	}
}

func (u *AzureUploader) Upload(
	ctx context.Context,
	reader io.ReadSeeker,
	length int64,
	url string,
) error {
	ctx, span := tracer.Start(ctx, "AzureUploader.Upload", trace.WithAttributes(
		attribute.String("url", url),
		attribute.Int64("length", length),
	))
	defer span.End()

	_, err := u.client.UploadStream(ctx, u.container, url, reader, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to upload reader")
		return err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "uploaded file")
	return nil
}

func (u *AzureUploader) Exists(ctx context.Context, url string) (bool, error) {
	ctx, span := tracer.Start(ctx, "AzureUploader.Exists", trace.WithAttributes(
		attribute.String("url", url),
	))
	defer span.End()

	_, err := u.client.ServiceClient().
		NewContainerClient(u.container).
		NewBlobClient(url).
		GetProperties(ctx, nil)
	if err != nil {
		// we hit an error only if the the error is not a blob not found error
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.ErrorCode == string(bloberror.BlobNotFound) {
			span.RecordError(nil)
			span.SetStatus(codes.Ok, "did not find blob")
			return false, nil
		}

		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to check file exists")
		return false, err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "found blob")
	return true, nil
}

func (u *AzureUploader) StoreIdentifier(_ context.Context) (string, error) {
	return u.container, nil
}

func (u *AzureUploader) PresignedReadURL(
	ctx context.Context,
	url string,
	duration time.Duration,
) (string, error) {
	_, span := tracer.Start(ctx, "AzureUploader.PresignedReadURL", trace.WithAttributes(
		attribute.String("url", url),
		attribute.String("duration", duration.String()),
	))
	defer span.End()

	presigned, err := u.client.ServiceClient().
		NewContainerClient(u.container).
		NewBlobClient(url).
		GetSASURL(sas.BlobPermissions{Read: true}, time.Now().Add(duration), nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get presigned url")
		return "", err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "got presigned url")
	return presigned, nil
}
