package jobrunner

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

// RunTest tests a given PoV, Patch, or combination
//
//	@Summary		Run Test Job
//	@Description	submit a PoV, Patch, or combination for testing.  Submitting both tests a PoV against a Patch to determine if the PoV crashes after the patch is applied.
//	@Tags			jobrunner
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Param			payload	body		types.JobArgs	true	"Submission body"
//
//	@Success		200		{object}	types.JobResponse
//
//	@Failure		400		{object}	types.Error
//	@Failure		401		{object}	types.Error
//	@Failure		404		{object}	types.Error
//	@Failure		422		{object}	types.Error
//	@Failure		500		{object}	types.Error
//
//	@Router			/jobrunner/job/ [post]
func RunTest(c echo.Context) error {
	type requestData struct {
		types.JobArgs
	}

	var rdata requestData

	err := c.Bind(&rdata)
	if err != nil {
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.StringError("failed to parse request data"),
		)
	}

	err = c.Validate(rdata)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, types.ValidationError(err))
	}

	return c.JSON(http.StatusOK, types.JobResponse{
		JobID:  uuid.New().String(),
		Status: types.SubmissionStatusAccepted,
	})
}
