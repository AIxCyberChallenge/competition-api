package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0008, Down0008)
}

func Up0008(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
CREATE TABLE task (
    id UUID PRIMARY KEY DEFAULT uuidv7_sub_ms(),
    type TEXT NOT NULL,
    deadline TIMESTAMP WITH TIME ZONE NOT NULL,
    source JSONB NOT NULL,
    round_id TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT current_timestamp,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT current_timestamp
);
`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `CREATE INDEX task_round_id_index ON task (round_id);`)
	if err != nil {
		return err
	}

	return nil
}

func Down0008(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP INDEX task_round_id_index;`)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DROP TABLE task;`)
	if err != nil {
		return err
	}

	return nil
}
