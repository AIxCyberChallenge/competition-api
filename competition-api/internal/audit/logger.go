package audit

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

type Context struct {
	TeamID  *string
	TaskID  *string
	RoundID string
}

func dispForStatus(status types.SubmissionStatus) Disposition {
	switch status {
	case types.SubmissionStatusAccepted:
		return DispositionNeutral
	case types.SubmissionStatusErrored:
		return DispositionBad
	case types.SubmissionStatusPassed:
		return DispositionGood
	case types.SubmissionStatusFailed:
		return DispositionBad
	case types.SubmissionStatusDeadlineExceeded:
		return DispositionBad
	default:
		return DispositionNeutral
	}
}

func LogFileArchived(
	c Context,
	bucketName string,
	objectName string,
	fileArchived types.ArchivedFile,
	fileArchivedEntity FileArchivedEntity,
	entityID string,
) {
	event := FileArchived{}
	event.Type = EvtFileArchived

	event.LogContext = logContext
	event.SchemaVersion = schemaVersion

	event.Timestamp = types.UnixMilli(time.Now().UTC().UnixMilli())
	event.RoundID = c.RoundID
	event.TeamID = c.TeamID
	event.TaskID = c.TaskID

	event.Disposition = DispositionNeutral

	event.Event.BucketName = bucketName
	event.Event.ObjectName = objectName
	event.Event.FileArchived = fileArchived
	event.Event.Entity = fileArchivedEntity
	event.Event.EntityID = entityID

	evtStr, err := json.Marshal(event)
	if err != nil {
		logger.Logger.Error(
			"could not serialize FileArchived event",
			"bucketName",
			bucketName,
			"objectName",
			objectName,
			"fileArchived",
			fileArchived,
			"entity",
			fileArchivedEntity,
			"entityID",
			entityID,
		)
		return
	}

	// TODO: should this go to stderr?
	fmt.Println(string(evtStr))
}

func LogNewDeltaScan(
	c Context,
	repoURL string,
	baseCommitHash string,
	deltaCommitHash string,
	deadline types.UnixMilli,
	ossFuzzURL string,
	ossFuzzHash string,
	challengeName string,
	unharnessed bool,
) {
	event := NewDeltaScan{}
	event.Type = EvtNewDeltaScan
	event.Event.TaskType = types.TaskTypeDelta

	event.LogContext = logContext
	event.SchemaVersion = schemaVersion

	event.Timestamp = types.UnixMilli(time.Now().UTC().UnixMilli())
	event.RoundID = c.RoundID
	event.TaskID = c.TaskID

	event.Disposition = DispositionNeutral

	event.Event.RepoURL = repoURL
	event.Event.DeltaCommitHash = deltaCommitHash
	event.Event.BaseCommitHash = baseCommitHash
	event.Event.Deadline = deadline
	event.Event.FuzzToolingURL = ossFuzzURL
	event.Event.FuzzToolingHash = ossFuzzHash
	event.Event.ChallengeName = challengeName
	event.Event.Unharnessed = unharnessed

	evtStr, err := json.Marshal(event)
	if err != nil {
		logger.Logger.Error(
			"could not serialize NewDeltaScan event",
			"repoURL",
			repoURL,
			"baseCommitHash",
			baseCommitHash,
			"deltaCommitHash",
			deltaCommitHash,
			"deadline",
			deadline,
			"challengeName",
			challengeName,
			"unharnessed",
			unharnessed,
		)
		return
	}

	// TODO: should this go to stderr?
	fmt.Println(string(evtStr))
}

