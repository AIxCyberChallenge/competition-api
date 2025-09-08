package migrations

import (
	"context"
	"database/sql"
	"errors"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0015, Down0015)
}

func Up0015(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
ALTER TABLE sarif_broadcast
ADD COLUMN task_id UUID;`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
UPDATE sarif_broadcast
SET task_id = vuln_submission.task_id
FROM vuln_submission
WHERE sarif_broadcast.vuln_id = vuln_submission.id;`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
ALTER TABLE sarif_broadcast
ALTER COLUMN task_id SET NOT NULL;`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
ALTER TABLE sarif_broadcast
ADD CONSTRAINT task_id
FOREIGN KEY (task_id)
REFERENCES task(id);`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`CREATE INDEX sarif_broadcast_task_id_index ON patch_submission (task_id);`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
ALTER TABLE sarif_broadcast
DROP COLUMN vuln_id;`,
	)
	if err != nil {
		return err
	}

	return nil
}

func Down0015(_ context.Context, _ *sql.Tx) error {
	return errors.New(
		"cannot undo dropping column vuln_id in sarif_broadcast table that information is gone",
	)
}
