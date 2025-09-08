package fetch

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Ensure AzureFetcher implements Fetcher interface.
var _ Fetcher = (*AzureFetcher)(nil)

// Azure blob backed fetcher
type AzureFetcher struct {
	az *azblob.Client
	// `container` in the storage count where the files are stored
	container string
}

// `container` must be part of the storage account provided
func NewAzureFetcher(accountName, accountKey, serviceURL, container string) (*AzureFetcher, error) {
	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, err
	}

	client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, cred, &azblob.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Retry: policy.RetryOptions{
				RetryDelay: time.Second,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if container == "" {
		return nil, errors.New("container is required")
	}

	return &AzureFetcher{
		az:        client,
		container: container,
	}, nil
}

// `container` must be part of the storage account provided
func NewAzureFetcherFromClient(client *azblob.Client, container string) *AzureFetcher {
	return &AzureFetcher{
		az:        client,
		container: container,
	}
}

func (a *AzureFetcher) Fetch(ctx context.Context, url string) (io.ReadCloser, error) {
	ctx, span := tracer.Start(ctx, "AzureFetcher.Fetch", trace.WithAttributes(
		attribute.String("url", url),
	))
	defer span.End()

	res, err := a.az.DownloadStream(ctx, a.container, url, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch")
		return nil, err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "fetched url")
	return res.Body, nil
}
