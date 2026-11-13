// Package data — tests for the Text item type.
//
// This file covers only the Text data type (constructor, metadata, Item base
// entity, validation and the Text branch of the type switches). The other
// item types (Credential, Binary, Card) are left for their own test files.
package data

import (
	"errors"
	"strings"
	"testing"
)

// mustNewText builds a Text item and fails the test if NewText returns an
// error. It returns a concrete *Text so tests can inspect the Content field.
func mustNewText(t *testing.T, content string, meta Metadata) *Text {
	t.Helper()
	item, err := NewText(content, meta)
	if err != nil {
		t.Fatalf("NewText: %v", err)
	}
	return item.(*Text)
}

// TestData_NewText covers the NewText constructor for the spec scenarios:
// valid text (simple, UTF-8, large), empty and whitespace-only text, and
// metadata that fails Validate().
func TestData_NewText(t *testing.T) {
	t.Helper()

	tests := []struct {
		name      string
		content   string
		meta      Metadata
		wantError error
	}{
		{"valid simple text", "hello world", Metadata{}, nil},
		{"valid utf-8 text", "привет — émoji 🚀", Metadata{}, nil},
		{"valid large text", strings.Repeat("a", 250_000) + "done", Metadata{}, nil},
		{"empty content", "", Metadata{}, ErrEmptyData},
		{"whitespace-only content", " \t\n  ", Metadata{}, ErrEmptyData},
		{"empty otp in metadata", "ok", Metadata{OPTs: []string{"", "123"}}, ErrInvalidField},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, got := NewText(tt.content, tt.meta)
			if got != nil {
				if tt.wantError == nil {
					t.Fatalf("NewText: unexpected error %v", got)
					return
				}
				if !errors.Is(got, tt.wantError) {
					t.Fatalf("NewText error %v, want %v", got, tt.wantError)
					return
				}
				return
			}
			if tt.wantError != nil {
				t.Fatalf("NewText: got item, want error %v", tt.wantError)
				return
			}
			text := item.(*Text)
			if text.Content != tt.content {
				t.Fatalf("Content = %q, want %q", text.Content, tt.content)
			}
			if text.Type() != TypeText {
				t.Fatalf("Type = %q, want %q", text.Type(), TypeText)
			}
		})
	}
}

// TestMetadata_Text verifies that a Text item stores and reads metadata
// correctly: empty, arbitrary keys, non-ASCII, very long values, duplicate
// keys (later value wins) and that metadata is deep-copied into the item.
func TestMetadata_Text(t *testing.T) {
	t.Helper()

	t.Run("empty", func(t *testing.T) {
		item := mustNewText(t, "hello", Metadata{})
		if !item.Metadata().IsEmpty() {
			t.Fatalf("expected empty metadata, got %+v", item.Metadata())
		}
	})

	t.Run("arbitrary keys", func(t *testing.T) {
		meta := Metadata{
			Website: "example.com",
			Person:  "Alice",
			Bank:    "TestBank",
			OPTs:    []string{"123456"},
			Extra:   map[string]string{"note": "secret"},
		}
		got := mustNewText(t, "hello", meta).Metadata()
		if got.Website != "example.com" {
			t.Fatalf("Website = %q, want example.com", got.Website)
		}
		if got.Person != "Alice" {
			t.Fatalf("Person = %q, want Alice", got.Person)
		}
		if got.Bank != "TestBank" {
			t.Fatalf("Bank = %q, want TestBank", got.Bank)
		}
		if len(got.OPTs) != 1 || got.OPTs[0] != "123456" {
			t.Fatalf("OPTs = %+v, want [123456]", got.OPTs)
		}
		if got.Get("note") != "secret" {
			t.Fatalf("Get(note) = %q, want secret", got.Get("note"))
		}
	})

	t.Run("non-ascii values", func(t *testing.T) {
		got := mustNewText(t, "hello", Metadata{Website: "пример.рф", Person: "Борис", Bank: "Сбер"}).Metadata()
		if got.Website != "пример.рф" || got.Person != "Борис" || got.Bank != "Сбер" {
			t.Fatalf("non-ascii metadata not preserved: %+v", got)
		}
	})

	t.Run("very long values", func(t *testing.T) {
		long := strings.Repeat("x", 200_000)
		got := mustNewText(t, "hello", Metadata{Extra: map[string]string{"long": long}}).Metadata()
		if got.Get("long") != long {
			t.Fatalf("long value not preserved (len=%d, want %d)", len(got.Get("long")), len(long))
		}
	})

	t.Run("duplicate keys keep latest on update", func(t *testing.T) {
		m := &Metadata{}
		if err := m.Set("tag", "v1"); err != nil {
			t.Fatalf("Set v1: %v", err)
		}
		if err := m.Set("tag", "v2"); err != nil {
			t.Fatalf("Set v2: %v", err)
		}
		if m.Get("tag") != "v2" {
			t.Fatalf("tag = %q, want v2", m.Get("tag"))
		}
	})

	t.Run("metadata is deep copied", func(t *testing.T) {
		extra := map[string]string{"tag": "v1"}
		item := mustNewText(t, "hello", Metadata{Extra: extra})
		extra["tag"] = "changed"
		if item.Metadata().Get("tag") != "v1" {
			t.Fatalf("item metadata must be a deep copy, got %q", item.Metadata().Get("tag"))
		}
	})
}

