package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0032, Down0032)
}

func Up0032(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE task ADD COLUMN memory_gb int NOT NULL DEFAULT 8;
`},
		statement{query: `
ALTER TABLE task ALTER COLUMN memory_gb DROP DEFAULT;
`},
		statement{query: `
UPDATE task SET memory_gb = 20 WHERE focus ILIKE '%tika%' OR focus ILIKE '%commons-compress%';
`})
}

func Down0032(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE task DROP COLUMN memory_gb;`})
}
