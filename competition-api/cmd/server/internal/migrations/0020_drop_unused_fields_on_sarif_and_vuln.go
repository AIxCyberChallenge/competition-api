package migrations

import (
	"context"
	"database/sql"
	"errors"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0020, Down0020)
}

func Up0020(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE vuln_submission DROP COLUMN commit_hash;`},
		statement{query: `
ALTER TABLE sarif_broadcast DROP COLUMN validity;`},
		statement{query: `
ALTER TABLE sarif_broadcast DROP COLUMN round_id;`},
	)
}

func Down0020(_ context.Context, _ *sql.Tx) error {
	return errors.New("cannot undo data deletion")
}
