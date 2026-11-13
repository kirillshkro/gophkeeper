package data

// ItemType enumerates the kinds of data stored by GophKeeper.
//
// A new item type is added by introducing a new constant and a new
// concrete type; existing types are never modified (Open/Closed).
type ItemType string

const (
	// TypeCredential is a login/password pair.
	TypeCredential ItemType = "credential"

	// TypeText is arbitrary text data.
	TypeText ItemType = "text"

	// TypeBinary is arbitrary binary data.
	TypeBinary ItemType = "binary"

	// TypeCard is bank card data.
	TypeCard ItemType = "card"
)

// IsKnown reports whether t is one of the known item types.
func (t ItemType) IsKnown() bool {
	switch t {
	case TypeCredential, TypeText, TypeBinary, TypeCard:
		return true
	default:
		return false
	}
}

// String returns the name of the item type.
func (t ItemType) String() string {
	return string(t)
}

// ValidateType reports an error if t is not one of the known item types.
//
// It is meant to be used when a type comes from outside the process, for
// example from a stored row or a serialized payload.
func ValidateType(t ItemType) error {
	if t.IsKnown() {
		return nil
	}
	return ErrUnknownType
}
