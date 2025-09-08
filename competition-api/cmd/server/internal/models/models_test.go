package models

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/migrations"
)

func TestUtilities(t *testing.T) {
	ctx := context.Background()

	postgresContainer, err := postgres.Run(ctx,
		"postgres:16.4-alpine",
		postgres.WithDatabase("competitionapi"),
		postgres.WithUsername("competitionapi"),
		postgres.WithPassword("competitionapi"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	defer func() {
		err = testcontainers.TerminateContainer(postgresContainer)
		assert.NoError(t, err, "failed to terminate container")
	}()
	require.NoError(t, err, "failed to start postgres container")

	dsn, err := postgresContainer.ConnectionString(ctx)
	require.NoError(t, err, "failed to get connection string to container")

	db, err := gorm.Open(gormpostgres.Open(dsn))
	require.NoError(t, err, "failed to connect to the database")

	err = migrations.Up(ctx, db)
	require.NoError(t, err, "failed to migrate db")

	auth := &Auth{
		Token:       "foobar",
		Note:        "foobar",
		Active:      NewNullFromData(true),
		Permissions: Permissions{CRS: true},
	}
	result := db.Create(auth)
	require.NoError(t, result.Error, "failed to write element to db")

	t.Run("ExistsByID", func(t *testing.T) {
		exists, err := Exists[Auth](context.Background(), db, "id = ?", auth.ID)
		require.NoError(t, err, "failed to check db for existence")

		assert.True(t, exists, "did not find the object")
	})

	t.Run("ExistsByNote", func(t *testing.T) {
		exists, err := Exists[Auth](context.Background(), db, "note = ?", auth.Note)
		require.NoError(t, err, "failed to check db for existence")

		assert.True(t, exists, "did not find the object")
	})

	t.Run("DoesNotExistByID", func(t *testing.T) {
		exists, err := Exists[Auth](context.Background(), db, "id = ?", uuid.New())
		require.NoError(t, err, "failed to check db for existence")

		assert.False(t, exists, "should not find object")
	})
}
