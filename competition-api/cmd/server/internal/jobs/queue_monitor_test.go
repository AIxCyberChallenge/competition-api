package jobs

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	sloggorm "github.com/imdatngo/slog-gorm/v2"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/mock/gomock"
	"gorm.io/datatypes"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/migrations"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	mockfetcher "github.com/aixcyberchallenge/competition-api/competition-api/internal/fetch/mock"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	mockuploader "github.com/aixcyberchallenge/competition-api/competition-api/internal/upload/mock"
)

type WorkerMsgHandlerTestSuite struct {
	suite.Suite

	pgContainer *postgres.PostgresContainer
	db          *gorm.DB
	tx          *gorm.DB
	archiver    *mockuploader.MockUploader
	fetcher     *mockfetcher.MockFetcher
	handler     *WorkerMsgHandler

	roundID uuid.UUID
}

func (s *WorkerMsgHandlerTestSuite) SetupSuite() {
	s.roundID = uuid.New()

	ct, err := postgres.Run(s.T().Context(),
		"postgres:16.4-alpine",
		postgres.WithDatabase("competitionapi"),
		postgres.WithUsername("competitionapi"),
		postgres.WithPassword("competitionapi"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	s.Require().NoError(err)
	s.pgContainer = ct

	connStr, err := s.pgContainer.ConnectionString(s.T().Context())
	s.Require().NoError(err)

	db, err := gorm.Open(gormpg.Open(connStr), &gorm.Config{
		Logger: sloggorm.New(),
	})
	s.Require().NoError(err)
	s.db = db

	s.Require().NoError(migrations.Up(s.T().Context(), s.db))
}

func (s *WorkerMsgHandlerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.archiver = mockuploader.NewMockUploader(ctrl)
	s.fetcher = mockfetcher.NewMockFetcher(ctrl)
	s.tx = s.db.Begin()

	s.handler = &WorkerMsgHandler{
		db:              s.tx,
		archiver:        s.archiver,
		artifactFetcher: s.fetcher,
	}
}

func (s *WorkerMsgHandlerTestSuite) TearDownTest() {
	s.handler.db = s.tx.Rollback()
}

func (s *WorkerMsgHandlerTestSuite) TearDownSuite() {
	s.Require().NoError(testcontainers.TerminateContainer(s.pgContainer))
}

func TestWorkerMsgHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(WorkerMsgHandlerTestSuite))
}

func (s *WorkerMsgHandlerTestSuite) Test_HandleArtifactMessage_TypeJob_NoJob() {
	err := s.handler.HandleArtifactMessage(s.T().Context(), &types.WorkerMsgArtifact{
		WorkerMsg: types.WorkerMsg{
			Entity:   types.JobTypeJob,
			EntityID: uuid.NewString(),
		},
	})

	s.Require().NoError(err)
}

func (s *WorkerMsgHandlerTestSuite) Test_HandleArtifactMessage_TypeJob_ExistingJob() {
	jobID := uuid.New()
	job := &models.Job{
		Model: models.Model{
			ID: jobID,
		},
	}
	s.Require().NoError(s.tx.Create(job).Error)

	artifact := types.JobArtifact{
		Blob: types.Blob{
			ObjectName: "foo",
		},
	}
	err := s.handler.HandleArtifactMessage(s.T().Context(), &types.WorkerMsgArtifact{
		WorkerMsg: types.WorkerMsg{
			Entity:   types.JobTypeJob,
			EntityID: jobID.String(),
		},
		Artifact: artifact,
	})
	s.Require().NoError(err)

	var got models.Job
	s.Require().NoError(s.tx.Model(&models.Job{}).Where("id = ?", jobID).Take(&got).Error)

	s.Contains(got.Artifacts, artifact)
}

