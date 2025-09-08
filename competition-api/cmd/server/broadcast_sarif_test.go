package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

func (s *ServerTestSuite) Test_SubmitSarifAssessment() {
	tests := []struct {
		name           string
		auth           *clientAuth
		bodyTester     func(t *testing.T, body map[string]any)
		taskID         string
		sarifID        string
		assessment     string
		description    string
		expectedStatus int
	}{
		{
			name:           "Valid",
			taskID:         sarifBroadcast.TaskID.String(),
			sarifID:        sarifBroadcast.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			assessment:     string(types.AssessmentCorrect),
			description:    longString(1 << 17),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:           "ValidUpper",
			taskID:         strings.ToUpper(sarifBroadcast.TaskID.String()),
			sarifID:        strings.ToUpper(sarifBroadcast.ID.String()),
			auth:           &clientAuth{auth.ID.String(), authToken},
			assessment:     string(types.AssessmentCorrect),
			description:    longString(1 << 17),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:           "ValidLower",
			taskID:         strings.ToLower(sarifBroadcast.TaskID.String()),
			sarifID:        strings.ToLower(sarifBroadcast.ID.String()),
			auth:           &clientAuth{auth.ID.String(), authToken},
			assessment:     string(types.AssessmentCorrect),
			description:    longString(1 << 17),
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:           "ValidInvalidAssessment",
			taskID:         sarifBroadcast.TaskID.String(),
			sarifID:        sarifBroadcast.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			assessment:     string(types.AssessmentIncorrect),
			description:    "very insightful",
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "accepted")
			},
		},
		{
			name:           "ValidExpiredTask",
			taskID:         sarifBroadcastExpired.TaskID.String(),
			sarifID:        sarifBroadcastExpired.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			assessment:     string(types.AssessmentCorrect),
			description:    "very insightful",
			expectedStatus: http.StatusOK,
			bodyTester: func(t *testing.T, body map[string]any) {
				assert.Contains(t, body, "status", "contains status key")
				assert.Contains(t, body["status"], "deadline_exceeded")
			},
		},
		{
			name:           "InvalidTaskID",
			taskID:         "invalid",
			sarifID:        sarifBroadcast.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			assessment:     string(types.AssessmentCorrect),
			description:    "very insightful",
			expectedStatus: http.StatusNotFound,
			bodyTester:     notFoundBodyTester,
		},
		{
			name:           "InvalidSarifID",
			taskID:         taskOpen.ID.String(),
			sarifID:        "foobar",
			auth:           &clientAuth{auth.ID.String(), authToken},
			assessment:     string(types.AssessmentCorrect),
			description:    "very insightful",
			expectedStatus: http.StatusNotFound,
			bodyTester:     notFoundBodyTester,
		},
		{
			name:           "InvalidMissingAssessment",
			taskID:         sarifBroadcast.TaskID.String(),
			sarifID:        sarifBroadcast.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			assessment:     "",
			description:    "very insightful",
			expectedStatus: http.StatusBadRequest,
			bodyTester: func(t *testing.T, body map[string]any) {
				assertErrorBodyWithFields(t, body)
				assert.Contains(t, body["message"], "validation error")
				assert.Contains(
					t,
					body["fields"].(map[string]any)["assessment"],
					"Failed to validate while checking condition: required",
				)
			},
		},
		{
			name:           "InvalidInvalidAssessment",
			taskID:         sarifBroadcast.TaskID.String(),
			sarifID:        sarifBroadcast.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			assessment:     "i am an invalid assessment",
			description:    "very insightful",
			expectedStatus: http.StatusBadRequest,
			bodyTester: func(t *testing.T, body map[string]any) {
				assertErrorBodyWithFields(t, body)
				assert.Contains(t, body["message"], "validation error")
				assert.Contains(
					t,
					body["fields"].(map[string]any)["assessment"],
					"Failed to validate while checking condition: eq=correct|eq=incorrect",
				)
			},
		},
		{
			name:           "InvalidMissingDescription",
			taskID:         sarifBroadcast.TaskID.String(),
			sarifID:        sarifBroadcast.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			assessment:     string(types.AssessmentCorrect),
			description:    "",
			expectedStatus: http.StatusBadRequest,
			bodyTester: func(t *testing.T, body map[string]any) {
				assertErrorBodyWithFields(t, body)
				assert.Contains(t, body["message"], "validation error")
				assert.Contains(
					t,
					body["fields"].(map[string]any)["description"],
					"Failed to validate while checking condition: required",
				)
			},
		},
		{
			name:           "InvalidLongDescription",
			taskID:         sarifBroadcast.TaskID.String(),
			sarifID:        sarifBroadcast.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			assessment:     string(types.AssessmentCorrect),
			description:    longString(1<<17 + 1),
			expectedStatus: http.StatusBadRequest,
			bodyTester: func(t *testing.T, body map[string]any) {
				assertErrorBodyWithFields(t, body)
				assert.Contains(t, body["message"], "validation error")
				assert.Contains(
					t,
					body["fields"].(map[string]any)["description"],
					"Failed to validate while checking condition: max",
				)
			},
		},
		{
			name:           "InvalidMissingAuth",
			taskID:         sarifBroadcast.TaskID.String(),
			sarifID:        sarifBroadcast.ID.String(),
			auth:           nil,
			assessment:     string(types.AssessmentCorrect),
			description:    "very insightful",
			expectedStatus: http.StatusUnauthorized,
			bodyTester:     unauthorizedBodyTester,
		},
		{
			name:           "InvalidAuthNotCompetitionManager",
			taskID:         sarifBroadcast.TaskID.String(),
			sarifID:        sarifBroadcast.ID.String(),
			auth:           &clientAuth{authCompetitionManager.ID.String(), authToken},
			assessment:     string(types.AssessmentCorrect),
			description:    "very insightful",
			expectedStatus: http.StatusUnauthorized,
			bodyTester:     unauthorizedBodyTester,
		},
		{
			name:           "InvalidNonMatchingBroadcastAndTask",
			taskID:         taskOpen.ID.String(),
			sarifID:        sarifBroadcastExpired.ID.String(),
			auth:           &clientAuth{auth.ID.String(), authToken},
			assessment:     string(types.AssessmentCorrect),
			description:    "very insightful",
			expectedStatus: http.StatusNotFound,
			bodyTester:     notFoundBodyTester,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			payload := fmt.Sprintf(
				`{"assessment":"%s","description":"%s"}`,
				tt.assessment,
				tt.description,
			)

			req, err := http.NewRequest(
				http.MethodPost,
				fmt.Sprintf(
					"%s/v1/task/%s/broadcast-sarif-assessment/%s",
					s.server.URL,
					tt.taskID,
					tt.sarifID,
				),
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
