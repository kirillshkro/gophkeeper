// JSON round-trip tests for the protocol payloads (tests-specs sections
// 3.3 and 3.5): marshal/unmarshal — identity, invalid JSON — ErrInvalidPayload,
// unknown fields ignored.
package protocol

import (
	"reflect"
	"testing"
	"time"

	"github.com/kirillshkro/gophkeeper/internal/data"
)

// fixedTime is a deterministic UTC instant for the tests.
var fixedTime = time.Date(2024, 1, 15, 12, 30, 45, 0, time.UTC)

// baseMeta is the common metadata for the item tests.
var baseMeta = data.Metadata{
	Website: "example.com",
	Person:  "Jane",
	Bank:    "Bank One",
	OPTs:    []string{"abc-123"},
	Extra:   map[string]string{"tag": "demo"},
}

// itemText returns a complete text item for the tests.
func itemText(t *testing.T) data.Item {
	t.Helper()
	item, err := data.Empty(data.TypeText)
	if err != nil {
		t.Fatalf("itemText: %v", err)
	}
	txt := item.(*data.Text)
	txt.SetID("text-1")
	txt.SetOwnerID("owner-1")
	txt.SetVersion(3)
	txt.SetCreatedAt(fixedTime.Add(-time.Hour))
	txt.SetUpdatedAt(fixedTime)
	txt.SetMetadata(baseMeta)
	txt.Content = "hello, world"
	return item
}

// itemCredential returns a complete credential item for the tests.
func itemCredential(t *testing.T) data.Item {
	t.Helper()
	item, err := data.Empty(data.TypeCredential)
	if err != nil {
		t.Fatalf("itemCredential: %v", err)
	}
	cred := item.(*data.Credential)
	cred.SetID("cred-1")
	cred.SetOwnerID("owner-1")
	cred.SetVersion(7)
	cred.SetCreatedAt(fixedTime.Add(-2 * time.Hour))
	cred.SetUpdatedAt(fixedTime)
	cred.SetMetadata(baseMeta)
	cred.Login = "user@site.com"
	cred.Password = "enc-0001"
	return item
}

// itemBinary returns a complete binary item for the tests.
func itemBinary(t *testing.T) data.Item {
	t.Helper()
	item, err := data.Empty(data.TypeBinary)
	if err != nil {
		t.Fatalf("itemBinary: %v", err)
	}
	bin := item.(*data.Binary)
	bin.SetID("bin-1")
	bin.SetOwnerID("owner-1")
	bin.SetVersion(2)
	bin.SetCreatedAt(fixedTime.Add(-3 * time.Hour))
	bin.SetUpdatedAt(fixedTime)
	bin.SetMetadata(baseMeta)
	bin.Data = []byte{0x00, 0x01, 0x02, 0xff, 0xfe}
	return item
}

// itemCard returns a complete card item for the tests.
func itemCard(t *testing.T) data.Item {
	t.Helper()
	item, err := data.Empty(data.TypeCard)
	if err != nil {
		t.Fatalf("itemCard: %v", err)
	}
	card := item.(*data.Card)
	card.SetID("card-1")
	card.SetOwnerID("owner-1")
	card.SetVersion(5)
	card.SetCreatedAt(fixedTime.Add(-4 * time.Hour))
	card.SetUpdatedAt(fixedTime)
	card.SetMetadata(baseMeta)
	card.Number = "4111111111111111"
	card.Expiry = "12/25"
	card.CVV = "123"
	card.Holder = "John Doe"
	return item
}

// roundTrip checks that Marshal and Unmarshal are inverse operations for the
// payload v: the bytes decoded into the same type are deep-equal to the
// original payload.
func roundTrip(t *testing.T, name string, v any) {
	b, err := Marshal(v)
	if err != nil {
		t.Fatalf("%s: Marshal: %v", name, err)
	}
	tv := reflect.New(reflect.TypeOf(v))
	if err := Unmarshal(b, tv.Interface()); err != nil {
		t.Fatalf("%s: Unmarshal: %v", name, err)
	}
	// tv.Interface() is a pointer; compare the underlying values instead,
	// since DeepEqual requires the same dynamic type on both sides.
	out := tv.Elem().Interface()
	if !reflect.DeepEqual(v, out) {
		t.Fatalf("%s: round trip not identity:\n  got  %v\n  want %v", name, out, v)
	}
}

