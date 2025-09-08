package models

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

type (
	Job struct {
		Status   types.SubmissionStatus `gorm:"type:text;default:'accepted'"`
		CacheKey string
		Model

		Results                   []types.JobResult   `gorm:"type:jsonb;serializer:json"`
		Artifacts                 []types.JobArtifact `gorm:"type:jsonb;serializer:json"`
		FunctionalityTestsPassing datatypes.Null[bool]
	}
)

func (Job) TableName() string {
	return "job"
}

func (j Job) GetID() uuid.UUID {
	return j.ID
}
