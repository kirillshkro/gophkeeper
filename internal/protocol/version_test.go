// Package protocol contains tests for the protocol versioning part
// (tests-specs: section 3.1 and 3.5 #3).
package protocol

import (
	"errors"
	"regexp"
	"testing"
)

// semverSpecRe is the spec regex for a valid protocol version
// (tests-specs 3.1 #1).
var semverSpecRe = regexp.MustCompile(`^\d+\.\d+\.\d+(-.*)?$`)

// checkErr asserts that got is nil or is wrapped from want.
//
// All error assertions use errors.Is (tests-specs, readiness criteria).
func checkErr(t *testing.T, name string, got, want error) {
	t.Helper()
	if (got == nil) != (want == nil) {
		t.Fatalf("%s: got err=%v, want err=%v", name, got, want)
	}
	if got != nil && !errors.Is(got, want) {
		t.Fatalf("%s: got %v, want %v", name, got, want)
	}
}

// TestProtocol_ProtocolVersion_Current verifies that the current version is a
// valid semver and is parseable (spec 3.1 #1).
func TestProtocol_ProtocolVersion_Current(t *testing.T) {
	t.Helper()
	if Version() != Current {
		t.Fatalf("Version() = %q, want Current %q", Version(), Current)
	}
	v := string(Version())
	if !semverSpecRe.MatchString(v) {
		t.Fatalf("current version %q does not match the spec semver regex", v)
	}
	if _, err := Parse(v); err != nil {
		t.Fatalf("Parse(current %q): %v", v, err)
	}
}

// TestProtocol_ProtocolVersion_Compatible covers supported versions: the
// supported version in a request (spec 3.1 #3), a new client against an old
// server and an old client against a new server (spec 3.1 #2, 3.5 #2).
func TestProtocol_ProtocolVersion_Compatible(t *testing.T) {
	t.Helper()
	tests := []struct {
		name      string
		version   string
		wantError error
	}{
		{"supported version in request", "1.0.0", nil},
		{"same major, higher minor (new client, old server)", "1.5.0", nil},
		{"same major, higher patch", "1.0.9", nil},
		{"same major, pre-release", "1.0.0-rc.1", nil},
		{"lower major (old client, new server)", "0.9.9", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkErr(t, "IsCompatible", IsCompatible(tt.version), tt.wantError)
		})
	}
}

// TestProtocol_ProtocolVersion_Unsupported covers unknown and incompatible
// versions (spec 3.1 #4, 3.5 #3): an incompatible major and malformed
// versions always return ErrUnsupportedVersion.
func TestProtocol_ProtocolVersion_Unsupported(t *testing.T) {
	t.Helper()
	// Incompatible major (spec 3.5 #3): the format is valid, so Parse
	// accepts it; only IsCompatible rejects it.
	for _, version := range []string{"2.0.0", "3.7.0", "2.0.0-rc.1"} {
		t.Run("incompatible major "+version, func(t *testing.T) {
			checkErr(t, "IsCompatible", IsCompatible(version), ErrUnsupportedVersion)
			parsed, err := Parse(version)
			if err != nil {
				t.Fatalf("Parse(%q): %v, want nil", version, err)
			}
			if string(parsed) != version {
				t.Fatalf("Parse(%q) = %q, want %q", version, parsed, version)
			}
		})
	}
	// Malformed (spec 3.1 #4): both Parse and IsCompatible reject.
	for _, version := range []string{"1.0", "1", "a.b.c", "1.0.0a", "1.0.0+build", "1.0.0 -rc.1", " 1.0.0", ""} {
		t.Run("malformed "+version, func(t *testing.T) {
			checkErr(t, "IsCompatible", IsCompatible(version), ErrUnsupportedVersion)
			if _, err := Parse(version); !errors.Is(err, ErrUnsupportedVersion) {
				t.Fatalf("Parse(%q) = %v, want ErrUnsupportedVersion", version, err)
			}
		})
	}
}

// TestProtocol_ProtocolVersion_Parse covers successful parsing and the
// parse/serialize/parse round-trip (spec 3.1, 3.5).
func TestProtocol_ProtocolVersion_Parse(t *testing.T) {
	t.Helper()
	tests := []struct {
		name  string
		input string
		want  ProtocolVersion
	}{
		{"exact current", "1.0.0", Current},
		{"large components", "12.34.56", "12.34.56"},
		{"pre-release", "1.0.0-rc.1", "1.0.0-rc.1"},
		{"pre-release with dots", "2.1.0-beta.2", "2.1.0-beta.2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("Parse(%q) = %q, want %q", tt.input, got, tt.want)
			}
			// Round-trip: a parsed version is parsed again and is identical.
			round, err := Parse(string(got))
			if err != nil || round != tt.want {
				t.Fatalf("Parse(string(Parse(%q))) = %q (err %v), want %q", tt.input, round, err, tt.want)
			}
		})
	}
}

// TestProtocol_majorOf covers extraction of the major part: valid semver
// strings and malformed input (the helper must be safe even if called
// without Parse first).
func TestProtocol_majorOf(t *testing.T) {
	t.Helper()
	tests := []struct {
		name    string
		version string
		want    int
		wantErr bool
	}{
		{"current", "1.0.0", 1, false},
		{"newer major", "2.0.0", 2, false},
		{"pre-release", "1.0.0-rc.1", 1, false},
		{"empty", "", 0, true},
		{"non-numeric", "a.b.c", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := majorOf(tt.version)
			if (err != nil) != tt.wantErr {
				t.Fatalf("majorOf(%q): err = %v, want err=%v", tt.version, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("majorOf(%q) = %d, want %d", tt.version, got, tt.want)
			}
		})
	}
}
