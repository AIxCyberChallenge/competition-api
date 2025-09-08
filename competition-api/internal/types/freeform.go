package types

type FreeformSubmission struct {
	// Base64 encoded arbitrary data
	//
	// 2MiB max size before Base64 encoding
	Submission string `json:"submission" validate:"required,base64"`
}

type FreeformResponse struct {
	// Schema-compliant submissions will only ever receive the statuses accepted or deadline_exceeded
	Status     SubmissionStatus `json:"status"      validate:"required,eq=accepted|eq=deadline_exceeded"`
	FreeformID string           `json:"freeform_id" validate:"required,uuid_rfc4122"                     format:"uuid"`
}
