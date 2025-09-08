package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/alexedwards/argon2id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/azure/azurite"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/mock/gomock"
	"gorm.io/datatypes"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/middleware"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/migrations"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/routes"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/routes/competition"
	routesv1 "github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/routes/v1"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/config"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/otel"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/types"
	mockuploader "github.com/aixcyberchallenge/competition-api/competition-api/internal/upload/mock"
)

const (
	authToken = "i am a very secure password"
)

var (
	taskOpen               models.Task
	taskExpired            models.Task
	auth                   models.Auth
	auth2                  models.Auth
	authInactive           models.Auth
	authCompetitionManager models.Auth
	vuln                   models.POVSubmission
	vulnExpired            models.POVSubmission
	patch                  models.PatchSubmission
	sarifBroadcast         models.SARIFBroadcast
	sarifBroadcastExpired  models.SARIFBroadcast
	sarifAssessment        models.SARIFAssessment
	sarifSubmitted         models.SARIFSubmission
	bundle                 models.Bundle
	bundleExpired          models.Bundle
	freeform               models.FreeformSubmission
)

type clientAuth struct {
	id    string
	token string
}

// FIXME: put real data in the storage containers with the db entries
func seedDB(db *gorm.DB, roundID string) error {
	taskOpen = models.Task{
		Type:     types.TaskTypeFull,
		Deadline: time.Date(3000, time.January, 1, 0, 0, 0, 0, time.Local),
		RoundID:  roundID,
		Commit:   "foobar",
		Source:   []models.Source{},
		UnstrippedSource: models.UnstrippedSources{
			HeadRepo: models.Source{
				Type:   string(types.SourceTypeRepo),
				URL:    "foobar",
				SHA256: "foobar",
			},
			FuzzTooling: models.Source{
				Type:   string(types.SourceTypeFuzzTooling),
				URL:    "foobar",
				SHA256: "foobar",
			},
		},
		HarnessesIncluded: true,
	}

	result := db.Create(&taskOpen)
	if result.Error != nil {
		return result.Error
	}

	taskExpired = models.Task{
		Type:     types.TaskTypeFull,
		Deadline: time.Date(1000, time.January, 1, 0, 0, 0, 0, time.Local),
		RoundID:  roundID,
		Commit:   "foobar",
		Source:   []models.Source{},
		UnstrippedSource: models.UnstrippedSources{
			HeadRepo: models.Source{
				Type:   string(types.SourceTypeRepo),
				URL:    "foobar",
				SHA256: "foobar",
			},
			FuzzTooling: models.Source{
				Type:   string(types.SourceTypeFuzzTooling),
				URL:    "foobar",
				SHA256: "foobar",
			},
		},
		HarnessesIncluded: true,
	}

	result = db.Create(&taskExpired)
	if result.Error != nil {
		return result.Error
	}

	hash, err := argon2id.CreateHash(authToken, argon2id.DefaultParams)
	if err != nil {
		return err
	}

	authToInsert := []*models.Auth{}
	auth = models.Auth{
		Token:  hash,
		Note:   "very testing team",
		Active: models.NewNullFromData(true),
		Permissions: models.Permissions{
			CRS: true,
		},
	}
	authToInsert = append(authToInsert, &auth)

	hash2, err := argon2id.CreateHash(authToken, argon2id.DefaultParams)
	if err != nil {
		return err
	}
	auth2 = models.Auth{
		Token:  hash2,
		Note:   "very testing team 2",
		Active: models.NewNullFromData(true),
		Permissions: models.Permissions{
			CRS: true,
		},
	}
	authToInsert = append(authToInsert, &auth2)

	authInactive = models.Auth{
		Token:  hash2,
		Note:   "very inactive auth",
		Active: models.NewNullFromData(false),
		Permissions: models.Permissions{
			CRS: true,
		},
	}
	authToInsert = append(authToInsert, &authInactive)

	authCompetitionManager = models.Auth{
		Token:  hash,
		Note:   "out of budget auth",
		Active: models.NewNullFromData(true),
		Permissions: models.Permissions{
			CompetitionManagement: true,
		},
	}
	authToInsert = append(authToInsert, &authCompetitionManager)

	result = db.Create(authToInsert)
	if result.Error != nil {
		return result.Error
	}

	vuln = models.POVSubmission{
		SubmitterID:  auth.ID,
		TaskID:       taskOpen.ID,
		TestcasePath: "https://example.com/foobar.tar.gz",
		FuzzerName:   "harness_1",
		Sanitizer:    "ADDRESS",
		Architecture: "i386",
		Status:       types.SubmissionStatusAccepted,
	}

	result = db.Create(&vuln)
	if result.Error != nil {
		return result.Error
	}

	vulnExpired = models.POVSubmission{
		SubmitterID:  auth.ID,
		TaskID:       taskExpired.ID,
		TestcasePath: "https://example.com/foobar.tar.gz",
		FuzzerName:   "harness_1",
		Sanitizer:    "ADDRESS",
		Architecture: "i386",
		Status:       types.SubmissionStatusAccepted,
	}

	result = db.Create(&vulnExpired)
	if result.Error != nil {
		return result.Error
	}

	patch = models.PatchSubmission{
		SubmitterID:   auth.ID,
		PatchFilePath: "https://example.com/foobar.tar.gz",
		Status:        types.SubmissionStatusPassed,
		TaskID:        vuln.TaskID,
	}

	result = db.Create(&patch)
	if result.Error != nil {
		return result.Error
	}

	sarifBroadcast = models.SARIFBroadcast{
		TaskID: taskOpen.ID,
		SARIF:  datatypes.JSON(`{"version": "2.1.0", "runs": []}`),
	}

	result = db.Create(&sarifBroadcast)
	if result.Error != nil {
		return result.Error
	}

	sarifBroadcastExpired = models.SARIFBroadcast{
		TaskID: taskExpired.ID,
		SARIF:  datatypes.JSON(`{"version": "2.1.0", "runs": []}`),
	}

	result = db.Create(&sarifBroadcastExpired)
	if result.Error != nil {
		return result.Error
	}

	sarifAssessment = models.SARIFAssessment{
		SubmitterID:      auth.ID,
		SARIFBroadcastID: sarifBroadcast.ID,
		Assessment:       types.AssessmentIncorrect,
		Status:           types.SubmissionStatusAccepted,
	}

	result = db.Create(&sarifAssessment)
	if result.Error != nil {
		return result.Error
	}

	sarifSubmitted = models.SARIFSubmission{
		SubmitterID: auth.ID,
		TaskID:      taskOpen.ID,
		SARIF:       datatypes.JSON(`{"version": "2.1.0", "runs": []}`),
	}

	result = db.Create(&sarifSubmitted)
	if result.Error != nil {
		return result.Error
	}

	bundle = models.Bundle{
		SubmitterID:      auth.ID,
		TaskID:           taskOpen.ID,
		POVID:            models.NewNullFromData(vuln.ID),
		Status:           types.SubmissionStatusAccepted,
		SubmittedSARIFID: models.NewNullFromData(sarifSubmitted.ID),
	}
	result = db.Create(&bundle)
	if result.Error != nil {
		return result.Error
	}

	bundleExpired = models.Bundle{
		SubmitterID: auth.ID,
		Status:      types.SubmissionStatusDeadlineExceeded,
		TaskID:      taskExpired.ID,
		POVID:       models.NewNullFromData(vulnExpired.ID),
		Description: models.NewNullFromData("description"),
	}
	result = db.Create(&bundleExpired)
	if result.Error != nil {
		return result.Error
	}

	freeform = models.FreeformSubmission{
		TaskID:      taskOpen.ID,
		SubmitterID: auth.ID,
		Submission:  "foobar",
	}

	result = db.Create(&freeform)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

type ServerTestSuite struct {
	suite.Suite

	archiver *mockuploader.MockUploader

	config       *config.Config
	azurite      *azurite.Container
	blobClient   *azblob.Client
	postgres     *postgres.PostgresContainer
	db           *gorm.DB
	tx           *gorm.DB
	otelShutdown func(context.Context) error
	server       *httptest.Server
}

func (s *ServerTestSuite) SetupSuite() {
	ctrl := gomock.NewController(s.T())
	s.archiver = mockuploader.NewMockUploader(ctrl)

	logger.InitSlog()

	cfg, err := config.GetConfig()
	s.Require().NoError(err, "failed getting config")
	s.config = cfg

	azuriteContainer, err := azurite.Run(
		s.T().Context(),
		"mcr.microsoft.com/azure-storage/azurite:latest",
		azurite.WithInMemoryPersistence(256),
	)
	s.Require().NoError(err, "failed to make azurite container")
	s.azurite = azuriteContainer

	azureCred, err := azblob.NewSharedKeyCredential(azurite.AccountName, azurite.AccountKey)
	s.Require().NoError(err, "failed to make azure cred")

	azureClient, err := azblob.NewClientWithSharedKeyCredential(s.AzureStorageURL(), azureCred, nil)
	s.Require().NoError(err, "failed to make azure client")
	s.blobClient = azureClient

	postgresContainer, err := postgres.Run(
		s.T().Context(),
		"postgres:16.4-alpine",
		postgres.WithDatabase("competitionapi"),
		postgres.WithUsername("competitionapi"),
		postgres.WithPassword("competitionapi"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	s.Require().NoError(err, "failed to start postgres container")
	s.postgres = postgresContainer

	dsn, err := s.postgres.ConnectionString(s.T().Context())
	s.Require().NoError(err, "failed to get connection string to container")

	db, err := gorm.Open(gormpostgres.Open(dsn))
	s.Require().NoError(err, "failed to connect to the database")
	s.db = db

	err = migrations.Up(s.T().Context(), db)
	s.Require().NoError(err, "failed to run up migrations")

	s.Require().NoError(seedDB(db, *cfg.RoundID), "failed to seed db")

	shutdownOTel, err := otel.SetupOTelSDK(s.T().Context(), false)
	s.Require().NoError(err, "could not setup otel")
	s.otelShutdown = shutdownOTel
}

func (s *ServerTestSuite) SetupTest() {
	s.archiver.EXPECT().Exists(gomock.Any(), gomock.Any()).AnyTimes()
	s.archiver.EXPECT().StoreIdentifier(gomock.Any()).AnyTimes()
	s.archiver.EXPECT().Upload(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	err := setupContainers(s.T().Context(), s.blobClient, s.config.Azure.StorageAccount.Containers)
	s.Require().NoError(err, "failed to setup containers")
	s.tx = s.db.Begin()

	v1Handler := routesv1.NewHandler(
		s.tx,
		nil,
		nil,
		nil,
		nil,
		s.config,
		s.archiver,
		s.archiver,
		s.archiver,
	)
	competitionHandler := competition.Handler{RoundID: *s.config.RoundID}
	middlewareHandler := middleware.Handler{DB: s.tx}

	e, err := routes.BuildEcho(logger.Logger)
	s.Require().NoError(err, "failed to construct router")

	v1Handler.AddRoutes(e, &middlewareHandler)
	competitionHandler.AddRoutes(e, &middlewareHandler)

	s.server = httptest.NewServer(e)
}

func (s *ServerTestSuite) TearDownTest() {
	s.Require().NoError(s.tx.Rollback().Error)
	s.server.Close()
}

func (s *ServerTestSuite) TearDownSuite() {
	s.Require().NoError(testcontainers.TerminateContainer(s.azurite))
	s.Require().NoError(testcontainers.TerminateContainer(s.postgres))
	s.Require().NoError(s.otelShutdown(s.T().Context()))
}

func (s *ServerTestSuite) AzureStorageURL() string {
	storageURLRaw, err := s.azurite.BlobServiceURL(s.T().Context())
	s.Require().NoError(err, "failed to get azure blob url")

	return fmt.Sprintf("%s/%s", storageURLRaw, azurite.AccountName)
}

func (s *ServerTestSuite) AzureQueueURL() string {
	queueURLRaw, err := s.azurite.QueueServiceURL(s.T().Context())
	s.Require().NoError(err, "failed to get azure queue url")

	return fmt.Sprintf("%s/%s", queueURLRaw, azurite.AccountName)
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

type resp struct {
	body string
	code int
}

func doRequest(t *testing.T, req *http.Request) (*resp, error) {
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "failed to send http request")
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err, "failed to read body")

	return &resp{body: string(body), code: res.StatusCode}, nil
}

func base64String(length int) string {
	arr := make([]byte, length)
	for i := range arr {
		arr[i] = 'a'
	}
	return base64.StdEncoding.EncodeToString(arr)
}

func longString(length int) string {
	arr := make([]byte, length)
	for i := range arr {
		arr[i] = 'a'
	}
	return string(arr)
}

func notFoundBodyTester(t *testing.T, body map[string]any) {
	assert.Contains(t, body, "message", "contains message key")
	assert.Contains(t, body["message"], "not found")
}

func unauthorizedBodyTester(t *testing.T, body map[string]any) {
	assert.Contains(t, body, "message", "contains message key")
	assert.Contains(t, body["message"], "Unauthorized")
}

func assertErrorBodyWithFields(t *testing.T, body map[string]any) {
	assert.Contains(t, body, "message", "contains message key")
	assert.Contains(t, body, "fields", "contains fields key")
}
