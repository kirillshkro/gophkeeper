// Package data contains the data model of the GophKeeper system.
//
// The package defines the types of stored private data:
//
// - login/password pairs (Credential, TypeCredential);
// - arbitrary text data (Text, TypeText);
// - arbitrary binary data (Binary, TypeBinary);
// - bank card data (Card, TypeCard).
//
// Every concrete type is an implementation of the Item interface and embeds
// the ItemCore base model, which holds the common fields: ID, OwnerID,
// ItemType, Metadata, Version, CreatedAt and UpdatedAt.
//
// For any item there is an option to store arbitrary textual metadata
// (Metadata): belonging to a website, person or bank, one-time activation
// codes (OTP) and arbitrary key/value pairs.
//
// Constructors (NewCredential, NewText, NewBinary, NewCard) validate the
// input and return (Item, error): empty fields yield ErrEmptyData, invalid
// formats yield ErrInvalidField and unknown item types yield ErrUnknownType.
//
// The package is a stable leaf layer: it depends only on the standard
// library and is independent of persistence and transport.
package data
