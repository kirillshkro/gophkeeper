// JSON error and extensibility tests for the protocol payloads (tests-specs
// section 3.3#7: invalid JSON — ErrInvalidPayload; 3.3#8 and 3.5: unknown fields
// ignored).
package protocol

import (
	"errors"
	"testing"

	"github.com/kirillshkro/gophkeeper/internal/data"
)

func TestProtocol_InvalidJson_Unmarshal(t *testing.T) {
	// Empty and nil bytes are rejected with the bare sentinel.
	if err := Unmarshal([]byte(""), &LoginRequest{}); err != ErrInvalidPayload {
		t.Fatalf("Unmarshal(empty): %v, want %v", err, ErrInvalidPayload)
	}
	if err := Unmarshal(nil, &LoginRequest{}); err != ErrInvalidPayload {
		t.Fatalf("Unmarshal(nil): %v, want %v", err, ErrInvalidPayload)
	}

	tests := []struct {
		name   string
		bytes  []byte
		target any
	}{
		{"garbage", []byte("not json at all"), &LoginRequest{}},
		{"truncated object", []byte(`{"login": "alice"`), &LoginRequest{}},
		{"array instead of object", []byte(`["alice", "s3cret"]`), &LoginRequest{}},
		{"null item in change", []byte(`{"item": null, "action": "create", "version": 1, "timestamp": "2024-01-15T12:30:45Z"}`), &Change{}},
		{"unknown item type", []byte(`{"action": "create", "owner_id": "owner-1", "item": {"type": "unknown"}, "version": 1}`), &DataRequest{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unmarshal(tt.bytes, tt.target)
			if !errors.Is(err, ErrInvalidPayload) {
				t.Fatalf("%s: got %v, want ErrInvalidPayload", tt.name, err)
			}
			var fe *FieldError
			if !errors.As(err, &fe) || fe.Field == "" {
				t.Fatalf("%s: got %q, want FieldError context", tt.name, err.Error())
			}
		})
	}

	// Marshal rejects an unencodable payload with ErrInvalidPayload.
	if _, err := Marshal(Change{}); !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("Marshal(Change{}): %v, want ErrInvalidPayload", err)
	}
	if _, err := Marshal(nil); !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("Marshal(nil): %v, want ErrInvalidPayload", err)
	}
}

// TestProtocol_UnknownFields_Ignored — 3.3#8 and 3.5: unknown fields must
// not prevent decoding (extensibility); the known fields stay intact.
func TestProtocol_UnknownFields_Ignored(t *testing.T) {
	t.Run("login request", func(t *testing.T) {
		b := []byte(`{"login":"alice","password":"s3cret","futureFeature":"x"}`)
		var req LoginRequest
		if err := Unmarshal(b, &req); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if req.Login != "alice" || req.Password != "s3cret" {
			t.Fatalf("known fields lost: %v", req)
		}
	})
	t.Run("change with unknown fields in item and on top", func(t *testing.T) {
		b := []byte(`{"item":{"type":"text","id":"text-1","owner_id":"owner-1","version":3,
			"created_at":"2024-01-15T11:30:45Z","updated_at":"2024-01-15T12:30:45Z",
			"text":{"content":"hello"},"legacy":{"old":"value"}},"action":"create","version":3,
			"timestamp":"2024-01-15T12:30:45Z","legacyTop":true}`)
		var ch Change
		if err := Unmarshal(b, &ch); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if ch.Action != ActionCreate || ch.Version != 3 {
			t.Fatalf("change fields lost: %v", ch)
		}
		txt, ok := ch.Item.(*data.Text)
		if !ok || txt.Content != "hello" || txt.ID() != "text-1" {
			t.Fatalf("item lost: %q", ch.Item)
		}
	})
	t.Run("minimal old payload", func(t *testing.T) {
		b := []byte(`{"item":{"type":"text","id":"text-1","owner_id":"owner-1","version":1,
			"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z",
			"text":{"content":"hi"}},"action":"create","version":1,"timestamp":"2024-01-01T00:00:00Z"}`)
		var ch Change
		if err := Unmarshal(b, &ch); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		txt, ok := ch.Item.(*data.Text)
		if !ok || txt.Content != "hi" {
			t.Fatalf("item lost: %q", ch.Item)
		}
	})
	t.Run("sync request", func(t *testing.T) {
		b := []byte(`{"owner_id":"owner-1","base_version":0,"changes":[],"newField":"x"}`)
		var req SyncRequest
		if err := Unmarshal(b, &req); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if req.OwnerID != "owner-1" || len(req.Changes) != 0 {
			t.Fatalf("request lost: %v", req)
		}
	})
}

// TestProtocol_FieldError_ErrorFormat — the message layout of FieldError is
// stable: "invalid payload: <field>: <reason>", the middle part is dropped
// when Field is empty, and the error always reports ErrInvalidPayload via
// errors.Is.
func TestProtocol_FieldError_ErrorFormat(t *testing.T) {
	tests := []struct {
		name string
		err  *FieldError
		want string
	}{
		{"with field", &FieldError{Field: "login", Reason: "must not be empty"}, "invalid payload: login: must not be empty"},
		{"empty field", &FieldError{Reason: "cannot unmarshal JSON"}, "invalid payload: cannot unmarshal JSON"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("FieldError.Error() = %q, want %q", got, tt.want)
			}
			if !errors.Is(tt.err, ErrInvalidPayload) {
				t.Fatalf("FieldError is not ErrInvalidPayload: %v", tt.err)
			}
		})
	}
}
