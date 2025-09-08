package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func (s *ServerTestSuite) Test_SubmitBundleTests() {
	tests := []struct {
		name           string
		auth           *clientAuth
		bodyTester     func(t *testing.T, body map[string]any)
		taskID         string
		payload        string
		expectedStatus int
	}{
		{
			name:   "Valid",
			taskID: taskOpen.ID.String(),
			auth:   &clientAuth{id: auth.ID.String(), token: authToken},
			payload: fmt.Sprintf(
				`{"pov_id": "%s", "patch_id": "%s", "broadcast_sarif_id": "%s", "submitted_sarif_id":"%s", "description": "%s", "freeform_id": "%s"}`,
				vuln.ID.String(),
				patch.ID.String(),
				sarifBroadcast.ID.String(),
				sarifSubmitted.ID.String(),
				"foobar",
				freeform.ID.String(),
			),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:   "ValidNullFields",
			taskID: taskOpen.ID.String(),
			auth:   &clientAuth{id: auth.ID.String(), token: authToken},
			payload: fmt.Sprintf(
				`{"broadcast_sarif_id":null,"description":null,"freeform_id":null,"patch_id":"%s","pov_id":"%s","submitted_sarif_id":null}`,
				patch.ID.String(),
				vuln.ID.String(),
			),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:   "ValidExpiredTask",
			taskID: taskExpired.ID.String(),
			auth:   &clientAuth{id: auth.ID.String(), token: authToken},
			payload: fmt.Sprintf(
				`{"pov_id": "%s", "description": "%s"}`,
				vulnExpired.ID.String(),
				"foobar",
			),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "deadline_exceeded")
			},
		},
		{
			name:   "InvalidUnknownPOV",
			taskID: taskOpen.ID.String(),
			auth:   &clientAuth{id: auth.ID.String(), token: authToken},
			payload: fmt.Sprintf(
				`{"pov_id": "%s", "description": "%s"}`,
				uuid.New().String(),
				"foobar",
			),
			expectedStatus: http.StatusNotFound,
			bodyTester:     notFoundBodyTester,
		},
		{
			name:           "InvalidLessThan2",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{id: auth.ID.String(), token: authToken},
			payload:        fmt.Sprintf(`{"pov_id": "%s"}`, vuln.ID.String()),
			expectedStatus: http.StatusBadRequest,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "message", "contains message key")
				assert.Contains(t, body["message"], "at least 2 fields")
			},
		},
		{
			name:   "InvalidNotMatchingVuln",
			taskID: taskOpen.ID.String(),
			auth:   &clientAuth{id: auth.ID.String(), token: authToken},
			payload: fmt.Sprintf(
				`{"pov_id": "%s", "description": "%s"}`,
				vulnExpired.ID.String(),
				"foobar",
			),
			expectedStatus: http.StatusNotFound,
			bodyTester:     notFoundBodyTester,
		},
		{
			name:   "InvalidWrongAuth",
			taskID: taskOpen.ID.String(),
			auth:   &clientAuth{id: auth2.ID.String(), token: authToken},
			payload: fmt.Sprintf(
				`{"pov_id": "%s", "description": "%s"}`,
				vuln.ID.String(),
				"foobar",
			),
			expectedStatus: http.StatusNotFound,
			bodyTester:     notFoundBodyTester,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req, err := http.NewRequest(
				http.MethodPost,
				fmt.Sprintf("%s/v1/task/%s/bundle/", s.server.URL, tt.taskID),
				strings.NewReader(tt.payload),
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

func (s *ServerTestSuite) Test_PatchBundleTests() {
	tests := []struct {
		name           string
		auth           *clientAuth
		bodyTester     func(t *testing.T, body map[string]any)
		taskID         string
		bundleID       string
		payload        string
		expectedStatus int
	}{
		{
			name:     "Valid",
			taskID:   taskOpen.ID.String(),
			bundleID: bundle.ID.String(),
			auth:     &clientAuth{id: auth.ID.String(), token: authToken},
			payload: fmt.Sprintf(
				`{"pov_id": "%s", "patch_id": "%s", "broadcast_sarif_id": "%s", "submitted_sarif_id":"%s", "description": "%s"}`,
				vuln.ID.String(),
				patch.ID.String(),
				sarifBroadcast.ID.String(),
				sarifSubmitted.ID.String(),
				"foobar",
			),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:     "InvalidExpiredTask",
			taskID:   taskExpired.ID.String(),
			bundleID: bundleExpired.ID.String(),
			auth:     &clientAuth{id: auth.ID.String(), token: authToken},
			payload: fmt.Sprintf(
				`{"pov_id": "%s", "description": "%s"}`,
				vulnExpired.ID.String(),
				"foobar",
			),
			expectedStatus: http.StatusBadRequest,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "message", "contains status key")
				assert.Contains(t, body["message"], "deadline to modify")
			},
		},
		{
			name:           "InvalidLessThan2",
			taskID:         taskOpen.ID.String(),
			bundleID:       bundle.ID.String(),
			auth:           &clientAuth{id: auth.ID.String(), token: authToken},
			payload:        `{"pov_id": null, "patch_id": null, "broadcast_sarif_id": null, "submitted_sarif_id":null}`,
			expectedStatus: http.StatusBadRequest,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "message", "contains message key")
				assert.Contains(t, body["message"], "at least 2 fields")
			},
		},
		{
			name:     "InvalidNotMatchingVuln",
			taskID:   taskOpen.ID.String(),
			bundleID: bundle.ID.String(),
			auth:     &clientAuth{id: auth.ID.String(), token: authToken},
			payload: fmt.Sprintf(
				`{"pov_id": "%s", "description": "%s"}`,
				vulnExpired.ID.String(),
				"foobar",
			),
			expectedStatus: http.StatusNotFound,
			bodyTester:     notFoundBodyTester,
		},
		{
			name:     "InvalidWrongAuth",
			taskID:   taskOpen.ID.String(),
			bundleID: bundle.ID.String(),
			auth:     &clientAuth{id: auth2.ID.String(), token: authToken},
			payload: fmt.Sprintf(
				`{"pov_id": "%s", "description": "%s"}`,
				vuln.ID.String(),
				"foobar",
			),
			expectedStatus: http.StatusNotFound,
			bodyTester:     notFoundBodyTester,
		},
		{
			name:     "InvalidWrongTask",
			taskID:   taskOpen.ID.String(),
			bundleID: bundleExpired.ID.String(),
			auth:     &clientAuth{id: auth.ID.String(), token: authToken},
			payload: fmt.Sprintf(
				`{"pov_id": "%s", "description": "%s"}`,
				vuln.ID.String(),
				"foobar",
			),
			expectedStatus: http.StatusNotFound,
			bodyTester:     notFoundBodyTester,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req, err := http.NewRequest(
				http.MethodPatch,
				fmt.Sprintf("%s/v1/task/%s/bundle/%s/", s.server.URL, tt.taskID, tt.bundleID),
				strings.NewReader(tt.payload),
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

func (s *ServerTestSuite) Test_GetBundleTests() {
	tests := []struct {
		name           string
		auth           *clientAuth
		bodyTester     func(t *testing.T, body map[string]any)
		taskID         string
		bundleID       string
		expectedStatus int
	}{
		{
			name:           "InvalidBundleID",
			taskID:         taskOpen.ID.String(),
			bundleID:       "foobar",
			auth:           &clientAuth{id: auth.ID.String(), token: authToken},
			expectedStatus: http.StatusNotFound,
			bodyTester:     notFoundBodyTester,
		},
		{
			name:           "InvalidWrongAuth",
			taskID:         taskOpen.ID.String(),
			bundleID:       bundle.ID.String(),
			auth:           &clientAuth{id: auth2.ID.String(), token: authToken},
			expectedStatus: http.StatusNotFound,
			bodyTester:     notFoundBodyTester,
		},
		{
			name:           "InvalidWrongTaskID",
			taskID:         taskExpired.ID.String(),
			bundleID:       bundle.ID.String(),
			auth:           &clientAuth{id: auth.ID.String(), token: authToken},
			expectedStatus: http.StatusNotFound,
			bodyTester:     notFoundBodyTester,
		},
		{
			name:           "ValidExpired",
			taskID:         taskExpired.ID.String(),
			bundleID:       bundleExpired.ID.String(),
			auth:           &clientAuth{id: auth.ID.String(), token: authToken},
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "deadline_exceeded")
			},
		},
		{
			name:           "Valid",
			taskID:         taskOpen.ID.String(),
			bundleID:       bundle.ID.String(),
			auth:           &clientAuth{id: auth.ID.String(), token: authToken},
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "accepted")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req, err := http.NewRequest(
				http.MethodGet,
				fmt.Sprintf("%s/v1/task/%s/bundle/%s", s.server.URL, tt.taskID, tt.bundleID),
				nil,
			)
			s.Require().NoError(err, "failed to construct http request")

			req.Header.Add("Content-Type", "application/json")

			if tt.auth != nil {
				req.SetBasicAuth(tt.auth.id, tt.auth.token)
			}

			resp, err := doRequest(s.T(), req)
			s.Require().NoError(err)

			s.Equal(tt.expectedStatus, resp.code, "incorrect status code")
			if resp.code == http.StatusNoContent {
				return
			}

			body := make(map[string]any)
			s.Require().NoError(json.Unmarshal([]byte(resp.body), &body))

			tt.bodyTester(s.T(), body)
		})
	}
}

