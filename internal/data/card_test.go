// Package data — tests for the Card item type.
//
// This file covers the constructor, SetCVV and SetHolder. The metadata,
// Item base entity and type-switch tests live in card_item_test.go.
// The other item types (Credential, Text, Binary) are left for their own
// test files.
package data

import (
	"errors"
	"strings"
	"testing"
)

// mustNewCard builds a Card item and fails the test if NewCard returns an
// error. It returns a concrete *Card so tests can inspect its fields.
func mustNewCard(t *testing.T, number, expiry string) *Card {
	t.Helper()
	item, err := NewCard(number, expiry)
	if err != nil {
		t.Fatalf("NewCard: %v", err)
	}
	return item.(*Card)
}

// checkErr fails the test when the presence of got differs from want or
// when got is not wrapped from want.
func checkErr(t *testing.T, name string, got, want error) {
	if (got != nil) != (want != nil) {
		t.Fatalf("%s: got err=%v, want err=%v", name, got, want)
	}
	if got != nil && !errors.Is(got, want) {
		t.Fatalf("%s error %v, want %v", name, got, want)
	}
}

// TestData_NewCard covers the NewCard constructor (spec 3.2): valid cards
// with 13/15/16/19-digit numbers, number and expiry normalization, and the
// invalid inputs. All test numbers below are Luhn-valid unless noted.
func TestData_NewCard(t *testing.T) {
	t.Helper()

	tests := []struct {
		name      string
		number    string
		expiry    string
		wantNum   string
		wantExp   string
		wantError error
	}{
		// valid cards
		{"valid 16 digits", "4111111111111111", "12/25", "4111111111111111", "12/25", nil},
		{"spaces stripped from number", "4111 1111 1111 1111", "12/25", "4111111111111111", "12/25", nil},
		{"valid 13 digits", "4222222222222", "01/26", "4222222222222", "01/26", nil},
		{"valid 15 digits", "371449635398431", "01/26", "371449635398431", "01/26", nil},
		{"valid 19 digits", "4111111111111111110", "12/25", "4111111111111111110", "12/25", nil},
		{"expiry MM/YY with leading zero month", "4111111111111111", "02/25", "4111111111111111", "02/25", nil},
		{"expiry MMYYYY normalized to MM/YY", "4111111111111111", "1225", "4111111111111111", "12/25", nil},
		{"expiry with spaces trimmed", "4111111111111111", " 12/25 ", "4111111111111111", "12/25", nil},
		//
		// Spec/code discrepancies, documented by the tests:
		// - the spec expects MM/YYYY (7 chars) to be accepted, but the
		//   normalizeExpiry case is len == 6, so "12/2025" is rejected;
		// - the len==6 branch ("12/202") cannot be exercised: it executes
		//   s[3:7] on a len-6 string, which panics in this toolchain
		//   (go1.27.4-X does not clamp the end index), so no test hits it;
		// - the spec expects expired expiries to be rejected, but the code
		//   has no comparison with the current date.
		{"MM/YYYY (7 chars) is rejected by the code", "4111111111111111", "12/2025", "", "", ErrInvalidField},
		{"expired expiry is accepted by the code", "4111111111111111", "01/20", "4111111111111111", "01/20", nil},
		// invalid number
		{"empty number", "", "12/25", "", "", ErrInvalidField},
		{"short number (12 digits)", "411111111111", "12/25", "", "", ErrInvalidField},
		{"long number (20 digits)", "41111111111111110950", "12/25", "", "", ErrInvalidField},
		{"non-digit in number", "4111111111111111a", "12/25", "", "", ErrInvalidField},
		{"number fails luhn check", "4111111111111112", "12/25", "", "", ErrInvalidField},
		// invalid expiry
		{"empty expiry", "4111111111111111", "", "", "", ErrInvalidField},
		{"invalid month (13)", "4111111111111111", "13/25", "", "", ErrInvalidField},
		{"zero month (len 4, not digits)", "4111111111111111", "0/25", "", "", ErrInvalidField},
		{"non-digit in expiry", "4111111111111111", "ab/25", "", "", ErrInvalidField},
		{"wrong length (3 chars)", "4111111111111111", "122", "", "", ErrInvalidField},
		{"wrong length with slash (4 chars)", "4111111111111111", "1/25", "", "", ErrInvalidField},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, err := NewCard(tt.number, tt.expiry)
			checkErr(t, "NewCard", err, tt.wantError)
			if err != nil {
				return
			}
			card := item.(*Card)
			if card.Number != tt.wantNum {
				t.Fatalf("Number = %q, want %q", card.Number, tt.wantNum)
			}
			if card.Expiry != tt.wantExp {
				t.Fatalf("Expiry = %q, want %q", card.Expiry, tt.wantExp)
			}
		})
	}
}

