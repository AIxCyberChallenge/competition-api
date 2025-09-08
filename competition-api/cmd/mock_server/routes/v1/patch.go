package v1

import (
	"encoding/base64"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/validator"
)

// SubmitPatch submits a patch for testing
//
//	@Summary		Submit Patch
//	@Description	submit a patch for testing
//	@Tags			patch
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Param			task_id	path		string					true	"Task ID"	Format(uuid)
//	@Param			payload	body		types.PatchSubmission	true	"Payload"
//
//	@Success		200		{object}	types.PatchSubmissionResponse
//
//	@Failure		400		{object}	types.Error
//	@Failure		401		{object}	types.Error
//	@Failure		404		{object}	types.Error
//	@Failure		500		{object}	types.Error
//
//	@Router			/v1/task/{task_id}/patch/ [post]
func SubmitPatch(c echo.Context) error {
	type requestData struct {
		TaskID string `param:"task_id" validate:"required,uuid_rfc4122"`
		types.PatchSubmission
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

	// 2^10 = 1kb
	if !validator.ValidatePatchSize(len(rdata.Patch)) {
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.Error{Message: "validation error", Fields: &map[string]string{
				"patch": "must be <= 100kb",
			}},
		)
	}

	_, err = base64.StdEncoding.DecodeString(rdata.Patch)
	if err != nil {
		// unreachable under nominal conditions
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.Error{Message: "failed to decode base64", Fields: &map[string]string{
				"patch": "must be valid base64",
			}},
		)
	}

	return c.JSON(http.StatusOK, types.PatchSubmissionResponse{
		PatchID: uuid.New().String(),
		Status:  types.SubmissionStatusAccepted})
}

// PatchStatus yields the status of the patch testing
//
//	@Summary		Patch Status
//	@Description	yield the status of patch testing
//	@Tags			patch
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Param			task_id		path		string	true	"Task ID"	Format(uuid)
//	@Param			patch_id	path		string	true	"Patch ID"	Format(uuid)
//
//	@Success		200			{object}	types.PatchSubmissionResponse
//
//	@Failure		400			{object}	types.Error
//	@Failure		401			{object}	types.Error
//	@Failure		404			{object}	types.Error
//	@Failure		500			{object}	types.Error
//
//	@Router			/v1/task/{task_id}/patch/{patch_id}/ [get]
func PatchStatus(c echo.Context) error {
	type requestData struct {
		TaskID  string `param:"task_id"  validate:"required,uuid_rfc4122"`
		PatchID string `param:"patch_id" validate:"required,uuid_rfc4122"`
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

	return c.JSON(http.StatusOK, types.PatchSubmissionResponse{
		PatchID: rdata.PatchID,
		Status:  types.SubmissionStatusPassed})
}
