package engine

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePatch(t *testing.T) {
	files, err := parsePatch("d4678cfc9822ca149668dad1552c794d8e1edaac7c1a40d2a08a5702633e9b8f")
	require.NoError(t, err, "failed to read patch file")

	patch, err := os.ReadFile("d4678cfc9822ca149668dad1552c794d8e1edaac7c1a40d2a08a5702633e9b8f")
	require.NoError(t, err, "failed to read patch file")
	require.Contains(t, string(patch), "\r", "should have at least one carriage return")

	for _, file := range files {
		assert.Equal(
			t,
			"pdfbox/src/main/java/org/apache/pdfbox/pdmodel/PDPageTree.java",
			file.NewName,
			"names should match",
		)
	}
}
