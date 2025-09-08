package queue_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azqueue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/azure/azurite"
	"go.uber.org/mock/gomock"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/queue"
	mockqueue "github.com/aixcyberchallenge/competition-api/competition-api/internal/queue/mock"
)

var queueName = "queue"

type message struct {
	Foo string `json:"foo"`
}

func TestAzure(t *testing.T) {
	ctx := t.Context()

	azuriteContainer, err := azurite.Run(
		ctx,
		"mcr.microsoft.com/azure-storage/azurite:latest",
		azurite.WithInMemoryPersistence(256),
	)
	require.NoError(t, err, "failed to make azurite container")
	defer func() {
		require.NoError(t, testcontainers.TerminateContainer(azuriteContainer))
	}()

	cred, err := azqueue.NewSharedKeyCredential(azurite.AccountName, azurite.AccountKey)
	require.NoError(t, err, "failed to get creds")

	serviceURL, err := azuriteContainer.QueueServiceURL(ctx)
	require.NoError(t, err, "failed to get serviceURL")
	serviceURL = fmt.Sprintf("%s/%s", serviceURL, azurite.AccountName)

	azclient, err := azqueue.NewServiceClientWithSharedKeyCredential(
		serviceURL,
		cred,
		nil,
	)
	require.NoError(t, err, "failed to make azure queue client")

	queueclient := azclient.NewQueueClient(queueName)
	_, err = queueclient.Create(ctx, nil)
	require.NoError(t, err, "failed to make queue")

	queuer, err := queue.NewAzureQueuer(
		azurite.AccountName,
		azurite.AccountKey,
		serviceURL,
		queueName,
	)
	require.NoError(t, err, "failed to construct uploader")

	t.Run("Enqueue", func(t *testing.T) {
		expected := message{Foo: "foo"}
		require.NoError(t, queuer.Enqueue(ctx, expected), "failed to queue message")

		dequeued, dqErr := queueclient.DequeueMessage(
			ctx,
			nil,
		)
		require.NoError(t, dqErr, "failed to dequeue message")

		assert.Len(t, dequeued.Messages, 1, "should remove 1 message")

		rawMessage := *dequeued.Messages[0].MessageText
		actual := message{}
		err = json.Unmarshal([]byte(rawMessage), &actual)
		require.NoError(t, err, "failed to unmarshal message")

		assert.Equal(t, expected, actual, "messages should match")
	})

	t.Run("Dequeue", func(t *testing.T) {
		t.Run("Empty", func(t *testing.T) {
			// Should not find something to dequeue before the context cancels
			ctrl := gomock.NewController(t)
			handler := mockqueue.NewMockMessageHandler(ctrl)

			handler.EXPECT().Handle(gomock.Any(), gomock.Any()).Times(0)

			cctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()

			require.Error(
				t,
				queuer.Dequeue(cctx, time.Minute, handler),
				"failed to handle a dequeue",
			)
		})

		t.Run("Something", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			handler := mockqueue.NewMockMessageHandler(ctrl)

			msg := "abc"
			_, err = queueclient.EnqueueMessage(ctx, msg, nil)
			require.NoError(t, err, "enqueing message")

			handler.EXPECT().Handle(gomock.Any(), gomock.Eq([]byte(msg))).Times(1)

			err := queuer.Dequeue(ctx, time.Minute, handler)
			require.NoError(t, err, "failed to dequeue message")
		})
	})
}