func LogNewFullScan(
	c Context,
	repoURL string,
	commitHash string,
	deadline types.UnixMilli,
	ossFuzzURL string,
	ossFuzzHash string,
	challengeName string,
	unharnessed bool,
) {
	event := NewFullScan{}
	event.Type = EvtNewFullScan
	event.Event.TaskType = types.TaskTypeFull

	event.LogContext = logContext
	event.SchemaVersion = schemaVersion

	event.Timestamp = types.UnixMilli(time.Now().UTC().UnixMilli())
	event.RoundID = c.RoundID
	event.TaskID = c.TaskID

	event.Disposition = DispositionNeutral

	event.Event.RepoURL = repoURL
	event.Event.CommitHash = commitHash
	event.Event.Deadline = deadline
	event.Event.FuzzToolingURL = ossFuzzURL
	event.Event.FuzzToolingHash = ossFuzzHash
	event.Event.ChallengeName = challengeName
	event.Event.Unharnessed = unharnessed

	evtStr, err := json.Marshal(event)
	if err != nil {
		logger.Logger.Error(
			"could not serialize NewFullScan event",
			"repoURL",
			repoURL,
			"commitHash",
			commitHash,
			"deadline",
			deadline,
			"unharnessed",
			unharnessed,
		)
		return
	}

	// TODO: should this go to stderr?
	fmt.Println(string(evtStr))
}

func LogNewSARIFBroadcast(c Context, repoURL string, commitHash *string, sarifID string) {
	event := NewSARIFBroadcast{}
	event.Type = EvtNewSARIFBroadcast

	event.LogContext = logContext
	event.SchemaVersion = schemaVersion

	event.Timestamp = types.UnixMilli(time.Now().UTC().UnixMilli())
	event.RoundID = c.RoundID
	event.TaskID = c.TaskID

	event.Disposition = DispositionNeutral

	event.Event.RepoURL = repoURL
	event.Event.SARIFID = sarifID

	if commitHash != nil {
		event.Event.CommitHash = *commitHash
	} else {
		event.Event.CommitHash = ""
	}

	evtStr, err := json.Marshal(event)
	if err != nil {
		logger.Logger.Error(
			"could not serialize NewSARIFBroadcast event",
			"repoURL",
			repoURL,
			"commitHash",
			commitHash,
			"sarifID",
			sarifID,
		)
		return
	}

	// TODO: should this go to stderr?
	fmt.Println(string(evtStr))
}

func LogPOVSubmission(
	c Context,
	povID string,
	status types.SubmissionStatus,
	fuzzerName string,
	testcaseSHA256 string,
	sanitizer string,
	architecture string,
	engine string,
) {
	event := POVSubmission{}
	event.Type = EvtPOVSubmission

	event.LogContext = logContext
	event.SchemaVersion = schemaVersion

	event.Timestamp = types.UnixMilli(time.Now().UTC().UnixMilli())
	event.RoundID = c.RoundID
	event.TeamID = c.TeamID
	event.TaskID = c.TaskID

	event.Disposition = DispositionNeutral

	event.Event.POVID = povID
	event.Event.Status = status
	event.Event.FuzzerName = fuzzerName
	event.Event.TestcaseSHA256 = testcaseSHA256
	event.Event.Sanitizer = sanitizer
	event.Event.Architecture = architecture
	event.Event.Engine = engine

	evtStr, err := json.Marshal(event)
	if err != nil {
		logger.Logger.Error(
			"could not serialize VulnSubmission event",
			"povID",
			povID,
			"fuzzerName",
			fuzzerName,
			"testcaseSHA256",
			testcaseSHA256,
			"sanitizer",
			sanitizer,
			"architecture",
			architecture,
			"engine",
			engine,
		)
		return
	}

	// TODO: should this go to stderr?
	fmt.Println(string(evtStr))
}

func LogPOVSubmissionResult(c Context, povID string, status types.SubmissionStatus) {
	event := POVSubmissionResult{}
	event.Type = EvtPOVSubmissionResult

	event.LogContext = logContext
	event.SchemaVersion = schemaVersion

	event.Timestamp = types.UnixMilli(time.Now().UTC().UnixMilli())
	event.RoundID = c.RoundID
	event.TeamID = c.TeamID
	event.TaskID = c.TaskID

	event.Disposition = dispForStatus(status)

	event.Event.POVID = povID
	event.Event.Status = status

	evtStr, err := json.Marshal(event)
	if err != nil {
		logger.Logger.Error(
			"could not serialize VulnSubmissionResult event",
			"povID",
			povID,
			"status",
			status,
		)
		return
	}

	// TODO: should this go to stderr?
	fmt.Println(string(evtStr))
}