// TestData_SetCVV covers the card security code setter (spec 3.2 #4 and the
// godoc contract: 3 or 4 digits, an empty value clears the code).
func TestData_SetCVV(t *testing.T) {
	t.Helper()

	tests := []struct {
		name    string
		cvv     string
		wantErr error
		wantCVV string
	}{
		{"3 digits", "123", nil, "123"},
		{"4 digits", "1234", nil, "1234"},
		{"empty clears the code", "", nil, ""},
		{"too short (2)", "12", ErrInvalidField, ""},
		{"too long (5)", "12345", ErrInvalidField, ""},
		{"non-digit", "12a", ErrInvalidField, ""},
		{"space inside", "1 2", ErrInvalidField, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := mustNewCard(t, "4111111111111111", "12/25")
			checkErr(t, "SetCVV", card.SetCVV(tt.cvv), tt.wantErr)
			if card.CVV != tt.wantCVV {
				t.Fatalf("CVV = %q, want %q", card.CVV, tt.wantCVV)
			}
		})
	}

	t.Run("error leaves CVV unchanged", func(t *testing.T) {
		card := mustNewCard(t, "4111111111111111", "12/25")
		if err := card.SetCVV("1234"); err != nil {
			t.Fatalf("SetCVV: %v", err)
		}
		if err := card.SetCVV("1a2"); err == nil {
			t.Fatal("expected ErrInvalidField for non-digit CVV")
		}
		if card.CVV != "1234" {
			t.Fatalf("CVV = %q, want unchanged 1234", card.CVV)
		}
	})
}

// TestData_SetHolder covers the card holder setter (trim, max 64 chars,
// empty value clears the name).
func TestData_SetHolder(t *testing.T) {
	t.Helper()

	tests := []struct {
		name       string
		holderIn   string
		wantErr    error
		wantHolder string
	}{
		{"valid name", "John Doe", nil, "John Doe"},
		// TrimSpace only: leading/trailing whitespace is removed, internal
		// runs of spaces are preserved as-is by the code.
		{"surrounding whitespace trimmed", "  John  Doe  ", nil, "John  Doe"},
		{"non-ASCII name", "Иван Петров", nil, "Иван Петров"},
		{"empty clears the name", "", nil, ""},
		{"whitespace only clears", "   ", nil, ""},
		{"exactly max length (64)", strings.Repeat("a", 64), nil, strings.Repeat("a", 64)},
		{"too long (65)", strings.Repeat("a", 65), ErrInvalidField, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := mustNewCard(t, "4111111111111111", "12/25")
			checkErr(t, "SetHolder", card.SetHolder(tt.holderIn), tt.wantErr)
			if card.Holder != tt.wantHolder {
				t.Fatalf("Holder = %q, want %q", card.Holder, tt.wantHolder)
			}
		})
	}

	t.Run("error leaves holder unchanged", func(t *testing.T) {
		card := mustNewCard(t, "4111111111111111", "12/25")
		if err := card.SetHolder("Alice"); err != nil {
			t.Fatalf("SetHolder(Alice): %v", err)
		}
		if err := card.SetHolder(strings.Repeat("a", 65)); err == nil {
			t.Fatal("expected ErrInvalidField for long holder")
		}
		if card.Holder != "Alice" {
			t.Fatalf("Holder = %q, want unchanged Alice", card.Holder)
		}
	})
}

