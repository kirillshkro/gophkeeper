// Package data — tests for the Credential item entity: metadata accessors,
// the Item base fields and the Credential branch of the package type
// switches. mustNewCredential and the shared checkErr helper live in
// credential_test.go / card_test.go (same package).
package data

import (
	"errors"
	"strings"
	"testing"
)

// TestMetadata_Credential verifies that a Credential item stores and reads
// metadata correctly (spec 3.3): empty, arbitrary keys, non-ASCII values,
// very long values and duplicate keys (later value wins). Metadata is set
// through SetMetadata because NewCredential does not accept metadata.
func TestMetadata_Credential(t *testing.T) {
	t.Helper()

	t.Run("empty", func(t *testing.T) {
		item := mustNewCredential(t, "user@example.com", "secret")
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
		item := mustNewCredential(t, "user@example.com", "secret")
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
		item := mustNewCredential(t, "user@example.com", "secret")
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
		item := mustNewCredential(t, "user@example.com", "secret")
		item.SetMetadata(Metadata{Extra: map[string]string{"long": long}})
		if got := item.Metadata().Get("long"); len(got) != len(long) {
			t.Fatalf("long value not preserved (len=%d, want %d)", len(got), len(long))
		}
	})

	t.Run("duplicate keys keep latest on update", func(t *testing.T) {
		item := mustNewCredential(t, "user@example.com", "secret")
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

// TestCredential_Item verifies the Item interface and the ItemCore base
// fields as exposed through a Credential item (spec 3.4).
func TestCredential_Item(t *testing.T) {
	t.Helper()

	t.Run("Type is TypeCredential", func(t *testing.T) {
		item := mustNewCredential(t, "user@example.com", "secret")
		if item.Type() != TypeCredential {
			t.Fatalf("Type = %q, want %q", item.Type(), TypeCredential)
		}
	})

	t.Run("OwnerID empty by default", func(t *testing.T) {
		item := mustNewCredential(t, "user@example.com", "secret")
		if item.OwnerID() != "" {
			t.Fatalf("OwnerID = %q, want empty", item.OwnerID())
		}
	})

	t.Run("OwnerID can be set", func(t *testing.T) {
		item := mustNewCredential(t, "user@example.com", "secret")
		item.SetOwnerID("owner-1")
		if item.OwnerID() != "owner-1" {
			t.Fatalf("OwnerID = %q, want owner-1", item.OwnerID())
		}
	})

	t.Run("Version is 1 at creation", func(t *testing.T) {
		item := mustNewCredential(t, "user@example.com", "secret")
		if item.Version() != 1 {
			t.Fatalf("Version = %d, want 1", item.Version())
		}
	})

	t.Run("Version increases on update", func(t *testing.T) {
		item := mustNewCredential(t, "user@example.com", "secret")
		before := item.Version()
		item.Bump()
		if item.Version() <= before {
			t.Fatalf("Version after bump = %d, want > %d", item.Version(), before)
		}
	})

	t.Run("timestamps set and ordered", func(t *testing.T) {
		item := mustNewCredential(t, "user@example.com", "secret")
		if item.CreatedAt().IsZero() || item.UpdatedAt().IsZero() {
			t.Fatal("CreatedAt or UpdatedAt is zero")
		}
		if item.UpdatedAt().Before(item.CreatedAt()) {
			t.Fatalf("UpdatedAt %s before CreatedAt %s", item.UpdatedAt(), item.CreatedAt())
		}
	})

	t.Run("IDs unique and non-empty", func(t *testing.T) {
		a := mustNewCredential(t, "user@example.com", "secret")
		b := mustNewCredential(t, "user@example.com", "secret")
		if a.ID() == "" || a.ID() == b.ID() {
			t.Fatal("IDs should be unique and non-empty")
		}
	})
}

// TestCredential_TypeSwitch covers the Credential branch of the package
// type switches (Empty, Summary) plus ItemType validation (spec 3.1 and
// 3.5): known types are accepted and the unknown-type fallback returns
// ErrUnknownType.
func TestCredential_TypeSwitch(t *testing.T) {
	t.Helper()

	t.Run("Empty(TypeCredential) builds a Credential", func(t *testing.T) {
		item, err := Empty(TypeCredential)
		if err != nil {
			t.Fatalf("Empty: %v", err)
		}
		cred, ok := item.(*Credential)
		if !ok {
			t.Fatalf("Empty: got %T, want *Credential", item)
		}
		if cred.Login != "" || cred.Password != "" {
			t.Fatalf("Empty item must be unpopulated: Login=%q Password=%q", cred.Login, cred.Password)
		}
	})

	t.Run("Empty(unknown type) falls back to ErrUnknownType", func(t *testing.T) {
		_, err := Empty("bogus")
		if !errors.Is(err, ErrUnknownType) {
			t.Fatalf("Empty: %v, want ErrUnknownType", err)
		}
	})

	t.Run("Summary shows the login but not the password", func(t *testing.T) {
		cred := mustNewCredential(t, "user@example.com", "super-secret-99")
		s := Summary(cred)
		if !strings.Contains(s, "credential "+cred.ID()) {
			t.Fatalf("Summary = %q, want 'credential %s ...'", s, cred.ID())
		}
		if !strings.Contains(s, "user@example.com") {
			t.Fatalf("Summary = %q, want the login", s)
		}
		if strings.Contains(s, "super-secret-99") {
			t.Fatalf("Summary leaks the password: %q", s)
		}
	})

	t.Run("ValidateType and IsKnown", func(t *testing.T) {
		if err := ValidateType(TypeCredential); err != nil {
			t.Fatalf("ValidateType(TypeCredential): %v", err)
		}
		if !TypeCredential.IsKnown() {
			t.Fatal("TypeCredential.IsKnown() = false, want true")
		}
		unknown := ItemType("bogus")
		if err := ValidateType(unknown); !errors.Is(err, ErrUnknownType) {
			t.Fatalf("ValidateType(%q): %v, want ErrUnknownType", unknown, err)
		}
	})
}
