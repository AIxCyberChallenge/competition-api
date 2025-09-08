package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0034, Down0034)
}

func Up0034(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE job ADD COLUMN cache_key TEXT NOT NULL DEFAULT 'abc123';
`},
		statement{query: `
ALTER TABLE job ALTER COLUMN cache_key DROP DEFAULT;
`},
		statement{query: `
CREATE INDEX job_cache_key ON job (cache_key);
`})
}

func Down0034(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
DROP INDEX job_cache_key;`},
		statement{query: `
ALTER TABLE job DROP COLUMN cache_key;`})
}
