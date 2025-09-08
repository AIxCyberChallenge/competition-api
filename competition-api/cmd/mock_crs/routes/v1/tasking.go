package v1

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"golang.org/x/sync/errgroup"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

// SubmitTask submits a task for work
//
//	@Summary		Submit Task
//	@Description	submit a task for work
//	@Tags			task
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Param			payload	body		types.Task	true	"Submission body"
//
//	@Success		202		{string}	string		"No Content"
//
//	@Router			/v1/task/ [post]
func SubmitTask(c echo.Context) error {
	type requestData struct {
		types.Task
	}

	var rdata requestData

	if err := c.Bind(&rdata); err != nil {
		return echo.NewHTTPError(
			http.StatusBadRequest,
			types.StringError("failed parsing request data"),
		)
	}

	if err := c.Validate(rdata); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, types.ValidationError(err))
	}

	go func() {
		eg := errgroup.Group{}

		vulnStatus := types.POVSubmissionResponse{}

		task := rdata.Tasks[0]

		eg.Go(func() error {
			povData, err := os.ReadFile("/data/data.bin")
			if err != nil {
				return err
			}
			data := base64.StdEncoding.EncodeToString(povData)
			vulnSubmission := types.POVSubmission{
				Testcase:     data,
				Engine:       "libfuzzer",
				FuzzerName:   "xml",
				Sanitizer:    "address",
				Architecture: "x86_64",
			}

			jsonVulnSubmission, fail := json.Marshal(vulnSubmission)
			if fail != nil {
				log.Println(fail)
				return fail
			}

			req, fail := http.NewRequest(
				http.MethodPost,
				fmt.Sprintf("%s/v1/task/%s/pov", apiURL, task.TaskID),
				bytes.NewReader(jsonVulnSubmission),
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
				log.Println("error sending PoV request", resp.StatusCode, string(reply))
				return fail
			}
			log.Printf("Vuln Submission: %s\n", string(reply))
			fail = json.Unmarshal(reply, &vulnStatus)
			if fail != nil {
				log.Println(fail)
				return fail
			}
			return nil
		})
		if err := eg.Wait(); err != nil {
			log.Println("Error in good POV Submission:")
			log.Println(err)
		}

		eg = errgroup.Group{}
		eg.Go(func() error {
			patchData, err := os.ReadFile("/data/patch.patch")
			if err != nil {
				return err
			}
			data := base64.StdEncoding.EncodeToString(patchData)
			patch := types.PatchSubmission{
				Patch: data,
			}
			patchBody, fail := json.Marshal(patch)
			if fail != nil {
				log.Println("Error marshaling patch data")
			}
			req, fail := http.NewRequest(
				http.MethodPost,
				fmt.Sprintf("%s/v1/task/%s/patch", apiURL, task.TaskID),
				bytes.NewReader(patchBody),
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
			if resp.StatusCode > 299 {
				log.Println(
					"error sending Patch submission request",
					resp.StatusCode,
					string(reply),
				)
				return fail
			}
			log.Printf("Patch Submission: %s\n", string(reply))
			if fail != nil {
				log.Println(fail)
				return fail
			}
			patchStatus := types.PatchSubmissionResponse{}
			fail = json.Unmarshal(reply, &patchStatus)
			if fail != nil {
				log.Println("Error unmarshaling patch status")
				return fail
			}
			log.Println("Patch status: ", patchStatus)
			return nil
		})
		if err := eg.Wait(); err != nil {
			log.Println("error sending patch")
			log.Println(err)
		}
	}()

	return c.NoContent(http.StatusAccepted)
}

// CancelTasks cancels all previously submitted tasks
//
//	@Summary		Cancel Tasks
//	@Description	Cancel all previously submitted tasks. This is meant for edge case recovery in case something goes wrong.
//	@Tags			task
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Success		200	{string}	string	"No Content"
//
//	@Router			/v1/task/ [delete]
func CancelTasks(c echo.Context) error {
	return c.NoContent(http.StatusOK)
}

// CancelTask cancels a task previously submitted for work
//
//	@Summary		Cancel Task
//	@Description	Cancel a task by id. This is meant for edge case recovery in case something goes wrong.
//	@Tags			task
//	@Accept			json
//	@Produce		json
//
//	@Security		BasicAuth
//
//	@Param			task_id	path		string	true	"Task ID"	Format(uuid)
//
//	@Success		200		{string}	string	"No Content"
//
//	@Router			/v1/task/{task_id}/ [delete]
func CancelTask(c echo.Context) error {
	type requestData struct {
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

	return c.NoContent(http.StatusOK)
}
