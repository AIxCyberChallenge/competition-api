package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0028, Down0028)
}

func Up0028(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE pov_submission ADD COLUMN engine TEXT NOT NULL DEFAULT 'libfuzzer';
`},
		statement{query: `
ALTER TABLE pov_submission ALTER COLUMN engine DROP DEFAULT;
`})
}

func Down0028(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE pov_submission DROP COLUMN engine;`})
}
