package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0011, Down0011)
}

var tables = []string{
	"vuln_submission",
	"auth",
	"patch_submission",
	"sarif_broadcast",
	"sarif_submission",
	"task",
}

func Up0011(ctx context.Context, tx *sql.Tx) error {
	for _, table := range tables {
		_, err := tx.ExecContext(ctx, fmt.Sprintf(`
CREATE TRIGGER touch_updated_at_trigger
BEFORE UPDATE ON %s
FOR EACH ROW EXECUTE PROCEDURE touch_updated_at();`,
			table))
		if err != nil {
			return err
		}
	}

	return nil
}

func Down0011(ctx context.Context, tx *sql.Tx) error {
	for _, table := range reverse(tables) {
		_, err := tx.ExecContext(
			ctx,
			fmt.Sprintf(`DROP TRIGGER touch_updated_at_trigger ON %s;`, table),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func reverse[T any](list []T) []T {
	for i, j := 0, len(list)-1; i < j; {
		list[i], list[j] = list[j], list[i]
		i++
		j--
	}
	return list
}
