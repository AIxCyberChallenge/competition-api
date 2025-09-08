package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0018, Down0018)
}

func Up0018(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE auth ALTER COLUMN active DROP DEFAULT`},
		statement{query: `
ALTER TABLE auth ADD COLUMN permissions JSONB NOT NULL DEFAULT '{"crs": false, "out_of_budget": false}';`},
		statement{query: `
ALTER TABLE auth ALTER COLUMN permissions DROP DEFAULT`},
	)
}

func Down0018(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE auth DROP COLUMN permissions`},
		statement{query: `
ALTER TABLE auth ALTER COLUMN permissions SET DEFAULT FALSE`},
	)
}
