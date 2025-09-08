package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func (s *ServerTestSuite) Test_PatchSubmission() {
	tests := []struct {
		name           string
		auth           *clientAuth
		bodyTester     func(t *testing.T, body map[string]any)
		taskID         string
		patch          string
		expectedStatus int
	}{
		{
			name:           "Valid",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			patch:          base64String(10),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:           "ValidUpper",
			taskID:         strings.ToUpper(taskOpen.ID.String()),
			auth:           &clientAuth{auth.ID.String(), authToken},
			patch:          base64String(10),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:           "ValidLower",
			taskID:         strings.ToLower(taskOpen.ID.String()),
			auth:           &clientAuth{auth.ID.String(), authToken},
			patch:          base64String(10),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:           "ValidRepeatData",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			patch:          base64String(10),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:           "ValidLongPatch",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			patch:          base64String((1 << 10) * 100),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:           "ValidNoDescription",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			patch:          base64String(10),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:           "ValidExpiredTask",
			taskID:         taskExpired.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			patch:          base64String(10),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body["status"], "deadline_exceeded")
			},
		},
		{
			name:           "InvalidTaskID",
			taskID:         "foobar",
			auth:           &clientAuth{auth.ID.String(), authToken},
			patch:          base64String(10),
			expectedStatus: http.StatusNotFound,
			bodyTester:     notFoundBodyTester,
		},
		{
			name:           "InvalidPatchNotBase64",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			patch:          "foobar",
			expectedStatus: http.StatusBadRequest,
			bodyTester: func(t *testing.T, body map[string]any) {
				assertErrorBodyWithFields(t, body)
				assert.Contains(t, body["message"], "validation error")
				assert.Contains(
					t,
					body["fields"].(map[string]any)["patch"],
					"Failed to validate while checking condition: base64",
				)
			},
		},
		{
			name:           "InvalidPatchTooLong",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			patch:          base64String((1<<10)*100 + 10),
			expectedStatus: http.StatusBadRequest,
			bodyTester: func(t *testing.T, body map[string]any) {
				assertErrorBodyWithFields(t, body)
				assert.Contains(t, body["message"], "validation error")
				assert.Contains(t, body["fields"].(map[string]any)["patch"], "must be <= 100kb")
			},
		},
		{
			name:           "InvalidPatchMissing",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			patch:          "",
			expectedStatus: http.StatusBadRequest,
			bodyTester: func(t *testing.T, body map[string]any) {
				assertErrorBodyWithFields(t, body)
				assert.Contains(t, body["message"], "validation error")
				assert.Contains(
					t,
					body["fields"].(map[string]any)["patch"],
					"Failed to validate while checking condition: required",
				)
			},
		},
		{
			name:           "InvalidNoAuth",
			taskID:         taskOpen.ID.String(),
			auth:           nil,
			patch:          base64String(10),
			expectedStatus: http.StatusUnauthorized,
			bodyTester:     unauthorizedBodyTester,
		},
		{
			name:           "InvalidAuthNotCompetitionManager",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{authCompetitionManager.ID.String(), authToken},
			patch:          base64String(10),
			expectedStatus: http.StatusUnauthorized,
			bodyTester:     unauthorizedBodyTester,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			payload := fmt.Sprintf(`{"patch":"%s"}`, tt.patch)

			req, err := http.NewRequest(
				http.MethodPost,
				fmt.Sprintf("%s/v1/task/%s/patch/", s.server.URL, tt.taskID),
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

func (s *ServerTestSuite) Test_PatchStatus() {
	tests := []struct {
		name         string
		auth         *clientAuth
		bodyTester   func(t *testing.T, body map[string]any)
		taskID       string
		patchID      string
		expectedCode int
	}{
		{
			name:         "Valid",
			taskID:       patch.TaskID.String(),
			patchID:      patch.ID.String(),
			auth:         &clientAuth{auth.ID.String(), authToken},
			expectedCode: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body["status"], "passed")
			},
		},
		{
			name:         "InvalidNoAuth",
			taskID:       patch.TaskID.String(),
			patchID:      patch.ID.String(),
			auth:         nil,
			expectedCode: http.StatusUnauthorized,
			bodyTester:   unauthorizedBodyTester,
		},
		{
			name:         "InvalidTaskID",
			taskID:       "foobar",
			patchID:      patch.ID.String(),
			auth:         &clientAuth{auth.ID.String(), authToken},
			expectedCode: http.StatusNotFound,
			bodyTester:   notFoundBodyTester,
		},
		{
			name:         "InvalidPatchID",
			taskID:       taskOpen.ID.String(),
			patchID:      "foobar",
			auth:         &clientAuth{auth.ID.String(), authToken},
			expectedCode: http.StatusNotFound,
			bodyTester:   notFoundBodyTester,
		},
		{
			name:         "InvalidWrongAuth",
			taskID:       patch.TaskID.String(),
			patchID:      patch.ID.String(),
			auth:         &clientAuth{auth2.ID.String(), authToken},
			expectedCode: http.StatusNotFound,
			bodyTester:   notFoundBodyTester,
		},
		{
			name:         "InvalidAuthNotCompetitionManager",
			taskID:       patch.TaskID.String(),
			patchID:      patch.ID.String(),
			auth:         &clientAuth{authCompetitionManager.ID.String(), authToken},
			expectedCode: http.StatusUnauthorized,
			bodyTester:   unauthorizedBodyTester,
		},
		{
			name:         "InvalidNonMatchingVulnAndPatch",
			taskID:       taskExpired.ID.String(),
			patchID:      patch.ID.String(),
			auth:         &clientAuth{auth.ID.String(), authToken},
			expectedCode: http.StatusNotFound,
			bodyTester:   notFoundBodyTester,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req, err := http.NewRequest(
				http.MethodGet,
				fmt.Sprintf("%s/v1/task/%s/patch/%s/", s.server.URL, tt.taskID, tt.patchID),
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
