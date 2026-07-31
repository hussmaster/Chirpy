package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

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

// Gets TOKENSTRING from the auth header
// Auth header should look like
// Bearer <TOKEN>
func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("Authorization header is empty")
	}
	// Strips out Bearer from Authorization header
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	return tokenString, nil

}

// Creates a refresh token for a user
func MakeRefreshToken() string {
	// Make a 32 byte random data string
	refreshTok := make([]byte, 32)
	rand.Read(refreshTok)
	// Conver and encode to Hex string
	encodedRefresh := hex.EncodeToString(refreshTok)
	return encodedRefresh
}

// Function to get API KEY from header with format being Authorization: ApiKey <Key>
func GetAPIKey(headers http.Header) (string, error) {
	apiHeader := headers.Get("Authorization")
	if apiHeader == "" {
		return "", fmt.Errorf("Authorization header is empty")
	}
	apiString := strings.TrimPrefix(apiHeader, "ApiKey ")
	return apiString, nil
}