func (s *WorkerMsgHandlerTestSuite) Test_HandleArtifactMessage_TypePOV() {
	pov := []byte("payload")

	s.fetcher.EXPECT().Fetch(gomock.Any(), "foo").Return(io.NopCloser(bytes.NewReader(pov)), nil)
	s.archiver.EXPECT().StoreIdentifier(gomock.Any()).Return("bucket", nil)
	s.archiver.EXPECT().
		Upload(gomock.Any(), bytes.NewReader(pov), int64(len(pov)), "foo").
		Return(nil)

	authID := uuid.New()
	s.Require().NoError(
		s.tx.Model(&models.Auth{}).Create(&models.Auth{
			Model: models.Model{
				ID: authID,
			},
			Active: datatypes.Null[bool]{V: true, Valid: true},
		}).Error,
	)

	taskID := uuid.New()
	s.Require().NoError(
		s.tx.Model(&models.Task{}).Create(&models.Task{
			Model: models.Model{
				ID: taskID,
			},
			Type:    types.TaskTypeDelta,
			RoundID: s.roundID.String(),
		}).Error,
	)

	povID := uuid.New()
	s.Require().NoError(
		s.tx.Model(&models.POVSubmission{}).Create(&models.POVSubmission{
			Model: models.Model{
				ID: povID,
			},
			SubmitterID: authID,
			TaskID:      taskID,
		}).Error,
	)

	err := s.handler.HandleArtifactMessage(s.T().Context(), &types.WorkerMsgArtifact{
		WorkerMsg: types.WorkerMsg{
			Entity:   types.JobTypePOV,
			EntityID: povID.String(),
		},
		Artifact: types.JobArtifact{
			Blob: types.Blob{
				ObjectName: "foo",
			},
		},
	})
	s.Require().NoError(err)
}

func (s *WorkerMsgHandlerTestSuite) Test_HandleArtifactMessage_TypePatch() {
	patch := []byte("payload")

	s.fetcher.EXPECT().Fetch(gomock.Any(), "foo").Return(io.NopCloser(bytes.NewReader(patch)), nil)
	s.archiver.EXPECT().StoreIdentifier(gomock.Any()).Return("bucket", nil)
	s.archiver.EXPECT().
		Upload(gomock.Any(), bytes.NewReader(patch), int64(len(patch)), "foo").
		Return(nil)

	authID := uuid.New()
	s.Require().NoError(
		s.tx.Model(&models.Auth{}).Create(&models.Auth{
			Model: models.Model{
				ID: authID,
			},
			Active: datatypes.Null[bool]{V: true, Valid: true},
		}).Error,
	)

	taskID := uuid.New()
	s.Require().NoError(
		s.tx.Model(&models.Task{}).Create(&models.Task{
			Model: models.Model{
				ID: taskID,
			},
			Type:    types.TaskTypeDelta,
			RoundID: s.roundID.String(),
		}).Error,
	)

	patchID := uuid.New()
	s.Require().NoError(
		s.tx.Model(&models.PatchSubmission{}).Create(&models.PatchSubmission{
			Model: models.Model{
				ID: patchID,
			},
			SubmitterID: authID,
			TaskID:      taskID,
		}).Error,
	)

	err := s.handler.HandleArtifactMessage(s.T().Context(), &types.WorkerMsgArtifact{
		WorkerMsg: types.WorkerMsg{
			Entity:   types.JobTypePatch,
			EntityID: patchID.String(),
		},
		Artifact: types.JobArtifact{
			Blob: types.Blob{
				ObjectName: "foo",
			},
		},
	})
	s.Require().NoError(err)
}

func (s *WorkerMsgHandlerTestSuite) Test_HandleCommandResultMessage_NotAJob() {
	s.Require().NoError(
		s.handler.HandleCommandResultMessage(s.T().Context(), &types.WorkerMsgCommandResult{
			WorkerMsg: types.WorkerMsg{
				Entity: types.JobTypePOV,
			},
		}),
	)
}

func (s *WorkerMsgHandlerTestSuite) Test_HandleCommandResultMessage_NoResults() {
	job := &models.Job{
		Model: models.Model{
			ID: uuid.New(),
		},
	}
	s.Require().NoError(
		s.tx.Model(&models.Job{}).Create(job).Error,
	)

	s.Require().ErrorContains(
		s.handler.HandleCommandResultMessage(s.T().Context(), &types.WorkerMsgCommandResult{
			WorkerMsg: types.WorkerMsg{
				Entity:   types.JobTypeJob,
				EntityID: job.ID.String(),
			},
		}),
		"empty command result message",
	)

	s.Require().NoError(s.tx.Model(job).First(job).Error)
	s.Empty(job.Results)
}

