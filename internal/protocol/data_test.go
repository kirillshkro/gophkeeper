// Tests for the CRUD data payload validation (protocol package,
// tests-specs section 3.2, ValidateData).
//
// Every error is checked through errors.Is against the sentinel base
// (protocol.ErrInvalidPayload, data.ErrUnknownType, data.ErrEmptyData,
// data.ErrInvalidField) and, for payload problems, through errors.As against
// the FieldError context, so the offending field and the reason are
// first-class values, not string fragments.
package protocol

import (
	"errors"
	"testing"
	"time"

	"github.com/kirillshkro/gophkeeper/internal/data"
)

// testOwnerID is the owner identifier used by every fixture of this file.
const testOwnerID = "owner"

// dataTestCase is a single table case for the ValidateData suite.
//
// wantErr is nil or the sentinel the returned error must match through
// errors.Is; wantField and wantReason are checked only when the error
// carries the FieldError context (the data sentinels do not).
type dataTestCase struct {
	name       string
	request    *DataRequest
	wantErr    error
	wantField  string
	wantReason string
}

// runValidateDataCases runs the table cases of ValidateData: a nil
// expectation asserts a nil error; an error is checked against the sentinel
// base and, when the context is available, against the FieldError field and
// reason.
func runValidateDataCases(t *testing.T, cases []dataTestCase) {
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateData(tt.request)
			if (err == nil) != (tt.wantErr == nil) {
				t.Fatalf("ValidateData: got err=%v, want err=%v", err, tt.wantErr)
			}
			if err == nil {
				return
			}
			// (a) the sentinel base.
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateData = %q, want errors.Is(%q)", err, tt.wantErr)
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

// newText returns a text item with a random id, the given owner, content
// and version.
func newText(t *testing.T, content string, version int) *data.Text {
	it, err := data.NewText(content, data.Metadata{})
	if err != nil {
		t.Fatalf("data.NewText(%q): %v", content, err)
	}
	x := it.(*data.Text)
	x.SetOwnerID(testOwnerID)
	if version > 0 {
		x.SetVersion(version)
	}
	return x
}

// newCredential returns a credential item with a random id, the given
// owner, login, password and version.
func newCredential(t *testing.T, login, password string, version int) *data.Credential {
	it, err := data.NewCredential(login, password)
	if err != nil {
		t.Fatalf("data.NewCredential(%q, %q): %v", login, password, err)
	}
	x := it.(*data.Credential)
	x.SetOwnerID(testOwnerID)
	if version > 0 {
		x.SetVersion(version)
	}
	return x
}

// unknownItem implements data.Item with an unsupported ItemType.
//
// It exercises the unknown-type branch of ValidateData (tests-specs 3.2
// #2): data.ValidateType rejects the type before the content is checked.
type unknownItem struct {
	id        string
	ownerID   string
	metadata  data.Metadata
	version   int
	createdAt time.Time
	updatedAt time.Time
}

func (i *unknownItem) ID() string { return i.id }
func (i *unknownItem) Type() data.ItemType {
	return data.ItemType("unsupported")
}
func (i *unknownItem) OwnerID() string         { return i.ownerID }
func (i *unknownItem) Metadata() data.Metadata { return i.metadata }
func (i *unknownItem) Version() int            { return i.version }
func (i *unknownItem) CreatedAt() time.Time    { return i.createdAt }
func (i *unknownItem) UpdatedAt() time.Time    { return i.updatedAt }

// TestProtocol_ValidateData covers the valid requests and the payload-level
// checks of ValidateData (tests-specs 3.2 #1 and #5): a valid create and
// delete are accepted; a nil request, a missing owner id, an unknown action,
// a nil item for create and a mismatching owner id are rejected with a
// FieldError naming the offending field.
func TestProtocol_ValidateData(t *testing.T) {
	t.Helper()

	text := newText(t, "hello world", 1)
	text.SetMetadata(data.Metadata{Website: "example.com", Extra: map[string]string{"note": "todo"}})

	cases := []dataTestCase{
		{
			name:    "valid create: text item with metadata",
			request: &DataRequest{Action: ActionCreate, OwnerID: testOwnerID, Item: text},
		},
		{
			name:    "valid delete: item is not required",
			request: &DataRequest{Action: ActionDelete, OwnerID: testOwnerID, ID: "item-1"},
		},
		{
			name:       "nil request",
			request:    nil,
			wantErr:    ErrInvalidPayload,
			wantField:  "request",
			wantReason: "must not be nil",
		},
		{
			name:       "whitespace-only owner id",
			request:    &DataRequest{Action: ActionCreate, OwnerID: " \t"},
			wantErr:    ErrInvalidPayload,
			wantField:  "owner_id",
			wantReason: "must not be empty",
		},
		{
			name:       "unknown action",
			request:    &DataRequest{Action: Action("bogus"), OwnerID: testOwnerID},
			wantErr:    ErrInvalidPayload,
			wantField:  "action",
			wantReason: "must be create, update or delete",
		},
		{
			name:       "create with nil item",
			request:    &DataRequest{Action: ActionCreate, OwnerID: testOwnerID},
			wantErr:    ErrInvalidPayload,
			wantField:  "item",
			wantReason: "must not be nil",
		},
		{
			name:       "owner id does not match item owner",
			request:    &DataRequest{Action: ActionCreate, OwnerID: "other", Item: text},
			wantErr:    ErrInvalidPayload,
			wantField:  "owner_id",
			wantReason: "does not match item owner",
		},
		{
			name:       "delete without item id",
			request:    &DataRequest{Action: ActionDelete, OwnerID: testOwnerID},
			wantErr:    ErrInvalidPayload,
			wantField:  "id",
			wantReason: "must not be empty for delete",
		},
	}

	runValidateDataCases(t, cases)
}

// TestProtocol_ValidateData_Item covers the item-level checks (tests-specs
// 3.2 #2 and #3): an unsupported item type is rejected with
// data.ErrUnknownType, empty content with data.ErrEmptyData and broken
// metadata with data.ErrInvalidField.
func TestProtocol_ValidateData_Item(t *testing.T) {
	t.Helper()

	emptyText := newText(t, "placeholder", 1)
	emptyText.Content = ""
	emptyPw := newCredential(t, "example", "secret", 1)
	emptyPw.Password = ""
	emptyMeta := newText(t, "hello world", 1)
	emptyMeta.SetMetadata(data.Metadata{OPTs: []string{"ok", ""}})

	cases := []dataTestCase{
		{
			name: "unknown item type",
			request: &DataRequest{
				Action:  ActionCreate,
				OwnerID: testOwnerID,
				Item:    &unknownItem{id: "item-1", ownerID: testOwnerID, version: 1},
			},
			wantErr: data.ErrUnknownType,
		},
		{
			name:    "empty text content",
			request: &DataRequest{Action: ActionCreate, OwnerID: testOwnerID, Item: emptyText},
			wantErr: data.ErrEmptyData,
		},
		{
			name:    "empty credential password",
			request: &DataRequest{Action: ActionCreate, OwnerID: testOwnerID, Item: emptyPw},
			wantErr: data.ErrEmptyData,
		},
		{
			name:    "empty otp code in metadata",
			request: &DataRequest{Action: ActionCreate, OwnerID: testOwnerID, Item: emptyMeta},
			wantErr: data.ErrInvalidField,
		},
	}

	runValidateDataCases(t, cases)
}

// TestProtocol_ValidateData_Update covers the update-specific checks
// (tests-specs 3.2 #4): a valid update (item version higher than the base
// version) is accepted; a missing id, a zero base version and a stale item
// version are rejected with a FieldError naming the field.
func TestProtocol_ValidateData_Update(t *testing.T) {
	t.Helper()

	item := newText(t, "hello world", 2)
	id := item.ID()

	cases := []dataTestCase{
		{
			name:    "valid update: item version higher than base version",
			request: &DataRequest{Action: ActionUpdate, ID: id, OwnerID: testOwnerID, Version: 1, Item: item},
		},
		{
			name:       "update without item id",
			request:    &DataRequest{Action: ActionUpdate, OwnerID: testOwnerID, Version: 1, Item: item},
			wantErr:    ErrInvalidPayload,
			wantField:  "id",
			wantReason: "must not be empty for update",
		},
		{
			name:       "update with zero base version",
			request:    &DataRequest{Action: ActionUpdate, ID: id, OwnerID: testOwnerID, Version: 0, Item: item},
			wantErr:    ErrInvalidPayload,
			wantField:  "version",
			wantReason: "must be positive for update",
		},
		{
			name:       "stale update: item version equal to base version",
			request:    &DataRequest{Action: ActionUpdate, ID: id, OwnerID: testOwnerID, Version: 2, Item: item},
			wantErr:    ErrInvalidPayload,
			wantField:  "version",
			wantReason: "item version must be higher than the base version",
		},
		{
			name:       "stale update: item version lower than base version",
			request:    &DataRequest{Action: ActionUpdate, ID: id, OwnerID: testOwnerID, Version: 5, Item: item},
			wantErr:    ErrInvalidPayload,
			wantField:  "version",
			wantReason: "item version must be higher than the base version",
		},
	}

	runValidateDataCases(t, cases)
}
