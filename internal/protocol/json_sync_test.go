// JSON round-trip tests for the synchronization, change and token payloads (tests-specs
// section 3.3: marshal/unmarshal — identity).
package protocol

import (
	"encoding/json"
	"testing"
	"time"
)

func TestProtocol_SyncRequest_MarshalRoundTrip(t *testing.T) {
	t.Run("with changes", func(t *testing.T) {
		req := SyncRequest{
			OwnerID:     "owner-1",
			BaseVersion: 0,
			Changes: []Change{
				{Item: itemText(t), Action: ActionCreate, Version: 3, Timestamp: fixedTime},
				{Item: itemCard(t), Action: ActionUpdate, Version: 6, Timestamp: fixedTime.Add(time.Minute)},
			},
		}
		roundTrip(t, "SyncRequest", req)
	})
	t.Run("empty changes (no-op)", func(t *testing.T) {
		roundTrip(t, "SyncRequest", SyncRequest{OwnerID: "owner-1"})
	})
}

// TestProtocol_SyncResponse_MarshalRoundTrip — 3.3#4 (response).
func TestProtocol_SyncResponse_MarshalRoundTrip(t *testing.T) {
	resp := SyncResponse{
		OwnerID:         "owner-1",
		NextBaseVersion: 4,
		Changes: []Change{
			{Item: itemText(t), Action: ActionCreate, Version: 4, Timestamp: fixedTime},
		},
	}
	roundTrip(t, "SyncResponse", resp)
}

// TestProtocol_Change_MarshalRoundTrip — 3.3#6: Change (item, action, version,
// timestamp) — marshal/unmarshal — identity for each known item type and
// each action.
func TestProtocol_Change_MarshalRoundTrip(t *testing.T) {
	t.Run("text/create", func(t *testing.T) {
		roundTrip(t, "Change", Change{
			Item:      itemText(t),
			Action:    ActionCreate,
			Version:   3,
			Timestamp: fixedTime,
		})
	})
	t.Run("credential/create", func(t *testing.T) {
		roundTrip(t, "Change", Change{
			Item:      itemCredential(t),
			Action:    ActionCreate,
			Version:   8,
			Timestamp: fixedTime,
		})
	})
	t.Run("binary/create", func(t *testing.T) {
		roundTrip(t, "Change", Change{
			Item:      itemBinary(t),
			Action:    ActionCreate,
			Version:   3,
			Timestamp: fixedTime,
		})
	})
	t.Run("card/create", func(t *testing.T) {
		roundTrip(t, "Change", Change{
			Item:      itemCard(t),
			Action:    ActionCreate,
			Version:   6,
			Timestamp: fixedTime,
		})
	})
	t.Run("text/update", func(t *testing.T) {
		roundTrip(t, "Change", Change{
			Item:      itemText(t),
			Action:    ActionUpdate,
			Version:   4,
			Timestamp: fixedTime,
		})
	})
	t.Run("text/delete", func(t *testing.T) {
		roundTrip(t, "Change", Change{
			Item:      itemText(t),
			Action:    ActionDelete,
			Version:   3,
			Timestamp: fixedTime,
		})
	})
}

// TestProtocol_Token_MarshalRoundTrip — 3.3#5: Token — marshal/unmarshal —
// identity, and the token carries only owner_id, expiry and signature (no
// secrets).
func TestProtocol_Token_MarshalRoundTrip(t *testing.T) {
	tok := Token{
		OwnerID:   "owner-1",
		Expiry:    fixedTime.Add(24 * time.Hour),
		Signature: "sig-0001",
	}
	roundTrip(t, "Token", tok)

	b, err := Marshal(tok)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var keys map[string]any
	if err := json.Unmarshal(b, &keys); err != nil {
		t.Fatalf("token keys: %v", err)
	}
	want := map[string]struct{}{"owner_id": {}, "expiry": {}, "signature": {}}
	if len(keys) != len(want) {
		t.Fatalf("token keys: got %v, want exactly %v", keys, want)
	}
	for k := range keys {
		if _, ok := want[k]; !ok {
			t.Fatalf("unexpected token field %q", k)
		}
	}
}

// TestProtocol_InvalidJson_Unmarshal — 3.3#7: invalid JSON is rejected with
// ErrInvalidPayload (errors.Is), with FieldError context (errors.As).
