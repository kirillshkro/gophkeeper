package protocol

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// ErrUnsupportedVersion is returned when a protocol version cannot be parsed
// or is incompatible with the current one: its major part exceeds the major
// part of Current.
var ErrUnsupportedVersion = errors.New("unsupported protocol version")

// ProtocolVersion is a version of the GophKeeper client-server protocol.
//
// The format is semver MAJOR.MINOR.PATCH with an optional pre-release suffix
// separated by a hyphen: "1.0.0", "1.5.0", "1.0.0-rc.1". Build suffixes
// ("+...") are not part of the protocol format.
type ProtocolVersion string

// Current is the current version of the protocol.
//
// Both the client and the server put this value into requests and responses.
// A server accepts any version compatible with Current (see IsCompatible).
const Current ProtocolVersion = "1.0.0"

// semverRe matches the whole version format: three numeric components
// separated by dots and an optional hyphenated pre-release suffix.
var semverRe = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(-.*)?$`)

// Version returns the current protocol version (spec 3.1 #1).
//
// The returned value is always a valid semver and is parseable by Parse.
//
// The function is named Version rather than ProtocolVersion because Go
// does not allow a type and a package-level function to share the same
// identifier name.
func Version() ProtocolVersion {
	return Current
}

// Parse parses s as a ProtocolVersion.
//
// It returns ErrUnsupportedVersion when s is empty or deviates from the
// format: missing components, non-numeric parts or a build suffix.
func Parse(s string) (ProtocolVersion, error) {
	if !semverRe.MatchString(s) {
		return "", ErrUnsupportedVersion
	}
	return ProtocolVersion(s), nil
}

// IsCompatible reports whether v is compatible with the current version
// (spec 3.1 #2-#4, 3.5 #3).
//
// A version is compatible when it can be parsed and its major part does not
// exceed the major part of Current. Minor and patch may be lower (old client
// to new server) or higher (new client to old server). An unknown, malformed
// or "newer major" version returns ErrUnsupportedVersion.
func IsCompatible(v string) error {
	if _, err := Parse(v); err != nil {
		return err
	}
	major, err := majorOf(v)
	if err != nil {
		return ErrUnsupportedVersion
	}
	currentMajor, err := majorOf(string(Current))
	if err != nil {
		return ErrUnsupportedVersion
	}
	if major > currentMajor {
		return ErrUnsupportedVersion
	}
	return nil
}

// majorOf returns the numeric major part of a semver string.
//
// It returns an error when the major part is not a valid decimal number.
// Callers that pass a string already validated by Parse never see an error;
// the error return keeps the function safe if it is reused without Parse.
func majorOf(v string) (int, error) {
	parts := strings.SplitN(v, ".", 3)
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}
	return n, nil
}
