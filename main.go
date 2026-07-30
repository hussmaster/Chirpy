package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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
	serversecret   string
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

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	serversecret := os.Getenv("SERVERSECRET")
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
		db:           dbQueries,
		platform:     platform,
		serversecret: serversecret,
	}
	//index.html is in the app folder, strip out app prefix before sending to fileserver
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("app")))))
	//Handles /healthz by forwarding to healthCheck func
	mux.HandleFunc("GET /api/healthz", healthCheck)
	mux.HandleFunc("GET /admin/metrics", apiCfg.getFileServerHits)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetFileServerHits)
	mux.HandleFunc("POST /api/chirps", apiCfg.postChirp)
	mux.HandleFunc("POST /api/users", apiCfg.addUser)
	mux.HandleFunc("GET /api/chirps", apiCfg.getAllChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getOneChirp)
	mux.HandleFunc("POST /api/login", apiCfg.userLogin)
	mux.HandleFunc("POST /api/refresh", apiCfg.refreshAccessToken)
	mux.HandleFunc("POST /api/revoke", apiCfg.revokeRefreshToken)
	mux.HandleFunc("PUT /api/users", apiCfg.updateUser)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.deleteChirp)
	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.chirpyredEnable)
	//Serve website
	http.ListenAndServe(server.Addr, server.Handler)

}
