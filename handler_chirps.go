package main

import (
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hussmaster/Chirpy/internal/auth"
	"github.com/hussmaster/Chirpy/internal/database"
)

// Posts chirp into database
func (cfg *apiConfig) postChirp(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	type requestBody struct {
		Body string `json:"body"`
	}
	type responseBody struct {
		Id        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserId    uuid.UUID `json:"user_id"`
	}
	authHeader, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 400, "no authorization header")
		log.Printf("error getting auth header from http headers: %v\n", err)
		return
	}
	validatedUUID, err := auth.ValidateJWT(authHeader, cfg.serversecret)
	if err != nil {
		respondWithError(w, 400, "unauthorized token")
		log.Printf("JWT was not validated: %v\n", err)
		return
	}
	decoder := json.NewDecoder(r.Body)
	//Create database postchirp parameter struct to decode into
	params := requestBody{}
	chirpParams := database.PostChirpParams{}
	err = decoder.Decode(&params)

	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		log.Printf("something went wrong: %v\n", err)
		return
	}
	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		log.Print("Chirp is too long")
		return
	} else if len(params.Body) < 1 {
		respondWithError(w, 400, "Chirp is too short")
		log.Print("Chirp is too short")
		return
	} else {

		//Clean body text for bad words
		cleaned := cleanText(params.Body)
		params.Body = cleaned
		// Post the chirp into the database
		chirpParams.Body = cleaned
		chirpParams.UserID = validatedUUID
		userPost, err := cfg.db.PostChirp(r.Context(), chirpParams)
		if err != nil {
			respondWithError(w, 500, "error posting chirp")
			log.Printf("error posting chrip: %v\n", err)
			return
		}
		//Create return json struct
		returnBody := responseBody{}
		returnBody.Id = userPost.ID
		returnBody.CreatedAt = userPost.CreatedAt
		returnBody.UpdatedAt = userPost.UpdatedAt
		returnBody.Body = userPost.Body
		returnBody.UserId = userPost.UserID

		respondWithJSON(w, 201, returnBody)
	}

}

// Function to clean text for profane words
func cleanText(body string) string {
	tempText := []string{}
	splitStrings := strings.Split(body, " ")
	profane := []string{"kerfuffle", "sharbert", "fornax"}
	for _, str := range splitStrings {
		tempLow := strings.ToLower(str)
		if slices.Contains(profane, tempLow) {
			tempText = append(tempText, "****")
		} else {
			tempText = append(tempText, str)
		}
	}
	replacementText := strings.Join(tempText, " ")
	return replacementText // change
}

// Function to retrieve all chirps from database
func (cfg *apiConfig) getAllChirps(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	// No need to ingest json here
	type returnBody struct {
		Id        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserId    uuid.UUID `json:"user_id"`
	}
	respBody := []returnBody{}
	//Get db rows in var
	allChirps, err := cfg.db.GetAllChirps(r.Context())
	if err != nil {
		respondWithError(w, 500, "error retrieving chirps")
		log.Printf("error retrieving chirps: %v\n", err)
		return
	}
	// Loop through dbRows, create temp struct and append to main array respBody struct
	for _, dbRow := range allChirps {
		tempBody := returnBody{
			Id:        dbRow.ID,
			CreatedAt: dbRow.CreatedAt,
			UpdatedAt: dbRow.UpdatedAt,
			Body:      dbRow.Body,
			UserId:    dbRow.UserID,
		}
		respBody = append(respBody, tempBody)
	}
	respondWithJSON(w, 200, respBody)
}

func (cfg *apiConfig) getOneChirp(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	type returnBody struct {
		Id        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserId    uuid.UUID `json:"user_id"`
	}
	respBody := returnBody{}
	chirpUUID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, 500, "something went wrong")
		log.Printf("unable to parse request string into uuid type: %v\n", err)
		return
	}
	oneChirp, err := cfg.db.GetOneChirp(r.Context(), chirpUUID)
	if err != nil {
		respondWithError(w, 404, "something went wrong")
		log.Printf("db query for single chirp failed: %v\n", err)
		return
	}
	respBody.Id = oneChirp.ID
	respBody.CreatedAt = oneChirp.CreatedAt
	respBody.UpdatedAt = oneChirp.UpdatedAt
	respBody.Body = oneChirp.Body
	respBody.UserId = oneChirp.UserID
	respondWithJSON(w, 200, respBody)
}
