package workererrors

import (
	"fmt"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

// Carries an exit code along with an error so the app can exit correctly
type ExitError struct {
	Err  error
	Code int
}

func (e ExitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("%d", e.Code)
	}

	return fmt.Sprintf("%d: %s", e.Code, e.Err.Error())
}

func (e ExitError) Unwrap() error {
	return e.Err
}

// Wrap an error with an exit code
func ExitErrorWrap(code int, err error) error {
	return ExitError{Code: code, Err: err}
}

// Carries the status and patch test status along with the error so that it can be handled and propagated back to the queue
type StatusError struct {
	Err              error
	Status           types.SubmissionStatus
	PatchTestsFailed bool
}

func (e StatusError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("%s %v", e.Status, e.PatchTestsFailed)
	}
	return fmt.Sprintf("%s %v: %s", e.Status, e.PatchTestsFailed, e.Err.Error())
}

func (e StatusError) Unwrap() error {
	return e.Err
}

// Wrap an error with a status and test state
//
// Semantically, `patchTestsFailed` is only meaningful when status == types.SubmissionStatusFailed
func StatusErrorWrap(status types.SubmissionStatus, patchTestsFailed bool, err error) error {
	return StatusError{Status: status, PatchTestsFailed: patchTestsFailed, Err: err}
}
