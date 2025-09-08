package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0013, Down0013)
}

func Up0013(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
ALTER TABLE task
ADD COLUMN unstripped_source JSONB NOT NULL DEFAULT '[]';`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
ALTER TABLE task
ALTER COLUMN unstripped_source DROP DEFAULT;`,
	)
	if err != nil {
		return err
	}

	return nil
}

func Down0013(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
ALTER TABLE task
DROP COLUMN unstripped_source;
`)
	if err != nil {
		return err
	}

	return nil
}
