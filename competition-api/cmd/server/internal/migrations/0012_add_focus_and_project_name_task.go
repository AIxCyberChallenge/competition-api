package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0012, Down0012)
}

func Up0012(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
ALTER TABLE task
ADD COLUMN focus TEXT NOT NULL DEFAULT 'FIXME';`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
ALTER TABLE task
ADD COLUMN project_name TEXT NOT NULL DEFAULT 'FIXME';`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
ALTER TABLE task
ALTER COLUMN focus DROP DEFAULT;`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
ALTER TABLE task
ALTER COLUMN project_name DROP DEFAULT;`,
	)
	if err != nil {
		return err
	}

	return nil
}

func Down0012(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
ALTER TABLE task
DROP COLUMN focus;
`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
ALTER TABLE task
DROP COLUMN project_name;
`)
	if err != nil {
		return err
	}

	return nil
}
