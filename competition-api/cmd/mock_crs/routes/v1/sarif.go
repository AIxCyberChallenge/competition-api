package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"golang.org/x/sync/errgroup"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

// SubmitSARIFBroadcast notify the CRS of a potential vulnerability to work on
//
//	@Summary		Submit Sarif Broadcast
//	@Description	submit a sarif broadcast
//	@Tags			sarif
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Param			payload	body		types.SARIFBroadcast	true	"Vulnerability Broadcast"
//
//	@Success		200		{string}	string					"No Content"
//
//	@Router			/v1/sarif/ [post]
func SubmitSARIFBroadcast(c echo.Context) error {
	type requestData struct {
		types.SARIFBroadcast
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

	eg := errgroup.Group{}

	eg.Go(func() error {
		broadcast := rdata.Broadcasts[0]
		sarifAssessment := types.SarifAssessmentSubmission{
			Assessment:  types.AssessmentCorrect,
			Description: "i am very smart crs",
		}

		sarifAssessmentPayload, fail := json.Marshal(sarifAssessment)
		if fail != nil {
			log.Println(fail)
			return fail
		}

		req, fail := http.NewRequest(
			http.MethodPost,
			fmt.Sprintf(
				"%s/v1/task/%s/broadcast-sarif-assessment/%s/",
				apiURL,
				broadcast.TaskID,
				broadcast.SARIFID,
			),
			bytes.NewReader(sarifAssessmentPayload),
		)
		if fail != nil {
			log.Println(fail)
			return fail
		}
		req.SetBasicAuth(apiCRSID, apiCRSToken)
		req.Header.Set("Content-Type", "application/json")
		resp, fail := http.DefaultClient.Do(req)
		if fail != nil {
			log.Println(fail)
			return fail
		}
		defer resp.Body.Close()
		reply, fail := io.ReadAll(resp.Body)
		if fail != nil {
			log.Println(fail)
			return fail
		}
		if resp.StatusCode > 299 {
			log.Println("error sending request", resp.StatusCode, string(reply))
			return fail
		}
		return nil
	})

	return c.NoContent(http.StatusOK)
}
