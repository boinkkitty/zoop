// Package password hashes and checks passwords with Argon2id, stored as PHC strings.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	memoryKiB = 19 * 1024
	time      = 2
	threads   = 1
	saltLen   = 16
	keyLen    = 32

	// MaxBytes caps password length so nobody can make the server hash megabytes.
	MaxBytes = 256

	// maxConcurrent bounds how many hashes run at once: each one holds 19 MiB.
	maxConcurrent = 4
)

// ErrTooLong means the password is longer than MaxBytes.
var ErrTooLong = errors.New("password: too long")

var slots = make(chan struct{}, maxConcurrent)

// idKey runs Argon2id, waiting for a free slot first.
func idKey(plain string, salt []byte, t, m uint32, p uint8, n uint32) []byte {
	slots <- struct{}{}
	defer func() { <-slots }()
	return argon2.IDKey([]byte(plain), salt, t, m, p, n)
}

// Hash returns an Argon2id PHC string for plain.
// It returns ErrTooLong for passwords over MaxBytes.
func Hash(plain string) (string, error) {
	if len(plain) > MaxBytes {
		return "", ErrTooLong
	}
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := idKey(plain, salt, time, memoryKiB, threads, keyLen)
	enc := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memoryKiB, time, threads, enc.EncodeToString(salt), enc.EncodeToString(key)), nil
}

// ErrMalformedHash means a stored hash isn't an Argon2id PHC string this package can read.
var ErrMalformedHash = errors.New("password: malformed hash")

// Verify reports whether plain matches hash, comparing in constant time.
// It reads the parameters from hash, so older hashes keep working if they change.
func Verify(hash, plain string) (bool, error) {
	if len(plain) > MaxBytes {
		return false, ErrTooLong
	}
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return false, ErrMalformedHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, ErrMalformedHash
	}
	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil || m == 0 || t == 0 || p == 0 {
		return false, ErrMalformedHash
	}
	enc := base64.RawStdEncoding
	salt, err := enc.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return false, ErrMalformedHash
	}
	want, err := enc.DecodeString(parts[5])
	if err != nil || len(want) == 0 {
		return false, ErrMalformedHash
	}
	got := idKey(plain, salt, t, m, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
