package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0014, Down0014)
}

func Up0014(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
ALTER TABLE patch_submission
ALTER COLUMN vuln_id DROP NOT NULL;`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
ALTER TABLE patch_submission
ADD COLUMN task_id UUID;`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
UPDATE patch_submission
SET task_id = vuln_submission.task_id
FROM vuln_submission
WHERE patch_submission.vuln_id = vuln_submission.id;`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
ALTER TABLE patch_submission
ALTER COLUMN task_id SET NOT NULL;`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
ALTER TABLE patch_submission
ADD CONSTRAINT task_id 
FOREIGN KEY (task_id) 
REFERENCES task(id);`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`CREATE INDEX patch_submission_task_id_index ON patch_submission (task_id);`,
	)
	if err != nil {
		return err
	}

	return nil
}

func Down0014(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP INDEX patch_submission_task_id_index;`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
ALTER TABLE patch_submission
DROP CONSTRAINT patch_sumbission_task_id_fkey;`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
ALTER TABLE patch_submission
ALTER COLUMN task_id DROP NOT NULL;`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
ALTER TABLE patch_submission
DROP COLUMN task_id;`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
ALTER TABLE patch_submission
ALTER COLUMN SET NOT NULL;
`)
	if err != nil {
		return err
	}

	return nil
}
