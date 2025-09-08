package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0035, Down0035)
}

func Up0035(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE task ADD COLUMN cpus int NOT NULL DEFAULT 2;
`},
		statement{query: `
ALTER TABLE task ALTER COLUMN cpus DROP DEFAULT;
`},
		statement{query: `
UPDATE task SET cpus = 8 WHERE focus ILIKE '%wireshark%' OR focus ILIKE '%poi%' or focus ILIKE '%curl%';
`})
}

func Down0035(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE task DROP COLUMN cpus;`})
}
