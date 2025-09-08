package common

import (
	"errors"
	"os"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/audit"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/queue"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/upload"
)

func GetAuditContext() (*audit.Context, error) {
	roundID := os.Getenv("ROUND_ID")
	if roundID == "" {
		return nil, errors.New("round id not set")
	}

	teamID := os.Getenv("TEAM_ID")
	if teamID == "" {
		return nil, errors.New("team id not set")
	}

	taskID := os.Getenv("TASK_ID")
	if taskID == "" {
		return nil, errors.New("task id not set")
	}

	return &audit.Context{
		RoundID: roundID,
		TeamID:  &teamID,
		TaskID:  &taskID,
	}, nil
}

func GetAzureBlobClient() (*upload.AzureUploader, error) {
	return upload.NewAzureUploader(
		os.Getenv("AZURE_STORAGE_ACCOUNT_NAME"),
		os.Getenv("AZURE_STORAGE_ACCOUNT_KEY"),
		os.Getenv("AZURE_STORAGE_ACCOUNT_CONTAINERS_URL"),
		os.Getenv("AZURE_STORAGE_ACCOUNT_CONTAINER"),
	)
}

func GetAzureQueueClient() (*queue.AzureQueuer, error) {
	return queue.NewAzureQueuer(
		os.Getenv("AZURE_STORAGE_ACCOUNT_NAME"),
		os.Getenv("AZURE_STORAGE_ACCOUNT_KEY"),
		os.Getenv("AZURE_STORAGE_ACCOUNT_QUEUES_URL"),
		os.Getenv("AZURE_STORAGE_ACCOUNT_RESULTS_QUEUE"),
	)
}
