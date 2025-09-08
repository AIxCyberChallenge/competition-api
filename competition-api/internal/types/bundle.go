package types

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type BundleSubmission struct {
	POVID            Optional[string] `json:"pov_id"             validate:"omitempty" format:"uuid" swaggertype:"string"`
	PatchID          Optional[string] `json:"patch_id"           validate:"omitempty" format:"uuid" swaggertype:"string"`
	SubmittedSARIFID Optional[string] `json:"submitted_sarif_id" validate:"omitempty" format:"uuid" swaggertype:"string"`
	BroadcastSARIFID Optional[string] `json:"broadcast_sarif_id" validate:"omitempty" format:"uuid" swaggertype:"string"`
	// optional plaintext description of the components of the bundle, such as would be found in a pull request description or bug report
	Description Optional[string] `json:"description"        validate:"omitempty"               swaggertype:"string"`
	FreeformID  Optional[string] `json:"freeform_id"        validate:"omitempty" format:"uuid" swaggertype:"string"`
}

type BundleSubmissionParsed struct {
	POVID            Optional[uuid.UUID] `json:"pov_id"             validate:"omitempty" format:"uuid" swaggertype:"string"`
	PatchID          Optional[uuid.UUID] `json:"patch_id"           validate:"omitempty" format:"uuid" swaggertype:"string"`
	SubmittedSARIFID Optional[uuid.UUID] `json:"submitted_sarif_id" validate:"omitempty" format:"uuid" swaggertype:"string"`
	BroadcastSARIFID Optional[uuid.UUID] `json:"broadcast_sarif_id" validate:"omitempty" format:"uuid" swaggertype:"string"`
	Description      Optional[string]    `json:"description"        validate:"omitempty"               swaggertype:"string"`
	FreeformID       Optional[uuid.UUID] `json:"freeform_id"        validate:"omitempty" format:"uuid" swaggertype:"string"`
}

type BundleSubmissionResponseBody struct {
	POVID            *uuid.UUID `json:"pov_id"             validate:"omitempty" format:"uuid" swaggertype:"string"`
	PatchID          *uuid.UUID `json:"patch_id"           validate:"omitempty" format:"uuid" swaggertype:"string"`
	SubmittedSARIFID *uuid.UUID `json:"submitted_sarif_id" validate:"omitempty" format:"uuid" swaggertype:"string"`
	BroadcastSARIFID *uuid.UUID `json:"broadcast_sarif_id" validate:"omitempty" format:"uuid" swaggertype:"string"`
	Description      *string    `json:"description"        validate:"omitempty"               swaggertype:"string"`
	FreeformID       *uuid.UUID `json:"freeform_id"        validate:"omitempty" format:"uuid" swaggertype:"string"`
}

func (b *BundleSubmission) ValidateFieldCount() error {
	countSet := 0
	if b.POVID.Defined && b.POVID.Value != nil {
		countSet++
	}
	if b.BroadcastSARIFID.Defined && b.BroadcastSARIFID.Value != nil {
		countSet++
	}
	if b.Description.Defined && b.Description.Value != nil {
		countSet++
	}
	if b.PatchID.Defined && b.PatchID.Value != nil {
		countSet++
	}
	if b.SubmittedSARIFID.Defined && b.SubmittedSARIFID.Value != nil {
		countSet++
	}
	if b.FreeformID.Defined && b.FreeformID.Value != nil {
		countSet++
	}

	if countSet < 2 {
		return errors.New("must set at least 2 fields")
	}

	return nil
}

func (b *BundleSubmission) Parse() (*BundleSubmissionParsed, error) {
	parseUUID := func(v *string) (*uuid.UUID, error) {
		if v == nil {
			return nil, nil
		}
		id, err := uuid.Parse(*v)
		if err != nil {
			return nil, fmt.Errorf("failed to parse uuid: %w", err)
		}

		return &id, nil
	}
	povID, err := Map(b.POVID, parseUUID)
	if err != nil {
		return nil, err
	}
	patchID, err := Map(b.PatchID, parseUUID)
	if err != nil {
		return nil, err
	}
	submittedSARIFID, err := Map(b.SubmittedSARIFID, parseUUID)
	if err != nil {
		return nil, err
	}
	broadcastSARIFID, err := Map(b.BroadcastSARIFID, parseUUID)
	if err != nil {
		return nil, err
	}
	description := b.Description
	freeformID, err := Map(b.FreeformID, parseUUID)
	if err != nil {
		return nil, err
	}
	return &BundleSubmissionParsed{
		POVID:            povID,
		PatchID:          patchID,
		SubmittedSARIFID: submittedSARIFID,
		BroadcastSARIFID: broadcastSARIFID,
		Description:      description,
		FreeformID:       freeformID,
	}, nil
}

type BundleSubmissionResponse struct {
	BundleID string `json:"bundle_id" validate:"required,uuid_rfc4122"                     format:"uuid"`
	// Schema-compliant submissions will only ever receive the statuses accepted or deadline_exceeded
	Status SubmissionStatus `json:"status"    validate:"required,eq=accepted|eq=deadline_exceeded"`
}

type BundleSubmissionResponseVerbose struct {
	BundleSubmissionResponseBody
	BundleSubmissionResponse
}
