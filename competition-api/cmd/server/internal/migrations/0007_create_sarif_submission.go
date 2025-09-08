package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0007, Down0007)
}

func Up0007(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
CREATE TABLE sarif_submission (
    id UUID PRIMARY KEY DEFAULT uuidv7_sub_ms(),
    submitter_id UUID NOT NULL,
    FOREIGN KEY (submitter_id) REFERENCES auth(id),
    sarif_broadcast_id UUID NOT NULL,
    FOREIGN KEY (sarif_broadcast_id) REFERENCES sarif_broadcast(id),
    assessment TEXT NOT NULL,
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
		`CREATE INDEX sarif_submission_submitter_id_index ON sarif_submission (submitter_id);`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`CREATE INDEX sarif_submission_sarif_broadcast_id_index ON sarif_submission (sarif_broadcast_id);`,
	)
	if err != nil {
		return err
	}

	return nil
}

func Down0007(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP INDEX sarif_submission_sarif_broadcast_id_index;`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DROP INDEX sarif_submission_submitter_id_index;`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DROP TABLE sarif_submission;`)
	if err != nil {
		return err
	}

	return nil
}
