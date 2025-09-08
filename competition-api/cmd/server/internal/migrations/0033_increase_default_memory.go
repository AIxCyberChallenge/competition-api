package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0033, Down0033)
}

func Up0033(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
UPDATE task SET memory_gb = 16 WHERE memory_gb = 8;
`})
}

func Down0033(_ context.Context, _ *sql.Tx) error {
	return nil
}
