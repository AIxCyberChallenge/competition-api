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

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/validator"
)

var sarifAssessmentTestTable = map[string]struct {
	taskID               string
	sarifID              string
	assessment           string
	description          string
	expectedBodyFragment string
	expectedStatus       int
}{
	"Valid": {
		taskID:               uuid.NewString(),
		sarifID:              uuid.NewString(),
		assessment:           string(types.AssessmentCorrect),
		description:          "very insightful",
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"ValidUpper": {
		taskID:               strings.ToUpper(uuid.NewString()),
		sarifID:              strings.ToUpper(uuid.NewString()),
		assessment:           string(types.AssessmentCorrect),
		description:          "very insightful",
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"ValidLower": {
		taskID:               strings.ToLower(uuid.NewString()),
		sarifID:              strings.ToLower(uuid.NewString()),
		assessment:           string(types.AssessmentCorrect),
		description:          "very insightful",
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"ValidInvalidAssessment": {
		taskID:               uuid.NewString(),
		sarifID:              uuid.NewString(),
		assessment:           string(types.AssessmentIncorrect),
		description:          "very insightful",
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"InvalidTaskID": {
		taskID:               "foobar",
		sarifID:              uuid.NewString(),
		assessment:           string(types.AssessmentCorrect),
		description:          "very insightful",
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: "task_id\":",
	},
	"InvalidSarifID": {
		taskID:               uuid.NewString(),
		sarifID:              "foobar",
		assessment:           string(types.AssessmentCorrect),
		description:          "very insightful",
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: "broadcast_sarif_id\":",
	},
	"InvalidMissingAssessment": {
		taskID:               uuid.NewString(),
		sarifID:              uuid.NewString(),
		assessment:           "",
		description:          "very insightful",
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: "assessment\":",
	},
	"InvalidInvalidAssessment": {
		taskID:               uuid.NewString(),
		sarifID:              uuid.NewString(),
		assessment:           "i am an invalid assessment",
		description:          "very insightful",
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: "assessment\":",
	},
	"InvalidMissingDescription": {
		taskID:               uuid.NewString(),
		sarifID:              uuid.NewString(),
		assessment:           string(types.AssessmentCorrect),
		description:          "",
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: "description\":",
	},
}

func TestSubmitSarifAssessment(t *testing.T) {
	e := echo.New()
	validate := validator.Create()
	e.Validator = &validate

	for testName, testData := range sarifAssessmentTestTable {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			payload := fmt.Sprintf(
				`{"assessment":"%s","description":"%s"}`,
				testData.assessment,
				testData.description,
			)

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			c := e.NewContext(req, rec)
			c.SetPath("/v1/task/:task_id/broadcast-sarif-assessment/:broadcast_sarif_id")
			c.SetParamNames("task_id", "broadcast_sarif_id")
			c.SetParamValues(testData.taskID, testData.sarifID)

			doRequest(e, c, SubmitSarifAssessment)

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
