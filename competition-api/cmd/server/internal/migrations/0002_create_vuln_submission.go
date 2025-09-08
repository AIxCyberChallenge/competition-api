package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0002, Down0002)
}

func Up0002(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
CREATE TABLE vuln_submission (
    id UUID PRIMARY KEY DEFAULT uuidv7_sub_ms(),
    submitter_id UUID NOT NULL,
    task_id UUID NOT NULL,
    data_file_path TEXT NOT NULL,
    harness_name TEXT NOT NULL,
    sanitizer TEXT NOT NULL,
    architecture TEXT NOT NULL,
    sarif JSONB DEFAULT NULL,
    status TEXT NOT NULL,
    commit_hash BYTEA DEFAULT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT current_timestamp,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT current_timestamp
);
`)
	if err != nil {
		return err
	}

	return nil
}

func Down0002(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP TABLE vuln_submission;`)
	if err != nil {
		return err
	}

	return nil
}
