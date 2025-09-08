package command

import (
	"context"
	"io"

	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer(
	"github.com/aixcyberchallenge/competition-api/competition-api/worker/internal/executor",
)

type Result struct {
	Cmd      []string
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

type Command struct {
	Stdin   io.Reader
	Program string
	Args    []string
}

func New(program string, args ...string) *Command {
	return &Command{
		Program: program,
		Args:    args,
	}
}

//go:generate mockgen -destination ./mock/mock.go -package mock . Executor

type Executor interface {
	Execute(ctx context.Context, cmd *Command) (*Result, error)
}
