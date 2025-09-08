package workerqueue_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/worker/internal/workerqueue"
	mockqueue "github.com/aixcyberchallenge/competition-api/competition-api/internal/queue/mock"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
)

var entityID = "entityID"
var entityType = types.JobTypeJob

func TestArtifact(t *testing.T) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	queuer := mockqueue.NewMockQueuer(ctrl)

	resultContext := types.ResultCtxBaseRepoTest
	filename := "filename"
	blob := types.Blob{
		ObjectName: "objectname",
	}

	artifact := types.JobArtifact{
		Context:  resultContext,
		Filename: filename,
		Blob:     blob,
	}

	expected := types.WorkerMsgArtifact{
		WorkerMsg: types.WorkerMsg{
			MsgType:  types.MsgTypeArtifact,
			Entity:   entityType,
			EntityID: entityID,
		},
		Artifact: artifact,
	}

	queuer.EXPECT().Enqueue(gomock.Any(), expected).Times(1)

	wq := workerqueue.NewWorkerQueue(entityID, entityType, queuer)
	err := wq.Artifact(ctx, artifact)
	if !assert.NoError(t, err, "failed to queue artifact") {
		return
	}
}

func TestFinalMessage(t *testing.T) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	queuer := mockqueue.NewMockQueuer(ctrl)

	status := types.SubmissionStatusPassed
	patchTestsFailed := true

	expected := types.WorkerMsgFinal{
		WorkerMsg: types.WorkerMsg{
			MsgType:  types.MsgTypeFinal,
			Entity:   entityType,
			EntityID: entityID,
		},
		Status:           status,
		PatchTestsFailed: patchTestsFailed,
	}

	queuer.EXPECT().Enqueue(gomock.Any(), expected).Times(1)

	wq := workerqueue.NewWorkerQueue(entityID, entityType, queuer)
	err := wq.FinalMessage(ctx, status, patchTestsFailed)
	if !assert.NoError(t, err, "failed to queue artifact") {
		return
	}
}

func TestCommandResult(t *testing.T) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	queuer := mockqueue.NewMockQueuer(ctrl)

	exitCode := 0
	resultContext := types.ResultCtxBaseRepoTest

	result := &types.JobResult{
		Cmd: []string{"echo", "a"},
		StdoutBlob: types.Blob{
			ObjectName: "stdout",
		},
		StderrBlob: types.Blob{
			ObjectName: "stderr",
		},
		ExitCode: &exitCode,
		Context:  resultContext,
	}

	expected := types.WorkerMsgCommandResult{
		WorkerMsg: types.WorkerMsg{
			MsgType:  types.MsgTypeCommandResult,
			Entity:   entityType,
			EntityID: entityID,
		},
		Result: result,
	}

	queuer.EXPECT().Enqueue(gomock.Any(), expected).Times(1)

	wq := workerqueue.NewWorkerQueue(entityID, entityType, queuer)
	err := wq.CommandResult(ctx, result)
	if !assert.NoError(t, err, "failed to queue artifact") {
		return
	}
}
