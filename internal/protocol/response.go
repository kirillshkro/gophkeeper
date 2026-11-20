// Package-level definitions for the response payloads of the protocol
// contract (tests-specs section 3.3: marshal/unmarshal — identity).
package protocol

import (
	"encoding/json"
	"time"

	"github.com/kirillshkro/gophkeeper/internal/data"
)

// Token is the bearer token issued by the server after a successful
// registration or login.
//
// A token carries no secrets: it contains only the owner identifier, the
// validity deadline (UTC) and the server signature. Signature verification
// is the auth package responsibility.
type Token struct {
	// OwnerID is the identifier of the account this token belongs to.
	OwnerID string `json:"owner_id"`
	// Expiry is the instant (UTC) at which the token stops being valid.
	Expiry time.Time `json:"expiry"`
	// Signature is the server signature over the token payload.
	Signature string `json:"signature"`
}

// RegisterResponse is the server response to a registration request.
//
// It is sent only after a successful registration and carries the bearer
// token of the newly created account.
type RegisterResponse struct {
	// Token is the bearer token of the newly created account.
	Token Token `json:"token"`
}

// LoginResponse is the server response to a login request.
//
// It is sent only after a successful login and carries the bearer token
// issued for the session.
type LoginResponse struct {
	// Token is the bearer token issued for the session.
	Token Token `json:"token"`
}

// dataResponseJSON is the JSON wire form of DataResponse; Item is a pointer
// so that an empty response encodes as {"item": null} and decodes back to a
// zero item.
type dataResponseJSON struct {
	Item *itemJSON `json:"item"`
}

// DataResponse is the server response to a data request.
//
// On success it carries the resulting item (with the server-assigned
// identifier for create); for delete it may be empty.
type DataResponse struct {
	// Item is the resulting item, if any.
	Item data.Item `json:"item"`
}

// MarshalJSON returns the JSON encoding of the data response.
func (r DataResponse) MarshalJSON() ([]byte, error) {
	var item *itemJSON
	if r.Item != nil {
		ij, err := itemJSONOf(r.Item)
		if err != nil {
			return nil, fieldError("response.item", err.Error())
		}
		item = &ij
	}
	return json.Marshal(dataResponseJSON{Item: item})
}

// UnmarshalJSON decodes the data response from its JSON encoding.
//
// The method restores the concrete item type selected by the "type"
// discriminator of the item wire form.
func (r *DataResponse) UnmarshalJSON(b []byte) error {
	var raw dataResponseJSON
	if err := json.Unmarshal(b, &raw); err != nil {
		return fieldError("response", "cannot unmarshal JSON")
	}
	r.Item = nil
	if raw.Item != nil {
		item, err := itemFromJSON(*raw.Item)
		if err != nil {
			return fieldError("response.item", err.Error())
		}
		r.Item = item
	}
	return nil
}
