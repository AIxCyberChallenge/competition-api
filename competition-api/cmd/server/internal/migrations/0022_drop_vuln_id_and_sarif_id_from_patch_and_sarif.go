package migrations

import (
	"context"
	"database/sql"
	"errors"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0022, Down0022)
}

func Up0022(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE patch_submission DROP COLUMN vuln_id;`},
		statement{query: `
ALTER TABLE patch_submission DROP COLUMN sarif_id;`},
		statement{query: `
ALTER TABLE patch_submission DROP COLUMN description;`},
		statement{query: `
ALTER TABLE vuln_submission DROP COLUMN sarif;`},
	)
}

func Down0022(_ context.Context, _ *sql.Tx) error {
	return errors.New("this information is gone it cannot be undone")
}
