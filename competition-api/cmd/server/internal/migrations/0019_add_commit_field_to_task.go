package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0019, Down0019)
}

func Up0019(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE task ADD COLUMN commit TEXT NOT NULL DEFAULT 'FIXME';`},
		statement{query: `
ALTER TABLE task ALTER COLUMN commit DROP DEFAULT`},
	)
}

func Down0019(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE auth DROP COLUMN commit`},
	)
}
