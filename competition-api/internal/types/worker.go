package types

import (
	"fmt"
)

type (
	WorkerMsg struct {
		MsgType
		Entity   JobType `json:"entity"`
		EntityID string  `json:"entity_id" validate:"uuid_rfc4122" format:"uuid"`
	}

	WorkerMsgArtifact struct {
		WorkerMsg
		Artifact JobArtifact `json:"artifact"`
	}

	WorkerMsgCommandResult struct {
		Result *JobResult `json:"artifact,omitempty"`
		WorkerMsg
	}

	WorkerMsgFinal struct {
		WorkerMsg
		Status SubmissionStatus `json:"status"`

		// only meaningful if Entity == Patch && Status == Failed
		PatchTestsFailed bool `json:"patch_tests_failed"`
	}

	MsgType string
	JobKind string
	JobType string
)

const (
	MsgTypeFinal         = "final"
	MsgTypeArtifact      = "artifact"
	MsgTypeCommandResult = "command_result"

	JobKindEval      = "eval"
	JobKindBroadcast = "broadcast"
	JobKindCancel    = "cancel"

	JobTypePOV   JobType = "pov"
	JobTypePatch JobType = "patch"
	// Jobrunner uses this generic type
	JobTypeJob JobType = "job"
)

func NewWorkerMsgArtifact(
	entity JobType,
	entityID string,
	artifact JobArtifact,
) WorkerMsgArtifact {
	return WorkerMsgArtifact{
		WorkerMsg: WorkerMsg{
			MsgType:  MsgTypeArtifact,
			Entity:   entity,
			EntityID: entityID,
		},
		Artifact: artifact,
	}
}

func NewWorkerMsgCommandResult(
	entity JobType,
	entityID string,
	result *JobResult,
) WorkerMsgCommandResult {
	return WorkerMsgCommandResult{
		WorkerMsg: WorkerMsg{
			MsgType:  MsgTypeCommandResult,
			Entity:   entity,
			EntityID: entityID,
		},
		Result: result,
	}
}

func NewWorkerMsgFinal(
	entity JobType,
	entityID string,
	status SubmissionStatus,
	patchTestsFailed *bool,
) WorkerMsgFinal {
	failed := false
	if patchTestsFailed != nil {
		failed = *patchTestsFailed
	}
	return WorkerMsgFinal{
		WorkerMsg: WorkerMsg{
			MsgType:  MsgTypeFinal,
			Entity:   entity,
			EntityID: entityID,
		},
		PatchTestsFailed: failed,
		Status:           status,
	}
}

func JobTypeFromString(s string) (*JobType, error) {
	var t JobType

	switch s {
	case string(JobTypePOV):
		t = JobTypePOV
	case string(JobTypePatch):
		t = JobTypePatch
	case string(JobTypeJob):
		t = JobTypeJob
	default:
		return nil, fmt.Errorf("%s is not a valid job type", s)
	}

	return &t, nil
}
