package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Ensure HTTPFetcher implements Fetcher interface.
var _ Fetcher = (*HTTPFetcher)(nil)

type HTTPFetcher struct {
	client *http.Client
}

func NewHTTPFetcher(client *http.Client) *HTTPFetcher {
	return &HTTPFetcher{
		client: client,
	}
}

func (f *HTTPFetcher) Fetch(ctx context.Context, url string) (io.ReadCloser, error) {
	ctx, span := tracer.Start(ctx, "HTTPFetcher.Fetch", trace.WithAttributes(
		attribute.String("url", url),
	))
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to construct request")
		return nil, err
	}
	resp, err := f.client.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to download file")
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("invalid status code: %d", resp.StatusCode)
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid status code")
		return nil, err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "fetched file by http")
	return resp.Body, nil
}