func (s *ServerTestSuite) Test_DeleteBundleTests() {
	tests := []struct {
		name           string
		auth           *clientAuth
		bodyTester     func(t *testing.T, body map[string]any)
		taskID         string
		bundleID       string
		expectedStatus int
	}{
		{
			name:           "InvalidBundleID",
			taskID:         taskOpen.ID.String(),
			bundleID:       "foobar",
			auth:           &clientAuth{id: auth.ID.String(), token: authToken},
			expectedStatus: http.StatusNotFound,
			bodyTester:     notFoundBodyTester,
		},
		{
			name:           "InvalidWrongAuth",
			taskID:         taskOpen.ID.String(),
			bundleID:       bundle.ID.String(),
			auth:           &clientAuth{id: auth2.ID.String(), token: authToken},
			expectedStatus: http.StatusNotFound,
			bodyTester:     notFoundBodyTester,
		},
		{
			name:           "InvalidWrongTaskID",
			taskID:         taskExpired.ID.String(),
			bundleID:       bundle.ID.String(),
			auth:           &clientAuth{id: auth.ID.String(), token: authToken},
			expectedStatus: http.StatusNotFound,
			bodyTester:     notFoundBodyTester,
		},
		{
			name:           "InvalidExpired",
			taskID:         taskExpired.ID.String(),
			bundleID:       bundleExpired.ID.String(),
			auth:           &clientAuth{id: auth.ID.String(), token: authToken},
			expectedStatus: http.StatusBadRequest,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "message", "contains message key")
				assert.Contains(t, body["message"], "deadline to modify submission passed")
			},
		},
		{
			name:           "Valid",
			taskID:         taskOpen.ID.String(),
			bundleID:       bundle.ID.String(),
			auth:           &clientAuth{id: auth.ID.String(), token: authToken},
			expectedStatus: http.StatusNoContent,
			bodyTester: func(_ *testing.T, _ map[string]any) {
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			req, err := http.NewRequest(
				http.MethodDelete,
				fmt.Sprintf("%s/v1/task/%s/bundle/%s", s.server.URL, tt.taskID, tt.bundleID),
				nil,
			)
			s.Require().NoError(err, "failed to construct http request")

			req.Header.Add("Content-Type", "application/json")

			if tt.auth != nil {
				req.SetBasicAuth(tt.auth.id, tt.auth.token)
			}

			resp, err := doRequest(s.T(), req)
			s.Require().NoError(err)

			s.Equal(tt.expectedStatus, resp.code, "incorrect status code")
			if resp.code == http.StatusNoContent {
				return
			}

			body := make(map[string]any)
			s.Require().NoError(json.Unmarshal([]byte(resp.body), &body))

			tt.bodyTester(s.T(), body)
		})
	}
}
