package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0006, Down0006)
}

func Up0006(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
CREATE TABLE sarif_broadcast (
    id UUID PRIMARY KEY DEFAULT uuidv7_sub_ms(),
    vuln_id UUID NOT NULL,
    FOREIGN KEY (vuln_id) REFERENCES vuln_submission(id),
    sarif JSONB NOT NULL,
    validity TEXT NOT NULL,
    round_id TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT current_timestamp,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT current_timestamp
);
`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`CREATE INDEX sarif_broadcast_vuln_id_index ON sarif_broadcast (vuln_id);`,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`CREATE INDEX sarif_broadcast_round_id_index ON sarif_broadcast (round_id);`,
	)
	if err != nil {
		return err
	}

	return nil
}

func Down0006(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP INDEX sarif_broadcast_round_id_index;`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DROP INDEX sarif_broadcast_vuln_id_index;`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DROP TABLE sarif_broadcast;`)
	if err != nil {
		return err
	}

	return nil
}
