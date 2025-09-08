package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0016, Down0016)
}

func Up0016(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
ALTER TABLE patch_submission
ADD COLUMN sarif_id UUID;`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
ALTER TABLE patch_submission
ADD CONSTRAINT sarif_id
FOREIGN KEY (sarif_id)
REFERENCES sarif_broadcast(id);`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`CREATE INDEX patch_submission_sarif_id_index ON patch_submission (sarif_id);`,
	)
	if err != nil {
		return err
	}

	return nil
}

func Down0016(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP INDEX patch_submission_sarif_id_index;`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
ALTER TABLE patch_submission
DROP CONSTRAINT patch_submission_sarif_id_fkey;`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
ALTER TABLE patch_submission
DROP COLUMN sarif_id;`,
	)
	if err != nil {
		return err
	}

	return nil
}
