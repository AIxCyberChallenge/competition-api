package models

import (
	"github.com/google/uuid"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/audit"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

type FreeformSubmission struct {
	Submission string
	Status     types.SubmissionStatus `gorm:"type:text"`
	Model
	TaskID      uuid.UUID
	SubmitterID uuid.UUID
}

var _ Submission = (*FreeformSubmission)(nil)

func (FreeformSubmission) TableName() string {
	return "freeform_submission"
}

func (s FreeformSubmission) GetID() uuid.UUID {
	return s.ID
}

func (s FreeformSubmission) GetSubmitterID() uuid.UUID {
	return s.SubmitterID
}

func (FreeformSubmission) AuditLogSubmissionResult(c audit.Context) {
	audit.LogFreeformSubmission(c)
}

func (s FreeformSubmission) GetTaskID() uuid.UUID {
	return s.TaskID
}
