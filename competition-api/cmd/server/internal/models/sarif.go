package models

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/audit"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

type (
	SARIFBroadcast struct {
		Model
		SARIF  datatypes.JSON `gorm:"type:jsonb;serializer:json"`
		TaskID uuid.UUID
	}

	SARIFAssessment struct {
		Assessment types.Assessment       `gorm:"type:text"`
		Status     types.SubmissionStatus `gorm:"type:text"`
		Model
		SubmitterID      uuid.UUID
		SARIFBroadcastID uuid.UUID
	}

	SARIFSubmission struct {
		Model
		Status      types.SubmissionStatus `gorm:"type:text"`
		SARIF       datatypes.JSON
		SubmitterID uuid.UUID
		TaskID      uuid.UUID
	}
)

var _ Submission = (*SARIFSubmission)(nil)

func (SARIFBroadcast) TableName() string {
	return "sarif_broadcast"
}

func (b SARIFBroadcast) GetID() uuid.UUID {
	return b.ID
}

func (SARIFAssessment) TableName() string {
	return "sarif_assessment"
}

func (s SARIFAssessment) GetID() uuid.UUID {
	return s.ID
}

func (s SARIFAssessment) GetSubmitterID() uuid.UUID {
	return s.SubmitterID
}

func (SARIFSubmission) TableName() string {
	return "sarif_submission"
}

func (s SARIFSubmission) GetID() uuid.UUID {
	return s.ID
}

func (s SARIFSubmission) GetSubmitterID() uuid.UUID {
	return s.SubmitterID
}

func (s SARIFSubmission) AuditLogSubmissionResult(c audit.Context) {
	audit.LogSARIFSubmission(c, s.ID.String(), s.Status)
}

func (s SARIFSubmission) GetTaskID() uuid.UUID {
	return s.TaskID
}
