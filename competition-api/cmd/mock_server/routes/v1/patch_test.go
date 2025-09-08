package v1

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/validator"
)

var patchSubmissionTestTable = map[string]struct {
	taskID               string
	patch                string
	expectedBodyFragment string
	expectedStatus       int
}{
	"Valid": {
		taskID:               uuid.NewString(),
		patch:                base64String(10),
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"ValidUpper": {
		taskID:               strings.ToUpper(uuid.NewString()),
		patch:                base64String(10),
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"ValidLower": {
		taskID:               strings.ToLower(uuid.NewString()),
		patch:                base64String(10),
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"ValidLongPatch": {
		taskID:               uuid.NewString(),
		patch:                base64String((1 << 10) * 100),
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"ValidNoDescription": {
		taskID:               uuid.NewString(),
		patch:                base64String(10),
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"InvalidTaskID": {
		taskID:               "foobar",
		patch:                base64String(10),
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: "task_id\":",
	},
	"InvalidPatchNotBase64": {
		taskID:               uuid.NewString(),
		patch:                "foobar",
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: "patch\":",
	},
	"InvalidPatchTooLong": {
		taskID:               uuid.NewString(),
		patch:                base64String((1<<10)*100 + 100),
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: "patch\":",
	},
	"InvalidPatchMissing": {
		taskID:               uuid.NewString(),
		patch:                "",
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: "patch\":",
	},
}

func TestPatchSubmission(t *testing.T) {
	e := echo.New()
	validate := validator.Create()
	e.Validator = &validate

	for testName, testData := range patchSubmissionTestTable {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			payload := fmt.Sprintf(`{"patch":"%s"}`, testData.patch)
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			c := e.NewContext(req, rec)
			c.SetPath("/v1/task/:task_id/patch/")
			c.SetParamNames("task_id")
			c.SetParamValues(testData.taskID)

			doRequest(e, c, SubmitPatch)

			assert.Equal(t, testData.expectedStatus, rec.Code, "Status code does not match")
			assert.Contains(
				t,
				rec.Body.String(),
				testData.expectedBodyFragment,
				"Body missing expected fragment",
			)
		})
	}
}

var patchStatusTestTable = map[string]struct {
	taskID               string
	patchID              string
	expectedBodyFragment string
	expectedStatus       int
}{
	"Valid": {
		taskID:               uuid.NewString(),
		patchID:              uuid.NewString(),
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"ValidUpper": {
		taskID:               strings.ToUpper(uuid.NewString()),
		patchID:              strings.ToUpper(uuid.NewString()),
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"ValidLower": {
		taskID:               strings.ToUpper(uuid.NewString()),
		patchID:              strings.ToLower(uuid.NewString()),
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"InvalidTaskID": {
		taskID:               "foobar",
		patchID:              uuid.NewString(),
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: "task_id\":",
	},
	"InvalidPatchID": {
		taskID:               uuid.NewString(),
		patchID:              "foobar",
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: "patch_id\":",
	},
}

func TestPatchStatus(t *testing.T) {
	e := echo.New()
	validate := validator.Create()
	e.Validator = &validate

	for testName, testData := range patchStatusTestTable {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(""))
			rec := httptest.NewRecorder()

			c := e.NewContext(req, rec)
			c.SetPath("/v1/task/:task_id/patch/:patch_id")
			c.SetParamNames("task_id", "patch_id")
			c.SetParamValues(testData.taskID, testData.patchID)

			doRequest(e, c, PatchStatus)

			assert.Equal(t, testData.expectedStatus, rec.Code, "Status code does not match")
			assert.Contains(
				t,
				rec.Body.String(),
				testData.expectedBodyFragment,
				"Body missing expected fragment",
			)
		})
	}
}
