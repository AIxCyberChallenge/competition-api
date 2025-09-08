package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0017, Down0017)
}

func Up0017(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE auth ADD COLUMN active BOOLEAN NOT NULL DEFAULT FALSE;`},
		statement{query: `
CREATE INDEX auth_active_index ON auth (active);`},
	)
}

func Down0017(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
DROP INDEX auth_active_index;`},
		statement{query: `
ALTER TABLE auth DROP COLUMN active;`},
	)
}
