package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/kaszta1274/macrorecipe/internal/database"
	"github.com/kaszta1274/macrorecipe/internal/seeder"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL must be set")
	}

	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		log.Fatal("ADMIN_EMAIL must be set")
	}

	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		log.Fatal("ADMIN_PASSWORD must be set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("could not connect to database: %s", err)
	}
	defer db.Close()

	dbQueries := database.New(db)

	if err := seeder.RunSeeders(dbQueries, adminEmail, adminPassword); err != nil {
		log.Fatalf("seeding error: %s", err)
	}

	log.Println("Seeding completed!")
}
