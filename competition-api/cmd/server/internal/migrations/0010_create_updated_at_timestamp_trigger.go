package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0010, Down0010)
}

func Up0010(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
CREATE FUNCTION touch_updated_at()
RETURNS TRIGGER AS $$
BEGIN
NEW.updated_at = current_timestamp;
RETURN NEW;
END;
$$ language 'plpgsql'; 
`)

	return err
}

func Down0010(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP FUNCTION touch_updated_at();`)
	return err
}
