package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Azure storage queues backed queuer
type AzureQueuer struct {
	az *azqueue.QueueClient
}

var _ Queuer = (*AzureQueuer)(nil)

// `queueName` must exist in the storage account
func NewAzureQueuer(storageAccountName string,
	storageAccountKey string,
	queueServiceURL string,
	queueName string,
) (*AzureQueuer, error) {
	azureCred, err := azqueue.NewSharedKeyCredential(storageAccountName, storageAccountKey)
	if err != nil {
		return nil, err
	}
	serviceClient, err := azqueue.NewServiceClientWithSharedKeyCredential(
		queueServiceURL,
		azureCred,
		&azqueue.ClientOptions{
			ClientOptions: policy.ClientOptions{
				Retry: policy.RetryOptions{
					MaxRetries: 5,
					RetryDelay: 500 * time.Millisecond,
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}

	client := serviceClient.NewQueueClient(queueName)

	return &AzureQueuer{az: client}, nil
}

func (q AzureQueuer) Enqueue(ctx context.Context, message any) error {
	ctx, span := tracer.Start(ctx, "Azure.Enqueue")
	defer span.End()

	msgJSON, err := json.Marshal(message)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to marshal message")
		return err
	}

	span.AddEvent("serialized_message", trace.WithAttributes(
		attribute.String("message", string(msgJSON)),
	))

	_, err = q.az.EnqueueMessage(ctx, string(msgJSON), &azqueue.EnqueueMessageOptions{})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to enqueue message")
		return err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "enqueued message")
	return nil
}

func (q AzureQueuer) Dequeue(
	ctx context.Context,
	timeout time.Duration,
	handler MessageHandler,
) error {
	ctx, span := tracer.Start(ctx, "Azure.Dequeue", trace.WithAttributes(
		attribute.Int64("timeoutSecs", int64(timeout.Seconds())),
	))
	defer span.End()

	// Gives us a bit of time to stop work before it is released after cancelling the context
	timeoutSeconds := int32(timeout.Seconds()) + 5

	var msg azqueue.DequeueMessagesResponse
loop:
	for {
		var err error
		msg, err = q.az.DequeueMessage(ctx, &azqueue.DequeueMessageOptions{
			VisibilityTimeout: &timeoutSeconds,
		})
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to dequeue message")
			return err
		}

		switch len(msg.Messages) {
		case 1:
			break loop
		case 0:
			select {
			// Allow early bail from sleep if context becomes cancelled
			case <-ctx.Done():
				span.RecordError(ctx.Err())
				span.SetStatus(codes.Error, "context cancelled")
				return ctx.Err()
			case <-time.After(time.Second * 30):
				continue
			}
		default:
			err = fmt.Errorf("unexpected number of messages: %d", len(msg.Messages))
			span.RecordError(err)
			span.SetStatus(codes.Error, "unexpected number of messages")
			return err
		}
	}

	msgInstance := msg.Messages[0]

	span.AddEvent("got_message", trace.WithAttributes(
		attribute.String("message", *msgInstance.MessageText),
	))

	handlerCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := handler.Handle(handlerCtx, []byte(*msgInstance.MessageText))
	if err != nil {
		var pe *PoisonError
		if !errors.As(err, &pe) {
			// let timeout expire and the message go back to the queue
			span.AddEvent("failed_message_handler", trace.WithAttributes(
				attribute.String("error", err.Error()),
			))
			// TODO: update message to reduce the timeout
			span.RecordError(nil)
			span.SetStatus(codes.Ok, "dequeued message but failed to handle")
			return nil
		}
	}

	_, err = q.az.DeleteMessage(ctx, *msgInstance.MessageID, *msgInstance.PopReceipt, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to remove message")
		return err
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "dequeued message")
	return nil
}
