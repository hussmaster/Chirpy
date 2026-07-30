package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hussmaster/Chirpy/internal/auth"
	"github.com/hussmaster/Chirpy/internal/database"
)

// Function to add users to database
func (cfg *apiConfig) addUser(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	type requestBody struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type responseBody struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := requestBody{}
	// Decode into the params struct
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
		return
	}
	// Create the user, passing in the HTTP context and the email
	returnBody := responseBody{}
	// Hash password from response/params struct
	hashedPass, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
		log.Printf("error hashing password: %v\n", err)
		return
	}
	// Create database user params struct to pass into CreateUser
	userParams := database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPass,
	}
	user, err := cfg.db.CreateUser(r.Context(), userParams)
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
		log.Printf("error creating user: %v\n", err)
		return
	}
	//Populate struct for returning
	returnBody.ID = user.ID
	returnBody.CreatedAt = user.CreatedAt
	returnBody.UpdatedAt = user.UpdatedAt
	returnBody.Email = user.Email
	err = respondWithJSON(w, 201, returnBody)
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
	}
}

// Function to login
func (cfg *apiConfig) userLogin(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	type requestBody struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type responseBody struct {
		ID           uuid.UUID `json:"id"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		Email        string    `json:"email"`
		Token        string    `json:"token"`
		RefreshToken string    `json:"refresh_token"`
	}

	decoder := json.NewDecoder(r.Body)
	params := requestBody{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
		log.Printf("error decoding params: %v\n", err)
		return
	}

	user, err := cfg.db.UserLookup(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, 401, "incorrect email or password")
		log.Printf("error on user lookup: %v\n", err)
		return
	}
	userPasswordCheck, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		respondWithError(w, 401, "incorrect email or password")
		log.Printf("error on hashing check: %v\n", err)
		return
	}
	if userPasswordCheck != true {
		respondWithError(w, 401, "incorrect email or password")
		log.Print("password hashes did not match")
		return
	}
	// Create JWT and pass it along on the login process
	// make sure to pass in server secret environment variable
	// expire tokens after 1 hour
	userJWT, err := auth.MakeJWT(user.ID, cfg.serversecret, time.Duration(3600)*time.Second)
	if err != nil {
		respondWithError(w, 400, "something went wrong")
		log.Printf("error making the jwt: %v\n", err)
		return
	}
	// Generate refresh token
	userRefreshToken := auth.MakeRefreshToken()
	userRefreshParams := database.RefreshTokenParams{
		Token:  userRefreshToken,
		UserID: user.ID,
	}
	// Save refresh token into refresh_tokens database
	cfg.db.RefreshToken(r.Context(), userRefreshParams)
	returnBody := responseBody{}
	returnBody.ID = user.ID
	returnBody.CreatedAt = user.CreatedAt
	returnBody.UpdatedAt = user.UpdatedAt
	returnBody.Email = user.Email
	returnBody.Token = userJWT
	returnBody.RefreshToken = userRefreshToken
	err = respondWithJSON(w, 200, returnBody)
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
	}
}

func (cfg *apiConfig) refreshAccessToken(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	refreshHeader, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 400, "no refresh token in header")
		log.Printf("error getting refresh token from header: %v\n", err)
		return
	}

	user, err := cfg.db.GetUserFromRefreshToken(r.Context(), refreshHeader)
	if err != nil {
		respondWithError(w, 404, "something went wrong")
		log.Printf("db query for userID by refresh token failed: %v\n", err)
		return
	}
	// Make sure the refresh token isn't revoked
	if user.RevokedAt.Valid {
		respondWithError(w, 401, "refresh token is revoked")
		log.Printf("refresh token is revoked: %v\n", user.RevokedAt.Time)
		return
	}
	// Make sure the refresh token isn't expired
	isExpired := time.Now().After(user.ExpiresAt)
	if isExpired {
		respondWithError(w, 401, "refresh token is expired")
		log.Printf("refresh token is expired: %v\n", err)
		return
	}
	newAccessTok, err := auth.MakeJWT(user.UserID, cfg.serversecret, time.Duration(3600)*time.Second)
	if err != nil {
		respondWithError(w, 500, "failed to generate JWT")
		log.Printf("failed to make the jwt: %v\n", err)
		return
	}
	// Return new access token
	type responseBody struct {
		Token string `json:"token"`
	}
	returnBody := responseBody{}
	returnBody.Token = newAccessTok
	// Send back new access token
	respondWithJSON(w, 200, returnBody)
}

// Revoke refresh token
func (cfg *apiConfig) revokeRefreshToken(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	refreshHeader, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 400, "no refresh token in header")
		log.Printf("error getting refresh token: %v\n", err)
		return
	}

	// Wrap time.Now in sql NullTime
	revokeParams := database.RevokeRefreshTokenParams{
		UpdatedAt: time.Now(),
		RevokedAt: sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		},
		Token: refreshHeader,
	}
	err = cfg.db.RevokeRefreshToken(r.Context(), revokeParams)
	if err != nil {
		respondWithError(w, 500, "failed to revoke refresh token")
		log.Printf("failed to revoke refresh token: %v\n", err)
		return
	}
	// Send back only status code 204
	w.WriteHeader(204)
}

// Function to allow updating of email of password
func (cfg *apiConfig) updateUser(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	type requestBody struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type responseBody struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}
	// Extract access token from header
	accessToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "no access token")
		log.Printf("no access token in header: %v\n", err)
		return
	}
	// Get UUID from access token
	validatedUUID, err := auth.ValidateJWT(accessToken, cfg.serversecret)
	if err != nil {
		respondWithError(w, 401, "unauthorized token")
		log.Printf("JWT was not validated: %v\n", err)
		return
	}
	decoder := json.NewDecoder(r.Body)
	params := requestBody{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "something went wrong")
		log.Printf("error decoding params: %v\n", err)
		return
	}
	// Lookup the user by validated UUID
	userDB, err := cfg.db.UserLookupByID(r.Context(), validatedUUID)
	if err != nil {
		respondWithError(w, 404, "user not found")
		log.Printf("user ID not in database: %v\n", err)
		return
	}
	// Hash the new password from the user
	newHashedPass, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, 500, "unable to hash password")
		log.Printf("unable to hash password: %v\n", err)
		return
	}

	userupdateParams := database.UpdateUserParams{
		Email:          params.Email,
		HashedPassword: newHashedPass,
		UpdatedAt:      time.Now(),
		ID:             validatedUUID,
	}
	// Update the user with the above values
	err = cfg.db.UpdateUser(r.Context(), userupdateParams)
	if err != nil {
		respondWithError(w, 500, "error updating user")
		log.Printf("error updating user in database: %v\n", err)
		return
	}

	returnBody := responseBody{
		ID:        validatedUUID,
		CreatedAt: userDB.CreatedAt,
		UpdatedAt: userupdateParams.UpdatedAt,
		Email:     params.Email,
	}

	err = respondWithJSON(w, 200, returnBody)
	if err != nil {
		respondWithError(w, 401, "something went wrong")
		log.Printf("error sending returnbody: %v\n", err)
	}

}
