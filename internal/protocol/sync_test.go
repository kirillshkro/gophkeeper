// Tests for the synchronization payload validation (protocol package,
// tests-specs section 3.2, ValidateSync).
//
// Every error is checked through errors.Is against the sentinel base
// (protocol.ErrInvalidPayload, data.ErrUnknownType, data.ErrEmptyData) and,
// for payload problems, through errors.As against the FieldError context, so
// the offending field and the reason are first-class values.
package protocol

import (
	"errors"
	"testing"
	"time"

	"github.com/kirillshkro/gophkeeper/internal/data"
)

// syncTestCase is a single table case for the ValidateSync suite.
//
// wantErr is nil or the sentinel the returned error must match through
// errors.Is; wantField and wantReason are checked only when the error carries
// the FieldError context (the data sentinels do not).
type syncTestCase struct {
	name       string
	request    *SyncRequest
	wantErr    error
	wantField  string
	wantReason string
}

// runValidateSyncCases runs the table cases of ValidateSync.
//
// A nil expectation asserts a nil error; an error is checked against the
// sentinel base and, when the context is available, against the FieldError
// field and reason.
func runValidateSyncCases(t *testing.T, cases []syncTestCase) {
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSync(tt.request)
			if (err == nil) != (tt.wantErr == nil) {
				t.Fatalf("ValidateSync: got err=%v, want err=%v", err, tt.wantErr)
			}
			if err == nil {
				return
			}
			// (a) the sentinel base.
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateSync = %q, want errors.Is(%q)", err, tt.wantErr)
			}
			// (b) payload problems expose the FieldError context.
			if tt.wantField == "" {
				return
			}
			var fe *FieldError
			if !errors.As(err, &fe) {
				t.Fatalf("error %q does not expose the FieldError context", err)
			}
			if fe.Field != tt.wantField {
				t.Fatalf("fe.Field = %q, want %q (err: %v)", fe.Field, tt.wantField, err)
			}
			if fe.Reason != tt.wantReason {
				t.Fatalf("fe.Reason = %q, want %q (err: %v)", fe.Reason, tt.wantReason, err)
			}
		})
	}
}

// newChange builds a well-formed change for the given action and version using
// a text item owned by testOwnerID.
func newChange(t *testing.T, action Action, version int) Change {
	it := newText(t, "hello world", version)
	return Change{
		Item:      it,
		Action:    action,
		Version:   version,
		Timestamp: time.Now().UTC(),
	}
}

// newChangeOwner builds a well-formed change whose item is owned by other.
func newChangeOwner(t *testing.T, other string, action Action, version int) Change {
	it := newText(t, "hello world", version)
	it.SetOwnerID(other)
	return Change{
		Item:      it,
		Action:    action,
		Version:   version,
		Timestamp: time.Now().UTC(),
	}
}

