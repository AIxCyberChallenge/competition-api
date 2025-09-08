package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0029, Down0029)
}

func Up0029(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE bundle ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
`})
}

func Down0029(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE bundle DROP COLUMN deleted_at;`})
}
