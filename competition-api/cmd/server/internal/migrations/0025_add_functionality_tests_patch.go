package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0025, Down0025)
}

func Up0025(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE patch_submission ADD COLUMN functionality_tests_passing BOOLEAN;
`})
}

func Down0025(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE patch_submission DROP COLUMN functionality_tests_passing;`})
}
