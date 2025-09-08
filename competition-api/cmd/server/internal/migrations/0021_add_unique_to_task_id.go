package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0021, Down0021)
}

func Up0021(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE sarif_broadcast ADD CONSTRAINT task_id_unique UNIQUE (task_id);`})
}

func Down0021(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE sarif_broadcast DROP CONSTRAINT task_id_unique;`})
}