func LogPatchSubmission(
	c Context,
	patchID string,
	status types.SubmissionStatus,
	patchSHA256 string,
) {
	event := PatchSubmission{}
	event.Type = EvtPatchSubmission

	event.LogContext = logContext
	event.SchemaVersion = schemaVersion

	event.Timestamp = types.UnixMilli(time.Now().UTC().UnixMilli())
	event.RoundID = c.RoundID
	event.TeamID = c.TeamID
	event.TaskID = c.TaskID

	event.Disposition = DispositionNeutral

	event.Event.PatchID = patchID
	event.Event.Status = status
	event.Event.PatchSHA256 = patchSHA256

	evtStr, err := json.Marshal(event)
	if err != nil {
		logger.Logger.Error(
			"could not serialize PatchSubmission event",
			"patchID",
			patchID,
			"patchSHA256",
			patchSHA256,
		)
		return
	}

	// TODO: should this go to stderr?
	fmt.Println(string(evtStr))
}

func LogPatchSubmissionResult(
	c Context,
	patchID string,
	status types.SubmissionStatus,
	functionalityTestsPass *bool,
) {
	event := PatchSubmissionResult{}
	event.Type = EvtPatchSubmissionResult

	event.LogContext = logContext
	event.SchemaVersion = schemaVersion

	event.Timestamp = types.UnixMilli(time.Now().UTC().UnixMilli())
	event.RoundID = c.RoundID
	event.TeamID = c.TeamID
	event.TaskID = c.TaskID

	event.Disposition = dispForStatus(status)

	event.Event.PatchID = patchID
	event.Event.Status = status
	event.Event.FunctionalityTestsPassing = functionalityTestsPass

	evtStr, err := json.Marshal(event)
	if err != nil {
		logger.Logger.Error(
			"could not serialize PatchSubmissionResult event",
			"patchID",
			patchID,
			"status",
			status,
		)
		return
	}

	// TODO: should this go to stderr?
	fmt.Println(string(evtStr))
}

func LogSARIFAssessment(
	c Context,
	assessmentID string,
	assessment string,
	sarifBroadcastID string,
) {
	event := SARIFAssessment{}
	event.Type = EvtSARIFAssessment

	event.LogContext = logContext
	event.SchemaVersion = schemaVersion

	event.Timestamp = types.UnixMilli(time.Now().UTC().UnixMilli())
	event.RoundID = c.RoundID
	event.TeamID = c.TeamID
	event.TaskID = c.TaskID

	event.Disposition = DispositionNeutral

	event.Event.AssessmentID = assessmentID
	event.Event.Assessment = assessment
	event.Event.SARIFBroadcastID = sarifBroadcastID

	evtStr, err := json.Marshal(event)
	if err != nil {
		logger.Logger.Error(
			"could not serialize SARIFAssessment event",
			"assessmentID",
			assessmentID,
			"assessment",
			assessment,
			"sarifBroadcastID",
			sarifBroadcastID,
		)
		return
	}

	// TODO: should this go to stderr?
	fmt.Println(string(evtStr))
}

func LogSARIFSubmission(c Context, submissionID string, status types.SubmissionStatus) {
	event := SARIFSubmission{}
	event.Type = EvtSARIFSubmission

	event.LogContext = logContext
	event.SchemaVersion = schemaVersion

	event.Timestamp = types.UnixMilli(time.Now().UTC().UnixMilli())
	event.RoundID = c.RoundID
	event.TeamID = c.TeamID
	event.TaskID = c.TaskID

	event.Disposition = dispForStatus(status)

	event.Event.SubmissionID = submissionID
	event.Event.Status = status

	evtStr, err := json.Marshal(event)
	if err != nil {
		logger.Logger.Error("could not serialize SARIFSubmission event", "status", status)
		return
	}

	// TODO: should this go to stderr?
	fmt.Println(string(evtStr))
}

