package security

import (
	"crypto/hmac"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	// argon2Version is the Argon2 format version embedded in the hash
	// ($argon2id$v=19).
	argon2Version = 19
	// argon2Time is the number of iterations (second option of
	// RFC 9106 section 4: time=3, memory=64 MiB, threads=4).
	argon2Time = 3
	// argon2Memory is the memory cost in KiB (second option of RFC 9106 section 4).
	argon2Memory = 64 * 1024
	// argon2Threads is the degree of parallelism (RFC 9106 section 4).
	argon2Threads = 4
	// argon2KeyLen is the hash output length in bytes.
	argon2KeyLen = 32
	// argon2SaltLen is the random salt length in bytes.
	argon2SaltLen = 16
)

// argon2idID is the identifier of the Argon2id hash (the first PHC field
// after the leading "$").
const argon2idID = "argon2id"

// errInvalidArgon2Format is returned when a hash cannot be parsed.
var errInvalidArgon2Format = errors.New("invalid argon2 hash format")

// argHash stores the parameters and result of an argon2id hash.
type argHash struct {
	version int
	memory  uint32
	time    uint32
	threads uint32
	keyLen  uint32
	salt    []byte
	key     []byte
}

// HashPassword hashes plain with the Argon2id key derivation function
// (RFC 9106, second recommended option: time=3, memory=64MiB, threads=4)
// using a random salt and returns the result in the standard PHC format.
//
// The password itself is never returned, logged or stored in clear:
// only the salted hash is.
func HashPassword(plain string) (string, error) {
	if plain == "" {
		return "", ErrEmptyData
	}
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	return hashWithSalt(plain, salt), nil
}

// VerifyPassword reports whether plain matches the encoded Argon2id hash.
//
// It returns ErrInvalidPasswordHash when the hash format is damaged or
// when plain does not match. The comparison is done in constant time
// (hmac.Equal).
func VerifyPassword(plain, hash string) error {
	d, err := decodeArgon2Hash(hash)
	if err != nil {
		return ErrInvalidPasswordHash
	}
	key := argon2.IDKey([]byte(plain), d.salt, d.time, d.memory, uint8(d.threads), d.keyLen)
	if !hmac.Equal(key, d.key) {
		slog.Default().Warn("password verification failed")
		return ErrInvalidPasswordHash
	}
	return nil
}

// hashWithSalt hashes plain with a fixed salt. It is used by HashPassword
// (which uses a random salt) and by the tests (which use a deterministic
// salt).
func hashWithSalt(plain string, salt []byte) string {
	key := argon2.IDKey([]byte(plain), salt, argon2Time, argon2Memory, uint8(argon2Threads), argon2KeyLen)
	return encodeArgon2(key, salt)
}

// encodeArgon2 returns the hash in the PHC format:
// "$argon2id$v=19$m=65536,t=3,p=4$<base64(salt)>$<base64(key)>".
func encodeArgon2(key, salt []byte) string {
	return fmt.Sprintf("$%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2idID, argon2Version, argon2Memory, argon2Time, argon2Threads,
		base64.StdEncoding.EncodeToString(salt),
		base64.StdEncoding.EncodeToString(key))
}

// decodeArgon2Hash parses a hash encoded by encodeArgon2.
func decodeArgon2Hash(hash string) (*argHash, error) {
	// The PHC string always starts with "$", so a split on "$" yields a
	// leading empty field: $argon2id$v=19$m=...$salt$key -> ["", "argon2id", ...].
	fields := strings.SplitN(hash, "$", 6)
	if len(fields) != 6 || fields[1] != argon2idID {
		return nil, errInvalidArgon2Format
	}
	version, err := parseArgon2Version(fields[2])
	if err != nil {
		return nil, errInvalidArgon2Format
	}
	params, err := parseArgon2Params(fields[3])
	if err != nil {
		return nil, errInvalidArgon2Format
	}
	salt, err := base64.StdEncoding.DecodeString(fields[4])
	if err != nil || len(salt) == 0 {
		return nil, errInvalidArgon2Format
	}
	key, err := base64.StdEncoding.DecodeString(fields[5])
	if err != nil || len(key) != argon2KeyLen {
		return nil, errInvalidArgon2Format
	}
	return &argHash{
		version: version,
		memory:  params.memory,
		time:    params.time,
		threads: params.threads,
		keyLen:  argon2KeyLen,
		salt:    salt,
		key:     key,
	}, nil
}

type argon2Params struct {
	memory  uint32
	time    uint32
	threads uint32
}

// parseArgon2Params parses the "m=65536,t=3,p=4" part of a hash.
func parseArgon2Params(s string) (argon2Params, error) {
	var p argon2Params
	for _, part := range strings.Split(s, ",") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return p, errInvalidArgon2Format
		}
		n, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return p, errInvalidArgon2Format
		}
		switch key {
		case "m":
			p.memory = uint32(n)
		case "t":
			p.time = uint32(n)
		case "p":
			p.threads = uint32(n)
		default:
			return p, errInvalidArgon2Format
		}
	}
	if p.memory == 0 || p.time == 0 || p.threads == 0 {
		return p, errInvalidArgon2Format
	}
	return p, nil
}

// parseArgon2Version parses the "v=19" part of a hash.
func parseArgon2Version(s string) (int, error) {
	prefix, after, ok := strings.Cut(s, "=")
	if !ok || prefix != "v" {
		return 0, errInvalidArgon2Format
	}
	n, err := strconv.Atoi(after)
	if err != nil || n != argon2Version {
		return 0, errInvalidArgon2Format
	}
	return n, nil
}
