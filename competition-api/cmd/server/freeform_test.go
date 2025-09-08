package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func (s *ServerTestSuite) Test_SubmitFreeform() {
	tests := []struct {
		name           string
		auth           *clientAuth
		bodyTester     func(t *testing.T, body map[string]any)
		taskID         string
		submission     string
		expectedStatus int
	}{
		{
			name:           "Valid",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			submission:     base64String(10),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:           "ValidTaskExpired",
			taskID:         taskExpired.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			submission:     base64String(10),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "deadline_exceeded")
			},
		},
		{
			name:           "ValidUpper",
			taskID:         strings.ToUpper(taskOpen.ID.String()),
			auth:           &clientAuth{auth.ID.String(), authToken},
			submission:     base64String(10),
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
			submission:     base64String(10),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:           "InvalidTooLong",
			taskID:         strings.ToLower(taskExpired.ID.String()),
			auth:           &clientAuth{auth.ID.String(), authToken},
			submission:     base64String(1<<21 + 100),
			expectedStatus: http.StatusBadRequest,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "message", "contains message key")
				assert.Contains(t, body["message"], "validation")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			payload := fmt.Sprintf(`{"submission": "%s"}`,
				tt.submission,
			)

			req, err := http.NewRequest(
				http.MethodPost,
				fmt.Sprintf("%s/v1/task/%s/freeform/", s.server.URL, tt.taskID),
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
