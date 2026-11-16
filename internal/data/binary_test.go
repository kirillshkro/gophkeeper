// Package data — tests for the Binary item type.
//
// This file covers the NewBinary constructor (spec 3.2). Metadata, Item base
// entity and the Binary branch of the type switches live in
// binary_item_test.go (same package).
package data

import (
	"bytes"
	"strings"
	"testing"
)

// mustNewBinary builds a Binary item and fails the test if NewBinary returns
// an error. It returns a concrete *Binary so tests can inspect the Data field.
func mustNewBinary(t *testing.T, data []byte, meta Metadata) *Binary {
	t.Helper()
	item, err := NewBinary(data, meta)
	if err != nil {
		t.Fatalf("NewBinary: %v", err)
	}
	return item.(*Binary)
}

// TestData_NewBinary covers the NewBinary constructor (spec 3.2): valid blobs
// (simple, raw non-ASCII bytes and a multi-MiB payload), an empty blob and
// metadata that fails Validate(). All payloads are fake.
func TestData_NewBinary(t *testing.T) {
	t.Helper()

	large := []byte(strings.Repeat("x", 2*1024*1024))

	tests := []struct {
		name      string
		dataIn    []byte
		meta      Metadata
		wantError error
	}{
		// spec 3.2 #1 — valid blob
		{"valid simple blob", []byte("gophkeeper"), Metadata{}, nil},
		{"valid raw non-ASCII bytes", []byte{0x00, 0xff, 0x80, 0x01}, Metadata{}, nil},
		// spec 3.2 #2 — very large payload
		{"valid large blob (2 MiB)", large, Metadata{}, nil},

		// spec 3.2 #3 — empty blob
		{"empty blob", []byte{}, Metadata{}, ErrEmptyData},

		// invalid metadata
		{"empty otp in metadata", []byte("x"), Metadata{OPTs: []string{"", "123"}}, ErrInvalidField},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, err := NewBinary(tt.dataIn, tt.meta)
			checkErr(t, "NewBinary", err, tt.wantError)
			if err != nil {
				return
			}
			b := item.(*Binary)
			if b.Type() != TypeBinary {
				t.Fatalf("Type = %q, want %q", b.Type(), TypeBinary)
			}
			if !bytes.Equal(b.Data, tt.dataIn) {
				t.Fatalf("Data len=%d, want len=%d", len(b.Data), len(tt.dataIn))
			}
		})
	}
}
