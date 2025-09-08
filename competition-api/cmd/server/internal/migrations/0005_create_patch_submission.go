package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0005, Down0005)
}

func Up0005(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
CREATE TABLE patch_submission (
    id UUID PRIMARY KEY DEFAULT uuidv7_sub_ms(),
    submitter_id UUID NOT NULL,
    FOREIGN KEY (submitter_id) REFERENCES auth(id),
    vuln_id UUID NOT NULL,
    FOREIGN KEY (vuln_id) REFERENCES vuln_submission(id),
    patch_file_path TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT current_timestamp,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT current_timestamp
);
`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`CREATE INDEX patch_submission_submitter_id_index ON patch_submission (submitter_id);`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`CREATE INDEX patch_submission_vuln_id_index ON patch_submission (vuln_id);`,
	)
	if err != nil {
		return err
	}

	return nil
}

func Down0005(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP INDEX patch_submission_vuln_id_index;`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DROP INDEX patch_submission_submitter_id_index;`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DROP TABLE patch_submission;`)
	if err != nil {
		return err
	}

	return nil
}
