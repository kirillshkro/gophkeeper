// Package data — tests for the Card item entity: metadata accessors,
// the Item base fields and the Card branch of the type switches. The
// mustNewCard and checkErr helpers live in card_test.go (same package).
package data

import (
	"errors"
	"strings"
	"testing"
)

// TestMetadata_Card verifies that a Card item stores and reads metadata
// (spec 3.3): empty, arbitrary keys, non-ASCII values, very long values
// and duplicate keys (later value wins).
func TestMetadata_Card(t *testing.T) {
	t.Helper()

	t.Run("empty", func(t *testing.T) {
		item := mustNewCard(t, "4111111111111111", "12/25")
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
		item := mustNewCard(t, "4111111111111111", "12/25")
		item.SetMetadata(meta)
		got := item.Metadata()
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

	t.Run("non-ASCII values", func(t *testing.T) {
		item := mustNewCard(t, "4111111111111111", "12/25")
		item.SetMetadata(Metadata{
			Person: "Иван Петров",
			Extra:  map[string]string{"tag": "привет 🚀"},
		})
		got := item.Metadata()
		if got.Person != "Иван Петров" {
			t.Fatalf("Person = %q, want non-ASCII value", got.Person)
		}
		if got.Get("tag") != "привет 🚀" {
			t.Fatalf("Get(tag) = %q, want non-ASCII value", got.Get("tag"))
		}
	})

	t.Run("very long values", func(t *testing.T) {
		long := strings.Repeat("x", 10_000)
		item := mustNewCard(t, "4111111111111111", "12/25")
		item.SetMetadata(Metadata{Extra: map[string]string{"long": long}})
		if got := item.Metadata().Get("long"); len(got) != len(long) {
			t.Fatalf("long value not preserved (len=%d, want %d)", len(got), len(long))
		}
	})

	t.Run("duplicate keys keep latest on update", func(t *testing.T) {
		item := mustNewCard(t, "4111111111111111", "12/25")
		m := Metadata{}
		if err := m.Set("tag", "v1"); err != nil {
			t.Fatalf("Set v1: %v", err)
		}
		if err := m.Set("tag", "v2"); err != nil {
			t.Fatalf("Set v2: %v", err)
		}
		item.SetMetadata(m)
		if item.Metadata().Get("tag") != "v2" {
			t.Fatalf("tag = %q, want v2", item.Metadata().Get("tag"))
		}
	})
}

// TestCard_Item verifies the Item interface and the ItemCore base fields
// as exposed through a Card item (spec 3.4).
func TestCard_Item(t *testing.T) {
	t.Helper()

	t.Run("Type is TypeCard", func(t *testing.T) {
		item := mustNewCard(t, "4111111111111111", "12/25")
		if item.Type() != TypeCard {
			t.Fatalf("Type = %q, want %q", item.Type(), TypeCard)
		}
	})

	t.Run("OwnerID empty by default", func(t *testing.T) {
		item := mustNewCard(t, "4111111111111111", "12/25")
		if item.OwnerID() != "" {
			t.Fatalf("OwnerID = %q, want empty", item.OwnerID())
		}
	})

	t.Run("OwnerID can be set", func(t *testing.T) {
		item := mustNewCard(t, "4111111111111111", "12/25")
		item.SetOwnerID("owner-1")
		if item.OwnerID() != "owner-1" {
			t.Fatalf("OwnerID = %q, want owner-1", item.OwnerID())
		}
	})

	t.Run("Version is 1 at creation", func(t *testing.T) {
		item := mustNewCard(t, "4111111111111111", "12/25")
		if item.Version() != 1 {
			t.Fatalf("Version = %d, want 1", item.Version())
		}
	})

	t.Run("Version increases on update", func(t *testing.T) {
		item := mustNewCard(t, "4111111111111111", "12/25")
		before := item.Version()
		item.Bump()
		if item.Version() <= before {
			t.Fatalf("Version after bump = %d, want > %d", item.Version(), before)
		}
	})

	t.Run("timestamps set and ordered", func(t *testing.T) {
		item := mustNewCard(t, "4111111111111111", "12/25")
		if item.CreatedAt().IsZero() || item.UpdatedAt().IsZero() {
			t.Fatal("CreatedAt or UpdatedAt is zero")
		}
		if item.UpdatedAt().Before(item.CreatedAt()) {
			t.Fatalf("UpdatedAt %s before CreatedAt %s", item.UpdatedAt(), item.CreatedAt())
		}
	})

	t.Run("IDs unique and non-empty", func(t *testing.T) {
		a := mustNewCard(t, "4111111111111111", "12/25")
		b := mustNewCard(t, "4111111111111111", "12/25")
		if a.ID() == "" || a.ID() == b.ID() {
			t.Fatal("IDs should be unique and non-empty")
		}
	})
}

// TestCard_TypeSwitch covers the Card branch of the package type switches
// (Empty, Summary, ValidateType) plus the unknown-type fallback (spec 3.5).
func TestCard_TypeSwitch(t *testing.T) {
	t.Helper()

	t.Run("Empty(TypeCard) builds a Card", func(t *testing.T) {
		item, err := Empty(TypeCard)
		if err != nil {
			t.Fatalf("Empty: %v", err)
		}
		if _, ok := item.(*Card); !ok {
			t.Fatalf("Empty: got %T, want *Card", item)
		}
	})

	t.Run("Empty(unknown type) falls back to ErrUnknownType", func(t *testing.T) {
		_, err := Empty("bogus")
		if !errors.Is(err, ErrUnknownType) {
			t.Fatalf("Empty: %v, want ErrUnknownType", err)
		}
	})

	t.Run("Summary shows only the last four digits", func(t *testing.T) {
		card := mustNewCard(t, "4111111111111111", "12/25")
		s := Summary(card)
		if !strings.Contains(s, "card "+card.ID()) {
			t.Fatalf("Summary = %q, want 'card %s ...'", s, card.ID())
		}
		if !strings.Contains(s, "****1111") {
			t.Fatalf("Summary = %q, want '****1111'", s)
		}
		if strings.Contains(s, "4111111111111111") {
			t.Fatalf("Summary leaks the full number: %q", s)
		}
	})

	t.Run("ValidateType and IsKnown", func(t *testing.T) {
		if err := ValidateType(TypeCard); err != nil {
			t.Fatalf("ValidateType(TypeCard): %v", err)
		}
		if !TypeCard.IsKnown() {
			t.Fatal("TypeCard.IsKnown() = false, want true")
		}
		unknown := ItemType("bogus")
		if err := ValidateType(unknown); !errors.Is(err, ErrUnknownType) {
			t.Fatalf("ValidateType(%q): %v, want ErrUnknownType", unknown, err)
		}
	})
}
