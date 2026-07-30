package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
)

func (cfg *apiConfig) chirpyredEnable(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// Webhook body
	type requestBody struct {
		Event string `json:"event"`
		Data  struct {
			UserID uuid.UUID `json:"user_id"`
		} `json:"data"`
	}

	decoder := json.NewDecoder(r.Body)
	params := requestBody{}
	err := decoder.Decode(&params)
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
