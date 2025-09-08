package v1

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

// SubmitBundle submits a bundle
//
//	@Summary		Submit Bundle
//	@Description	submits a bundle
//	@Tags			bundle
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Param			task_id	path		string					true	"Task ID"	Format(uuid)
//	@Param			payload	body		types.BundleSubmission	true	"Submission Body"
//
//	@Success		200		{object}	types.BundleSubmissionResponse
//
//	@Failure		400		{object}	types.Error
//	@Failure		401		{object}	types.Error
//	@Failure		404		{object}	types.Error
//	@Failure		500		{object}	types.Error
//
//	@Router			/v1/task/{task_id}/bundle/ [post]
func SubmitBundle(c echo.Context) error {
	type requestData struct {
		types.BundleSubmission
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

	err = rdata.ValidateFieldCount()
	if err != nil {
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.StringError("must set at least 2 fields"),
		)
	}

	_, err = rdata.Parse()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, types.StringError("failed to parse fields"))
	}

	return c.JSON(
		http.StatusOK,
		types.BundleSubmissionResponse{
			BundleID: uuid.New().String(),
			Status:   types.SubmissionStatusAccepted,
		},
	)
}

// PatchBundle updates a bundle
//
//	@Summary		Update Bundle
//	@Description	updates a bundle
//	@Tags			bundle
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Param			task_id		path		string					true	"Task ID"	Format(uuid)
//	@Param			bundle_id	path		string					true	"Bundle ID"	Format(uuid)
//	@Param			payload		body		types.BundleSubmission	true "Submission Body"
//
//	@Success		200			{object}	types.BundleSubmissionResponseVerbose
//
//	@Failure		400			{object}	types.Error
//	@Failure		401			{object}	types.Error
//	@Failure		404			{object}	types.Error
//	@Failure		500			{object}	types.Error
//
//	@Router			/v1/task/{task_id}/bundle/{bundle_id}/ [patch]
func PatchBundle(c echo.Context) error {
	type requestData struct {
		types.BundleSubmission
		TaskID   string `param:"task_id"   validate:"required,uuid_rfc4122"`
		BundleID string `param:"bundle_id" validate:"required,uuid_rfc4122"`
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

	parsed, err := rdata.Parse()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, types.StringError("failed to parse fields"))
	}

	bundleID, err := uuid.Parse(rdata.BundleID)
	if err != nil {
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.StringError("failed to parse bundle id"),
		)
	}

	return c.JSON(http.StatusOK, types.BundleSubmissionResponseVerbose{
		BundleSubmissionResponse: types.BundleSubmissionResponse{
			BundleID: bundleID.String(),
			Status:   types.SubmissionStatusAccepted,
		},
		BundleSubmissionResponseBody: types.BundleSubmissionResponseBody{
			POVID:            parsed.POVID.Value,
			PatchID:          parsed.PatchID.Value,
			BroadcastSARIFID: parsed.BroadcastSARIFID.Value,
			SubmittedSARIFID: parsed.SubmittedSARIFID.Value,
			Description:      parsed.Description.Value,
			FreeformID:       parsed.FreeformID.Value,
		},
	})
}

// GetBundle gets a bundle
//
//	@Summary		Get Bundle
//	@Description	get a bundle
//	@Tags			bundle
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Param			task_id		path		string	true	"Task ID"	Format(uuid)
//	@Param			bundle_id	path		string	true	"Bundle ID"	Format(uuid)
//
//	@Success		200			{object}	types.BundleSubmissionResponseVerbose
//
//	@Failure		400			{object}	types.Error
//	@Failure		401			{object}	types.Error
//	@Failure		404			{object}	types.Error
//	@Failure		500			{object}	types.Error
//
//	@Router			/v1/task/{task_id}/bundle/{bundle_id}/ [get]
func GetBundle(c echo.Context) error {
	type requestData struct {
		TaskID   string `param:"task_id"   validate:"required,uuid_rfc4122"`
		BundleID string `param:"bundle_id" validate:"required,uuid_rfc4122"`
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

	bundleID, err := uuid.Parse(rdata.BundleID)
	if err != nil {
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.StringError("failed to parse bundle id"),
		)
	}

	povID := uuid.New()
	patchID := uuid.New()
	return c.JSON(http.StatusOK, types.BundleSubmissionResponseVerbose{
		BundleSubmissionResponse: types.BundleSubmissionResponse{
			BundleID: bundleID.String(),
			Status:   types.SubmissionStatusAccepted,
		},
		BundleSubmissionResponseBody: types.BundleSubmissionResponseBody{
			POVID:   &povID,
			PatchID: &patchID,
		},
	})
}

// DeleteBundle deletes a bundle
//
//	@Summary		Delete Bundle
//	@Description	delete a bundle
//	@Tags			bundle
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Param			task_id		path		string	true	"Task ID"	Format(uuid)
//	@Param			bundle_id	path		string	true	"Bundle ID"	Format(uuid)
//
//	@Success		204			{object}	string	"No Content"
//
//	@Failure		400			{object}	types.Error
//	@Failure		401			{object}	types.Error
//	@Failure		404			{object}	types.Error
//	@Failure		500			{object}	types.Error
//
//	@Router			/v1/task/{task_id}/bundle/{bundle_id}/ [delete]
func DeleteBundle(c echo.Context) error {
	type requestData struct {
		TaskID   string `param:"task_id"   validate:"required,uuid_rfc4122"`
		BundleID string `param:"bundle_id" validate:"required,uuid_rfc4122"`
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

	return c.NoContent(http.StatusNoContent)
}
