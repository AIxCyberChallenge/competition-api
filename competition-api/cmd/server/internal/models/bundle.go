package models

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/audit"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

type Bundle struct {
	Model
	DeletedAt        gorm.DeletedAt
	Status           types.SubmissionStatus `gorm:"type:text"`
	Description      datatypes.Null[string]
	POVID            datatypes.Null[uuid.UUID] `gorm:"column:pov_id"`
	PatchID          datatypes.Null[uuid.UUID]
	BroadcastSARIFID datatypes.Null[uuid.UUID] `gorm:"column:broadcast_sarif_id"`
	SubmittedSARIFID datatypes.Null[uuid.UUID] `gorm:"column:submitted_sarif_id"`
	FreeformID       datatypes.Null[uuid.UUID] `gorm:"column:freeform_submission_id"`
	SubmitterID      uuid.UUID
	TaskID           uuid.UUID
}

var _ Submission = (*Bundle)(nil)

func (Bundle) TableName() string {
	return "bundle"
}

func (s Bundle) GetID() uuid.UUID {
	return s.ID
}

func (s Bundle) GetSubmitterID() uuid.UUID {
	return s.SubmitterID
}

func (s Bundle) GetTaskID() uuid.UUID {
	return s.TaskID
}

func (s Bundle) AuditLogSubmissionResult(c audit.Context) {
	audit.LogBundleSubmission(
		c,
		s.ID.String(),
		PtrFromNull(s.POVID),
		PtrFromNull(s.PatchID),
		PtrFromNull(s.BroadcastSARIFID),
		PtrFromNull(s.SubmittedSARIFID),
		PtrFromNull(s.Description),
		PtrFromNull(s.FreeformID),
		s.Status,
	)
}

// Checks that all elements attempting to be set on bundle are owned by the correct task and submitter
func (s *Bundle) CheckAndSetRelations(
	db *gorm.DB,
	parsed *types.BundleSubmissionParsed,
) (bool, error) {
	exists := make([]string, 0, 4)
	values := make(map[string]any, 6)
	if parsed.POVID.Defined {
		if parsed.POVID.Value != nil {
			exists = append(
				exists,
				"EXISTS (SELECT 1 FROM pov_submission WHERE pov_submission.id = @pov_id AND pov_submission.submitter_id = @submitter_id AND pov_submission.task_id = @task_id)",
			)
			values["pov_id"] = *parsed.POVID.Value
		}
		s.POVID = NewNull(parsed.POVID.Value)
	}
	if parsed.PatchID.Defined {
		if parsed.PatchID.Value != nil {
			exists = append(
				exists,
				"EXISTS (SELECT 1 FROM patch_submission WHERE patch_submission.id = @patch_id AND patch_submission.submitter_id = @submitter_id AND patch_submission.task_id = @task_id)",
			)
			values["patch_id"] = *parsed.PatchID.Value
		}
		s.PatchID = NewNull(parsed.PatchID.Value)
	}
	if parsed.BroadcastSARIFID.Defined {
		if parsed.BroadcastSARIFID.Value != nil {
			exists = append(
				exists,
				"EXISTS (SELECT 1 FROM sarif_broadcast WHERE sarif_broadcast.id = @broadcast_sarif_id AND sarif_broadcast.task_id = @task_id)",
			)
			values["broadcast_sarif_id"] = *parsed.BroadcastSARIFID.Value
		}
		s.BroadcastSARIFID = NewNull(parsed.BroadcastSARIFID.Value)
	}
	if parsed.SubmittedSARIFID.Defined {
		if parsed.SubmittedSARIFID.Value != nil {
			exists = append(
				exists,
				"EXISTS (SELECT 1 FROM sarif_submission WHERE sarif_submission.id = @submitted_sarif_id AND sarif_submission.submitter_id = @submitter_id AND sarif_submission.task_id = @task_id)",
			)
			values["submitted_sarif_id"] = *parsed.SubmittedSARIFID.Value
		}
		s.SubmittedSARIFID = NewNull(parsed.SubmittedSARIFID.Value)
	}
	if parsed.Description.Defined {
		s.Description = NewNull(parsed.Description.Value)
	}
	if parsed.FreeformID.Defined {
		if parsed.FreeformID.Value != nil {
			exists = append(
				exists,
				"EXISTS (SELECT 1 FROM freeform_submission WHERE freeform_submission.id = @freeform_submission_id AND freeform_submission.submitter_id = @submitter_id AND freeform_submission.task_id = @task_id)",
			)
			values["freeform_submission_id"] = *parsed.FreeformID.Value
		}
		s.FreeformID = NewNull(parsed.FreeformID.Value)
	}

	values["task_id"] = s.TaskID
	values["submitter_id"] = s.SubmitterID

	query := fmt.Sprintf("SELECT %s;", strings.Join(exists, " AND "))
	val := false
	err := db.Raw(query, values).Scan(&val).Error
	if err != nil {
		return false, err
	}

	return val, nil
}

// Counts the number of fields on `bundle` and `parsed` that contain explicitly set data
func (s *Bundle) CountNotNull(parsed *types.BundleSubmissionParsed) int {
	count := 0
	if (s.POVID.Valid && !parsed.POVID.Defined) ||
		(parsed.POVID.Defined && parsed.POVID.Value != nil) {
		count++
	}
	if (s.PatchID.Valid && !parsed.PatchID.Defined) ||
		(parsed.PatchID.Defined && parsed.PatchID.Value != nil) {
		count++
	}
	if (s.BroadcastSARIFID.Valid && !parsed.BroadcastSARIFID.Defined) ||
		(parsed.BroadcastSARIFID.Defined && parsed.BroadcastSARIFID.Value != nil) {
		count++
	}
	if (s.SubmittedSARIFID.Valid && !parsed.SubmittedSARIFID.Defined) ||
		(parsed.SubmittedSARIFID.Defined && parsed.SubmittedSARIFID.Value != nil) {
		count++
	}
	if (s.Description.Valid && !parsed.Description.Defined) ||
		(parsed.Description.Defined && parsed.Description.Value != nil) {
		count++
	}
	if (s.FreeformID.Valid && !parsed.FreeformID.Defined) ||
		(parsed.FreeformID.Defined && parsed.FreeformID.Value != nil) {
		count++
	}

	return count
}