// TestProtocol_ValidateSync covers every scenario from tests-specs 3.2 for
// ValidateSync: valid changes are accepted, an empty changes list is a no-op,
// an unknown action or a non-positive version is rejected, and a nil request
// is rejected. Additional cases cover an empty owner identifier, a nil item, an
// unsupported item type, empty item content, a mismatching item owner, and an
// invalid change at a non-zero index.
func TestProtocol_ValidateSync(t *testing.T) {
	t.Helper()

	// Text item whose content was cleared (empty-content case).
	empty := newText(t, "placeholder", 1)
	empty.Content = ""

	cases := []syncTestCase{
		// 1. valid changes.
		{
			name: "valid single change",
			request: &SyncRequest{
				OwnerID: testOwnerID,
				Changes: []Change{newChange(t, ActionCreate, 1)},
			},
		},
		{
			name: "valid multi-change request",
			request: &SyncRequest{
				OwnerID: testOwnerID,
				Changes: []Change{
					newChange(t, ActionCreate, 1),
					newChange(t, ActionUpdate, 2),
					newChange(t, ActionDelete, 3),
				},
			},
		},
		// 2. empty changes (no-op).
		{
			name:    "empty changes list is a no-op",
			request: &SyncRequest{OwnerID: testOwnerID},
		},
		// 4. nil request.
		{
			name:       "nil request",
			request:    nil,
			wantErr:    ErrInvalidPayload,
			wantField:  "request",
			wantReason: "must not be nil",
		},
		// 3. unknown action.
		{
			name: "unknown action in change",
			request: &SyncRequest{
				OwnerID: testOwnerID,
				Changes: []Change{{Action: Action("bogus")}},
			},
			wantErr:    ErrInvalidPayload,
			wantField:  "changes[0].action",
			wantReason: "must be create, update or delete",
		},
		{
			name: "missing action in change",
			request: &SyncRequest{
				OwnerID: testOwnerID,
				Changes: []Change{{}},
			},
			wantErr:    ErrInvalidPayload,
			wantField:  "changes[0].action",
			wantReason: "must be create, update or delete",
		},
		// 3. bad version.
		{
			name: "zero version in change",
			request: &SyncRequest{
				OwnerID: testOwnerID,
				Changes: []Change{newChange(t, ActionCreate, 0)},
			},
			wantErr:    ErrInvalidPayload,
			wantField:  "changes[0].version",
			wantReason: "must be positive",
		},
		{
			name: "negative version in change",
			request: &SyncRequest{
				OwnerID: testOwnerID,
				Changes: []Change{newChange(t, ActionCreate, -1)},
			},
			wantErr:    ErrInvalidPayload,
			wantField:  "changes[0].version",
			wantReason: "must be positive",
		},
		// additional: owner.
		{
			name:       "empty owner identifier",
			request:    &SyncRequest{},
			wantErr:    ErrInvalidPayload,
			wantField:  "owner_id",
			wantReason: "must not be empty",
		},
		{
			name:       "whitespace-only owner identifier",
			request:    &SyncRequest{OwnerID: " \t "},
			wantErr:    ErrInvalidPayload,
			wantField:  "owner_id",
			wantReason: "must not be empty",
		},
		// additional: nil item.
		{
			name: "nil item in change",
			request: &SyncRequest{
				OwnerID: testOwnerID,
				Changes: []Change{{Action: ActionCreate, Version: 1}},
			},
			wantErr:    ErrInvalidPayload,
			wantField:  "changes[0].item",
			wantReason: "must not be nil",
		},
		// additional: unsupported item type.
		{
			name: "unsupported item type in change",
			request: &SyncRequest{
				OwnerID: testOwnerID,
				Changes: []Change{
					{
						Action:    ActionCreate,
						Version:   1,
						Item:      &unknownItem{id: "item-1", ownerID: testOwnerID, version: 1},
						Timestamp: time.Now().UTC(),
					},
				},
			},
			wantErr: data.ErrUnknownType,
		},
		// additional: empty item content.
		{
			name: "empty item content in change",
			request: &SyncRequest{
				OwnerID: testOwnerID,
				Changes: []Change{
					{
						Action:    ActionCreate,
						Version:   1,
						Item:      empty,
						Timestamp: time.Now().UTC(),
					},
				},
			},
			wantErr: data.ErrEmptyData,
		},
		// additional: owner mismatch.
		{
			name: "item owner does not match request owner",
			request: &SyncRequest{
				OwnerID: testOwnerID,
				Changes: []Change{newChangeOwner(t, "someone-else", ActionCreate, 1)},
			},
			wantErr:    ErrInvalidPayload,
			wantField:  "changes[0].item.owner_id",
			wantReason: "does not match the request owner",
		},
		// additional: error at a non-zero index.
		{
			name: "invalid action in the second change",
			request: &SyncRequest{
				OwnerID: testOwnerID,
				Changes: []Change{
					newChange(t, ActionCreate, 1),
					{Action: Action("bogus")},
				},
			},
			wantErr:    ErrInvalidPayload,
			wantField:  "changes[1].action",
			wantReason: "must be create, update or delete",
		},
	}

	runValidateSyncCases(t, cases)
}
