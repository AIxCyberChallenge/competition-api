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

var bundleSubmitTestTable = map[string]struct {
	taskID               string
	povID                string
	patchID              string
	expectedBodyFragment string
	expectedStatus       int
}{
	"Valid": {
		taskID:               uuid.NewString(),
		povID:                fmt.Sprintf(`"%s"`, uuid.NewString()),
		patchID:              fmt.Sprintf(`"%s"`, uuid.NewString()),
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: `status":`,
	},
	"ValidUpper": {
		taskID:               uuid.NewString(),
		povID:                strings.ToUpper(fmt.Sprintf(`"%s"`, uuid.NewString())),
		patchID:              strings.ToUpper(fmt.Sprintf(`"%s"`, uuid.NewString())),
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: `status":`,
	},
	"ValidLower": {
		taskID:               uuid.NewString(),
		povID:                strings.ToLower(fmt.Sprintf(`"%s"`, uuid.NewString())),
		patchID:              strings.ToLower(fmt.Sprintf(`"%s"`, uuid.NewString())),
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: `status":`,
	},
	"InvalidTaskID": {
		taskID:               "foobar",
		povID:                fmt.Sprintf(`"%s"`, uuid.NewString()),
		patchID:              fmt.Sprintf(`"%s"`, uuid.NewString()),
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: "task_id\":",
	},
	"InvalidPOVID": {
		taskID:               uuid.NewString(),
		povID:                `"foobar"`,
		patchID:              fmt.Sprintf(`"%s"`, uuid.NewString()),
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: `failed to parse fields`,
	},
	"InvalidNot2": {
		taskID:               uuid.NewString(),
		povID:                `null`,
		patchID:              fmt.Sprintf(`"%s"`, uuid.NewString()),
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: `must set at least 2 fields`,
	},
}

