// Package-level definitions for the CRUD data payload and its validation
// (protocol contract, tests-specs section 3.2, ValidateData).
package protocol

import (
	"strings"

	"github.com/kirillshkro/gophkeeper/internal/data"
)

// Action enumerates the CRUD operations supported by the data endpoint.
type Action string

const (
	// ActionCreate creates a new item; the server assigns its identifier.
	ActionCreate Action = "create"

	// ActionUpdate replaces an existing item; the server rejects it with a
	// conflict when the stored copy has a higher version (optimistic
	// concurrency control).
	ActionUpdate Action = "update"

	// ActionDelete removes an item owned by the requesting user.
	ActionDelete Action = "delete"
)

// IsKnown reports whether a is one of the known actions.
func (a Action) IsKnown() bool {
	switch a {
	case ActionCreate, ActionUpdate, ActionDelete:
		return true
	default:
		return false
	}
}

// DataRequest is the payload of the CRUD data endpoint.
//
// It carries the operation and its arguments: the owner identifier, the item
// identifier (required for update/delete; assigned by the server on create),
// the base version the client updates from (optimistic concurrency control)
// and the item payload with metadata for create/update. The protocol version
// is negotiated separately (see IsCompatible) and is not checked by
// ValidateData.
type DataRequest struct {
	// Action is the requested operation.
	Action Action `json:"action"`

	// ID is the item identifier.
	ID string `json:"id"`

	// OwnerID is the identifier of the item owner.
	OwnerID string `json:"owner_id"`

	// Version is the item version the client updates from. Required for
	// update; the item being sent must carry a higher version.
	Version int `json:"version"`

	// Item is the item payload (with metadata). Required for create and
	// update.
	Item data.Item `json:"item"`
}

// ValidateData reports whether req is a well-formed CRUD data request.
//
// It returns nil when the action, owner identifier, item and version fields
// are mutually consistent. A nil request, an empty or mismatching owner
// identifier, an unknown action, a nil item (for create/update), an unsupported
// item type, empty item content, invalid item metadata, a missing item
// identifier or a non-positive or stale version for an update each yield an
// error. Payload problems are FieldError values wrapping ErrInvalidPayload;
// they are detected with errors.Is (the base) and errors.As (the field and
// reason). Unsupported item types and empty content are detected as the
// data.ErrUnknownType and data.ErrEmptyData sentinels of the data package.
func ValidateData(req *DataRequest) error {
	if req == nil {
		return fieldError("request", "must not be nil")
	}
	if strings.TrimSpace(req.OwnerID) == "" {
		return fieldError("owner_id", "must not be empty")
	}
	if !req.Action.IsKnown() {
		return fieldError("action", "must be create, update or delete")
	}

	needsItem := req.Action == ActionCreate || req.Action == ActionUpdate
	if needsItem {
		if req.Item == nil {
			return fieldError("item", "must not be nil")
		}
		if err := validateDataItem(req.Item); err != nil {
			return err
		}
		if req.Item.OwnerID() != req.OwnerID {
			return fieldError("owner_id", "does not match item owner")
		}
	}

	switch req.Action {
	case ActionUpdate:
		if req.ID == "" {
			return fieldError("id", "must not be empty for update")
		}
		if req.Version < 1 {
			return fieldError("version", "must be positive for update")
		}
		if req.Item.Version() <= req.Version {
			return fieldError("version", "item version must be higher than the base version")
		}
	case ActionDelete:
		if req.ID == "" {
			return fieldError("id", "must not be empty for delete")
		}
	}
	return nil
}

// validateDataItem checks a concrete item payload: an unsupported item type
// yields data.ErrUnknownType, empty content yields data.ErrEmptyData and
// invalid metadata yields data.ErrInvalidField.
//
// It repeats the invariants of the data package constructors for items that
// arrive over the wire: even an untrusted or reconstructed serialized form
// must not violate them (a malformed item is rejected at the contract level).
func validateDataItem(item data.Item) error {
	if err := data.ValidateType(item.Type()); err != nil {
		return err
	}
	if err := item.Metadata().Validate(); err != nil {
		return err
	}
	switch v := item.(type) {
	case *data.Text:
		if strings.TrimSpace(v.Content) == "" {
			return data.ErrEmptyData
		}
	case *data.Binary:
		if len(v.Data) == 0 {
			return data.ErrEmptyData
		}
	case *data.Credential:
		if strings.TrimSpace(v.Login) == "" || strings.TrimSpace(v.Password) == "" {
			return data.ErrEmptyData
		}
	case *data.Card:
		if strings.TrimSpace(v.Number) == "" {
			return data.ErrEmptyData
		}
	default:
		return data.ErrUnknownType
	}
	return nil
}
