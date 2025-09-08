package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func (s *ServerTestSuite) Test_POVSubmission() {
	tests := []struct {
		name           string
		auth           *clientAuth
		bodyTester     func(t *testing.T, body map[string]any)
		taskID         string
		dataFile       string
		harnessName    string
		sanitizer      string
		architecture   string
		engine         string
		expectedStatus int
	}{
		{
			name:           "Valid",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			dataFile:       base64String(10),
			harnessName:    longString(4096),
			sanitizer:      longString(4096),
			architecture:   "x86_64",
			engine:         "libfuzzer",
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:           "ValidUpper",
			taskID:         strings.ToUpper(taskOpen.ID.String()),
			auth:           &clientAuth{auth.ID.String(), authToken},
			dataFile:       base64String(10),
			harnessName:    longString(4096),
			sanitizer:      longString(4096),
			architecture:   "x86_64",
			engine:         "libfuzzer",
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:           "ValidLower",
			taskID:         strings.ToLower(taskOpen.ID.String()),
			auth:           &clientAuth{auth.ID.String(), authToken},
			dataFile:       base64String(10),
			harnessName:    longString(4096),
			sanitizer:      longString(4096),
			architecture:   "x86_64",
			engine:         "libfuzzer",
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:           "ValidRepeatData",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			dataFile:       base64String(10),
			harnessName:    "harness_1",
			sanitizer:      "address",
			architecture:   "x86_64",
			engine:         "libfuzzer",
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:   "ValidLongDataFile",
			taskID: taskOpen.ID.String(),
			auth:   &clientAuth{auth.ID.String(), authToken},
			// FIXME: this no longer tests the upper limit
			dataFile:       base64String(1 << 21),
			harnessName:    "harness_1",
			sanitizer:      "address",
			architecture:   "x86_64",
			engine:         "libfuzzer",
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:           "ValidAfterDeadline",
			taskID:         taskExpired.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			dataFile:       base64String(10),
			harnessName:    "harness",
			sanitizer:      "address",
			architecture:   "x86_64",
			engine:         "libfuzzer",
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "deadline_exceeded")
			},
		},
		{
			name:           "InvalidTestcaseTooLong",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			dataFile:       base64String((1 << 21) + 100),
			harnessName:    "harness_1",
			sanitizer:      "address",
			architecture:   "x86_64",
			engine:         "libfuzzer",
			expectedStatus: http.StatusBadRequest,
			bodyTester: func(t *testing.T, body map[string]any) {
				assertErrorBodyWithFields(t, body)
				assert.Contains(t, body["message"], "validation error")
				assert.Contains(t, body["fields"].(map[string]any)["testcase"], "must be <= 2mb")
			},
		},
		{
			name:           "InvalidFuzzerTooLong",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			dataFile:       base64String(10),
			harnessName:    longString(4097),
			sanitizer:      "address",
			architecture:   "x86_64",
			engine:         "libfuzzer",
			expectedStatus: http.StatusBadRequest,
			bodyTester: func(t *testing.T, body map[string]any) {
				assertErrorBodyWithFields(t, body)
				assert.Contains(t, body["message"], "validation error")
				assert.Contains(t, body["fields"].(map[string]any)["fuzzer_name"], "max")
			},
		},
		{
			name:           "InvalidSanitizerTooLong",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			dataFile:       base64String(10),
			harnessName:    "harness",
			sanitizer:      longString(4097),
			architecture:   "x86_64",
			engine:         "libfuzzer",
			expectedStatus: http.StatusBadRequest,
			bodyTester: func(t *testing.T, body map[string]any) {
				assertErrorBodyWithFields(t, body)
				assert.Contains(t, body["message"], "validation error")
				assert.Contains(t, body["fields"].(map[string]any)["sanitizer"], "max")
			},
		},
		{
			name:           "InvalidTestcaseNotBase64",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			dataFile:       "foobar",
			harnessName:    "harness_1",
			sanitizer:      "address",
			architecture:   "x86_64",
			engine:         "libfuzzer",
			expectedStatus: http.StatusBadRequest,
			bodyTester: func(t *testing.T, body map[string]any) {
				assertErrorBodyWithFields(t, body)
				assert.Contains(t, body["message"], "validation error")
				assert.Contains(
					t,
					body["fields"].(map[string]any)["testcase"],
					"Failed to validate while checking condition: base64",
				)
			},
		},
		{
			name:           "InvalidMissingFuzzerName",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			dataFile:       base64String(10),
			harnessName:    "",
			sanitizer:      "address",
			architecture:   "x86_64",
			engine:         "libfuzzer",
			expectedStatus: http.StatusBadRequest,
			bodyTester: func(t *testing.T, body map[string]any) {
				assertErrorBodyWithFields(t, body)
				assert.Contains(t, body["message"], "validation error")
				assert.Contains(
					t,
					body["fields"].(map[string]any)["fuzzer_name"],
					"Failed to validate while checking condition: required",
				)
			},
		},
		{
			name:           "InvalidMissingSanitizer",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			dataFile:       base64String(10),
			harnessName:    "harness",
			sanitizer:      "",
			architecture:   "x86_64",
			engine:         "libfuzzer",
			expectedStatus: http.StatusBadRequest,
			bodyTester: func(t *testing.T, body map[string]any) {
				assertErrorBodyWithFields(t, body)
				assert.Contains(t, body["message"], "validation error")
				assert.Contains(
					t,
					body["fields"].(map[string]any)["sanitizer"],
					"Failed to validate while checking condition: required",
				)
			},
		},
		{
			name:           "InvalidMissingArchitecture",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			dataFile:       base64String(10),
			harnessName:    "harness",
			sanitizer:      "address",
			architecture:   "",
			engine:         "libfuzzer",
			expectedStatus: http.StatusBadRequest,
			bodyTester: func(t *testing.T, body map[string]any) {
				assertErrorBodyWithFields(t, body)
				assert.Contains(t, body["message"], "validation error")
				assert.Contains(
					t,
					body["fields"].(map[string]any)["architecture"],
					"Failed to validate while checking condition: required",
				)
			},
		},
		{
			name:           "InvalidNoAuth",
			taskID:         taskOpen.ID.String(),
			auth:           nil,
			dataFile:       base64String(10),
			harnessName:    "harness_1",
			sanitizer:      "address",
			architecture:   "x86_64",
			engine:         "libfuzzer",
			expectedStatus: http.StatusUnauthorized,
			bodyTester:     unauthorizedBodyTester,
		},
		{
			name:           "InvalidAuthNotCompetitionManager",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{authCompetitionManager.ID.String(), authToken},
			dataFile:       base64String(10),
			harnessName:    longString(10),
			sanitizer:      longString(10),
			architecture:   "x86_64",
			engine:         "libfuzzer",
			expectedStatus: http.StatusUnauthorized,
			bodyTester:     unauthorizedBodyTester,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			payload := fmt.Sprintf(
				`{"testcase": "%s", "fuzzer_name": "%s", "sanitizer": "%s", "architecture": "%s", "engine": "%s"}`,
				tt.dataFile,
				tt.harnessName,
				tt.sanitizer,
				tt.architecture,
				tt.engine,
			)

			req, err := http.NewRequest(
				http.MethodPost,
				fmt.Sprintf("%s/v1/task/%s/pov/", s.server.URL, tt.taskID),
				strings.NewReader(payload),
			)
			s.Require().NoError(err, "failed to construct http request")

			req.Header.Add("Content-Type", "application/json")

			if tt.auth != nil {
				req.SetBasicAuth(tt.auth.id, tt.auth.token)
			}

			resp, err := doRequest(s.T(), req)
			s.Require().NoError(err)

			s.Equal(tt.expectedStatus, resp.code, "incorrect status code")
			body := make(map[string]any)
			s.Require().NoError(json.Unmarshal([]byte(resp.body), &body))

			tt.bodyTester(s.T(), body)
		})
	}
}

