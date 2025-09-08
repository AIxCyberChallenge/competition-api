package v1

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/santhosh-tekuri/jsonschema/v5"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/sarif"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

// SubmitSarifAssessment submits a sarif assessment
//
//	@Summary		Submit a CRS generated SARIF
//	@Description	Submit a CRS generated SARIF
//	@Tags			submitted-sarif
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Param			task_id	path		string					true	"Task ID"	Format(uuid)
//	@Param			payload	body		types.SARIFSubmission	true	"Submission body"
//
//	@Success		200		{object}	types.SARIFSubmissionResponse
//
//	@Failure		400		{object}	types.Error
//	@Failure		401		{object}	types.Error
//	@Failure		404		{object}	types.Error
//	@Failure		500		{object}	types.Error
//
//	@Router			/v1/task/{task_id}/submitted-sarif/ [post]
func SubmitSarif(c echo.Context) error {
	type requestData struct {
		// Enum for assessment is not currently enforced by the validator
		types.SARIFSubmission
		TaskID string `param:"task_id" validate:"required,uuid_rfc4122"`
	}

	var rdata requestData

	err := c.Bind(&rdata)
	if err != nil {
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.StringError("failed parsing request data"),
		)
	}

	err = c.Validate(rdata)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, types.ValidationError(err))
	}

	err = sarif.Schema.Validate(*rdata.SARIF)
	if validationErr, ok := err.(*jsonschema.ValidationError); ok {
		errs := validationErr.BasicOutput().Errors
		fieldMap := make(map[string]string, len(errs))
		for _, err := range errs {
			fieldMap[err.KeywordLocation] = err.Error
		}
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.Error{Message: "sarif failed to validate", Fields: &fieldMap},
		)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, types.StringError(err.Error()))
	}

	return c.JSON(
		http.StatusOK,
		types.SARIFSubmissionResponse{
			Status:           types.SubmissionStatusAccepted,
			SubmittedSARIFID: uuid.New().String(),
		},
	)
}
