package middleware

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
)

func TestAuthorization(t *testing.T) {
	l := logger.Logger
	t.Run("NeedsOneHasNone", func(t *testing.T) {
		hasPerm := hasPermission(
			context.TODO(),
			&models.Permissions{CRS: true},
			&models.Permissions{},
			l,
		)
		assert.False(t, hasPerm, "needs crs but does not have")
	})

	t.Run("NeedsOneHasExtra", func(t *testing.T) {
		hasPerm := hasPermission(
			context.TODO(),
			&models.Permissions{CRS: true},
			&models.Permissions{CRS: true, CompetitionManagement: true},
			l,
		)
		assert.True(t, hasPerm, "needs crs and has it")
	})

	t.Run("NeedsManyHasMany", func(t *testing.T) {
		hasPerm := hasPermission(
			context.TODO(),
			&models.Permissions{CRS: true, CompetitionManagement: true},
			&models.Permissions{CRS: true, CompetitionManagement: true},
			l,
		)
		assert.True(t, hasPerm, "needs crs and has it")
	})

	t.Run("NeedsOneHasOther", func(t *testing.T) {
		hasPerm := hasPermission(
			context.TODO(),
			&models.Permissions{CRS: true},
			&models.Permissions{CompetitionManagement: true},
			l,
		)
		assert.False(t, hasPerm, "needs crs but does not have it")
	})

	t.Run("HasOneNeedsOneWrongOrder", func(t *testing.T) {
		hasPerm := hasPermission(
			context.TODO(),
			&models.Permissions{CRS: true},
			&models.Permissions{CompetitionManagement: false, CRS: true},
			l,
		)
		assert.True(t, hasPerm, "needs crs and has it")
	})
}
