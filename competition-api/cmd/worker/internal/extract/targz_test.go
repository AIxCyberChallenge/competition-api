package extract_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/command"
	mockexecutor "github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/command/mock"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/extract"
)

func TestConcrete(t *testing.T) {
	ctx := context.Background()

	t.Run("Valid", func(t *testing.T) {
		outDir := t.TempDir()

		extractor := extract.NewTarGzExtractor(command.NewShellExecutor())
		tar, err := os.Open("./test.tar.gz")
		require.NoError(t, err, "failed to open tar")
		defer tar.Close()

		err = extractor.Extract(ctx, tar, outDir)
		require.NoError(t, err, "failed to execute unpack")

		entries, err := os.ReadDir(outDir)
		require.NoError(t, err, "failed to read unpacked dir")

		names := make([]string, len(entries))
		for i, entry := range entries {
			fmt.Println(entry.Name())
			names[i] = entry.Name()
		}

		require.Contains(t, names, "dind", "did not find child dir")

		contents, err := os.ReadFile(filepath.Join(outDir, "dind", "Containerfile"))
		require.NoError(t, err, "failed to read Containerfile")

		assert.Contains(t, string(contents), "FROM", "missing expected fragment")
	})

	t.Run("Invalid Outdir", func(t *testing.T) {
		outDir := "foobar"
		extractor := extract.NewTarGzExtractor(command.NewShellExecutor())
		tar, err := os.Open("./test.tar.gz")
		require.NoError(t, err, "failed to open tar")
		defer tar.Close()

		err = extractor.Extract(ctx, tar, outDir)
		require.Error(t, err, "should fail to unpack")
	})

	t.Run("Not a Tar", func(t *testing.T) {
		outDir := t.TempDir()

		extractor := extract.NewTarGzExtractor(command.NewShellExecutor())
		tar := bytes.NewBufferString("hello world")
		err := extractor.Extract(ctx, tar, outDir)
		assert.Error(t, err, "should fail")
	})
}

func TestMock(t *testing.T) {
	ctx := context.Background()

	t.Run("Command Failed to Execute", func(t *testing.T) {
		outDir := t.TempDir()

		ctrl := gomock.NewController(t)
		executor := mockexecutor.NewMockExecutor(ctrl)
		executor.EXPECT().
			Execute(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("failed to execute")).
			Times(1)
		extractor := extract.NewTarGzExtractor(executor)
		tar, err := os.Open("./test.tar.gz")
		require.NoError(t, err, "failed to open tar")
		defer tar.Close()

		err = extractor.Extract(ctx, tar, outDir)
		require.Error(t, err, "expecting error")
	})
}
