package types

type (
	Status struct {
		//  This is optional arbitrary content that may be logged in error cases, but is mainly for interactive troubleshooting.
		//
		Details map[string]string `json:"details"`
		// Version string for verification and reproducibility.
		//
		// - git commit
		//
		// - SemVer
		//
		// - etc
		Version string `json:"version" validate:"required"`
		// Last time task and submission stats were reset.  Unix timestamp at millisecond resolution.
		Since UnixMilli `json:"since"   validate:"required"`
		// State of the currently running tasks and submissions
		State StatusState `json:"state"   validate:"required"`
		// Boolean indicating if the CRS is prepared to work on tasks. Do not return true unless you have successfully tested connectivity to the Competition API via /v1/ping/
		Ready bool `json:"ready"   validate:"required"`
	}

	StatusState struct {
		Tasks StatusTasksState `json:"tasks" validate:"required"`
	}

	StatusTasksState struct {
		// Number of tasks that the CRS has not started work on
		Pending int32 `json:"pending"    validate:"required"`
		// Number of tasks that the CRS encountered an unrecoverable issue for
		Errored int32 `json:"errored"    validate:"required"`
		// Number of tasks that the CRS is currently processing
		Processing int32 `json:"processing" validate:"required"`
		// Number of tasks that competition infrastructure has cancelled
		Canceled int32 `json:"canceled"   validate:"required"`
		// Number of submissions that the competition infrastructure is currently testing
		Waiting int32 `json:"waiting"    validate:"required"`
		// Number of submissions that the competition infrastructure marked passed
		Succeeded int32 `json:"succeeded"  validate:"required"`
		// Number of submissions that the competition infrastructure marked failed
		Failed int32 `json:"failed"     validate:"required"`
	}
)
