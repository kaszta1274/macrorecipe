package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/kaszta1274/macrorecipe/internal/api"
	"github.com/kaszta1274/macrorecipe/internal/database"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL must be set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT must be set")
	}

	tokenSecret := os.Getenv("TOKEN_SECRET")
	if tokenSecret == "" {
		log.Fatal("TOKEN_SECRET must be set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("could not connect to database: %s", err)
	}
	defer db.Close()

	dbQueries := database.New(db)

	cfg := api.ApiConfig{
		DB:          dbQueries,
		TokenSecret: tokenSecret,
	}

	r := cfg.Router()

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}
	log.Fatal(srv.ListenAndServe())
}
