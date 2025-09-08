package types

type (
	SourceDetail struct {
		Type SourceType `json:"type"   validate:"required,eq=repo|eq=fuzz-tooling|eq=diff"`
		// URL to fetch the source gzipped tarball
		URL string `json:"url"    validate:"required"`
		// Integrity hash of the gzipped tarball
		SHA256 string `json:"sha256" validate:"required"`
	}

	TaskMetadata struct {
		//nolint:tagliatelle // compatibility
		TaskID string `json:"task.id"  validate:"required,uuid_rfc4122" format:"uuid"`
		//nolint:tagliatelle // compatibility
		RoundID string `json:"round.id" validate:"required"`
	}

	TaskDetail struct {
		// String to string map containing data that should be attached to outputs like log messages and OpenTelemetry trace attributes for traceability
		Metadata TaskMetadata `json:"metadata"           validate:"required"                  swaggertype:"object"`
		TaskID   string       `json:"task_id"            validate:"required,uuid_rfc4122"                          format:"uuid"`
		Type     TaskType     `json:"type"               validate:"required,eq=full|eq=delta"`
		// OSS Fuzz project name
		ProjectName string `json:"project_name"       validate:"required"`
		// The folder in the type repo source tarball containing the main project.
		//
		// This is the project the CRS is meant to submit vulns, patches, and SARIF assessments against.
		Focus string `json:"focus"              validate:"required"`
		// List of sources needed to evaluate a task
		Source []SourceDetail `json:"source"             validate:"required"`
		// UNIX millisecond timestamp by which any submissions for this task must be in
		Deadline          UnixMilli `json:"deadline"           validate:"required"`
		HarnessesIncluded bool      `json:"harnesses_included" validate:"required"`
	}

	Task struct {
		// Unique ID for this message.  The system will retry sending messages if it does not receive a 200 response code.
		// Use this to determine if you have already processed a message.
		MessageID string       `json:"message_id"   validate:"required,uuid_rfc4122" format:"uuid"`
		Tasks     []TaskDetail `json:"tasks"        validate:"required"`
		// UNIX millisecond timestamp for when the message was sent
		MessageTime UnixMilli `json:"message_time" validate:"required"`
	}

	SARIFBroadcastMetadata struct {
		TaskID  string `json:"task_id"  validate:"required,uuid_rfc4122" format:"uuid"`
		SARIFID string `json:"sarif_id" validate:"required,uuid_rfc4122" format:"uuid"`
		RoundID string `json:"round_id" validate:"required"`
	}

	SARIFBroadcastDetail struct {
		TaskID  string `json:"task_id"  validate:"required,uuid_rfc4122"  format:"uuid"`
		SARIFID string `json:"sarif_id" validate:"required, uuid_rfc4122" format:"uuid"`
		// SARIF Report compliant with provided schema
		SARIF any `json:"sarif"    validate:"required"                             swaggertype:"object"`
		// String to string map containing data that should be attached to outputs like log messages and OpenTelemetry trace attributes for traceability
		Metadata SARIFBroadcastMetadata `json:"metadata" validate:"required"                             swaggertype:"object"`
	}

	SARIFBroadcast struct {
		// Unique ID for this message.  The system will retry sending messages if it does not receive a 200 response code.
		// Use this to determine if you have already processed a message.
		MessageID  string                 `json:"message_id"   validate:"required,uuid_rfc4122" format:"uuid"`
		Broadcasts []SARIFBroadcastDetail `json:"broadcasts"   validate:"required"`
		// UNIX millisecond timestamp for when the message was sent
		MessageTime UnixMilli `json:"message_time" validate:"required"`
	}
)
