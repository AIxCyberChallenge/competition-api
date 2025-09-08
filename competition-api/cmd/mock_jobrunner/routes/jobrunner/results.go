package jobrunner

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

// GetJobResults returns the current status & any available results for a job
//
//	@Summary		Get Job Results
//	@Description	Get the current status & any available results for a job
//
//	@Tags			jobrunner
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Param			job_id	path		string	true	"Job ID"	Format(uuid)
//
//	@Success		200		{object}	types.JobResponse
//
//	@Failure		400		{object}	types.Error
//	@Failure		401		{object}	types.Error
//	@Failure		404		{object}	types.Error
//	@Failure		500		{object}	types.Error
//
//	@Router			/jobrunner/job/{job_id} [post]
func GetJobResults(c echo.Context) error {
	fakePresignedURL := "https://example.com/38e0b9de817f645c4bec37c0d4a3e58baecccb040f5718dc069a72c7385a0bed"
	fakeBlob := types.Blob{
		ObjectName:   "38e0b9de817f645c4bec37c0d4a3e58baecccb040f5718dc069a72c7385a0bed",
		PresignedURL: &fakePresignedURL,
	}
	fakeExitCode := types.ExitNormal

	resp := types.JobResponse{
		JobID:  c.Param("job_id"),
		Status: types.SubmissionStatusPassed,
		Artifacts: []types.JobArtifact{
			{
				Blob:         fakeBlob,
				Context:      types.ResultCtxHeadRepoTest,
				Filename:     "fuzz.out",
				ArchivedFile: types.FileFuzzOutHead,
			},
			{
				Blob:         fakeBlob,
				Context:      types.ResultCtxBaseRepoTest,
				Filename:     "fuzz.out",
				ArchivedFile: types.FileFuzzOutBase,
			},
		},
		Results: []types.JobResult{
			{
				Cmd:        []string{"./run_pov.sh", "--asdf", "asdf", "--asdf", "asdf"},
				StdoutBlob: fakeBlob,
				StderrBlob: fakeBlob,
				ExitCode:   &fakeExitCode,
				Context:    types.ResultCtxHeadRepoTest,
			},
			{
				Cmd:        []string{"./run_pov.sh", "--asdf", "asdf", "--asdf", "asdf"},
				StdoutBlob: fakeBlob,
				StderrBlob: fakeBlob,
				ExitCode:   &fakeExitCode,
				Context:    types.ResultCtxBaseRepoTest,
			},
		},
	}

	return c.JSON(http.StatusOK, resp)
}
