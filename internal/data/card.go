package data

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

const (
	minCardNumberLen = 13
	maxCardNumberLen = 19
	maxHolderLen     = 64
)

// Card is the data of a bank card.
//
// Number, Expiry and CVV are stored encrypted by the caller before
// persistence (see the security and storage packages).
type Card struct {
	ItemCore
	// Number is the card number; stored as digits without spaces.
	Number string
	// Expiry is the card expiry, normalized to MM/YY.
	Expiry string
	// CVV is the card security code, three or four digits.
	CVV string
	// Holder is the card holder name.
	Holder string
}

// NewCard creates a new card item with the given number and expiry.
//
// The number may contain spaces; it is stored as a continuous digit string
// (13..19 digits, Luhn check). The expiry accepts MM/YY, MM/YYYY and MMYY
// and is stored as MM/YY.
func NewCard(number, expiry string) (Item, error) {
	core, err := newItem(TypeCard)
	if err != nil {
		return nil, err
	}
	n, err := normalizeNumber(number)
	if err != nil {
		return nil, err
	}
	e, err := normalizeExpiry(expiry)
	if err != nil {
		return nil, err
	}
	return &Card{
		ItemCore: core,
		Number:   n,
		Expiry:   e,
	}, nil
}

// SetCVV sets the card security code.
//
// An empty value clears the code; a non-empty value must be three or four
// digits.
func (c *Card) SetCVV(cvv string) error {
	if cvv == "" {
		c.CVV = ""
		return nil
	}
	if len(cvv) < 3 || len(cvv) > 4 {
		return errInvalidField("cvv", "must be 3 or 4 digits")
	}
	for _, r := range cvv {
		if !unicode.IsDigit(r) {
			return errInvalidField("cvv", "must contain only digits")
		}
	}
	c.CVV = cvv
	return nil
}

// SetHolder sets the card holder name.
//
// An empty value clears the name; the trimmed name must not be longer than
// maxHolderLen characters.
func (c *Card) SetHolder(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		c.Holder = ""
		return nil
	}
	if len(name) > maxHolderLen {
		return errInvalidField("holder", "too long")
	}
	c.Holder = name
	return nil
}

// normalizeNumber strips spaces, validates and returns the number as a
// continuous digit string of 13..19 digits passing the Luhn check.
func normalizeNumber(s string) (string, error) {
	var sb strings.Builder
	var hasInvalid byte
	for _, r := range s {
		switch {
		case unicode.IsDigit(r):
			sb.WriteRune(r)
		case !unicode.IsSpace(r):
			hasInvalid = 1
		}
	}
	if hasInvalid != 0 {
		return "", errInvalidField("number", "must contain only digits and spaces")
	}
	n := sb.String()
	if len(n) < minCardNumberLen || len(n) > maxCardNumberLen {
		return "", errInvalidField("number", "must be 13..19 digits long")
	}
	if !luhnValid(n) {
		return "", errInvalidField("number", "failed luhn check")
	}
	return n, nil
}

// normalizeExpiry validates and normalizes the expiry string to MM/YY.
func normalizeExpiry(s string) (string, error) {
	s = strings.TrimSpace(s)
	var month, year int
	switch {
	case len(s) == 5 && s[2] == '/':
		if !allDigits(s[:2]) || !allDigits(s[3:5]) {
			return "", errInvalidField("expiry", "must contain only digits")
		}
		month, year = atoi(s[:2]), atoi(s[3:5])
	case len(s) == 6 && s[2] == '/':
		if !allDigits(s[:2]) || !allDigits(s[3:7]) {
			return "", errInvalidField("expiry", "must contain only digits")
		}
		month, year = atoi(s[:2]), atoi(s[3:7])
	case len(s) == 4:
		if !allDigits(s) {
			return "", errInvalidField("expiry", "must contain only digits")
		}
		month, year = atoi(s[:2]), atoi(s[2:])
	default:
		return "", errInvalidField("expiry", "must be MM/YY, MM/YYYY or MMYY")
	}
	if month < 1 || month > 12 {
		return "", errInvalidField("expiry", "invalid month")
	}
	return fmt.Sprintf("%02d/%02d", month, year%100), nil
}

// atoi parses the non-empty digit string s; the caller guarantees that all
// runes are digits.
func atoi(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

// allDigits reports whether s contains only digit runes.
func allDigits(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// luhnValid reports whether the digit string s passes the Luhn check.
func luhnValid(s string) bool {
	sum := 0
	doubled := false
	for i := len(s) - 1; i >= 0; i-- {
		d := int(s[i] - '0')
		if doubled {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		doubled = !doubled
	}
	return sum%10 == 0
}
