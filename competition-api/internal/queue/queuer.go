package queue

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer(
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/queuer",
)

//go:generate mockgen -destination ./mock/mock.go -package mock . Queuer,MessageHandler

// Generic tasking interface for enqueuing or dequeuing work
type Queuer interface {
	// May block while queuing data
	Enqueue(ctx context.Context, message any) error
	// May block while waiting for data to dequeue
	//
	// If handler returns poison error message should not be requeued, other errors are non fatal for a message.
	Dequeue(ctx context.Context, timeout time.Duration, handler MessageHandler) error
}

type MessageHandler interface {
	Handle(ctx context.Context, message []byte) error
}

// Mark a message as unprocessable. It will not be requeued.
type PoisonError struct {
	Err error
}

func (p PoisonError) Error() string {
	return fmt.Sprintf("Poisoned message: %v", p.Err)
}

func (p PoisonError) Unwrap() error {
	return p.Err
}

func WrapPoisonError(err error) error {
	return &PoisonError{Err: err}
}
