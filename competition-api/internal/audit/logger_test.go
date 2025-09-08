package audit

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

func ptr[T any](v T) *T {
	return &v
}

func captureStdout(fn func()) (string, error) {
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	os.Stdout = w

	fn()

	if err := w.Close(); err != nil {
		return "", err
	}
	os.Stdout = orig

	var buf bytes.Buffer
	if _, err = io.Copy(&buf, r); err != nil {
		return "", err
	}

	if err := r.Close(); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func TestLogFileArchived(t *testing.T) {
	ctx := Context{
		TeamID:  ptr("team"),
		TaskID:  ptr("task"),
		RoundID: "round",
	}
	got, err := captureStdout(func() {
		LogFileArchived(ctx, "bucket", "object", types.FileDiffTarball, EntityPOV, "entity")
	})
	require.NoError(t, err)

	expect := regexp.MustCompile(
		`{"event":{"bucket_name":"bucket","object_name":"object","file_archived":"diff_tarball","entity":"pov","entity_id":"entity"},"task_id":"task","team_id":"team","log_context":"audit","version":"\d\.\d\.\d","round_id":"round","disposition":"neutral","event_type":"file_archived","timestamp":\d+}`,
	)
	assert.Regexp(t, expect, got)
}

func TestLogNewDeltaScan(t *testing.T) {
	ctx := Context{
		TeamID:  ptr("team"),
		TaskID:  ptr("task"),
		RoundID: "round",
	}
	got, err := captureStdout(func() {
		LogNewDeltaScan(
			ctx,
			"repo_url",
			"base_commit",
			"delta_commit",
			0,
			"oss_fuzz_url",
			"oss_fuzz_hash",
			"challenge",
			false,
		)
	})
	require.NoError(t, err)

	expect := regexp.MustCompile(
		`{"task_id":"task","team_id":null,"log_context":"audit","version":"\d.\d.\d","round_id":"round","disposition":"neutral","event_type":"new_delta_scan","timestamp":\d+,"event":{"task_type":"delta","repo_url":"repo_url","base_commit_hash":"base_commit","delta_commit_hash":"delta_commit","fuzz_tooling_url":"oss_fuzz_url","fuzz_tooling_hash":"oss_fuzz_hash","challenge_name":"challenge","deadline":0,"unharnessed":false}}`,
	)
	assert.Regexp(t, expect, got)
}

func TestLogNewFullScan(t *testing.T) {
	ctx := Context{
		TeamID:  ptr("team"),
		TaskID:  ptr("task"),
		RoundID: "round",
	}
	got, err := captureStdout(func() {
		LogNewFullScan(
			ctx,
			"repo_url",
			"commit_hash",
			0,
			"oss_fuzz_url",
			"oss_fuzz_hash",
			"challenge",
			false,
		)
	})
	require.NoError(t, err)

	expect := regexp.MustCompile(
		`{"task_id":"task","team_id":null,"log_context":"audit","version":"\d.\d.\d","round_id":"round","disposition":"neutral","event_type":"new_full_scan","timestamp":\d+,"event":{"task_type":"full","repo_url":"repo_url","commit_hash":"commit_hash","fuzz_tooling_url":"oss_fuzz_url","fuzz_tooling_hash":"oss_fuzz_hash","challenge_name":"challenge","deadline":0,"unharnessed":false}}`,
	)
	assert.Regexp(t, expect, got)
}

func TestLogNewSARIFBroadcast(t *testing.T) {
	ctx := Context{
		TeamID:  ptr("team"),
		TaskID:  ptr("task"),
		RoundID: "round",
	}
	got, err := captureStdout(func() {
		LogNewSARIFBroadcast(ctx, "repo_url", ptr("commit_hash"), "sarif_id")
	})
	require.NoError(t, err)

	expect := regexp.MustCompile(
		`{"event":{"repo_url":"repo_url","commit_hash":"commit_hash","sarif_id":"sarif_id"},"task_id":"task","team_id":null,"log_context":"audit","version":"\d.\d.\d","round_id":"round","disposition":"neutral","event_type":"new_sarif_broadcast","timestamp":\d+`,
	)
	assert.Regexp(t, expect, got)
}

func TestLogPOVSubmission(t *testing.T) {
	ctx := Context{
		TeamID:  ptr("team"),
		TaskID:  ptr("task"),
		RoundID: "round",
	}
	got, err := captureStdout(func() {
		LogPOVSubmission(
			ctx,
			"pov_id",
			types.SubmissionStatusAccepted,
			"fuzzer",
			"sha123456",
			"sanitizer",
			"arch",
			"engine",
		)
	})
	require.NoError(t, err)

	expect := regexp.MustCompile(
		`{"event":{"pov_id":"pov_id","fuzzer_name":"fuzzer","testcase_sha256":"sha123456","sanitizer":"sanitizer","architecture":"arch","status":"accepted","engine":"engine"},"task_id":"task","team_id":"team","log_context":"audit","version":"\d.\d.\d","round_id":"round","disposition":"neutral","event_type":"pov_submission","timestamp":\d+}`,
	)
	assert.Regexp(t, expect, got)
}

func TestLogPOVSubmissionResult(t *testing.T) {
	ctx := Context{
		TeamID:  ptr("team"),
		TaskID:  ptr("task"),
		RoundID: "round",
	}
	got, err := captureStdout(func() {
		LogPOVSubmissionResult(ctx, "pov_id", types.SubmissionStatusAccepted)
	})
	require.NoError(t, err)

	expect := regexp.MustCompile(
		`{"event":{"pov_id":"pov_id","status":"accepted"},"task_id":"task","team_id":"team","log_context":"audit","version":"\d.\d.\d","round_id":"round","disposition":"neutral","event_type":"pov_submission_result","timestamp":\d+}`,
	)
	assert.Regexp(t, expect, got)
}

func TestLogPatchSubmission(t *testing.T) {
	ctx := Context{
		TeamID:  ptr("team"),
		TaskID:  ptr("task"),
		RoundID: "round",
	}
	got, err := captureStdout(func() {
		LogPatchSubmission(ctx, "patch_id", types.SubmissionStatusAccepted, "sha123456")
	})
	require.NoError(t, err)

	expect := regexp.MustCompile(
		`{"event":{"patch_id":"patch_id","patch_sha256":"sha123456","status":"accepted"},"task_id":"task","team_id":"team","log_context":"audit","version":"\d.\d.\d","round_id":"round","disposition":"neutral","event_type":"patch_submission","timestamp":\d+`,
	)
	assert.Regexp(t, expect, got)
}

func TestLogPatchSubmissionResult(t *testing.T) {
	ctx := Context{
		TeamID:  ptr("team"),
		TaskID:  ptr("task"),
		RoundID: "round",
	}
	got, err := captureStdout(func() {
		LogPatchSubmissionResult(ctx, "patch_id", types.SubmissionStatusAccepted, ptr(true))
	})
	require.NoError(t, err)

	expect := regexp.MustCompile(
		`{"event":{"functionality_tests_passing":true,"patch_id":"patch_id","status":"accepted"},"task_id":"task","team_id":"team","log_context":"audit","version":"\d.\d.\d","round_id":"round","disposition":"neutral","event_type":"patch_submission_result","timestamp":\d+}`,
	)
	assert.Regexp(t, expect, got)
}

func TestLogSARIFAssessment(t *testing.T) {
	ctx := Context{
		TeamID:  ptr("team"),
		TaskID:  ptr("task"),
		RoundID: "round",
	}
	got, err := captureStdout(func() {
		LogSARIFAssessment(ctx, "assessment_id", "assessment", "sarif_id")
	})
	require.NoError(t, err)

	expect := regexp.MustCompile(
		`{"event":{"assessment_id":"assessment_id","assessment":"assessment","sarif_broadcast_id":"sarif_id"},"task_id":"task","team_id":"team","log_context":"audit","version":"\d.\d.\d","round_id":"round","disposition":"neutral","event_type":"sarif_assessment","timestamp":\d+}`,
	)
	assert.Regexp(t, expect, got)
}

func TestLogBundleSubmission(t *testing.T) {
	ctx := Context{
		TeamID:  ptr("team"),
		TaskID:  ptr("task"),
		RoundID: "round",
	}
	zeroID := uuid.MustParse("00000000-0000-0000-0000-000000000000")
	got, err := captureStdout(func() {
		LogBundleSubmission(
			ctx,
			"bundle_id",
			&zeroID,
			&zeroID,
			&zeroID,
			&zeroID,
			ptr("description"),
			&zeroID,
			types.SubmissionStatusAccepted,
		)
	})
	require.NoError(t, err)

	uuidRegex := `[\d\w]{8}-[\d\w]{4}-[\d\w]{4}-[\d\w]{4}-[\d\w]{12}`
	expect := regexp.MustCompile(
		fmt.Sprintf(
			`{"event":{"bundle_id":"bundle_id","pov_id":"%s","patch_id":"%s","submitted_sarif_id":"%s","broadcast_sarif_id":"%s","description":"description","freeform_id":"%s","status":"accepted"},"task_id":"task","team_id":"team","log_context":"audit","version":"0.1.0","round_id":"round","disposition":"neutral","event_type":"bundle_submission","timestamp":\d+}`,
			uuidRegex,
			uuidRegex,
			uuidRegex,
			uuidRegex,
			uuidRegex,
		),
	)
	assert.Regexp(t, expect, got)
}

func TestLogBundleDelete(t *testing.T) {
	ctx := Context{
		TeamID:  ptr("team"),
		TaskID:  ptr("task"),
		RoundID: "round",
	}
	got, err := captureStdout(func() {
		LogBundleDelete(ctx, "bundle_id")
	})
	require.NoError(t, err)

	expect := regexp.MustCompile(
		`{"event":{"bundle_id":"bundle_id"},"task_id":"task","team_id":"team","log_context":"audit","version":"\d.\d.\d","round_id":"round","disposition":"neutral","event_type":"bundle_delete","timestamp":\d+}`,
	)
	assert.Regexp(t, expect, got)
}

func TestLogOutOfBudget(t *testing.T) {
	ctx := Context{
		TeamID:  ptr("team"),
		TaskID:  ptr("task"),
		RoundID: "round",
	}
	got, err := captureStdout(func() {
		LogOutOfBudget(ctx)
	})
	require.NoError(t, err)

	expect := regexp.MustCompile(
		`{"event":{},"task_id":"task","team_id":"team","log_context":"audit","version":"\d.\d.\d","round_id":"round","disposition":"bad","event_type":"out_of_budget","timestamp":\d+}`,
	)
	assert.Regexp(t, expect, got)
}

func TestLogCRSStatus(t *testing.T) {
	ctx := Context{
		TeamID:  ptr("team"),
		TaskID:  ptr("task"),
		RoundID: "round",
	}
	got, err := captureStdout(func() {
		LogCRSStatus(ctx, "crs_url", ptr(types.Status{
			Version: "0.0.0",
		}))
	})
	require.NoError(t, err)

	expect := regexp.MustCompile(
		`{"task_id":null,"team_id":"team","log_context":"audit","version":"\d.\d.\d","round_id":"round","disposition":"bad","event_type":"crs_status_check","timestamp":\d+,"event":{"version":"0.0.0","details":null,"state":{"tasks":{"pending":0,"errored":0,"processing":0,"canceled":0,"waiting":0,"succeeded":0,"failed":0}},"error":null,"crs_url":"crs_url","ready":false}}`,
	)
	assert.Regexp(t, expect, got)
}

func TestLogCRSStatusFailed(t *testing.T) {
	ctx := Context{
		TeamID:  ptr("team"),
		TaskID:  ptr("task"),
		RoundID: "round",
	}
	got, err := captureStdout(func() {
		LogCRSStatusFailed(ctx, "crs_url", "error")
	})
	require.NoError(t, err)

	expect := regexp.MustCompile(
		`{"task_id":null,"team_id":"team","log_context":"audit","version":"\d.\d.\d","round_id":"round","disposition":"bad","event_type":"crs_status_check","timestamp":\d+,"event":{"version":null,"details":null,"state":null,"error":"error","crs_url":"crs_url","ready":false}}`,
	)
	assert.Regexp(t, expect, got)
}

func TestLogBroadcastSucceeded(t *testing.T) {
	ctx := Context{
		TeamID:  ptr("team"),
		TaskID:  ptr("task"),
		RoundID: "round",
	}
	got, err := captureStdout(func() {
		LogBroadcastSucceeded(ctx, 0)
	})
	require.NoError(t, err)

	expect := regexp.MustCompile(
		`{"task_id":"task","team_id":"team","log_context":"audit","version":"\d.\d.\d","round_id":"round","disposition":"good","event_type":"broadcast_succeeded","timestamp":\d+,"event":{"retries":0}}`,
	)
	assert.Regexp(t, expect, got)
}

func TestLogBroadcastFailed(t *testing.T) {
	ctx := Context{
		TeamID:  ptr("team"),
		TaskID:  ptr("task"),
		RoundID: "round",
	}
	got, err := captureStdout(func() {
		LogBroadcastFailed(ctx, "payload", 0)
	})
	require.NoError(t, err)

	expect := regexp.MustCompile(
		`{"task_id":"task","team_id":"team","log_context":"audit","version":"\d.\d.\d","round_id":"round","disposition":"bad","event_type":"broadcast_failed","timestamp":\d+,"event":{"payload":"payload","retries":0}}`,
	)
	assert.Regexp(t, expect, got)
}

func TestLogFreeformSubmission(t *testing.T) {
	ctx := Context{
		TeamID:  ptr("team"),
		TaskID:  ptr("task"),
		RoundID: "round",
	}
	got, err := captureStdout(func() {
		LogFreeformSubmission(ctx)
	})
	require.NoError(t, err)

	expect := regexp.MustCompile(
		`{"event":{},"task_id":"task","team_id":"team","log_context":"audit","version":"\d.\d.\d","round_id":"round","disposition":"neutral","event_type":"freeform_submission","timestamp":\d+}`,
	)
	assert.Regexp(t, expect, got)
}
