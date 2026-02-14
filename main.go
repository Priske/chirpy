package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/Priske/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("warning: no .env file found (continuing)")
	}
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL is not set")
	}
	secret := os.Getenv("SECRET")
	if secret == "" {
		log.Fatal("Secret jwl is not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("error opening DB: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("error connecting to DB: %v", err)
	}
	dbQueries := database.New(db)

	mux := http.NewServeMux()

	cfg := &apiConfig{
		dbQueries: dbQueries,
		platform:  os.Getenv("PLATFORM"),
		polkaKey:  os.Getenv("POLKA_KEY"),
	}
	// 1) Readiness endpoint: /healthz (any method)
	mux.HandleFunc("GET /api/healthz", handlerReadiness)
	mux.HandleFunc("PUT /api/users", requireMethod(http.MethodPut, cfg.handlerUpdateLogin))
	mux.HandleFunc("POST /api/login", requireMethod(http.MethodPost, cfg.handlerLogin))
	mux.HandleFunc("POST /api/refresh", requireMethod(http.MethodPost, cfg.handlerRefresh))
	mux.HandleFunc("POST /api/revoke", requireMethod(http.MethodPost, cfg.handlerRevoke))
	//mux.HandleFunc("POST /api/validate_chirp", requireMethod(http.MethodPost, handlerValidateChirp))
	mux.HandleFunc("POST /api/chirps", requireMethod(http.MethodPost, cfg.handlerCreateChirp))
	mux.HandleFunc("GET /api/chirps", requireMethod(http.MethodGet, cfg.handlerFetchAllChirps))
	mux.HandleFunc("GET /api/chirps/{chirpID}", requireMethod(http.MethodGet, cfg.handlerGetChirpByID))
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", requireMethod(http.MethodDelete, cfg.handlerDeleteChirpByID))
	mux.HandleFunc("GET /admin/metrics", cfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", cfg.handlerReset)
	mux.HandleFunc("POST /api/users", requireMethod(http.MethodPost, cfg.handlerUsers))

	mux.HandleFunc("POST /api/polka/webhooks", requireMethod(http.MethodPost, cfg.handlerWebhooks))

	// 2) File server moved to /app/
	fileServer := http.FileServer(http.Dir("."))
	mux.Handle("/app/", cfg.middlewareMetricsInc(middlewareLog(http.StripPrefix("/app/", fileServer))))

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Printf("Serving on port: %s\n", server.Addr)
	log.Fatal(server.ListenAndServe())
}

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
	jwtSecret      string
	polkaKey       string
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}

func toAPIUser(u database.User) apiUser {
	return apiUser{
		ID: u.ID, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt, Email: u.Email, IsChirpyRed: u.IsChirpyRed,
	}
}
