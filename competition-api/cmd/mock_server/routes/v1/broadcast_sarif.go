package v1

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

// SubmitSarifAssessment submits a sarif assessment
//
//	@Summary		Submit a SARIF Assessment
//	@Description	Submit a SARIF assessment
//	@Tags			broadcast-sarif-assessment
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Param			task_id				path		string							true	"Task ID"				Format(uuid)
//	@Param			broadcast_sarif_id	path		string							true	"Broadcast SARIF ID"	Format(uuid)
//	@Param			payload				body		types.SarifAssessmentSubmission	true	"Submission body"
//
//	@Success		200					{object}	types.SarifAssessmentResponse
//
//	@Failure		400					{object}	types.Error
//	@Failure		401					{object}	types.Error
//	@Failure		404					{object}	types.Error
//	@Failure		500					{object}	types.Error
//
//	@Router			/v1/task/{task_id}/broadcast-sarif-assessment/{broadcast_sarif_id}/ [post]
func SubmitSarifAssessment(c echo.Context) error {
	type requestData struct {
		TaskID  string `param:"task_id"            validate:"required,uuid_rfc4122"`
		SarifID string `param:"broadcast_sarif_id" validate:"required,uuid_rfc4122"`
		// Enum for assessment is not currently enforced by the validator
		types.SarifAssessmentSubmission
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

	return c.JSON(
		http.StatusOK,
		types.SarifAssessmentResponse{Status: types.SubmissionStatusAccepted},
	)
}