func LogBundleSubmission(
	c Context,
	bundleID string,
	povID *uuid.UUID,
	patchID *uuid.UUID,
	broadcastSARIFID *uuid.UUID,
	submittedSARIFID *uuid.UUID,
	description *string,
	freeformID *uuid.UUID,
	status types.SubmissionStatus,
) {
	event := BundleSubmission{}
	event.Type = EvtBundleSubmission

	event.LogContext = logContext
	event.SchemaVersion = schemaVersion

	event.Timestamp = types.UnixMilli(time.Now().UTC().UnixMilli())
	event.RoundID = c.RoundID
	event.TeamID = c.TeamID
	event.TaskID = c.TaskID

	event.Disposition = dispForStatus(status)

	event.Event.BundleID = bundleID
	event.Event.POVID = povID
	event.Event.PatchID = patchID
	event.Event.BroadcastSARIFID = broadcastSARIFID
	event.Event.SubmittedSARIFID = submittedSARIFID
	event.Event.Description = description
	event.Event.Status = status
	event.Event.FreeformID = freeformID

	evtStr, err := json.Marshal(event)
	if err != nil {
		logger.Logger.Error(
			"could not serialize BundleSubmission event",
			"bundleID",
			bundleID,
			"povID",
			povID,
			"patchID",
			patchID,
			"broadcastSARIFID",
			broadcastSARIFID,
			"submittedSARIFID",
			submittedSARIFID,
			"description",
			description,
			"freeformId",
			freeformID,
			"status",
			status,
		)
		return
	}

	// TODO: should this go to stderr?
	fmt.Println(string(evtStr))
}

func LogBundleDelete(c Context, bundleID string) {
	event := BundleDelete{}
	event.Type = EvtBundleDelete

	event.LogContext = logContext
	event.SchemaVersion = schemaVersion

	event.Timestamp = types.UnixMilli(time.Now().UTC().UnixMilli())
	event.RoundID = c.RoundID
	event.TeamID = c.TeamID
	event.TaskID = c.TaskID

	event.Disposition = DispositionNeutral

	event.Event.BundleID = bundleID

	evtStr, err := json.Marshal(event)
	if err != nil {
		logger.Logger.Error("could not serialize BundleDelete event", "bundleID", bundleID)
		return
	}

	// TODO: should this go to stderr?
	fmt.Println(string(evtStr))
}

func LogOutOfBudget(c Context) {
	event := OutOfBudget{}
	event.Type = EvtOutOfBudget

	event.LogContext = logContext
	event.SchemaVersion = schemaVersion

	event.Timestamp = types.UnixMilli(time.Now().UTC().UnixMilli())
	event.RoundID = c.RoundID
	event.TeamID = c.TeamID
	event.TaskID = c.TaskID
	event.Disposition = DispositionBad

	evtStr, err := json.Marshal(event)
	if err != nil {
		var teamID string
		if c.TeamID != nil {
			teamID = *c.TeamID
		}
		logger.Logger.Error("could not serialize OutOfBudget event", "team", teamID)
		return
	}

	// TODO: should this go to stderr?
	fmt.Println(string(evtStr))
}

func LogCRSStatus(c Context, crsURL string, status *types.Status) {
	event := CRSStatus{}
	event.Type = EvtCRSStatus

	event.LogContext = logContext
	event.SchemaVersion = schemaVersion

	event.Timestamp = types.UnixMilli(time.Now().UTC().UnixMilli())
	event.RoundID = c.RoundID
	event.TeamID = c.TeamID

	if status.Ready {
		event.Disposition = DispositionGood
	} else {
		event.Disposition = DispositionBad
	}

	event.Event.CRSURL = crsURL
	event.Event.Ready = status.Ready
	event.Event.Version = &status.Version
	if status.Details != nil {
		event.Event.Details = &status.Details
	}
	event.Event.State = &status.State

	evtStr, err := json.Marshal(event)
	if err != nil {
		var teamID string
		if c.TeamID != nil {
			teamID = *c.TeamID
		}
		logger.Logger.Error(
			"could not serialize CRSReady event",
			"team",
			teamID,
			"crs",
			crsURL,
			"ready",
			status.Ready,
			"version",
			status.Version,
			"details",
			status.Details,
			"state",
			status.State,
		)
		return
	}

	// TODO: should this go to stderr?
	fmt.Println(string(evtStr))
}