func (s *WorkerMsgHandlerTestSuite) Test_HandleCommandResultMessage_Success() {
	job := &models.Job{
		Model: models.Model{
			ID: uuid.New(),
		},
	}
	s.Require().NoError(
		s.tx.Model(&models.Job{}).Create(job).Error,
	)

	res := types.JobResult{
		Cmd: []string{"echo", `"hello world"`},
		StdoutBlob: types.Blob{
			ObjectName: "foo_stdout",
		},
		StderrBlob: types.Blob{
			ObjectName: "foo_stderr",
		},
	}
	s.Require().NoError(
		s.handler.HandleCommandResultMessage(s.T().Context(), &types.WorkerMsgCommandResult{
			WorkerMsg: types.WorkerMsg{
				Entity:   types.JobTypeJob,
				EntityID: job.ID.String(),
			},
			Result: &res,
		}),
	)

	s.Require().NoError(s.tx.Model(job).First(job).Error)
	s.Require().Len(job.Results, 1)
	s.Equal(res, job.Results[0])
}

func (s *WorkerMsgHandlerTestSuite) Test_HandleFinalMessage_InvalidUUID() {
	s.Require().ErrorContains(
		s.handler.HandleFinalMessage(s.T().Context(), &types.WorkerMsgFinal{
			WorkerMsg: types.WorkerMsg{
				MsgType:  types.MsgTypeFinal,
				Entity:   types.JobTypePOV,
				EntityID: "notauuid",
			},
		}),
		"failed to parse entity ID as UUID",
	)
}

func (s *WorkerMsgHandlerTestSuite) Test_HandleFinalMessage_POV() {
	taskID := uuid.New()
	s.Require().NoError(
		s.tx.Model(&models.Task{}).Create(&models.Task{
			Model: models.Model{
				ID: taskID,
			},
		}).Error,
	)

	authID := uuid.New()
	s.Require().NoError(
		s.tx.Model(&models.Auth{}).Create(&models.Auth{
			Model: models.Model{
				ID: authID,
			},
			Active: datatypes.Null[bool]{V: true, Valid: true},
		}).Error,
	)

	povSubID := uuid.New()
	s.Require().NoError(
		s.tx.Model(&models.POVSubmission{}).Create(&models.POVSubmission{
			Model: models.Model{
				ID: povSubID,
			},
			TaskID:      taskID,
			SubmitterID: authID,
			Status:      types.SubmissionStatusAccepted,
		}).Error,
	)

	s.Require().NoError(s.handler.HandleFinalMessage(s.T().Context(), &types.WorkerMsgFinal{
		WorkerMsg: types.WorkerMsg{
			MsgType:  types.MsgTypeFinal,
			Entity:   types.JobTypePOV,
			EntityID: povSubID.String(),
		},
		Status: types.SubmissionStatusPassed,
	}))
}

func (s *WorkerMsgHandlerTestSuite) Test_HandleFinalMessage_Patch() {
	taskID := uuid.New()
	s.Require().NoError(
		s.tx.Model(&models.Task{}).Create(&models.Task{
			Model: models.Model{
				ID: taskID,
			},
		}).Error,
	)

	authID := uuid.New()
	s.Require().NoError(
		s.tx.Model(&models.Auth{}).Create(&models.Auth{
			Model: models.Model{
				ID: authID,
			},
			Active: datatypes.Null[bool]{V: true, Valid: true},
		}).Error,
	)

	patchSubID := uuid.New()
	s.Require().NoError(
		s.tx.Model(&models.PatchSubmission{}).Create(&models.PatchSubmission{
			Model: models.Model{
				ID: patchSubID,
			},
			TaskID:      taskID,
			SubmitterID: authID,
			Status:      types.SubmissionStatusAccepted,
		}).Error,
	)

	s.Require().NoError(s.handler.HandleFinalMessage(s.T().Context(), &types.WorkerMsgFinal{
		WorkerMsg: types.WorkerMsg{
			MsgType:  types.MsgTypeFinal,
			Entity:   types.JobTypePatch,
			EntityID: patchSubID.String(),
		},
		Status: types.SubmissionStatusPassed,
	}))
}

func (s *WorkerMsgHandlerTestSuite) Test_HandleFinalMessage_Job() {
	patchSubID := uuid.New()
	s.Require().NoError(
		s.tx.Model(&models.Job{}).Create(&models.Job{
			Model: models.Model{
				ID: patchSubID,
			},
			Status: types.SubmissionStatusAccepted,
		}).Error,
	)

	s.Require().NoError(s.handler.HandleFinalMessage(s.T().Context(), &types.WorkerMsgFinal{
		WorkerMsg: types.WorkerMsg{
			MsgType:  types.MsgTypeFinal,
			Entity:   types.JobTypeJob,
			EntityID: patchSubID.String(),
		},
		Status: types.SubmissionStatusPassed,
	}))
}
