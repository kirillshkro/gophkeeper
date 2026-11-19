// Package-level definitions for the synchronization payload and its
// validation (protocol contract, tests-specs section 3.2, ValidateSync).
package protocol

import (
	"fmt"
	"strings"
	"time"

	"github.com/kirillshkro/gophkeeper/internal/data"
)

// Change is one element of the incremental synchronization delta.
//
// It flows in both directions of the protocol: the client sends changes to the
// server (SyncUp) to publish local edits, and the server sends changes to the
// client (SyncDown) so the client can apply the server-side edits. Every change
// pairs an operation (create/update/delete) with the item it affects, the item
// version the change results in, and the moment the change occurred (UTC) —
// that timestamp is what a deterministic conflict resolution is built from.
type Change struct {
	// Item is the item payload (with metadata) the change applies to.
	//
	// It is always present: even a delete change carries a marker item so the
	// receiver knows which item the operation is about.
	Item data.Item `json:"item"`

	// Action is the operation the change encodes.
	Action Action `json:"action"`

	// Version is the item version the change results in.
	//
	// It is positive for every operation: for create/update it is the version
	// the item moves to; for delete it is the version that gets invalidated.
	Version int `json:"version"`

	// Timestamp is the change time in UTC, used to order changes and break
	// conflicts deterministically.
	Timestamp time.Time `json:"timestamp"`
}

// SyncRequest is the payload the client sends to publish local changes (SyncUp).
//
// It names the owner of the changes, the base version the client has already
// synchronized (so the server can reply with only the delta after it), and the
// list of local changes to publish. An empty changes list is a legal no-op
// (nothing to publish).
type SyncRequest struct {
	// OwnerID is the identifier of the owner whose changes are being published.
	OwnerID string `json:"owner_id"`

	// BaseVersion is the last version the client successfully synchronized.
	//
	// It is 0 on the very first synchronization and is not validated by
	// ValidateSync (it is only passed through to the server).
	BaseVersion int `json:"base_version"`

	// Changes is the list of local changes to publish.
	Changes []Change `json:"changes"`
}

// SyncResponse is the payload the server sends back after a synchronization
// (SyncDown).
//
// It names the owner and returns, after the requested base version, the list
// of server-side changes the client must apply together with the next base
// version (the latest version the server now holds) to use as the base on the
// next synchronization.
type SyncResponse struct {
	// OwnerID is the identifier of the owner the changes belong to.
	OwnerID string `json:"owner_id"`

	// Changes is the list of server-side changes to apply.
	Changes []Change `json:"changes"`

	// NextBaseVersion is the version the client should use as the base on the
	// next synchronization.
	NextBaseVersion int `json:"next_base_version"`
}

// ValidateSync reports whether req is a well-formed synchronization request.
//
// It returns nil when the owner identifier is present and every change in the
// delta is well-formed: it carries a known action, a positive version, a non-
// nil item, and the item is a valid payload owned by the requester. A nil
// request, an empty or whitespace-only owner identifier, or any single invalid
// change yields a FieldError wrapping ErrInvalidPayload and naming the
// offending change index and field. Unsupported item types and empty item
// content are detected as the data.ErrUnknownType and data.ErrEmptyData sentinels
// of the data package. An empty changes list is accepted as a no-op.
func ValidateSync(req *SyncRequest) error {
	if req == nil {
		return fieldError("request", "must not be nil")
	}
	if strings.TrimSpace(req.OwnerID) == "" {
		return fieldError("owner_id", "must not be empty")
	}
	for i, ch := range req.Changes {
		if err := validateChange(i, req.OwnerID, ch); err != nil {
			return err
		}
	}
	return nil
}

// validateChange checks a single change in the delta.
//
// It returns nil when the action is known, the version is positive, the item is
// non-nil and a valid payload, and the item's owner matches the requester's.
// Every violation yields a FieldError naming the change index and the
// offending field.
func validateChange(idx int, ownerID string, ch Change) error {
	if !ch.Action.IsKnown() {
		return fieldError(changeField("action", idx), "must be create, update or delete")
	}
	if ch.Version < 1 {
		return fieldError(changeField("version", idx), "must be positive")
	}
	if ch.Item == nil {
		return fieldError(changeField("item", idx), "must not be nil")
	}
	if err := validateDataItem(ch.Item); err != nil {
		return err
	}
	if ch.Item.OwnerID() != ownerID {
		return fieldError(changeField("item.owner_id", idx), "does not match the request owner")
	}
	return nil
}

// changeField builds the fully-qualified name of a change field, including the
// change index so that messages identify the offending element unambiguously.
func changeField(field string, idx int) string {
	return fmt.Sprintf("changes[%d].%s", idx, field)
}
