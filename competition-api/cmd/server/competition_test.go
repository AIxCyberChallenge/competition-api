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

func (s *ServerTestSuite) Test_OutOfBudget() {
	tests := []struct {
		name           string
		auth           *clientAuth
		bodyTester     func(t *testing.T, body map[string]any)
		id             string
		expectedStatus int
	}{
		{
			name:           "Valid",
			auth:           &clientAuth{authCompetitionManager.ID.String(), authToken},
			id:             fmt.Sprintf(`"%s"`, auth.ID.String()),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "body should contain status")
				assert.Contains(t, body["status"], "ok", "message missing fragment")
			},
		},
		{
			name:           "ValidUpper",
			auth:           &clientAuth{authCompetitionManager.ID.String(), authToken},
			id:             fmt.Sprintf(`"%s"`, strings.ToUpper(auth.ID.String())),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "body should contain status")
				assert.Contains(t, body["status"], "ok", "message missing fragment")
			},
		},
		{
			name:           "ValidLower",
			auth:           &clientAuth{authCompetitionManager.ID.String(), authToken},
			id:             fmt.Sprintf(`"%s"`, strings.ToLower(auth.ID.String())),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "body should contain status")
				assert.Contains(t, body["status"], "ok", "message missing fragment")
			},
		},
		{
			name:           "ValidFakeID",
			auth:           &clientAuth{authCompetitionManager.ID.String(), authToken},
			id:             fmt.Sprintf(`"%s"`, uuid.New().String()),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "body should contain status")
				assert.Contains(t, body["status"], "ok", "message missing fragment")
			},
		},
		{
			name:           "InvalidNonUUIDID",
			auth:           &clientAuth{authCompetitionManager.ID.String(), authToken},
			id:             fmt.Sprintf(`"%s"`, "foobar"),
			expectedStatus: http.StatusBadRequest,
			bodyTester: func(t *testing.T, body map[string]any) {
				assertErrorBodyWithFields(t, body)
				assert.Contains(t, body["message"], "validation error")
				assert.Contains(t, body["fields"].(map[string]any)["competitor_id"], "uuid")
			},
		},
		{
			name:           "InvalidNonCompetitionManager",
			auth:           &clientAuth{auth.ID.String(), authToken},
			id:             fmt.Sprintf(`"%s"`, auth.ID.String()),
			expectedStatus: http.StatusUnauthorized,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "message", "body does not contain message")
				assert.Contains(t, body["message"], "Unauthorized", "incorrect body fragment")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			payload := fmt.Sprintf(`{"competitor_id": %s}`,
				tt.id,
			)

			req, err := http.NewRequest(
				http.MethodPost,
				fmt.Sprintf("%s/competition/out-of-budget/", s.server.URL),
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
