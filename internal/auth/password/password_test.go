package password_test

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/boinkkitty/zoop/internal/auth/password"
)

func TestHashIsAnArgon2idPHCStringWithTheAgreedParameters(t *testing.T) {
	hash, err := password.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	// $argon2id$v=19$m=19456,t=2,p=1$<salt>$<key>
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Fatalf("hash %q has %d $-parts, want 6", hash, len(parts))
	}
	if parts[1] != "argon2id" || parts[2] != "v=19" || parts[3] != "m=19456,t=2,p=1" {
		t.Errorf("hash header = %q, want argon2id v=19 m=19456,t=2,p=1", strings.Join(parts[1:4], "$"))
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) != 16 {
		t.Errorf("salt %q decodes to %d bytes (err %v), want 16", parts[4], len(salt), err)
	}
}

func TestVerifyAcceptsOnlyThePasswordThatWasHashed(t *testing.T) {
	hash, err := password.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := password.Verify(hash, "correct horse battery staple"); err != nil || !ok {
		t.Errorf("Verify(right password) = %v, %v; want true, nil", ok, err)
	}
	if ok, err := password.Verify(hash, "Correct horse battery staple"); err != nil || ok {
		t.Errorf("Verify(wrong password) = %v, %v; want false, nil", ok, err)
	}
}

func TestVerifyRejectsAMalformedHash(t *testing.T) {
	for _, bad := range []string{"", "plain", "$argon2id$v=19$m=19456,t=2,p=1$!!$!!", "$bcrypt$x$y$z$w"} {
		if ok, err := password.Verify(bad, "anything"); err == nil || ok {
			t.Errorf("Verify(%q) = %v, %v; want false and an error", bad, ok, err)
		}
	}
}

func TestPasswordsLongerThanTheCapAreRefused(t *testing.T) {
	atCap := strings.Repeat("a", password.MaxBytes)
	if _, err := password.Hash(atCap); err != nil {
		t.Errorf("Hash(%d bytes) = %v, want it accepted", password.MaxBytes, err)
	}
	tooLong := atCap + "a"
	if _, err := password.Hash(tooLong); !errors.Is(err, password.ErrTooLong) {
		t.Errorf("Hash(%d bytes) error = %v, want ErrTooLong", len(tooLong), err)
	}
	hash, _ := password.Hash("short")
	if ok, err := password.Verify(hash, tooLong); ok || !errors.Is(err, password.ErrTooLong) {
		t.Errorf("Verify(too long) = %v, %v; want false, ErrTooLong", ok, err)
	}
}
