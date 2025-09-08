package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0030, Down0030)
}

func Up0030(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
CREATE TABLE job (
    id UUID PRIMARY KEY DEFAULT uuidv7_sub_ms(),
	status TEXT NOT NULL,
	artifacts JSONB DEFAULT '[]'::jsonb,
	results JSONB DEFAULT '{}'::jsonb,
	created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT current_timestamp,
	updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT current_timestamp
);
`)
	if err != nil {
		return err
	}

	return nil
}

func Down0030(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP TABLE job;`)
	if err != nil {
		return err
	}

	return nil
}
