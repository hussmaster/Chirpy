package auth

import (
	"log"

	"github.com/alexedwards/argon2id"
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

	log.Printf("Match: %v\n", match)
	return match, nil
}
