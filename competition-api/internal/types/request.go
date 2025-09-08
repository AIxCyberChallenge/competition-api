package types

type RequestSubmission struct {
	// Time in seconds until a task should expire. If not provided, defaults to 3600.
	DurationSecs *int64 `json:"duration_secs"`
}

type RequestListResponse struct {
	// List of challenges that competitors may task themselves with
	Challenges []string `json:"challenges" validate:"required"`
}
