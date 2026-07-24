package auth

import (
	"testing"
)

func TestCorrectPasswordHash(t *testing.T) {
	password := "password01"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to create hash: %v\n", err)
	}

	match, err := CheckPasswordHash(password, hash)
	if err != nil {
		t.Fatalf("failed to compare password and hash: %v\n", err)
	}

	if !match {
		t.Errorf("expected password to match the hash")
	}
}

func TestIncorrectPassword(t *testing.T) {
	password := "password01"
	wrong := "password02"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to create hash: %v\n", err)
	}

	match, err := CheckPasswordHash(wrong, hash)
	if err != nil {
		t.Fatalf("failed to compare the password and hash: %v\n", err)
	}
	if match {
		t.Errorf("expected password to not match the hash")
	}
}

func TestInvalidHashFormat(t *testing.T) {
	_, err := CheckPasswordHash("testPassword", "invalid")
	if err == nil {
		t.Errorf("expected error for malformed hash string, got nil")
	}
}
