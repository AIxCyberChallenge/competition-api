package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up0023, Down0023)
}

func Up0023(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
ALTER TABLE sarif_submission RENAME TO sarif_assessment;
`},
		statement{query: `
ALTER TABLE vuln_submission RENAME TO pov_submission;
`},
		statement{query: `
ALTER TABLE pov_submission RENAME COLUMN data_file_path TO testcase_path;
`},
		statement{query: `
ALTER TABLE pov_submission RENAME COLUMN harness_name TO fuzzer_name;
`},
		statement{query: `
CREATE TABLE sarif_submission (
	id UUID PRIMARY KEY DEFAULT uuidv7_sub_ms(),
	submitter_id UUID NOT NULL,
	FOREIGN KEY (submitter_id) REFERENCES auth(id),
	task_id UUID NOT NULL,
	FOREIGN KEY (task_id) REFERENCES task(id),
	sarif JSONB NOT NULL,
	created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT current_timestamp,
	updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT current_timestamp
);`},
		statement{query: `
CREATE TABLE bundle (
	id UUID PRIMARY KEY DEFAULT uuidv7_sub_ms(),
	submitter_id UUID NOT NULL,
	FOREIGN KEY (submitter_id) REFERENCES auth(id),
	task_id UUID NOT NULL,
	FOREIGN KEY (task_id) REFERENCES task(id),
	pov_id UUID,
	FOREIGN KEY (pov_id) REFERENCES pov_submission(id),
	patch_id UUID,
	FOREIGN KEY (patch_id) REFERENCES patch_submission(id),
	broadcast_sarif_id UUID,
	FOREIGN KEY (broadcast_sarif_id) REFERENCES sarif_broadcast(id),
	submitted_sarif_id UUID,
	FOREIGN KEY (submitted_sarif_id) REFERENCES sarif_submission(id),
	description TEXT,
	created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT current_timestamp,
	updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT current_timestamp
);`},
		statement{query: `
CREATE TRIGGER touch_updated_at_trigger
BEFORE UPDATE ON sarif_submission
FOR EACH ROW EXECUTE PROCEDURE touch_updated_at();`},
		statement{query: `
CREATE TRIGGER touch_updated_at_trigger
BEFORE UPDATE ON bundle
FOR EACH ROW EXECUTE PROCEDURE touch_updated_at();`})
}

func Down0023(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx,
		statement{query: `
DROP TABLE envelope;`},
		statement{query: `
DROP TABLE sarif_submission;`},
		statement{query: `
ALTER TABLE pov_submission RENAME COLUMN harness_name TO fuzzer_name;`},
		statement{query: `
ALTER TABLE pov_submission RENAME COLUMN testcase_path TO data_file_path;`},
		statement{query: `
ALTER TABLE pov_submission RENAME TO vuln_submission;`},
		statement{query: `
ALTER TABLE sarif_assessment RENAME TO sarif_submission;`})
}
