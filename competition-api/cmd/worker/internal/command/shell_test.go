package command_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/command"
)

func TestExecute(t *testing.T) {
	t.Run("ZeroExitCode", func(t *testing.T) {
		ctx := context.Background()
		shell := command.NewShellExecutor()

		expected := &command.Result{
			Cmd:      []string{"echo", "-n", "a"},
			Stdout:   []byte("a"),
			Stderr:   []byte{},
			ExitCode: 0,
		}

		cmd := command.New("echo", "-n", "a")
		result, err := shell.Execute(ctx, cmd)
		require.NoError(t, err, "failed to run command")
		assert.Equal(t, expected, result, "command result did not match")
	})

	t.Run("NonzeroExitCode", func(t *testing.T) {
		ctx := context.Background()
		shell := command.NewShellExecutor()

		expected := &command.Result{
			Cmd:    []string{"grep", "-y"},
			Stdout: []byte{},
			Stderr: []byte(`Usage: grep [OPTION]... PATTERNS [FILE]...
Try 'grep --help' for more information.
`),
			ExitCode: 2,
		}

		cmd := command.New("grep", "-y")
		result, err := shell.Execute(ctx, cmd)
		require.NoError(t, err, "failed to run command")
		assert.Equal(t, expected, result, "command result did not match")
	})

	t.Run("Cancel context graceful shutdown", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
		defer cancel()

		shell := command.NewShellExecutor()

		cmd := command.New("sleep", "10")
		result, err := shell.Execute(ctx, cmd)
		require.NoError(t, err, "context cancel sets return code -1")
		assert.Equal(t, -1, result.ExitCode, "context cancel sets return code to -1")
	})
}
