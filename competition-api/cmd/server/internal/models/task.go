package models

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/upload"
)

type (
	Source struct {
		Type   string `json:"type"` // TODO: fix enum
		URL    string `json:"url"`
		SHA256 string `json:"sha256"`
	}

	UnstrippedSources struct {
		BaseRepo    *Source `json:"base_repo"`
		HeadRepo    Source  `json:"head_repo"`
		FuzzTooling Source  `json:"fuzz_tooling"`
	}

	Task struct {
		UnstrippedSource UnstrippedSources `gorm:"type:jsonb;serializer:json"`
		Model
		Deadline          time.Time
		Type              types.TaskType
		RoundID           string
		ProjectName       string
		Focus             string
		Commit            string
		Source            datatypes.JSONSlice[Source]
		MemoryGB          int `gorm:"column:memory_gb"`
		CPUs              int `gorm:"column:cpus"`
		HarnessesIncluded bool
	}
)

func (Task) TableName() string {
	return "task"
}

func (t Task) GetID() uuid.UUID {
	return t.ID
}

type PresignedSourceURLs struct {
	HeadRepo    string
	FuzzTooling string
	BaseRepo    string
}

// Gets presigned urls for the sources that make up a task
func (t *Task) GetSourceURLs(
	ctx context.Context,
	u upload.Uploader,
	duration time.Duration,
) (*PresignedSourceURLs, error) {
	_, span := tracer.Start(ctx, "Task.GetSourceURLs")
	defer span.End()

	headRepo, err := u.PresignedReadURL(ctx, t.UnstrippedSource.HeadRepo.URL, duration)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to make head repo url")
		return nil, err
	}

	fuzzTooling, err := u.PresignedReadURL(ctx, t.UnstrippedSource.FuzzTooling.URL, duration)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to make fuzztooling repo url")
		return nil, err
	}

	presignedURLs := &PresignedSourceURLs{
		HeadRepo:    headRepo,
		FuzzTooling: fuzzTooling,
	}

	if t.Type == types.TaskTypeDelta {
		if t.UnstrippedSource.BaseRepo == nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "missing base repo")
			return nil, errors.New("missing base repo")
		}
		baseRepo, err := u.PresignedReadURL(ctx, t.UnstrippedSource.BaseRepo.URL, duration)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to make base repo url")
			return nil, err
		}
		presignedURLs.BaseRepo = baseRepo
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "generated source urls")
	return presignedURLs, nil
}