// TestProtocol_RegisterRequest_MarshalRoundTrip — 3.3#1: RegisterRequest
// (login, password, profile) — marshal/unmarshal — identity, both the
// present and the optional absent profile.
func TestProtocol_RegisterRequest_MarshalRoundTrip(t *testing.T) {
	t.Run("with profile", func(t *testing.T) {
		req := RegisterRequest{
			Login:    "alice",
			Password: "s3cret",
			Profile:  Profile{DisplayName: "Alice Example", Email: "alice@example.com"},
		}
		roundTrip(t, "RegisterRequest", req)
	})
	t.Run("without profile", func(t *testing.T) {
		roundTrip(t, "RegisterRequest", RegisterRequest{Login: "bob", Password: "s3cret"})
	})
}

// TestProtocol_LoginRequest_MarshalRoundTrip — 3.3#2: LoginRequest.
func TestProtocol_LoginRequest_MarshalRoundTrip(t *testing.T) {
	roundTrip(t, "LoginRequest", LoginRequest{Login: "alice", Password: "s3cret"})
}

// TestProtocol_RegisterResponse_MarshalRoundTrip — 3.3#1 (response).
func TestProtocol_RegisterResponse_MarshalRoundTrip(t *testing.T) {
	resp := RegisterResponse{Token: Token{
		OwnerID:   "owner-1",
		Expiry:    fixedTime.Add(24 * time.Hour),
		Signature: "sig-0001",
	}}
	roundTrip(t, "RegisterResponse", resp)
}

// TestProtocol_LoginResponse_MarshalRoundTrip — 3.3#2 (response).
func TestProtocol_LoginResponse_MarshalRoundTrip(t *testing.T) {
	resp := LoginResponse{Token: Token{
		OwnerID:   "owner-1",
		Expiry:    fixedTime.Add(24 * time.Hour),
		Signature: "sig-0002",
	}}
	roundTrip(t, "LoginResponse", resp)
}

// TestProtocol_DataRequest_MarshalRoundTrip — 3.3#3: DataRequest (item with
// metadata) — marshal/unmarshal — identity for each known item type and each
// action.
func TestProtocol_DataRequest_MarshalRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		req  DataRequest
	}{
		{"text/create", DataRequest{Action: ActionCreate, OwnerID: "owner-1", Item: itemText(t)}},
		{"credential/create", DataRequest{Action: ActionCreate, OwnerID: "owner-1", Item: itemCredential(t)}},
		{"binary/create", DataRequest{Action: ActionCreate, OwnerID: "owner-1", Item: itemBinary(t)}},
		{"card/create", DataRequest{Action: ActionCreate, OwnerID: "owner-1", Item: itemCard(t)}},
		{"text/update", DataRequest{
			Action:  ActionUpdate,
			ID:      "text-1",
			OwnerID: "owner-1",
			Version: 2,
			Item:    itemText(t),
		}},
		{"text/delete", DataRequest{Action: ActionDelete, ID: "text-1", OwnerID: "owner-1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roundTrip(t, tt.name, tt.req)
		})
	}
}

// TestProtocol_DataResponse_MarshalRoundTrip — 3.3#3 (response):
// DataResponse with an item and without.
func TestProtocol_DataResponse_MarshalRoundTrip(t *testing.T) {
	t.Run("with item", func(t *testing.T) {
		roundTrip(t, "DataResponse", DataResponse{Item: itemText(t)})
	})
	t.Run("without item", func(t *testing.T) {
		roundTrip(t, "DataResponse", DataResponse{})
	})
}
