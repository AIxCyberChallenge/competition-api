package migrations

import (
	"context"
	"database/sql"
	"errors"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0027, Down0027)
}

type Source27 struct {
	Type   string `json:"type"` // TODO: fix enum
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

func Up0027(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE task ADD COLUMN harnesses_included boolean NOT NULL DEFAULT TRUE;
			`},
		statement{query: `
ALTER TABLE task ALTER COLUMN harnesses_included DROP DEFAULT;`,
		},
		statement{query: `
UPDATE task
SET unstripped_source = jsonb_build_object(
    'head_repo', unstripped_source->0,
    'fuzz_tooling', unstripped_source->1,
    'base_repo', unstripped_source->2
)
WHERE jsonb_typeof(unstripped_source) = 'array';
`})
}

func Down0027(_ context.Context, _ *sql.Tx) error {
	return errors.New("cannot unmigrate since some objects may not have fuzz harness")
}