func TestSubmitBundle(t *testing.T) {
	e := echo.New()
	validate := validator.Create()
	e.Validator = &validate

	for testName, testData := range bundleSubmitTestTable {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			payload := fmt.Sprintf(`{"pov_id":%s,"patch_id":%s}`, testData.povID, testData.patchID)

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			c := e.NewContext(req, rec)
			c.SetPath("/v1/task/:task_id/bundle/")
			c.SetParamNames("task_id")
			c.SetParamValues(testData.taskID)

			doRequest(e, c, SubmitBundle)

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

var bundlePatchTestTable = map[string]struct {
	taskID               string
	bundleID             string
	povID                string
	patchID              string
	expectedBodyFragment string
	expectedStatus       int
}{
	"Valid": {
		taskID:               uuid.NewString(),
		bundleID:             uuid.NewString(),
		povID:                fmt.Sprintf(`"%s"`, uuid.NewString()),
		patchID:              fmt.Sprintf(`"%s"`, uuid.NewString()),
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: `status":`,
	},
	"ValidUpper": {
		taskID:               uuid.NewString(),
		bundleID:             uuid.NewString(),
		povID:                strings.ToUpper(fmt.Sprintf(`"%s"`, uuid.NewString())),
		patchID:              strings.ToUpper(fmt.Sprintf(`"%s"`, uuid.NewString())),
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: `status":`,
	},
	"ValidLower": {
		taskID:               uuid.NewString(),
		bundleID:             uuid.NewString(),
		povID:                strings.ToLower(fmt.Sprintf(`"%s"`, uuid.NewString())),
		patchID:              strings.ToLower(fmt.Sprintf(`"%s"`, uuid.NewString())),
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: `status":`,
	},
	"ValidAllNull": {
		taskID:               uuid.NewString(),
		bundleID:             uuid.NewString(),
		povID:                `null`,
		patchID:              `null`,
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: `status`,
	},
	"InvalidTaskID": {
		taskID:               "foobar",
		bundleID:             uuid.NewString(),
		povID:                fmt.Sprintf(`"%s"`, uuid.NewString()),
		patchID:              fmt.Sprintf(`"%s"`, uuid.NewString()),
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: "task_id\":",
	},
	"InvalidPOVID": {
		taskID:               uuid.NewString(),
		bundleID:             uuid.NewString(),
		povID:                `"foobar"`,
		patchID:              fmt.Sprintf(`"%s"`, uuid.NewString()),
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: `failed to parse fields`,
	},
}

func TestPatchBundle(t *testing.T) {
	e := echo.New()
	validate := validator.Create()
	e.Validator = &validate

	for testName, testData := range bundlePatchTestTable {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			payload := fmt.Sprintf(`{"pov_id":%s,"patch_id":%s}`, testData.povID, testData.patchID)

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			c := e.NewContext(req, rec)
			c.SetPath("/v1/task/:task_id/bundle/:bundle_id/")
			c.SetParamNames("task_id", "bundle_id")
			c.SetParamValues(testData.taskID, testData.bundleID)

			doRequest(e, c, PatchBundle)

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

var bundleGetTestTable = map[string]struct {
	taskID               string
	bundleID             string
	expectedBodyFragment string
	expectedStatus       int
}{
	"Valid": {
		taskID:               uuid.NewString(),
		bundleID:             uuid.NewString(),
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: `status":`,
	},
	"ValidUpper": {
		taskID:               strings.ToUpper(uuid.NewString()),
		bundleID:             strings.ToUpper(uuid.NewString()),
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: `status":`,
	},
	"ValidLower": {
		taskID:               strings.ToLower(uuid.NewString()),
		bundleID:             strings.ToLower(uuid.NewString()),
		expectedStatus:       http.StatusOK,
		expectedBodyFragment: `status":`,
	},
	"InvalidTaskID": {
		taskID:               "foobar",
		bundleID:             uuid.NewString(),
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: "task_id\":",
	},
	"InvalidBundleID": {
		taskID:               uuid.NewString(),
		bundleID:             "foobar",
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: `bundle_id":`,
	},
}

func TestGetBundle(t *testing.T) {
	e := echo.New()
	validate := validator.Create()
	e.Validator = &validate

	for testName, testData := range bundleGetTestTable {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodDelete, "/", nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			c := e.NewContext(req, rec)
			c.SetPath("/v1/task/:task_id/bundle/:bundle_id")
			c.SetParamNames("task_id", "bundle_id")
			c.SetParamValues(testData.taskID, testData.bundleID)

			doRequest(e, c, GetBundle)

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

var bundleDeleteTestTable = map[string]struct {
	taskID               string
	bundleID             string
	expectedBodyFragment string
	expectedStatus       int
}{
	"Valid": {
		taskID:               uuid.NewString(),
		bundleID:             uuid.NewString(),
		expectedStatus:       http.StatusNoContent,
		expectedBodyFragment: "",
	},
	"ValidUpper": {
		taskID:               strings.ToUpper(uuid.NewString()),
		bundleID:             strings.ToUpper(uuid.NewString()),
		expectedStatus:       http.StatusNoContent,
		expectedBodyFragment: "",
	},
	"ValidLower": {
		taskID:               strings.ToLower(uuid.NewString()),
		bundleID:             strings.ToLower(uuid.NewString()),
		expectedStatus:       http.StatusNoContent,
		expectedBodyFragment: "",
	},
	"InvalidTaskID": {
		taskID:               "foobar",
		bundleID:             uuid.NewString(),
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: "task_id\":",
	},
	"InvalidBundleID": {
		taskID:               uuid.NewString(),
		bundleID:             "foobar",
		expectedStatus:       http.StatusBadRequest,
		expectedBodyFragment: `bundle_id":`,
	},
}

func TestDeleteBundle(t *testing.T) {
	e := echo.New()
	validate := validator.Create()
	e.Validator = &validate

	for testName, testData := range bundleDeleteTestTable {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodDelete, "/", nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			c := e.NewContext(req, rec)
			c.SetPath("/v1/task/:task_id/bundle/:bundle_id")
			c.SetParamNames("task_id", "bundle_id")
			c.SetParamValues(testData.taskID, testData.bundleID)

			doRequest(e, c, DeleteBundle)

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
