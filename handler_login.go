package main

import (
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
		Email            string        `json:"email"`
		Password         string        `json:"password"`
		ExpiresInSeconds time.Duration `json:"expires_in_seconds"`
	}
	type responseBody struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
		Token     string    `json:"token"`
	}

	decoder := json.NewDecoder(r.Body)
	// Set default expiry to 1 hour
	params := requestBody{
		ExpiresInSeconds: time.Duration(3600) * time.Second,
	}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
		log.Printf("error decoding params: %v\n", err)
		return
	}
	// Prevent tokens expiry longer than 1 hour
	if params.ExpiresInSeconds > 3600 {
		params.ExpiresInSeconds = time.Duration(3600) * time.Second
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
	userJWT, err := auth.MakeJWT(user.ID, cfg.serversecret, time.Duration(params.ExpiresInSeconds))
	if err != nil {
		respondWithError(w, 400, "something went wrong")
		log.Printf("error making the jwt: %v\n", err)
		return
	}
	returnBody := responseBody{}
	returnBody.ID = user.ID
	returnBody.CreatedAt = user.CreatedAt
	returnBody.UpdatedAt = user.UpdatedAt
	returnBody.Email = user.Email
	returnBody.Token = userJWT
	err = respondWithJSON(w, 200, returnBody)
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
	}
}
