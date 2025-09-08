package otel

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
)

func TestEnvCarrier(t *testing.T) {
	t.Run("ImplementsInterface", func(*testing.T) {
		var _ propagation.TextMapCarrier = CreateEnvCarrier()
	})

	t.Run("RoundTrip", func(t *testing.T) {
		var e propagation.TextMapCarrier = CreateEnvCarrier()

		e.Set("a", "b")

		assert.Equal(t, "b", e.Get("a"), "failed to retrieve set value")
		assert.Equal(t, []string{"a"}, e.Keys(), "failed to get set keys")
	})

	t.Run("FromEnv", func(t *testing.T) {
		var e propagation.TextMapCarrier = CreateEnvCarrier()

		require.NoError(t, os.Setenv(mapKey("a"), "b"))

		assert.Equal(t, "b", e.Get("a"), "failed to get from env")
		assert.Equal(t, []string{"a"}, e.Keys(), "failed to get set keys")
	})
}
