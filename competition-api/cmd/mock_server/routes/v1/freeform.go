package v1

import (
	"encoding/base64"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/validator"
)

// SubmitFreeform submits a freeform pov.
//
//	@Summary		Submit Freeform
//	@Description	CRSs may submit anything to this endpoint as a base64'd string.  This is the only way to submit PoVs and Patches for unharnessed challenges, but will be open for other challenges also.   Nothing submitted to this endpoint is evaluated automatically.  Bundles can only contain one freeform_id, so pack as much info into the Freeform as you need.
//	@Tags			freeform
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Param			task_id	path		string						true	"Task ID"	Format(uuid)
//	@Param			payload	body		types.FreeformSubmission	true	"Submission Body"
//
//	@Success		200		{object}	types.FreeformResponse
//
//	@Failure		400		{object}	types.Error
//	@Failure		401		{object}	types.Error
//	@Failure		404		{object}	types.Error
//	@Failure		500		{object}	types.Error
//
//	@Router			/v1/task/{task_id}/freeform/ [post]
func SubmitFreeform(c echo.Context) error {
	type requestData struct {
		types.FreeformSubmission
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

	if !validator.ValidateFreeformSize(len(rdata.Submission)) {
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.Error{Message: "validation error", Fields: &map[string]string{
				"submission": "must be <= 2mb",
			}},
		)
	}

	_, err = base64.StdEncoding.DecodeString(rdata.Submission)
	if err != nil {
		// unreachable under nominal conditions
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.Error{Message: "failed to decode base64", Fields: &map[string]string{
				"submission": "must be valid base64",
			}},
		)
	}

	return c.JSON(http.StatusOK, types.FreeformResponse{
		FreeformID: uuid.New().String(),
		Status:     types.SubmissionStatusAccepted})
}
