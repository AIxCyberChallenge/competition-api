package fetch_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/fetch"
)

func TestHttp(t *testing.T) {
	ctx := context.Background()

	e := echo.New()
	rootContent := "hello world"
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, rootContent)
	})

	server := httptest.NewServer(e)

	t.Run("ValidPath", func(t *testing.T) {
		expected := []byte(rootContent)
		fetcher := fetch.NewHTTPFetcher(retryablehttp.NewClient().StandardClient())
		body, err := fetcher.Fetch(ctx, fmt.Sprintf("%s/", server.URL))
		require.NoError(t, err, "failed to fetch")
		defer body.Close()

		actual, err := io.ReadAll(body)
		require.NoError(t, err, "failed to read content")

		require.Equal(t, expected, actual, "wrong body fetched")
	})

	t.Run("InvalidPath", func(t *testing.T) {
		fetcher := fetch.NewHTTPFetcher(retryablehttp.NewClient().StandardClient())
		_, err := fetcher.Fetch(ctx, fmt.Sprintf("%s/foobar", server.URL))
		require.Error(t, err, "expected to fail")
	})
}
