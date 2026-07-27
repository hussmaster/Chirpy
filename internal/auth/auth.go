package auth

import (
	"log"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var hmacSecret []byte

// Hashes plain text string to hashed value
func HashPassword(password string) (string, error) {
	// Hash using defaults
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}
	return hash, nil

}

// Checks string against hash in database
func CheckPasswordHash(password, hash string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, err
	}

	log.Printf("Match: %v\n", match)
	return match, nil
}

// Function to make Json Web Token, JWT
func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	newTok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn).UTC()),
		Subject:   userID.String(),
	})
	tokenString, err := newTok.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// Function to validate Json Web Token
func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	// Create empty struct to decode into
	claims := jwt.RegisteredClaims{}
	// Takes in token string, claims reference, uses a function for the verification key using the tokenSecret
	// Provided no error, can then reference data from the claims struct
	// Keep first variable if extra data is needed, token.Valid, token.Header etc
	_, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.Nil, err

	}
	parsedUUID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, err
	}
	return parsedUUID, nil
}
