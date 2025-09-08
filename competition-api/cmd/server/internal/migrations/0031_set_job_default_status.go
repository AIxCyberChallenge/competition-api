package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0031, Down0031)
}

func Up0031(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
ALTER TABLE job ALTER COLUMN status SET DEFAULT 'accepted';
`)
	if err != nil {
		return err
	}

	return nil
}

func Down0031(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `ALTER TABLE job ALTER COLUMN status DROP DEFAULT;`)
	if err != nil {
		return err
	}

	return nil
}
