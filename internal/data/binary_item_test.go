// Package data — tests for the Binary item entity: metadata accessors, the
// Item base fields and the Binary branch of the package type switches. The
// mustNewBinary helper lives in binary_test.go (same package).
package data

import (
	"errors"
	"strings"
	"testing"
)

// TestMetadata_Binary verifies that a Binary item stores and reads metadata
// correctly (spec 3.3): empty, arbitrary keys, non-ASCII values, very long
// values, duplicate keys (later value wins) and deep copying of the input
// metadata.
func TestMetadata_Binary(t *testing.T) {
	t.Helper()

	t.Run("empty", func(t *testing.T) {
		item := mustNewBinary(t, []byte("x"), Metadata{})
		if !item.Metadata().IsEmpty() {
			t.Fatalf("expected empty metadata, got %+v", item.Metadata())
		}
	})

	t.Run("arbitrary keys are stored and read back", func(t *testing.T) {
		meta := Metadata{
			Website: "example.com",
			Person:  "Alice",
			Bank:    "TestBank",
			OPTs:    []string{"123456"},
			Extra:   map[string]string{"note": "secret"},
		}
		item := mustNewBinary(t, []byte("x"), Metadata{})
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

	t.Run("non-ASCII values are preserved", func(t *testing.T) {
		item := mustNewBinary(t, []byte("x"), Metadata{})
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

	t.Run("very long values are preserved", func(t *testing.T) {
		long := strings.Repeat("x", 10_000)
		item := mustNewBinary(t, []byte("x"), Metadata{})
		item.SetMetadata(Metadata{Extra: map[string]string{"long": long}})
		if got := item.Metadata().Get("long"); len(got) != len(long) {
			t.Fatalf("long value not preserved (len=%d, want %d)", len(got), len(long))
		}
	})

	t.Run("duplicate keys keep latest on update", func(t *testing.T) {
		item := mustNewBinary(t, []byte("x"), Metadata{})
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

	t.Run("metadata is deep copied", func(t *testing.T) {
		extra := map[string]string{"tag": "v1"}
		item := mustNewBinary(t, []byte("x"), Metadata{Extra: extra})
		extra["tag"] = "changed"
		if item.Metadata().Get("tag") != "v1" {
			t.Fatalf("item metadata must be a deep copy, got %q", item.Metadata().Get("tag"))
		}
	})
}

// TestBinary_Item verifies the Item interface and the ItemCore base fields
// as exposed through a Binary item (spec 3.4).
func TestBinary_Item(t *testing.T) {
	t.Helper()

	t.Run("Type is TypeBinary", func(t *testing.T) {
		item := mustNewBinary(t, []byte("hi"), Metadata{})
		if item.Type() != TypeBinary {
			t.Fatalf("Type = %q, want %q", item.Type(), TypeBinary)
		}
	})

	t.Run("OwnerID empty by default", func(t *testing.T) {
		item := mustNewBinary(t, []byte("hi"), Metadata{})
		if item.OwnerID() != "" {
			t.Fatalf("OwnerID = %q, want empty", item.OwnerID())
		}
	})

	t.Run("OwnerID can be set", func(t *testing.T) {
		item := mustNewBinary(t, []byte("hi"), Metadata{})
		item.SetOwnerID("owner-1")
		if item.OwnerID() != "owner-1" {
			t.Fatalf("OwnerID = %q, want owner-1", item.OwnerID())
		}
	})

	t.Run("Version is 1 at creation", func(t *testing.T) {
		item := mustNewBinary(t, []byte("hi"), Metadata{})
		if item.Version() != 1 {
			t.Fatalf("Version = %d, want 1", item.Version())
		}
	})

	t.Run("Version increases on update", func(t *testing.T) {
		item := mustNewBinary(t, []byte("hi"), Metadata{})
		before := item.Version()
		item.Bump()
		if item.Version() <= before {
			t.Fatalf("Version after bump = %d, want > %d", item.Version(), before)
		}
	})

	t.Run("timestamps set and ordered", func(t *testing.T) {
		item := mustNewBinary(t, []byte("hi"), Metadata{})
		if item.CreatedAt().IsZero() || item.UpdatedAt().IsZero() {
			t.Fatal("CreatedAt or UpdatedAt is zero")
		}
		if item.UpdatedAt().Before(item.CreatedAt()) {
			t.Fatalf("UpdatedAt %s before CreatedAt %s", item.UpdatedAt(), item.CreatedAt())
		}
	})

	t.Run("IDs unique and non-empty", func(t *testing.T) {
		a := mustNewBinary(t, []byte("hi"), Metadata{})
		b := mustNewBinary(t, []byte("hi"), Metadata{})
		if a.ID() == "" || a.ID() == b.ID() {
			t.Fatal("IDs should be unique and non-empty")
		}
	})
}

// TestBinary_TypeSwitch covers the Binary branch of the package type switches
// (Empty, Summary) plus ItemType validation (spec 3.1 and 3.5): known types
// are accepted and the unknown-type fallback returns ErrUnknownType. Field
// validation (empty data -> ErrEmptyData, invalid metadata -> ErrInvalidField)
// is covered by TestData_NewBinary.
func TestBinary_TypeSwitch(t *testing.T) {
	t.Helper()

	t.Run("Empty(TypeBinary) builds a Binary", func(t *testing.T) {
		item, err := Empty(TypeBinary)
		if err != nil {
			t.Fatalf("Empty: %v", err)
		}
		b, ok := item.(*Binary)
		if !ok {
			t.Fatalf("Empty: got %T, want *Binary", item)
		}
		if b.Data != nil {
			t.Fatalf("Empty item must be unpopulated: Data = %v", b.Data)
		}
	})

	t.Run("Empty(unknown type) falls back to ErrUnknownType", func(t *testing.T) {
		_, err := Empty("bogus")
		if !errors.Is(err, ErrUnknownType) {
			t.Fatalf("Empty: %v, want ErrUnknownType", err)
		}
	})

	t.Run("Summary shows the byte count but not the payload", func(t *testing.T) {
		payload := []byte("super-secret-payload")
		b := mustNewBinary(t, payload, Metadata{})
		s := Summary(b)
		if !strings.Contains(s, "binary "+b.ID()) {
			t.Fatalf("Summary = %q, want 'binary %s ...'", s, b.ID())
		}
		if !strings.Contains(s, "bytes") {
			t.Fatalf("Summary = %q, want '... bytes'", s)
		}
		if strings.Contains(s, "super-secret-payload") {
			t.Fatalf("Summary leaks the payload: %q", s)
		}
	})

	t.Run("ValidateType and IsKnown", func(t *testing.T) {
		if err := ValidateType(TypeBinary); err != nil {
			t.Fatalf("ValidateType(TypeBinary): %v", err)
		}
		if !TypeBinary.IsKnown() {
			t.Fatal("TypeBinary.IsKnown() = false, want true")
		}
		unknown := ItemType("bogus")
		if err := ValidateType(unknown); !errors.Is(err, ErrUnknownType) {
			t.Fatalf("ValidateType(%q): %v, want ErrUnknownType", unknown, err)
		}
	})
}
