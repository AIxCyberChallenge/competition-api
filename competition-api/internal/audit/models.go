package audit

import (
	"github.com/google/uuid"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

var schemaVersion = "0.1.0"
var logContext = "audit"

type Disposition string

const (
	DispositionNeutral Disposition = "neutral"
	DispositionGood    Disposition = "good"
	DispositionBad     Disposition = "bad"
)

type FileArchivedEntity string

const (
	EntityTask            = "task"
	EntityPOV             = "pov"
	EntityPatch           = "patch"
	EntitySARIFSubmission = "sarif_submission"
	EntitySARIFBroadcast  = "sarif_broadcast"
	EntityFreeformPOV     = "freeform_pov"
)

type EventType string

const (
	EvtNewDeltaScan          EventType = "new_delta_scan"
	EvtNewFullScan           EventType = "new_full_scan"
	EvtNewSARIFBroadcast     EventType = "new_sarif_broadcast"
	EvtPatchSubmission       EventType = "patch_submission"
	EvtPatchSubmissionResult EventType = "patch_submission_result"
	EvtPOVSubmission         EventType = "pov_submission"
	EvtPOVSubmissionResult   EventType = "pov_submission_result"
	EvtSARIFAssessment       EventType = "sarif_assessment"
	EvtSARIFSubmission       EventType = "sarif_submission"
	EvtOutOfBudget           EventType = "out_of_budget"
	EvtFileArchived          EventType = "file_archived"
	EvtCRSStatus             EventType = "crs_status_check"
	EvtBundleSubmission      EventType = "bundle_submission"
	EvtBundleDelete          EventType = "bundle_delete"
	EvtBroadcastSucceeded    EventType = "broadcast_succeeded"
	EvtBroadcastFailed       EventType = "broadcast_failed"
	EvtFreeformSubmission    EventType = "freeform_submission"
)

type Message struct {
	TaskID        *string     `json:"task_id"`
	TeamID        *string     `json:"team_id"`
	LogContext    string      `json:"log_context" validate:"required"`
	SchemaVersion string      `json:"version"     validate:"required"`
	RoundID       string      `json:"round_id"    validate:"required"`
	Disposition   Disposition `json:"disposition" validate:"required"`
	Type          EventType   `json:"event_type"  validate:"required"`

	Timestamp types.UnixMilli `json:"timestamp" validate:"required"`
}

type FileArchivedEvent struct {
	BucketName   string             `json:"bucket_name"   validate:"required"`
	ObjectName   string             `json:"object_name"   validate:"required"`
	FileArchived types.ArchivedFile `json:"file_archived" validate:"required"`
	Entity       FileArchivedEntity `json:"entity"        validate:"required"`
	EntityID     string             `json:"entity_id"     validate:"required"` // the ID for the entity called out in the context (PoV ID or Patch ID)
}

type FileArchived struct {
	Event FileArchivedEvent `json:"event" validate:"required"`
	Message
}

type NewDeltaScanEvent struct {
	TaskType        types.TaskType  `json:"task_type"         validate:"required"`
	RepoURL         string          `json:"repo_url"          validate:"required"`
	BaseCommitHash  string          `json:"base_commit_hash"  validate:"required"`
	DeltaCommitHash string          `json:"delta_commit_hash" validate:"required"`
	FuzzToolingURL  string          `json:"fuzz_tooling_url"`
	FuzzToolingHash string          `json:"fuzz_tooling_hash"`
	ChallengeName   string          `json:"challenge_name"`
	Deadline        types.UnixMilli `json:"deadline"          validate:"required"`
	Unharnessed     bool            `json:"unharnessed"`
}

type NewDeltaScan struct {
	Message
	Event NewDeltaScanEvent `json:"event" validate:"required"`
}

type NewFullScanEvent struct {
	TaskType        types.TaskType  `json:"task_type"         validate:"required"`
	RepoURL         string          `json:"repo_url"          validate:"required"`
	CommitHash      string          `json:"commit_hash"       validate:"required"`
	FuzzToolingURL  string          `json:"fuzz_tooling_url"`
	FuzzToolingHash string          `json:"fuzz_tooling_hash"`
	ChallengeName   string          `json:"challenge_name"`
	Deadline        types.UnixMilli `json:"deadline"          validate:"required"`
	Unharnessed     bool            `json:"unharnessed"`
}

type NewFullScan struct {
	Message
	Event NewFullScanEvent `json:"event" validate:"required"`
}

type NewSARIFBroadcastEvent struct {
	RepoURL    string `json:"repo_url"    validate:"required"`
	CommitHash string `json:"commit_hash" validate:"required"`
	SARIFID    string `json:"sarif_id"    validate:"required"`
}
type NewSARIFBroadcast struct {
	Event NewSARIFBroadcastEvent `json:"event" validate:"required"`
	Message
}

type POVSubmissionEvent struct {
	POVID          string                 `json:"pov_id"          validate:"required"`
	FuzzerName     string                 `json:"fuzzer_name"     validate:"required"`
	TestcaseSHA256 string                 `json:"testcase_sha256" validate:"required"`
	Sanitizer      string                 `json:"sanitizer"       validate:"required"`
	Architecture   string                 `json:"architecture"    validate:"required"`
	Status         types.SubmissionStatus `json:"status"`
	Engine         string                 `json:"engine"`
}

type POVSubmission struct {
	Event POVSubmissionEvent `json:"event" validate:"required"`
	Message
}

type POVSubmissionResultEvent struct {
	POVID  string                 `json:"pov_id" validate:"required"`
	Status types.SubmissionStatus `json:"status" validate:"required"`
}

type POVSubmissionResult struct {
	Event POVSubmissionResultEvent `json:"event" validate:"required"`
	Message
}

type PatchSubmissionEvent struct {
	PatchID     string                 `json:"patch_id"     validate:"required"`
	PatchSHA256 string                 `json:"patch_sha256" validate:"required"`
	Status      types.SubmissionStatus `json:"status"`
}

type PatchSubmission struct {
	Event PatchSubmissionEvent `json:"event" validate:"required"`
	Message
}

type PatchSubmissionResultEvent struct {
	FunctionalityTestsPassing *bool                  `json:"functionality_tests_passing"`
	PatchID                   string                 `json:"patch_id"                    validate:"required"`
	Status                    types.SubmissionStatus `json:"status"                      validate:"required"`
}

type PatchSubmissionResult struct {
	Event PatchSubmissionResultEvent `json:"event" validate:"required"`
	Message
}

type SARIFAssessmentEvent struct {
	AssessmentID     string `json:"assessment_id"      validate:"required"`
	Assessment       string `json:"assessment"         validate:"required"`
	SARIFBroadcastID string `json:"sarif_broadcast_id" validate:"required"`
}

type SARIFAssessment struct {
	Event SARIFAssessmentEvent `json:"event" validate:"required"`
	Message
}

type SARIFSubmissionEvent struct {
	SubmissionID string                 `json:"submission_id"`
	Status       types.SubmissionStatus `json:"status"`
}

type SARIFSubmission struct {
	Event SARIFSubmissionEvent `json:"event"`
	Message
}

// Exists to maintain existing contract structure of audit log
type OutOfBudgetEvent struct{}

type OutOfBudget struct {
	Event OutOfBudgetEvent `json:"event" validate:"required"`
	Message
}

type CRSStatusEvent struct {
	Version *string            `json:"version"`
	Details *map[string]string `json:"details"`
	State   *types.StatusState `json:"state"`
	Error   *string            `json:"error"`
	CRSURL  string             `json:"crs_url"`
	Ready   bool               `json:"ready"`
}

type CRSStatus struct {
	Message
	Event CRSStatusEvent `json:"event" validate:"required"`
}

type BundleSubmissionEvent struct {
	BundleID         string                 `json:"bundle_id"`
	POVID            *uuid.UUID             `json:"pov_id"`
	PatchID          *uuid.UUID             `json:"patch_id"`
	SubmittedSARIFID *uuid.UUID             `json:"submitted_sarif_id"`
	BroadcastSARIFID *uuid.UUID             `json:"broadcast_sarif_id"`
	Description      *string                `json:"description"`
	FreeformID       *uuid.UUID             `json:"freeform_id"`
	Status           types.SubmissionStatus `json:"status"`
}

type BundleSubmission struct {
	Event BundleSubmissionEvent `json:"event"`
	Message
}

type BundleDeleteEvent struct {
	BundleID string `json:"bundle_id"`
}

type BundleDelete struct {
	Event BundleDeleteEvent `json:"event"`
	Message
}

type BroadcastSucceededEvent struct {
	Retries int `json:"retries"`
}

type BroadcastSucceeded struct {
	Message
	Event BroadcastSucceededEvent `json:"event"`
}

type BroadcastFailedEvent struct {
	Payload string `json:"payload"`
	Retries int    `json:"retries"`
}

type BroadcastFailed struct {
	Message
	Event BroadcastFailedEvent `json:"event"`
}

// Exists to maintain existing contract structure of audit log
type FreeformSubmissionEvent struct{}

type FreeformSubmission struct {
	Event FreeformSubmissionEvent `json:"event"`
	Message
}
