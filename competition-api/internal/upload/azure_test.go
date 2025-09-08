package upload_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/azure/azurite"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/upload"
)

var container = "container"

func TestAzure(t *testing.T) {
	ctx := context.Background()

	azuriteContainer, err := azurite.Run(
		ctx,
		"mcr.microsoft.com/azure-storage/azurite:latest",
		azurite.WithInMemoryPersistence(256),
	)
	require.NoError(t, err, "failed to make azurite container")
	defer func() {
		require.NoError(t, testcontainers.TerminateContainer(azuriteContainer))
	}()

	cred, err := azblob.NewSharedKeyCredential(azurite.AccountName, azurite.AccountKey)
	require.NoError(t, err, "failed to get creds")

	serviceURL, err := azuriteContainer.BlobServiceURL(ctx)
	require.NoError(t, err, "failed to get serviceURL")
	serviceURL = fmt.Sprintf("%s/%s", serviceURL, azurite.AccountName)

	azclient, err := azblob.NewClientWithSharedKeyCredential(
		serviceURL,
		cred,
		nil,
	)
	require.NoError(t, err, "failed to make azure blob client")

	_, err = azclient.CreateContainer(ctx, container, nil)
	require.NoError(t, err, "failed to make container")

	uploader, err := upload.NewAzureUploader(
		azurite.AccountName,
		azurite.AccountKey,
		serviceURL,
		container,
	)
	require.NoError(t, err, "failed to construct uploader")

	t.Run("NotExists", func(t *testing.T) {
		expected := false
		exists, err := uploader.Exists(ctx, "abc")
		require.NoError(t, err, "failed to check if file exists")

		assert.Equal(t, expected, exists, "file should not exist")
	})

	t.Run("Exists", func(t *testing.T) {
		url := uuid.NewString()
		_, err := azclient.UploadBuffer(
			ctx,
			container,
			url,
			[]byte("hello world"),
			nil,
		)
		require.NoError(t, err, "failed to upload file for testing")

		expected := true
		exists, err := uploader.Exists(ctx, url)
		require.NoError(t, err, "failed to check if file exists")

		assert.Equal(t, expected, exists, "file should not exist")
	})

	t.Run("Upload", func(t *testing.T) {
		url := uuid.NewString()
		expected := "abc"
		err := uploader.Upload(
			ctx,
			strings.NewReader(expected),
			int64(len(expected)),
			url,
		)
		require.NoError(t, err, "failed to upload file")

		buffer := make([]byte, len(expected))
		_, err = azclient.DownloadBuffer(ctx, container, url, buffer, nil)
		require.NoError(t, err, "failed to download file to buffer")

		assert.Equal(t, expected, string(buffer), "content of file should match")
	})
}
