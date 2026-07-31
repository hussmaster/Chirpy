package main

import (
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"sort"
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
		respondWithError(w, 401, "unauthorized token")
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
	// Look for optional author_id parameter in GET request
	authorID := r.URL.Query().Get("author_id")
	// Look for optional sort parameter
	sorting := r.URL.Query().Get("sort")

	// Declare variables outside of if/else block
	var allChirps []database.Chirp
	var err error
	log.Printf("header: %v\n", r.URL)
	if authorID != "" {
		// Parse authorID string to uuid
		authorUUID, err := uuid.Parse(authorID)
		if err != nil {
			respondWithError(w, 500, "error parsing author into uuid")
			log.Printf("error parsing uuid from author id: %v\n", err)
			return
		}
		allChirps, err = cfg.db.GetAllChirpsByID(r.Context(), authorUUID)
		if err != nil {
			respondWithError(w, 500, "error retrieving chirps")
			log.Printf("error retrieving chrips for %v: %v\n", authorID, err)
			return
		}

	} else {
		//Get db rows in var
		allChirps, err = cfg.db.GetAllChirps(r.Context())
		if err != nil {
			respondWithError(w, 500, "error retrieving chirps")
			log.Printf("error retrieving chirps: %v\n", err)
			return
		}
	}
	// Sort the chirps
	// Default is ASC
	switch sorting {
	case "desc":
		sort.Slice(allChirps, func(i, j int) bool { return allChirps[j].CreatedAt.Before(allChirps[i].CreatedAt) })
	case "asc":
		sort.Slice(allChirps, func(i, j int) bool { return allChirps[i].CreatedAt.Before(allChirps[j].CreatedAt) })
	case "":
		sort.Slice(allChirps, func(i, j int) bool { return allChirps[i].CreatedAt.Before(allChirps[j].CreatedAt) })
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

func (cfg *apiConfig) deleteChirp(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
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

	// Parse chirp ID into uuid from the URL
	chirpUUID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, 500, "something went wrong")
		log.Printf("unable to parse chirpID string into uuid: %v\n", err)
		return
	}
	// Check if chirp exists in the database
	validChirp, err := cfg.db.GetOneChirp(r.Context(), chirpUUID)
	if err != nil {
		respondWithError(w, 404, "chirp is not found")
		log.Printf("chirp not found in the db: %v\n", err)
		return
	}

	if validChirp.UserID != validatedUUID {
		respondWithError(w, 403, "unauthorized")
		log.Printf("unauthorized chirp deletion attempt: %v\n", err)
		return
	}

	// Delete chirp
	err = cfg.db.DeleteChirp(r.Context(), chirpUUID)
	if err != nil {
		respondWithError(w, 403, "unauthorized")
		log.Printf("unauthorzied chirp deletion attempt: %v\n", err)
		return
	}
	// Successful delete
	w.WriteHeader(204)

}
