package data

import "strings"

// Metadata stores arbitrary textual metadata for an item:
//
//   - Website, Person and Bank describe the affiliation of the item (a web
//     site, a person or a bank);
//   - OTPs is the list of one-time activation codes;
//   - Extra is a map of arbitrary key/value pairs.
type Metadata struct {
	// Website is the web site the item belongs to.
	Website string
	// Person is the person the item belongs to.
	Person string
	// Bank is the bank the item belongs to.
	Bank string
	// OTPs is a list of one-time activation codes related to the item.
	OPTs []string
	// Extra is arbitrary textual key/value pairs.
	Extra map[string]string
}

// knownMetaKeys are the keys reserved for the named Metadata fields; they
// must not be used in Extra.
var knownMetaKeys = map[string]struct{}{
	"website": {},
	"person":  {},
	"bank":    {},
	"otp":     {},
}

// Clone returns a copy of m in which the OTPs slice and the Extra map are
// detached from m, so later mutations of either copy do not affect the
// other.
func (m Metadata) Clone() Metadata {
	out := Metadata{
		Website: m.Website,
		Person:  m.Person,
		Bank:    m.Bank,
	}
	if len(m.Extra) > 0 {
		out.Extra = make(map[string]string, len(m.Extra))
		for k, v := range m.Extra {
			out.Extra[k] = v
		}
	}
	if len(m.OPTs) > 0 {
		out.OPTs = make([]string, len(m.OPTs))
		copy(out.OPTs, m.OPTs)
	}
	return out
}

// Get returns the Extra value for key or an empty string when key is absent.
func (m Metadata) Get(key string) string {
	return m.Extra[key]
}

// Set adds or replaces an Extra pair (key, value) for the receiver.
//
// Set returns an error when key is empty or reserved for a named field
// (website, person, bank, otp).
func (m *Metadata) Set(key, value string) error {
	if m == nil {
		return nil
	}
	if key == "" {
		return errInvalidField("extra", "key must not be empty")
	}
	if _, reserved := knownMetaKeys[strings.ToLower(key)]; reserved {
		return errInvalidField("extra", "key is reserved")
	}
	if m.Extra == nil {
		m.Extra = make(map[string]string)
	}
	m.Extra[key] = value
	return nil
}

// IsEmpty reports whether all Metadata fields are empty.
func (m Metadata) IsEmpty() bool {
	return m.Website == "" &&
		m.Person == "" &&
		m.Bank == "" &&
		len(m.OPTs) == 0 &&
		len(m.Extra) == 0
}

// Validate checks the Metadata fields: OTPs must not contain empty codes and
// Extra must not contain empty keys.
func (m Metadata) Validate() error {
	for _, code := range m.OPTs {
		if strings.TrimSpace(code) == "" {
			return errInvalidField("otps", "codes must not be empty")
		}
	}
	for key := range m.Extra {
		if key == "" {
			return errInvalidField("extra", "key must not be empty")
		}
	}
	return nil
}
