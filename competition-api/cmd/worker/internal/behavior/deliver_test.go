package behavior_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/behavior"
)

var route = "/foo/bar/testing"
var username = "username"
var password = "password"

func basicAuthValidator(reqUser, reqPassword string, _ echo.Context) (bool, error) {
	return reqUser == username && reqPassword == password, nil
}

func createEcho() *echo.Echo {
	e := echo.New()
	e.Use(middleware.BasicAuth(basicAuthValidator))

	return e
}

func TestDeliver(t *testing.T) {
	ctx := context.Background()

	t.Run("Receive Post Data", func(t *testing.T) {
		t.Parallel()

		e := createEcho()

		type payload struct {
			Foo string `json:"foo"`
			Bar string `json:"bar"`
		}

		rawPayload := `{"foo": "foo", "bar": "bar"}`
		expectedPayload := payload{
			Foo: "foo",
			Bar: "bar",
		}

		counter := 0
		e.POST(route, func(c echo.Context) error {
			counter++
			var rdata payload
			err := c.Bind(&rdata)
			require.NoError(t, err, "failed to read payload")

			assert.Equal(t, expectedPayload, rdata, "payloads do not match")

			return c.NoContent(http.StatusOK)
		})

		server := httptest.NewServer(e)
		defer server.Close()
		baseURL, err := url.Parse(server.URL)
		require.NoError(t, err, "failed to parse url")

		err = behavior.Deliver(
			ctx,
			http.MethodPost,
			route,
			&rawPayload,
			time.Now().Add(time.Second*10),
			func(_ *behavior.DeliveryTarget, retries int, err error) {
				require.NoError(t, err, "request should deliver")
				assert.Equal(t, 0, retries, "should take no retries")
			},
			&behavior.DeliveryTarget{
				BaseURL:  baseURL,
				Username: username,
				Password: password,
			},
		)
		require.NoError(t, err, "failed to deliver")

		assert.Equal(t, 1, counter, "counter should be 1")
	})

	t.Run("Receive Delete Data", func(t *testing.T) {
		t.Parallel()

		e := createEcho()

		counter := 0
		e.DELETE(route, func(c echo.Context) error {
			counter++

			return c.NoContent(http.StatusOK)
		})

		server := httptest.NewServer(e)
		defer server.Close()
		baseURL, err := url.Parse(server.URL)
		require.NoError(t, err, "failed to parse url")

		err = behavior.Deliver(
			ctx,
			http.MethodDelete,
			route,
			nil,
			time.Now().Add(time.Second*10),
			func(_ *behavior.DeliveryTarget, retries int, err error) {
				require.NoError(t, err, "request should deliver")
				assert.Equal(t, 0, retries, "should take no retries")
			},
			&behavior.DeliveryTarget{
				BaseURL:  baseURL,
				Username: username,
				Password: password,
			},
		)
		require.NoError(t, err, "failed to deliver")

		assert.Equal(t, 1, counter, "counter should be 1")
	})

	t.Run("Multiple Tries", func(t *testing.T) {
		t.Parallel()

		e := createEcho()

		counter := 0
		e.DELETE(route, func(c echo.Context) error {
			counter++

			if counter < 5 {
				return c.NoContent(http.StatusBadRequest)
			}

			return c.NoContent(http.StatusOK)
		})

		server := httptest.NewServer(e)
		defer server.Close()
		baseURL, err := url.Parse(server.URL)
		require.NoError(t, err, "failed to parse url")

		err = behavior.Deliver(
			ctx,
			http.MethodDelete,
			route,
			nil,
			time.Now().Add(time.Second*10),
			func(_ *behavior.DeliveryTarget, retries int, err error) {
				require.NoError(t, err, "request should deliver")
				assert.Equal(t, 4, retries, "should take 4 retries")
			},
			&behavior.DeliveryTarget{
				BaseURL:  baseURL,
				Username: username,
				Password: password,
			},
		)
		require.NoError(t, err, "failed to deliver")

		assert.Equal(t, 5, counter, "counter should be 1")
	})
}
