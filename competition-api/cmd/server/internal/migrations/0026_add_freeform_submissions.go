package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0026, Down0026)
}

func Up0026(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
			CREATE TABLE freeform_submission (
				id UUID PRIMARY KEY DEFAULT uuidv7_sub_ms(),
				submitter_id UUID NOT NULL,
				FOREIGN KEY (submitter_id) REFERENCES auth(id),
				task_id UUID NOT NULL,
				FOREIGN KEY (task_id) REFERENCES task(id),
				submission TEXT NOT NULL,
				status TEXT NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT current_timestamp,
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT current_timestamp
);`},
		statement{query: `
ALTER TABLE bundle ADD COLUMN freeform_submission_id UUID;`},
		statement{query: `
ALTER TABLE bundle
ADD CONSTRAINT bundle_freeform_submission_id_fk
FOREIGN KEY (freeform_submission_id) REFERENCES freeform_submission(id);
`})
}

func Down0026(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE bundle DROP CONSTRAINT bundle_freeform_submission_id_fk;`},
		statement{query: `
ALTER TABLLE bundle DROP COLUMN freeform_submission_id;`},
		statement{query: `
DROP TABLE freeform_submission;`})
}
