package models

import (
	"github.com/google/uuid"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/audit"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

type POVSubmission struct {
	TestcasePath string // path in Azure Blob Container
	FuzzerName   string // OSS Fuzz name for harness
	Sanitizer    string
	Architecture string
	Status       types.SubmissionStatus `gorm:"type:text"`
	Engine       string
	Model
	SubmitterID uuid.UUID // TODO: figure out gorm associations. the database has the constraits from manual migrations
	TaskID      uuid.UUID // TODO: figure out gorm associations. the database has the constraits from manual migrations
}

var _ Submission = (*POVSubmission)(nil)

func (POVSubmission) TableName() string {
	return "pov_submission"
}

func (v POVSubmission) GetID() uuid.UUID {
	return v.ID
}

func (v POVSubmission) GetTaskID() uuid.UUID {
	return v.TaskID
}

func (v POVSubmission) GetSubmitterID() uuid.UUID {
	return v.SubmitterID
}

func (v POVSubmission) AuditLogSubmissionResult(c audit.Context) {
	audit.LogPOVSubmissionResult(c, v.ID.String(), v.Status)
}
