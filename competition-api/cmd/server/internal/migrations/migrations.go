package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

var tracer = otel.Tracer(
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/migrations",
)

func Up(ctx context.Context, db *gorm.DB) error {
	ctx, span := tracer.Start(ctx, "Up")
	defer span.End()

	rawDB, err := db.DB()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to bring migrations up")
		return err
	}

	err = goose.UpContext(ctx, rawDB, ".")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to bring migrations up")
		return err
	}

	span.AddEvent("migrated_up")

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "brought migrations up")
	return nil
}

func Down(ctx context.Context, db *gorm.DB) error {
	rawDB, err := db.DB()
	if err != nil {
		return err
	}

	err = goose.DownToContext(ctx, rawDB, ".", 0)
	if err != nil {
		return err
	}

	return nil
}

type statement struct {
	query string
	args  []any
}

func execStatements(ctx context.Context, tx *sql.Tx, statements ...statement) error {
	for _, statement := range statements {
		_, err := tx.ExecContext(ctx, statement.query, statement.args...)
		if err != nil {
			return err
		}
	}

	return nil
}
