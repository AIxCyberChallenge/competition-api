package middleware

import (
	"context"
	"errors"
	"os"
	"reflect"

	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"

	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/models"
	"github.com/aixcyberchallenge/competition-api/competition-api/cmd/server/internal/response"
	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
)

// Used when doing a fake compare in the error case of BasicAuthValidator
var defaultHashForError string

const name string = "github.com/aixcyberchallenge/competition-api/competition-api/server/middleware"

var tracer = otel.Tracer(name)

// Generate a hash
func init() {
	var err error

	defaultHashForError, err = argon2id.CreateHash(
		"bnZSraUCS+nZh3MI8F3iiXbKFBcAyJhvAB6u/GBJzhC00ZPAQlyYVpQ+aryw7QvE2ZI=",
		argon2id.DefaultParams,
	)
	if err != nil {
		logger.Logger.Error("error creating default hash", "error", err)
		os.Exit(1)
	}
}

// Does a fake hash and compare for a hard coded password. Used when BasicAuthValidator hits an error or a nonexistent user.
func fakePasswordHash(ctx context.Context) {
	_, span := tracer.Start(ctx, "fakePasswordHash")
	defer span.End()

	_, err := argon2id.ComparePasswordAndHash("i am a very real password", defaultHashForError)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to compare fake password with default hash for error")
		return
	}

	span.AddEvent("compared fake password and default hash for error")
}

// Queries a nonexistent user from the database. Used when BasicAuthValidator is provided an invalid UUID.
func fakeDBQuery(ctx context.Context, db *gorm.DB) {
	ctx, span := tracer.Start(ctx, "fakeDBQuery")
	defer span.End()

	db = db.WithContext(ctx)

	fakeID := uuid.New()
	_, err := models.ByID[models.Auth](ctx, db, fakeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return
		}

		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to make fake db query")
		return
	}

	span.AddEvent("completed database query for fake id")
}

// Validates a basic auth against the database
func (h *Handler) BasicAuthValidator(rawID, token string, c echo.Context) (bool, error) {
	ctx, span := tracer.Start(c.Request().Context(), "BasicAuthValidator")
	defer span.End()

	db := h.DB.WithContext(ctx)

	span.SetAttributes(
		attribute.String("id.raw", rawID),
	)

	span.AddEvent("parsing rawID as uuid")

	id, err := uuid.Parse(rawID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to parse rawID as a uuid")
		// Waste time for invalid UUID
		fakeDBQuery(ctx, db)
		fakePasswordHash(ctx)
		return false, nil
	}

	span.SetAttributes(attribute.String("id.parsed", id.String()))

	span.AddEvent("getting auth by id")
	auth, err := models.ByID[models.Auth](ctx, db, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "db error when search for api key id")

		fakePasswordHash(ctx)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// ok because Ok > Error
			span.SetStatus(codes.Ok, "api key not found")
			return false, nil
		}

		return false, response.InternalServerError
	}

	span.SetAttributes(
		attribute.String("note", auth.Note),
		attribute.Bool("active.valid", auth.Active.Valid),
		attribute.Bool("active.value", auth.Active.V),
	)

	span.AddEvent("checking hash")
	comparison, oldParams, err := argon2id.CheckHash(token, auth.Token)
	// All expensive ops have been performed that may result in a forbidden
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to check token")
		return false, response.InternalServerError
	}

	if !auth.Active.Valid || !auth.Active.V {
		span.AddEvent("auth is not active")
		return false, nil
	}

	if !reflect.DeepEqual(oldParams, argon2id.DefaultParams) {
		span.AddEvent("updating auth with the new params")
		newHash, err := argon2id.CreateHash(token, argon2id.DefaultParams)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to create new hash for token")
			return false, response.InternalServerError
		}

		auth.Token = newHash

		span.AddEvent("saving new auth to the database")
		err = db.Save(auth).Error
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to save new hash to the database")
			return false, response.InternalServerError
		}
	}

	if comparison {
		span.AddEvent("successful login attempt")
		c.Set("auth", auth)
	} else {
		span.AddEvent("failed login attempt")
	}

	return comparison, nil
}
