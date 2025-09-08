package v1

import (
	"encoding/base64"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/validator"
)

// SubmitVuln submits a vulnerability for testing
//
//	@Summary		Submit Vulnerability
//	@Description	submit a vulnerability for testing
//	@Tags			pov
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Param			task_id	path		string				true	"Task ID"	Format(uuid)
//	@Param			payload	body		types.POVSubmission	true	"Submission body"
//
//	@Success		200		{object}	types.POVSubmissionResponse
//
//	@Failure		400		{object}	types.Error
//	@Failure		401		{object}	types.Error
//	@Failure		404		{object}	types.Error
//	@Failure		500		{object}	types.Error
//
//	@Router			/v1/task/{task_id}/pov/ [post]
func SubmitPOV(c echo.Context) error {
	type requestData struct {
		TaskID string `param:"task_id" validate:"required,uuid_rfc4122"`
		types.POVSubmission
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

	// 2^20 = 1mb
	if !validator.ValidateTriggerSize(len(rdata.Testcase)) {
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.Error{Message: "validation error", Fields: &map[string]string{
				"testcase": "must be <= 2mb",
			}},
		)
	}

	_, err = base64.StdEncoding.DecodeString(rdata.Testcase)
	if err != nil {
		// unreachable under nominal conditions
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.Error{Message: "failed to decode base64", Fields: &map[string]string{
				"testcase": "must be valid base64",
			}},
		)
	}

	return c.JSON(http.StatusOK, types.POVSubmissionResponse{
		POVID:  uuid.New().String(),
		Status: types.SubmissionStatusAccepted})
}

// VulnStatus yields the status of the vuln testing
//
//	@Summary		Vulnerability Status
//	@Description	yield the status of vuln testing
//	@Tags			pov
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Param			task_id	path		string	true	"Task ID"	Format(uuid)
//	@Param			pov_id	path		string	true	"POV ID"	Format(uuid)
//
//	@Success		200		{object}	types.POVSubmissionResponse
//
//	@Failure		400		{object}	types.Error
//	@Failure		401		{object}	types.Error
//	@Failure		404		{object}	types.Error
//	@Failure		500		{object}	types.Error
//
//	@Router			/v1/task/{task_id}/pov/{pov_id}/ [get]
func POVStatus(c echo.Context) error {
	type requestData struct {
		TaskID         string `param:"task_id" validate:"uuid_rfc4122,required"`
		VulnerabiliyID string `param:"pov_id"  validate:"uuid_rfc4122,required"`
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

	return c.JSON(http.StatusOK, types.POVSubmissionResponse{
		POVID:  rdata.VulnerabiliyID,
		Status: types.SubmissionStatusPassed})
}
