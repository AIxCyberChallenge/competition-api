package models

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/audit"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

type PatchSubmission struct {
	PatchFilePath string
	Status        types.SubmissionStatus `gorm:"type:text"`
	Model
	SubmitterID               uuid.UUID // TODO: figure out gorm associations. fk constraints in place due to migrations
	TaskID                    uuid.UUID
	FunctionalityTestsPassing datatypes.Null[bool]
}

var _ Submission = (*PatchSubmission)(nil)

func (PatchSubmission) TableName() string {
	return "patch_submission"
}

func (p PatchSubmission) GetID() uuid.UUID {
	return p.ID
}

func (p PatchSubmission) GetTaskID() uuid.UUID {
	return p.TaskID
}

func (p PatchSubmission) GetSubmitterID() uuid.UUID {
	return p.SubmitterID
}

func (p PatchSubmission) AuditLogSubmissionResult(c audit.Context) {
	audit.LogPatchSubmissionResult(
		c,
		p.ID.String(),
		p.Status,
		PtrFromNull(p.FunctionalityTestsPassing),
	)
}
