package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/hussmaster/Chirpy/internal/auth"
)

// Function to enable Chirpy red on an account
func (cfg *apiConfig) chirpyredEnable(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	// Get POLKA API Key
	polkaKey := os.Getenv("POLKA_KEY")
	// Webhook body
	type requestBody struct {
		Event string `json:"event"`
		Data  struct {
			UserID uuid.UUID `json:"user_id"`
		} `json:"data"`
	}
	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		respondWithError(w, 401, "no api key")
		log.Printf("no api key in header: %v\n", err)
		return
	}
	if polkaKey != apiKey {
		respondWithError(w, 401, "invalid api key")
		log.Printf("api key and polka api key do not match: %v\n", err)
		return
	}
	decoder := json.NewDecoder(r.Body)
	params := requestBody{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 400, "something went wrong")
		log.Printf("unable to decode params: %v\n", err)
		return
	}

	// Return status code 204 if user.upgraded isn't the event
	if params.Event != "user.upgraded" {
		w.WriteHeader(204)
		return
	}

	// Enable chirpy red for the user
	err = cfg.db.ChirpyRedEnable(r.Context(), params.Data.UserID)
	if err != nil {
		respondWithError(w, 404, "user not found")
		log.Printf("user not found in database: %v\n", err)
		return
	}
	w.WriteHeader(204)
}
