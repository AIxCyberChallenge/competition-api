package models

import (
	"context"
	"fmt"

	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/config"
)

type Permissions struct {
	CRS                   bool `json:"crs"`
	CompetitionManagement bool `json:"competition_management"`
	JobRunner             bool `json:"job_runner"`
}

type Auth struct {
	Token string // argon2id hash
	Note  string // will be logged nonsensitive
	Model
	Permissions Permissions `gorm:"type:jsonb;serializer:json"`
	Active      datatypes.Null[bool]
}

func (Auth) TableName() string {
	return "auth"
}

func (a Auth) GetID() uuid.UUID {
	return a.ID
}

// Config is the authoritative api keys
//
// 1. Upsert auth data
// 2. Disable keys not currently contained in the config
func LoadAPIKeysFromConfig(ctx context.Context, db *gorm.DB, teams []config.Team) error {
	ctx, span := tracer.Start(ctx, "LoadAPIKeysFromConfig")
	defer span.End()

	db = db.WithContext(ctx)

	keysToUpsert := make([]*Auth, len(teams))
	keysInConfig := make([]uuid.UUID, len(teams))
	for i, team := range teams {
		hash, err := argon2id.CreateHash(team.APIKey.Token, argon2id.DefaultParams)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "error creating hash for api key")
			span.SetAttributes(attribute.String("failedTeam", team.ID))
			return err
		}

		teamID, err := uuid.Parse(team.ID)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "error creating hash for api key")
			span.SetAttributes(attribute.String("failedTeam", team.ID))
			return err
		}

		newModel := Auth{
			Model: Model{
				ID: teamID,
			},
			Token:  hash,
			Note:   team.Note,
			Active: NewNull(team.APIKey.Active),
			Permissions: Permissions{
				CRS:                   team.APIKey.Permissions.CRS,
				CompetitionManagement: team.APIKey.Permissions.CompetitionManagement,
				JobRunner:             team.APIKey.Permissions.JobRunner,
			},
		}

		keysToUpsert[i] = &newModel
		keysInConfig[i] = newModel.ID
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		//nolint:govet // shadow: intentionally shadow ctx and span to avoid using the incorrect one.
		ctx, span := tracer.Start(ctx, "LoadApiKeysFromConfig/Transaction")
		defer span.End()

		tx = tx.WithContext(ctx)

		if len(keysToUpsert) != 0 {
			span.AddEvent("upserting defined auths")
			result := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(keysToUpsert)
			if result.Error != nil {
				span.RecordError(result.Error)
				span.SetStatus(codes.Error, "failed to upsert defined auths")
				return fmt.Errorf("failed to upsert defined auths: %w", result.Error)
			}
			if result.RowsAffected != int64(len(teams)) {
				span.AddEvent("updated rows did not equal configured api key count")
				span.SetAttributes(
					attribute.Int64("rowsAffected", result.RowsAffected),
					attribute.Int64("teams", int64(len(teams))),
				)
			}
		} else {
			span.AddEvent("no defined auths to upsert")
		}

		span.AddEvent("setting all rows not in defined auth inactive")

		result := tx.Model(&Auth{}).
			Where("id NOT IN ?", keysInConfig).
			Updates(&Auth{Active: NewNullFromData(false)})
		if result.Error != nil {
			span.RecordError(result.Error)
			span.SetStatus(codes.Error, "failed to set all rows not in defined auth inactive")
			return fmt.Errorf(
				"failed to set all rows not in defined auth inactive: %w",
				result.Error,
			)
		}

		span.RecordError(nil)
		span.SetStatus(codes.Ok, "updated auths")
		return nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update auth")
		return fmt.Errorf("failed to update auth: %w", err)
	}

	span.RecordError(nil)
	span.SetStatus(codes.Ok, "updated auth")
	return nil
}
