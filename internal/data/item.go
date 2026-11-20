package data

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// Item is a single stored item of GophKeeper.
//
// Item is satisfied by *Credential, *Text, *Binary and *Card. Every concrete
// type embeds ItemCore and adds its own fields. A new item type is
// introduced by defining a new concrete type and a new ItemType constant
// without changing existing types (Open/Closed).
type Item interface {
	// ID returns the unique identifier of the item.
	ID() string
	// Type returns the kind of the item.
	Type() ItemType
	// OwnerID returns the identifier of the item owner.
	OwnerID() string
	// Metadata returns the item metadata.
	Metadata() Metadata
	// Version returns the data version used by synchronization.
	Version() int
	// CreatedAt returns the creation time of the item.
	CreatedAt() time.Time
	// UpdatedAt returns the last update time of the item.
	UpdatedAt() time.Time
}

// ItemCore is the base model shared by every item type.
//
// The fields are unexported; the state is read through the getters of the
// Item interface and changed through the Set* and Bump methods.
type ItemCore struct {
	// id is the unique identifier of the item.
	id string
	// ownerID is the identifier of the item owner.
	ownerID string
	// itemType is the kind of the item.
	itemType ItemType
	// metadata is the item metadata.
	metadata Metadata
	// version is the data version, used by synchronization.
	version int
	// createdAt is the creation time of the item.
	createdAt time.Time
	// updatedAt is the last update time of the item.
	updatedAt time.Time
}

// ID returns the unique identifier of the item.
func (c *ItemCore) ID() string {
	return c.id
}

// Type returns the kind of the item.
func (c *ItemCore) Type() ItemType {
	return c.itemType
}

// OwnerID returns the identifier of the item owner.
func (c *ItemCore) OwnerID() string {
	return c.ownerID
}

// Metadata returns the item metadata.
func (c *ItemCore) Metadata() Metadata {
	return c.metadata
}

// Version returns the data version used by synchronization.
func (c *ItemCore) Version() int {
	return c.version
}

// CreatedAt returns the creation time of the item.
func (c *ItemCore) CreatedAt() time.Time {
	return c.createdAt
}

// UpdatedAt returns the last update time of the item.
func (c *ItemCore) UpdatedAt() time.Time {
	return c.updatedAt
}

// SetOwnerID sets the owner identifier of the item.
func (c *ItemCore) SetOwnerID(ownerID string) {
	c.ownerID = ownerID
}

// SetMetadata replaces the item metadata with a copy of m.
func (c *ItemCore) SetMetadata(m Metadata) {
	c.metadata = m.Clone()
}

// SetVersion sets the item data version.
func (c *ItemCore) SetVersion(version int) {
	c.version = version
}

// SetID sets the unique identifier of the item.
//
// SetID is part of the reconstruction path: an item decoded from a
// serialized form (see the protocol package) fills in the field that
// Empty does not set.
func (c *ItemCore) SetID(id string) {
	c.id = id
}

// SetCreatedAt sets the creation time of the item.
//
// SetCreatedAt is part of the reconstruction path (see SetID).
func (c *ItemCore) SetCreatedAt(t time.Time) {
	c.createdAt = t
}

// SetUpdatedAt sets the last update time of the item.
//
// SetUpdatedAt is part of the reconstruction path (see SetID).
func (c *ItemCore) SetUpdatedAt(t time.Time) {
	c.updatedAt = t
}

// Bump increments the data version and refreshes the UpdatedAt time.
func (c *ItemCore) Bump() {
	c.version++
	c.updatedAt = time.Now().UTC()
}

// newItem creates the base of a new item of the given type: a random ID,
// version 1 and current UTC timestamps.
func newItem(t ItemType) (ItemCore, error) {
	id, err := newID()
	if err != nil {
		return ItemCore{}, err
	}
	now := time.Now().UTC()
	return ItemCore{
		itemType:  t,
		id:        id,
		version:   1,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// newID generates a random 16-byte identifier in hexadecimal form.
func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

// Empty creates an unpopulated item of the given type.
//
// Empty is meant for reconstructing items from a serialized form: the
// caller fills in the fields and the type-specific payload afterwards.
func Empty(t ItemType) (Item, error) {
	if err := ValidateType(t); err != nil {
		return nil, err
	}
	core := ItemCore{itemType: t}
	switch t {
	case TypeCredential:
		return &Credential{ItemCore: core}, nil
	case TypeText:
		return &Text{ItemCore: core}, nil
	case TypeBinary:
		return &Binary{ItemCore: core}, nil
	case TypeCard:
		return &Card{ItemCore: core}, nil
	default:
		return nil, ErrUnknownType
	}
}

// Summary returns a short human-readable summary of the item.
//
// Summary never reveals secrets: passwords are omitted and only the last
// four digits of a card number are shown.
func Summary(item Item) string {
	switch v := item.(type) {
	case *Credential:
		return fmt.Sprintf("credential %s: login=%q", v.ID(), v.Login)
	case *Text:
		return fmt.Sprintf("text %s: %d characters", v.ID(), len(v.Content))
	case *Binary:
		return fmt.Sprintf("binary %s: %d bytes", v.ID(), len(v.Data))
	case *Card:
		return fmt.Sprintf("card %s: ****%s", v.ID(), cardTail(v.Number))
	default:
		return fmt.Sprintf("unknown item %s", item.ID())
	}
}

// cardTail returns the last four digits of n or an empty string when n is
// shorter than four digits.
func cardTail(n string) string {
	if len(n) <= 4 {
		return ""
	}
	return n[len(n)-4:]
}

var (
	_ Item = (*Credential)(nil)
	_ Item = (*Text)(nil)
	_ Item = (*Binary)(nil)
	_ Item = (*Card)(nil)
)