// TestText_Item verifies the Item interface and the ItemCore base fields
// as exposed through a Text item (type, owner, version and timestamps).
func TestText_Item(t *testing.T) {
	t.Helper()

	t.Run("Type is TypeText", func(t *testing.T) {
		item := mustNewText(t, "hi", Metadata{})
		if item.Type() != TypeText {
			t.Fatalf("Type = %q, want %q", item.Type(), TypeText)
		}
	})

	t.Run("OwnerID empty by default", func(t *testing.T) {
		item := mustNewText(t, "hi", Metadata{})
		if item.OwnerID() != "" {
			t.Fatalf("OwnerID = %q, want empty", item.OwnerID())
		}
	})

	t.Run("OwnerID can be set", func(t *testing.T) {
		item := mustNewText(t, "hi", Metadata{})
		item.SetOwnerID("owner-1")
		if item.OwnerID() != "owner-1" {
			t.Fatalf("OwnerID = %q, want owner-1", item.OwnerID())
		}
	})

	t.Run("Version is 1 at creation", func(t *testing.T) {
		item := mustNewText(t, "hi", Metadata{})
		if item.Version() != 1 {
			t.Fatalf("Version = %d, want 1", item.Version())
		}
	})

	t.Run("Version increases on update", func(t *testing.T) {
		item := mustNewText(t, "hi", Metadata{})
		before := item.Version()
		item.Bump()
		if item.Version() <= before {
			t.Fatalf("Version after bump = %d, want > %d", item.Version(), before)
		}
	})

	t.Run("timestamps set and ordered", func(t *testing.T) {
		item := mustNewText(t, "hi", Metadata{})
		if item.CreatedAt().IsZero() || item.UpdatedAt().IsZero() {
			t.Fatal("CreatedAt or UpdatedAt is zero")
		}
		if item.UpdatedAt().Before(item.CreatedAt()) {
			t.Fatalf("UpdatedAt %s before CreatedAt %s", item.UpdatedAt(), item.CreatedAt())
		}
	})

	t.Run("IDs unique and non-empty", func(t *testing.T) {
		a := mustNewText(t, "hi", Metadata{})
		b := mustNewText(t, "hi", Metadata{})
		if a.ID() == "" {
			t.Fatal("ID is empty")
		}
		if a.ID() == b.ID() {
			t.Fatal("IDs should be unique, got duplicate")
		}
	})
}

// TestText_TypeSwitch covers the Text branch of the package type switches
// (Empty and Summary) so the Text arm of each switch is exercised.
func TestText_TypeSwitch(t *testing.T) {
	t.Helper()

	t.Run("Empty builds a Text item", func(t *testing.T) {
		item, err := Empty(TypeText)
		if err != nil {
			t.Fatalf("Empty: %v", err)
		}
		if _, ok := item.(*Text); !ok {
			t.Fatalf("Empty: got %T, want *Text", item)
		}
	})

	t.Run("Summary renders a Text item", func(t *testing.T) {
		item := mustNewText(t, "hi", Metadata{})
		if s := Summary(item); !strings.Contains(s, "characters") {
			t.Fatalf("Summary = %q, want '... characters'", s)
		}
	})
}
