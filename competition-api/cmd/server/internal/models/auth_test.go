package models

import (
	"context"
	"fmt"
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
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/config"
)

func TestAuth(t *testing.T) {
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

	require.NoError(t, migrations.Up(ctx, db), "failed to migrate db")

	tru := true
	fal := false
	teams := []config.Team{
		{
			ID:   uuid.New().String(),
			Note: "Key 0",
			APIKey: config.APIKey{
				Token: "abcdefg",
				Permissions: config.APIKeyPermissions{
					CRS:                   fal,
					CompetitionManagement: fal,
				},
				Active: &tru,
			},
			CRS: &config.CRSConfig{
				URL:         "http://mock-crs.mock-crs",
				APIKeyID:    "test",
				APIKeyToken: "abcdefg",
			},
		},
		{
			ID:   uuid.New().String(),
			Note: "Key 1",
			APIKey: config.APIKey{
				Token: "abcdefg",
				Permissions: config.APIKeyPermissions{
					CRS:                   fal,
					CompetitionManagement: fal,
				},
				Active: &tru,
			},
			CRS: &config.CRSConfig{
				URL:         "http://mock-crs.mock-crs",
				APIKeyID:    "test",
				APIKeyToken: "abcdefg",
			},
		},
		{
			ID:   uuid.New().String(),
			Note: "Key 2",
			APIKey: config.APIKey{
				Token: "abcdefg",
				Permissions: config.APIKeyPermissions{
					CRS:                   fal,
					CompetitionManagement: fal,
				},
				Active: &tru,
			},
			CRS: &config.CRSConfig{
				URL:         "http://mock-crs.mock-crs",
				APIKeyID:    "test",
				APIKeyToken: "abcdefg",
			},
		},
	}

	t.Run("LoadManyNoPerms", func(t *testing.T) {
		err = LoadAPIKeysFromConfig(context.Background(), db, teams)
		require.NoError(t, err, "failed to upsert keys")
		checkKeys(t, db, teams, true, Permissions{})
	})

	t.Run("LoadManyLessOne", func(t *testing.T) {
		modifiedTeams := make([]config.Team, len(teams))
		copy(modifiedTeams, teams)

		err = LoadAPIKeysFromConfig(context.Background(), db, modifiedTeams[1:])
		require.NoError(t, err, "failed to upsert keys")

		checkKeys(t, db, modifiedTeams[1:], true, Permissions{})
		checkKeys(t, db, modifiedTeams[0:1], false, Permissions{})
	})

	t.Run("LoadManyMarkOneInactive", func(t *testing.T) {
		modifiedTeams := make([]config.Team, len(teams))
		copy(modifiedTeams, teams)

		modifiedTeams[0].APIKey.Active = &fal

		err = LoadAPIKeysFromConfig(context.Background(), db, modifiedTeams)
		require.NoError(t, err, "failed to upsert keys")

		checkKeys(t, db, modifiedTeams[1:], true, Permissions{})
		checkKeys(t, db, modifiedTeams[0:1], false, Permissions{})
	})

	t.Run("LoadManyAddPermissions", func(t *testing.T) {
		modifiedTeams := make([]config.Team, len(teams))
		copy(modifiedTeams, teams)

		modifiedTeams[0].APIKey.Permissions = config.APIKeyPermissions{CRS: true}

		err = LoadAPIKeysFromConfig(context.Background(), db, modifiedTeams)
		require.NoError(t, err, "failed to upsert keys")

		checkKeys(t, db, modifiedTeams[0:1], true, Permissions{CRS: true})
		checkKeys(t, db, modifiedTeams[1:], true, Permissions{})
	})

	t.Run("LoadManyAddPermissionsAndRemove", func(t *testing.T) {
		modifiedTeams := make([]config.Team, len(teams))
		copy(modifiedTeams, teams)

		modifiedTeams[0].APIKey.Permissions = config.APIKeyPermissions{CRS: true}

		err = LoadAPIKeysFromConfig(context.Background(), db, modifiedTeams)
		require.NoError(t, err, "failed to upsert keys")

		checkKeys(t, db, modifiedTeams[0:1], true, Permissions{CRS: true})
		checkKeys(t, db, modifiedTeams[1:], true, Permissions{})

		modifiedTeams[0].APIKey.Permissions = config.APIKeyPermissions{CompetitionManagement: true}

		err = LoadAPIKeysFromConfig(context.Background(), db, modifiedTeams)
		require.NoError(t, err, "failed to upsert keys")

		checkKeys(t, db, modifiedTeams[0:1], true, Permissions{CompetitionManagement: true})
		checkKeys(t, db, modifiedTeams[1:], true, Permissions{})
	})
}

func checkKeys(t *testing.T, db *gorm.DB, teams []config.Team, a bool, p Permissions) {
	for _, team := range teams {
		m, err := ByID[Auth](context.Background(), db, uuid.MustParse(team.ID))
		require.NoError(t, err, "failed to get key from db")

		assert.True(t, m.Active.Valid, "active is not valid")
		assert.Equalf(t, a, m.Active.V, "active not expected state: %s", team.Note)
		fmt.Println(m)
		assert.Equalf(t, p, m.Permissions, "permissions not expected state: %s", team.Note)
	}
}
