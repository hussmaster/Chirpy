package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
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

func TestValidJWT(t *testing.T) {
	testUUID := uuid.New()
	testSecret := "car-horse-battery-staple"
	expiresIn := 60 * time.Second

	jwtToken, err := MakeJWT(testUUID, testSecret, expiresIn)
	if err != nil {
		t.Fatalf("unexpected error: %v\n", err)
	}

	checkUUID, err := ValidateJWT(jwtToken, testSecret)
	if err != nil {
		t.Fatalf("unexpected error: %v\n", err)
	}

	if testUUID != checkUUID {
		t.Errorf("got %v, want %v\n", testUUID, checkUUID)
	}
}

func TestInvalidJWT(t *testing.T) {
	testUUID := uuid.New()
	testSecret := "car-horse-battery-staple"
	expiresIn := 60 * time.Second

	jwtToken, err := MakeJWT(testUUID, testSecret, expiresIn)
	if err != nil {
		t.Fatalf("unexpected error: %v\n", err)
	}
	secondSecret := "car-horse-house-staple"
	_, err = ValidateJWT(jwtToken, secondSecret)
	if err == nil {
		t.Fatalf("unexpected error: %v\n", err)
	}

}

func TestExpiredJWT(t *testing.T) {
	testUUID := uuid.New()
	testSecret := "car-horse-battery-staple"
	expiresIn := 1 * time.Second

	jwtToken, err := MakeJWT(testUUID, testSecret, expiresIn)
	if err != nil {
		t.Fatalf("unexpected error: %v\n", err)
	}
	time.Sleep(2 * time.Second)
	_, err = ValidateJWT(jwtToken, testSecret)
	if err == nil {
		t.Fatalf("unexpected error: %v\n", err)
	}

}

func TestBearerToken(t *testing.T) {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer bingbong")

	result, err := GetBearerToken(headers)
	if err != nil {
		t.Fatalf("unexepected error:%v \n", err)
	}
	if result != "bingbong" {
		t.Fatalf("expected json, got %s", result)
	}
}
