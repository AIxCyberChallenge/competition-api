package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func (s *ServerTestSuite) Test_SubmitSARIF() {
	tests := []struct {
		name           string
		auth           *clientAuth
		bodyTester     func(t *testing.T, body map[string]any)
		taskID         string
		sarif          string
		expectedStatus int
	}{
		{
			name:           "Valid",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{id: auth.ID.String(), token: authToken},
			sarif:          `{"version": "2.1.0", "runs": []}`,
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:           "ValidResult",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{id: auth.ID.String(), token: authToken},
			sarif:          `{"version": "2.1.0", "runs": [{"tool": {"driver": {"name": "foobar"}}, "results":[{"message": {"text": "foobar"}, "ruleId": "abc"}]}]}`,
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:           "ValidUpper",
			taskID:         strings.ToUpper(taskOpen.ID.String()),
			auth:           &clientAuth{id: auth.ID.String(), token: authToken},
			sarif:          `{"version": "2.1.0", "runs": []}`,
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:           "ValidLower",
			taskID:         strings.ToLower(taskOpen.ID.String()),
			auth:           &clientAuth{id: auth.ID.String(), token: authToken},
			sarif:          `{"version": "2.1.0", "runs": []}`,
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:           "ValidTaskExpired",
			taskID:         taskExpired.ID.String(),
			auth:           &clientAuth{id: auth.ID.String(), token: authToken},
			sarif:          `{"version": "2.1.0", "runs": []}`,
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "deadline_exceeded")
			},
		},
		{
			name:           "InvalidTaskID",
			taskID:         "invalid",
			auth:           &clientAuth{id: auth.ID.String(), token: authToken},
			sarif:          `{"version": "2.1.0", "runs": []}`,
			expectedStatus: http.StatusNotFound,
			bodyTester:     notFoundBodyTester,
		},
		{
			name:           "InvalidMissingAuth",
			taskID:         taskOpen.ID.String(),
			auth:           nil,
			sarif:          `{"version": "2.1.0", "runs": []}`,
			expectedStatus: http.StatusUnauthorized,
			bodyTester:     unauthorizedBodyTester,
		},
		{
			name:           "InvalidAuthNotCRS",
			taskID:         taskOpen.ID.String(),
			auth:           &clientAuth{id: authCompetitionManager.ID.String(), token: authToken},
			sarif:          `{"version": "2.1.0", "runs": []}`,
			expectedStatus: http.StatusUnauthorized,
			bodyTester:     unauthorizedBodyTester,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			payload := fmt.Sprintf(`{"sarif": %s}`, tt.sarif)

			req, err := http.NewRequest(
				http.MethodPost,
				fmt.Sprintf("%s/v1/task/%s/submitted-sarif/", s.server.URL, tt.taskID),
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
