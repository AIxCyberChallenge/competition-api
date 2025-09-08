package models

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/audit"
)

const name string = "github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"

var tracer = otel.Tracer(name)

// Derived from gorm.Model
type Model struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	ID        uuid.UUID `gorm:"primaryKey;default:uuidv7_sub_ms()"`
}

type CompetitionAPIModel interface {
	GetID() uuid.UUID
}

type Submission interface {
	GetTaskID() uuid.UUID
	GetSubmitterID() uuid.UUID
	AuditLogSubmissionResult(c audit.Context)
}

// gets the round id from an object by selecting the task
func GetRoundIDForSubmission(ctx context.Context, db *gorm.DB, s Submission) (string, error) {
	task, err := ByID[Task](ctx, db, s.GetTaskID())
	if err != nil {
		return "", err
	}

	return task.RoundID, nil
}

// gets an object by id from the db
func ByID[T CompetitionAPIModel](ctx context.Context, db *gorm.DB, id uuid.UUID) (*T, error) {
	var data T

	ctx, span := tracer.Start(ctx, "ByID")
	defer span.End()

	db = db.WithContext(ctx)

	span.SetAttributes(
		attribute.String("id", id.String()),
		attribute.String("type", reflect.TypeOf(data).String()),
	)

	span.AddEvent("getting object by id")
	err := db.First(&data, id).Error
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get object by id")
		return nil, err
	}

	return &data, nil
}

// checks if an object exists in the db
func Exists[T CompetitionAPIModel](
	ctx context.Context,
	db *gorm.DB,
	query any,
	args ...any,
) (bool, error) {
	ctx, span := tracer.Start(ctx, "Exists")
	defer span.End()

	argStrings := make([]string, 0, len(args))
	for _, arg := range args {
		argStrings = append(argStrings, fmt.Sprint(arg))
	}

	span.SetAttributes(
		attribute.String("query", fmt.Sprint(query)),
		attribute.StringSlice("args", argStrings),
		attribute.String("type", reflect.TypeOf((*T)(nil)).Elem().String()),
	)

	db = db.WithContext(ctx)

	var data T
	var exists bool

	span.AddEvent("checking if element matching conditions exists")
	result := db.Model(&data).Select("1").Where(query, args).First(&exists)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}

		span.RecordError(result.Error)
		span.SetStatus(codes.Error, "failed to fetch from the db")
		return false, fmt.Errorf("failed to fetch from the db: %w", result.Error)
	}

	return exists, nil
}

// Transmutes a pointer into a [datatypes.Null]
func NewNull[T any](d *T) datatypes.Null[T] {
	if d != nil {
		return datatypes.NewNull(*d)
	}

	return datatypes.Null[T]{}
}

// Transmutes data into valid [datatypes.Null]
func NewNullFromData[T any](d T) datatypes.Null[T] {
	return datatypes.NewNull(d)
}

// Maps a [datatypes.Null] back into a pointer
func PtrFromNull[T any](d datatypes.Null[T]) *T {
	if !d.Valid {
		return nil
	}

	return &d.V
}
