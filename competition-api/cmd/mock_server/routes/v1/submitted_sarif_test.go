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

var sarifSubmissionTestTable = map[string]struct {
	taskID               string
	sarif                string
	expectedBodyFragment string
	expectedStatus       int
}{
	"Valid": {
		taskID:               uuid.NewString(),
		sarif:                `{"version": "2.1.0", "runs":[]}`,
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"ValidUpper": {
		taskID:               strings.ToUpper(uuid.NewString()),
		sarif:                `{"version": "2.1.0", "runs":[]}`,
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"ValidLower": {
		taskID:               strings.ToLower(uuid.NewString()),
		sarif:                `{"version": "2.1.0", "runs":[]}`,
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"InvalidTaskID": {
		taskID:               "foobar",
		sarif:                `{"version": "2.1.0", "runs":[]}`,
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: "task_id\":",
	},
	"InvalidInvalidSARIF": {
		taskID:               uuid.NewString(),
		sarif:                `{}`,
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: `missing properties: 'version', 'runs'`,
	},
	"InvalidMissingSARIF": {
		taskID:               uuid.NewString(),
		sarif:                `null`,
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: `"sarif":"Failed to validate`,
	},
}

func TestSubmitSarif(t *testing.T) {
	e := echo.New()
	validate := validator.Create()
	e.Validator = &validate

	for testName, testData := range sarifSubmissionTestTable {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			payload := fmt.Sprintf(`{"sarif":%s}`, testData.sarif)

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			c := e.NewContext(req, rec)
			c.SetPath("/v1/task/:task_id/submitted-sarif/")
			c.SetParamNames("task_id")
			c.SetParamValues(testData.taskID)

			doRequest(e, c, SubmitSarif)

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
