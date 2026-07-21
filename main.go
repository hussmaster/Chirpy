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

	"github.com/hussmaster/Chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Struct to keep track of web server hits
type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
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
	cfg.fileserverHits.Store(0)
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

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
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
		db: dbQueries,
	}
	//index.html is in the app folder, strip out app prefix before sending to fileserver
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("app")))))
	//Handles /healthz by forwarding to healthCheck func
	mux.HandleFunc("GET /api/healthz", healthCheck)
	mux.HandleFunc("GET /admin/metrics", apiCfg.getFileServerHits)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetFileServerHits)
	mux.HandleFunc("POST /api/validate_chirp", validateChirp)
	//Serve website
	http.ListenAndServe(server.Addr, server.Handler)

}