func LogCRSStatusFailed(c Context, crsURL string, errStr string) {
	event := CRSStatus{}
	event.Type = EvtCRSStatus

	event.LogContext = logContext
	event.SchemaVersion = schemaVersion

	event.Timestamp = types.UnixMilli(time.Now().UTC().UnixMilli())
	event.RoundID = c.RoundID
	event.TeamID = c.TeamID
	event.Disposition = DispositionBad

	event.Event.CRSURL = crsURL
	event.Event.Ready = false
	event.Event.Error = &errStr

	evtStr, err := json.Marshal(event)
	if err != nil {
		var teamID string
		if c.TeamID != nil {
			teamID = *c.TeamID
		}
		logger.Logger.Error(
			"could not serialize CRSReady event",
			"team",
			teamID,
			"crs",
			crsURL,
			"error",
			errStr,
		)
		return
	}

	// TODO: should this go to stderr?
	fmt.Println(string(evtStr))
}

func LogBroadcastSucceeded(c Context, retries int) {
	event := BroadcastSucceeded{}
	event.Type = EvtBroadcastSucceeded

	event.LogContext = logContext
	event.SchemaVersion = schemaVersion

	event.Timestamp = types.UnixMilli(time.Now().UTC().UnixMilli())
	event.RoundID = c.RoundID
	event.TeamID = c.TeamID
	event.TaskID = c.TaskID

	event.Disposition = DispositionGood

	event.Event.Retries = retries

	evtStr, err := json.Marshal(event)
	if err != nil {
		var teamID string
		if c.TeamID != nil {
			teamID = *c.TeamID
		}
		logger.Logger.Error(
			"could not serialize BroadcastSucceeded event",
			"team",
			teamID,
			"roundId",
			c.RoundID,
			"taskId",
			c.TaskID,
			"retries",
			retries,
		)
		return
	}

	// TODO: should this go to stderr?
	fmt.Println(string(evtStr))
}

func LogBroadcastFailed(c Context, payload string, retries int) {
	event := BroadcastFailed{}
	event.Type = EvtBroadcastFailed

	event.LogContext = logContext
	event.SchemaVersion = schemaVersion

	event.Timestamp = types.UnixMilli(time.Now().UTC().UnixMilli())
	event.RoundID = c.RoundID
	event.TeamID = c.TeamID
	event.TaskID = c.TaskID

	event.Disposition = DispositionBad

	event.Event.Payload = payload
	event.Event.Retries = retries

	evtStr, err := json.Marshal(event)
	if err != nil {
		var teamID string
		if c.TeamID != nil {
			teamID = *c.TeamID
		}
		logger.Logger.Error(
			"could not serialize BroadcastSucceeded event",
			"team",
			teamID,
			"roundId",
			c.RoundID,
			"taskId",
			c.TaskID,
			"payload",
			payload,
			"retries",
			retries,
		)
		return
	}

	// TODO: should this go to stderr?
	fmt.Println(string(evtStr))
}

func LogFreeformSubmission(c Context) {
	event := FreeformSubmission{}
	event.Type = EvtFreeformSubmission

	event.LogContext = logContext
	event.SchemaVersion = schemaVersion

	event.Timestamp = types.UnixMilli(time.Now().UTC().UnixMilli())
	event.RoundID = c.RoundID
	event.TeamID = c.TeamID
	event.TaskID = c.TaskID

	event.Disposition = DispositionNeutral

	evtStr, err := json.Marshal(event)
	if err != nil {
		var teamID string
		if c.TeamID != nil {
			teamID = *c.TeamID
		}
		logger.Logger.Error(
			"could not serialize FreeformSubmission event",
			"team",
			teamID,
			"roundId",
			c.RoundID,
			"taskId",
			c.TaskID,
		)
		return
	}

	// TODO: should this go to stderr?
	fmt.Println(string(evtStr))
}
