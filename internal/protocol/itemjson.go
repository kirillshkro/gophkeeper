// Package-level definitions for the JSON wire form of data items (tests-specs
// section 3.3: marshal/unmarshal — identity).
package protocol

import (
	"errors"
	"time"

	"github.com/kirillshkro/gophkeeper/internal/data"
)

// itemJSON is the JSON wire form of a data item.
//
// The "type" discriminator selects which payload sub-object is present; the
// core fields are always present so the decoded item is deep-equal to the
// original (marshal → unmarshal — identity). Unknown fields in the item
// object are ignored (extensibility).
type itemJSON struct {
	Type       data.ItemType   `json:"type"`
	ID         string          `json:"id"`
	OwnerID    string          `json:"owner_id"`
	Version    int             `json:"version"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	Metadata   data.Metadata   `json:"metadata,omitempty"`
	Credential *credentialJSON `json:"credential,omitempty"`
	Text       *textJSON       `json:"text,omitempty"`
	Binary     *binaryJSON     `json:"binary,omitempty"`
	Card       *cardJSON       `json:"card,omitempty"`
}

// credentialJSON is the JSON payload of a *data.Credential item.
type credentialJSON struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// textJSON is the JSON payload of a *data.Text item.
type textJSON struct {
	Content string `json:"content"`
}

// binaryJSON is the JSON payload of a *data.Binary item; encoding/json
// base64-encodes Data.
type binaryJSON struct {
	Data []byte `json:"data"`
}

// cardJSON is the JSON payload of a *data.Card item.
type cardJSON struct {
	Number string `json:"number"`
	Expiry string `json:"expiry"`
	CVV    string `json:"cvv"`
	Holder string `json:"holder"`
}

// itemJSONOf converts an item to its JSON wire form.
//
// It returns an error when the item is nil or is a type this package does
// not know about.
func itemJSONOf(item data.Item) (itemJSON, error) {
	if item == nil {
		return itemJSON{}, errors.New("item must not be nil")
	}
	out := itemJSON{
		Type:      item.Type(),
		ID:        item.ID(),
		OwnerID:   item.OwnerID(),
		Version:   item.Version(),
		CreatedAt: item.CreatedAt(),
		UpdatedAt: item.UpdatedAt(),
		Metadata:  item.Metadata(),
	}
	switch v := item.(type) {
	case *data.Credential:
		out.Credential = &credentialJSON{Login: v.Login, Password: v.Password}
	case *data.Text:
		out.Text = &textJSON{Content: v.Content}
	case *data.Binary:
		out.Binary = &binaryJSON{Data: v.Data}
	case *data.Card:
		out.Card = &cardJSON{Number: v.Number, Expiry: v.Expiry, CVV: v.CVV, Holder: v.Holder}
	default:
		return itemJSON{}, errors.New("unknown item type")
	}
	return out, nil
}

// itemFromJSON reconstructs an item from its JSON wire form.
//
// The method restores the concrete item type selected by the "type"
// discriminator and fills in the core fields and the type-specific payload.
func itemFromJSON(i itemJSON) (data.Item, error) {
	item, err := data.Empty(i.Type)
	if err != nil {
		return nil, err
	}
	core := itemCoreOf(item)
	if core == nil {
		return nil, errors.New("unknown item type")
	}
	core.SetID(i.ID)
	core.SetOwnerID(i.OwnerID)
	core.SetVersion(i.Version)
	core.SetCreatedAt(i.CreatedAt)
	core.SetUpdatedAt(i.UpdatedAt)
	core.SetMetadata(i.Metadata)

	switch v := item.(type) {
	case *data.Credential:
		if p := i.Credential; p != nil {
			v.Login, v.Password = p.Login, p.Password
		}
	case *data.Text:
		if p := i.Text; p != nil {
			v.Content = p.Content
		}
	case *data.Binary:
		if p := i.Binary; p != nil {
			v.Data = p.Data
		}
	case *data.Card:
		if p := i.Card; p != nil {
			v.Number, v.Expiry, v.CVV, v.Holder = p.Number, p.Expiry, p.CVV, p.Holder
		}
	}
	return item, nil
}

// itemCoreOf returns the embedded *data.ItemCore of a concrete item type, or
// nil when the type is unknown.
func itemCoreOf(item data.Item) *data.ItemCore {
	switch v := item.(type) {
	case *data.Credential:
		return &v.ItemCore
	case *data.Text:
		return &v.ItemCore
	case *data.Binary:
		return &v.ItemCore
	case *data.Card:
		return &v.ItemCore
	default:
		return nil
	}
}
