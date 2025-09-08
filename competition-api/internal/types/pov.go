package types

type (
	POVSubmission struct {
		// Base64 encoded vuln trigger
		//
		// 2MiB max size before Base64 encoding
		Testcase string `json:"testcase"     validate:"required,base64"       format:"base64"`
		// Fuzz Tooling fuzzer that exercises this vuln
		//
		// 4KiB max size
		FuzzerName string `json:"fuzzer_name"  validate:"required,max=4096"`
		// Fuzz Tooling Sanitizer that exercises this vuln
		//
		// 4KiB max size
		Sanitizer    string        `json:"sanitizer"    validate:"required,max=4096"`
		Architecture Architecture  `json:"architecture" validate:"required,eq=x86_64"`
		Engine       FuzzingEngine `json:"engine"       validate:"required,eq=libfuzzer"`
	}

	POVSubmissionResponse struct {
		POVID  string           `json:"pov_id" format:"uuid" validate:"required,uuid_rfc4122"`
		Status SubmissionStatus `json:"status"               validate:"required,eq=accepted|eq=errored|eq=passed|eq=failed|eq=deadline_exceeded"`
	}
)
