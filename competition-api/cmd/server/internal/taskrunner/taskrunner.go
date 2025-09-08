package taskrunner

import (
	"context"
	"errors"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

const name = "github.com/aixcyberchallenge/competition-api/competition-api/server/taskrunner"

var tracer = otel.Tracer(name)

// Provides a wrapper around [sync.WaitGroup] that has [Shutdown] vs timeout racing functionality
type Client struct {
	running sync.WaitGroup
}

func Create() *Client {
	return &Client{}
}

// Invokes the provided function as a go routine while tracking its state.
// This allows for us to wait for our tasks to finish before terminating gracefully.
// This is only as safe as the forceful shutdown timeout.
func (c *Client) Run(ctx context.Context, a func(context.Context)) {
	c.running.Add(1)
	go func() {
		defer c.running.Done()

		//nolint:govet // shadow: intentionally shadow ctx to avoid using the incorrect one.
		ctx, span := tracer.Start(ctx, "Run")
		defer span.End()

		a(context.WithoutCancel(ctx))

		span.RecordError(nil)
		span.SetStatus(codes.Ok, "ran task")
	}()
}

// Will race waiting for all of the tasks finishing and `ctx` becoming "done"
func (c *Client) Shutdown(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "Shutdown")
	defer span.End()

	done := make(chan struct{})
	go func() {
		// is this a leak... do we care
		c.running.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		span.AddEvent("hit_timeout")
		span.RecordError(errors.New("error shutting down in time"))
		span.SetStatus(codes.Error, "error shutting down in time")
		return errors.New("error shutting down in time")
	case <-done:
		span.AddEvent("done")
		span.RecordError(nil)
		span.SetStatus(codes.Ok, "finished shutting down")
		return nil
	}
}
