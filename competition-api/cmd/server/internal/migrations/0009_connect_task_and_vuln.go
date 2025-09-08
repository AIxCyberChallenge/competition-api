package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0009, Down0009)
}

func Up0009(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
ALTER TABLE vuln_submission
ADD CONSTRAINT vuln_submission_task_id_fk
FOREIGN KEY (task_id) REFERENCES task(id);
`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`CREATE INDEX vuln_submission_task_id_index ON vuln_submission (task_id);`,
	)
	if err != nil {
		return err
	}

	return nil
}

func Down0009(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP INDEX vuln_submission_task_id_index;`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
ALTER TABLE vuln_submission
DROP CONSTRAINT vuln_submission_task_id_fk;
`)
	if err != nil {
		return err
	}

	return nil
}
