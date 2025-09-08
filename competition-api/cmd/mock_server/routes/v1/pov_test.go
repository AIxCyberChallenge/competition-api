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

var vulnSubmissionTestTable = map[string]struct {
	taskID               string
	dataFile             string
	harnessName          string
	sanitizer            string
	architecture         string
	engine               string
	expectedBodyFragment string
	expectedStatus       int
}{
	"Valid": {
		taskID:               uuid.NewString(),
		dataFile:             base64String(10),
		harnessName:          "harness_1",
		sanitizer:            "address",
		architecture:         "x86_64",
		engine:               "libfuzzer",
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"ValidUpper": {
		taskID:               strings.ToUpper(uuid.NewString()),
		dataFile:             base64String(10),
		harnessName:          "harness_1",
		sanitizer:            "address",
		architecture:         "x86_64",
		engine:               "libfuzzer",
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"ValidLower": {
		taskID:               strings.ToLower(uuid.NewString()),
		dataFile:             base64String(10),
		harnessName:          "harness_1",
		sanitizer:            "address",
		architecture:         "x86_64",
		engine:               "libfuzzer",
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"ValidSarifIncluded": {
		taskID:               uuid.NewString(),
		dataFile:             base64String(10),
		harnessName:          "harness_1",
		sanitizer:            "address",
		architecture:         "x86_64",
		engine:               "libfuzzer",
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"ValidLongDataFile": {
		taskID:               uuid.NewString(),
		dataFile:             base64String(1 << 21),
		harnessName:          "harness_1",
		sanitizer:            "address",
		architecture:         "x86_64",
		engine:               "libfuzzer",
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"InvalidTestcaseTooLong": {
		taskID:               uuid.NewString(),
		dataFile:             base64String(1<<21 + 100),
		harnessName:          "harness_1",
		sanitizer:            "address",
		architecture:         "x86_64",
		engine:               "libfuzzer",
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: "testcase\":",
	},
	"InvalidTestcaseNotBase64": {
		taskID:               uuid.NewString(),
		dataFile:             "foobar",
		harnessName:          "harness_1",
		sanitizer:            "address",
		architecture:         "x86_64",
		engine:               "libfuzzer",
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: "testcase\":",
	},
	"InvalidMissingFuzzerName": {
		taskID:               uuid.NewString(),
		dataFile:             base64String(10),
		harnessName:          "",
		sanitizer:            "address",
		architecture:         "x86_64",
		engine:               "libfuzzer",
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: `"fuzzer_name":`,
	},
	"InvalidMissingSanitizer": {
		taskID:               uuid.NewString(),
		dataFile:             base64String(10),
		harnessName:          "harness",
		sanitizer:            "",
		architecture:         "x86_64",
		engine:               "libfuzzer",
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: `"sanitizer":`,
	},
	"InvalidMissingArchitecture": {
		taskID:               uuid.NewString(),
		dataFile:             base64String(10),
		harnessName:          "harness",
		sanitizer:            "address",
		architecture:         "",
		engine:               "libfuzzer",
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: `"architecture":`,
	},
}

func TestPOVSubmission(t *testing.T) {
	e := echo.New()
	validate := validator.Create()
	e.Validator = &validate

	for testName, testData := range vulnSubmissionTestTable {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			payload := fmt.Sprintf(
				`{"testcase":"%s","fuzzer_name":"%s","sanitizer":"%s","architecture":"%s", "engine": "%s"}`,
				testData.dataFile,
				testData.harnessName,
				testData.sanitizer,
				testData.architecture,
				testData.engine,
			)
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			c := e.NewContext(req, rec)
			c.SetPath("/v1/task/:task_id/vuln/")
			c.SetParamNames("task_id")
			c.SetParamValues(testData.taskID)

			doRequest(e, c, SubmitPOV)

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

var vulnStatusTestTable = map[string]struct {
	taskID               string
	vulnID               string
	expectedBodyFragment string
	expectedStatus       int
}{
	"Valid": {
		taskID:               uuid.NewString(),
		vulnID:               uuid.NewString(),
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"ValidUpper": {
		taskID:               strings.ToUpper(uuid.NewString()),
		vulnID:               strings.ToUpper(uuid.NewString()),
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"ValidLower": {
		taskID:               strings.ToLower(uuid.NewString()),
		vulnID:               strings.ToLower(uuid.NewString()),
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: "status\":",
	},
	"InvalidTaskID": {
		taskID:               "foobar",
		vulnID:               uuid.NewString(),
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: "task_id\":",
	},
	"InvalidVulnID": {
		taskID:               uuid.NewString(),
		vulnID:               "foobar",
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: "pov_id\":",
	},
}

func TestPOVStatus(t *testing.T) {
	e := echo.New()
	validate := validator.Create()
	e.Validator = &validate

	for testName, testData := range vulnStatusTestTable {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(""))
			rec := httptest.NewRecorder()

			c := e.NewContext(req, rec)
			c.SetPath("/v1/task/:task_id/vuln/:pov_id/")
			c.SetParamNames("task_id", "pov_id")
			c.SetParamValues(testData.taskID, testData.vulnID)

			doRequest(e, c, POVStatus)

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
