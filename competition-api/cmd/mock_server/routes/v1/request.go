package v1

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

// RequestChallenge initiates a static delta testing task
//
//	@Summary		Send a task to the source of this request
//	@Description	Send a task to the source of this request
//	@Tags			request
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Param			challenge_name	path		string					true	"Challenge Name"
//	@Param			payload			body		types.RequestSubmission	true	"Submission Body"
//
//	@Success		200				{object}	types.Message
//
//	@Failure		400				{object}	types.Error
//	@Failure		401				{object}	types.Error
//	@Failure		404				{object}	types.Error
//	@Failure		500				{object}	types.Error
//
//	@Router			/v1/request/{challenge_name} [post]
func RequestChallenge(c echo.Context) error {
	type requestData struct {
		types.RequestSubmission
		ChallengeName string `param:"challenge_name" validate:"required"`
	}

	var hourSecs int64 = 3600
	rdata := requestData{
		RequestSubmission: types.RequestSubmission{DurationSecs: &hourSecs},
	}

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
		types.Message{Message: "received task request on mock server.  not doing anything"},
	)
}

// RequestList gets the list of challenges a team can request to task
//
//	@Summary		Get a list of available challenges to task
//	@Description	Get a list of available challenges to task
//	@Tags			request
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Success		200	{object}	types.RequestListResponse
//
//	@Failure		400	{object}	types.Error
//	@Failure		401	{object}	types.Error
//	@Failure		404	{object}	types.Error
//	@Failure		500	{object}	types.Error
//
//	@Router			/v1/request/list/ [get]
func RequestList(c echo.Context) error {
	challengeList := []string{"challenge1", "challenge2"}
	return c.JSON(http.StatusOK, types.RequestListResponse{Challenges: challengeList})
}
