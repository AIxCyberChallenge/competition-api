package types

type SubmissionStatus string

const (
	SubmissionStatusAccepted         SubmissionStatus = "accepted"          // Successfully submitted
	SubmissionStatusPassed           SubmissionStatus = "passed"            // Successfully evaluated submission
	SubmissionStatusFailed           SubmissionStatus = "failed"            // Submission failed testing
	SubmissionStatusDeadlineExceeded SubmissionStatus = "deadline_exceeded" // Task deadline exceeded. All submissions marked accepted before the deadline will be evaluated.
	SubmissionStatusErrored          SubmissionStatus = "errored"           // Server side error when testing submission
	SubmissionStatusInconclusive     SubmissionStatus = "inconclusive"      // Test continued running beyond timeout and will be manually reviewed after the round.  As a result this status is inconclusive.
)

const (
	ExitNormal        int = 0
	ExitErrored       int = 1
	ExitFailedMin     int = 2
	ExitFailedMax     int = 3
	ExitFailedDefault int = 2
	ExitFailedTests   int = 3
)
