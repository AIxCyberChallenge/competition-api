package types

type (
	SarifAssessmentSubmission struct {
		Assessment Assessment `json:"assessment"  validate:"required,eq=correct|eq=incorrect"`
		// Plain text reasoning for the assessment.
		//
		// Must be nonempty.
		//
		// 128KiB max size
		Description string `json:"description" validate:"required,max=131072"`
	}

	SarifAssessmentResponse struct {
		Status SubmissionStatus `json:"status" validate:"required,eq=accepted|eq=errored|eq=passed|eq=failed|eq=deadline_exceeded"`
	}

	SARIFSubmission struct {
		// SARIF object compliant with the provided schema
		SARIF *any `json:"sarif" validate:"required" swaggertype:"object"`
	}

	SARIFSubmissionResponse struct {
		SubmittedSARIFID string `json:"submitted_sarif_id" validate:"required,uuid_rfc4122"                     format:"uuid"`
		// Schema-compliant submissions will only ever receive the statuses accepted or deadline_exceeded
		Status SubmissionStatus `json:"status"             validate:"required,eq=accepted|eq=deadline_exceeded"`
	}
)