func (s *ServerTestSuite) Test_POVStatus() {
	tests := []struct {
		name         string
		auth         *clientAuth
		bodyTester   func(t *testing.T, body map[string]any)
		taskID       string
		vulnID       string
		expectedCode int
	}{
		{
			name:         "Valid",
			taskID:       taskOpen.ID.String(),
			vulnID:       vuln.ID.String(),
			auth:         &clientAuth{auth.ID.String(), authToken},
			expectedCode: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:         "ValidUpper",
			taskID:       strings.ToUpper(taskOpen.ID.String()),
			vulnID:       strings.ToUpper(vuln.ID.String()),
			auth:         &clientAuth{auth.ID.String(), authToken},
			expectedCode: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:         "ValidLower",
			taskID:       strings.ToLower(taskOpen.ID.String()),
			vulnID:       strings.ToLower(vuln.ID.String()),
			auth:         &clientAuth{auth.ID.String(), authToken},
			expectedCode: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:         "InvalidInactiveAuth",
			taskID:       taskOpen.ID.String(),
			vulnID:       vuln.ID.String(),
			auth:         &clientAuth{authInactive.ID.String(), authInactive.Token},
			expectedCode: http.StatusUnauthorized,
			bodyTester:   unauthorizedBodyTester,
		},
		{
			name:         "InvalidNoAuth",
			taskID:       taskOpen.ID.String(),
			vulnID:       vuln.ID.String(),
			auth:         nil,
			expectedCode: http.StatusUnauthorized,
			bodyTester:   unauthorizedBodyTester,
		},
		{
			name:         "InvalidTaskID",
			taskID:       "foobar",
			vulnID:       vuln.ID.String(),
			auth:         &clientAuth{auth.ID.String(), authToken},
			expectedCode: http.StatusNotFound,
			bodyTester:   notFoundBodyTester,
		},
		{
			name:         "InvalidVulnID",
			taskID:       taskOpen.ID.String(),
			vulnID:       "foobar",
			auth:         &clientAuth{auth.ID.String(), authToken},
			expectedCode: http.StatusNotFound,
			bodyTester:   notFoundBodyTester,
		},
		{
			name:         "InvalidWrongAuth",
			taskID:       taskOpen.ID.String(),
			vulnID:       vuln.ID.String(),
			auth:         &clientAuth{auth2.ID.String(), authToken},
			expectedCode: http.StatusNotFound,
			bodyTester:   notFoundBodyTester,
		},
		{
			name:         "InvalidAuthNotCompetitionManager",
			taskID:       taskOpen.ID.String(),
			vulnID:       vuln.ID.String(),
			auth:         &clientAuth{authCompetitionManager.ID.String(), authToken},
			expectedCode: http.StatusUnauthorized,
			bodyTester:   unauthorizedBodyTester,
		},
		{
			name:         "InvlaidNonMatchingVulnAndTask",
			taskID:       taskExpired.ID.String(),
			vulnID:       vuln.ID.String(),
			auth:         &clientAuth{auth.ID.String(), authToken},
			expectedCode: http.StatusNotFound,
			bodyTester:   notFoundBodyTester,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req, err := http.NewRequest(
				http.MethodGet,
				fmt.Sprintf("%s/v1/task/%s/pov/%s/", s.server.URL, tt.taskID, tt.vulnID),
				nil,
			)
			s.Require().NoError(err, "failed to construct http request")

			if tt.auth != nil {
				req.SetBasicAuth(tt.auth.id, tt.auth.token)
			}

			resp, err := doRequest(s.T(), req)
			s.Require().NoError(err)

			s.Equal(tt.expectedCode, resp.code, "incorrect status code")
			body := make(map[string]any)
			s.Require().NoError(json.Unmarshal([]byte(resp.body), &body))

			tt.bodyTester(s.T(), body)
		})
	}
}
