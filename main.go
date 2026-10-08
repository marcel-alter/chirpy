package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/marcel-alter/chirpy.git/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
}

func main() {
	ctx := context.Background()
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	db, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to postgreSQL database with '%s' Error: %v", &dbURL, err)
	}
	queries := database.New(db)
	defer db.Close(ctx)
	const filepathRoot = "."
	const port = "8080"
	var cfg apiConfig = apiConfig{dbQueries: queries, platform: platform}
	mux := http.NewServeMux()

	fileHandler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", cfg.middlewareMetricsInc(fileHandler))
	mux.HandleFunc("GET /api/healthz", handlerStatus)
	mux.HandleFunc("POST /api/chirps", cfg.handlerAddChirp)
	mux.HandleFunc("GET /api/chirps/{chirpID}", cfg.handlerGetOneChirp)
	mux.HandleFunc("GET /api/chirps", cfg.handlerGetAllChirps)
	mux.HandleFunc("GET /admin/metrics", cfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", cfg.handlerReset)
	mux.HandleFunc("POST /api/users", cfg.handlerPostUser)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())
}
