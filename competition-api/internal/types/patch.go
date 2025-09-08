package types

type (
	PatchSubmission struct {
		// Base64 encoded patch in unified diff format
		//
		// 100KiB max size before Base64 encoding
		Patch string `json:"patch" validate:"required,base64" format:"base64"`
	}

	PatchSubmissionResponse struct {
		// null indicates the tests have not been run
		FunctionalityTestsPassing *bool            `json:"functionality_tests_passing"`
		PatchID                   string           `json:"patch_id"                    format:"uuid" validate:"required,uuid_rfc4122"`
		Status                    SubmissionStatus `json:"status"                                    validate:"required,eq=accepted|eq=errored|eq=passed|eq=failed|eq=deadline_exceeded"`
	}
)
