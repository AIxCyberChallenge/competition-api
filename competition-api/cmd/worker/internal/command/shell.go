package command

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
)

// Ensure ShellExecutor implements Executor interface.
var _ Executor = (*ShellExecutor)(nil)

// Executes the command via fork / subprocess
type ShellExecutor struct{}

func NewShellExecutor() *ShellExecutor {
	return &ShellExecutor{}
}

func (*ShellExecutor) Execute(ctx context.Context, command *Command) (*Result, error) {
	ctx, span := tracer.Start(ctx, "ShellExecutor.Execute", trace.WithAttributes(
		attribute.String("program", command.Program),
		attribute.StringSlice("args", command.Args),
	))
	defer span.End()

	var stdout, stderr bytes.Buffer

	//nolint:gosec // G204: not controllable by sanitizing here; callers should ensure sanitization
	cmd := exec.CommandContext(ctx, command.Program, command.Args...)
	cmd.Stdin = command.Stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.WaitDelay = time.Second

	err := cmd.Run()
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to execute command")
			return nil, err
		}
	}

	stdoutBytes := stdout.Bytes()
	stderrBytes := stderr.Bytes()

	scanner := bufio.NewScanner(bytes.NewReader(stdoutBytes))
	for scanner.Scan() {
		line := scanner.Text()
		logger.Logger.DebugContext(ctx, "stdout", "line", line)
	}
	scanner = bufio.NewScanner(bytes.NewReader(stderrBytes))
	for scanner.Scan() {
		line := scanner.Text()
		logger.Logger.DebugContext(ctx, "stderr", "line", line)
	}

	span.AddEvent("executed", trace.WithAttributes(
		attribute.Int("exitCode", cmd.ProcessState.ExitCode()),
	))

	executed := make([]string, 0, len(command.Args)+1)
	executed = append(executed, command.Program)
	executed = append(executed, command.Args...)

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "successfully executed command")
	return &Result{
		Cmd:      executed,
		Stdout:   stdoutBytes,
		Stderr:   stderrBytes,
		ExitCode: cmd.ProcessState.ExitCode(),
	}, nil
}
