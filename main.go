package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/hussmaster/Chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Struct to keep track of web server hits
type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
}

// User struct for adding to database
type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

// Displays site status code
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type: text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK) // Set 200
	w.Write([]byte("OK"))        // Write byte for work OK
}

// Middleware for adding to site hits
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

// Displays number of site hits
func (cfg *apiConfig) getFileServerHits(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits.Load())))
}

// Resets hit counter
func (cfg *apiConfig) resetFileServerHits(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type: text/plain; charset=utf-8")
		w.WriteHeader(http.StatusForbidden) // Set 403
		w.Write([]byte("403 Forbidden"))
	} else {
		cfg.fileserverHits.Store(0)
		_, err := cfg.db.DeleteUsers(r.Context())
		if err != nil {
			log.Fatalf("error cleaing users table: %v\n", err)
		}
	}

}

// Function that responds with JSON
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) error {
	response, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	w.Write(response)
	return nil
}

// Function that checks JSON for errors
func respondWithError(w http.ResponseWriter, code int, msg string) error {
	return respondWithJSON(w, code, map[string]string{"error": msg})
}

// Make sure Chirp is no more than 140 characters and has valid json
/*
func validateChirp(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	type requestBody struct {
		Body string `json:"body"`
	}
	type responseBody struct {
		CleanedBody string `json:"cleaned_body"`
	}
	decoder := json.NewDecoder(r.Body)
	params := requestBody{}
	//Change to output cleaned body text
	err := decoder.Decode(&params)
	cleaned := cleanText(params.Body)
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
	}
	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
	} else {
		respondWithJSON(w, 200, responseBody{CleanedBody: cleaned})
	}
}
*/

// Posts chirp into database
func (cfg *apiConfig) postChirp(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	type requestBody struct {
		Body   string    `json:"body"`
		UserId uuid.UUID `json:"user_id"`
	}
	type responseBody struct {
		Id        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserId    uuid.UUID `json:"user_id"`
	}
	decoder := json.NewDecoder(r.Body)
	//Create database postchirp parameter struct to decode into
	params := requestBody{}
	chirpParams := database.PostChirpParams{}
	err := decoder.Decode(&params)

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
		chirpParams.UserID = params.UserId
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

// Function to add users to database
func (cfg *apiConfig) addUser(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	type requestBody struct {
		Email string `json:"email"`
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
	}
	//Create the user, passing in the HTTP context and the email
	returnBody := responseBody{}
	user, err := cfg.db.CreateUser(r.Context(), params.Email)
	if err != nil {
		log.Fatalf("error creating user: %v\n", err)
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

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	if dbURL == "" {
		log.Fatalf("DBURL is an empty string \n")
	}
	//open DB connection
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("error opening database connection %v\n", err)
	}
	// Create sqlc connection for queries
	dbQueries := database.New(db)
	//Create http mux
	mux := http.NewServeMux()
	//http server struct
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	apiCfg := apiConfig{
		// assign db connection to sqlc connection
		db:       dbQueries,
		platform: platform,
	}
	//index.html is in the app folder, strip out app prefix before sending to fileserver
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("app")))))
	//Handles /healthz by forwarding to healthCheck func
	mux.HandleFunc("GET /api/healthz", healthCheck)
	mux.HandleFunc("GET /admin/metrics", apiCfg.getFileServerHits)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetFileServerHits)
	//mux.HandleFunc("POST /api/validate_chirp", validateChirp)
	mux.HandleFunc("POST /api/chirps", apiCfg.postChirp)
	mux.HandleFunc("POST /api/users", apiCfg.addUser)
	mux.HandleFunc("GET /api/chirps", apiCfg.getAllChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getOneChirp)
	//Serve website
	http.ListenAndServe(server.Addr, server.Handler)

}
