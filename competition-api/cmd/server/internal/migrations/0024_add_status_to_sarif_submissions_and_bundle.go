package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0024, Down0024)
}

func Up0024(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE sarif_submission ADD COLUMN status TEXT NOT NULL DEFAULT 'accepted';
`},
		statement{query: `
ALTER TABLE sarif_submission ALTER COLUMN status DROP DEFAULT;
`},
		statement{query: `
ALTER TABLE bundle ADD COLUMN status TEXT NOT NULL DEFAULT 'accepted';
`},
		statement{query: `
ALTER TABLE bundle ALTER COLUMN status DROP DEFAULT;
`})
}

func Down0024(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE bundle DROP COLUMN status;`},
		statement{query: `
ALTER TABLE sarif_submission DROP COLUMN status;`})
}
