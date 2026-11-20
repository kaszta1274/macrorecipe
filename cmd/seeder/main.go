package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/kaszta1274/macrorecipe/internal/database"
	"github.com/kaszta1274/macrorecipe/internal/seeder"
	_ "github.com/lib/pq"
)

func main() {
	mode := flag.String("mode", "prod", "seed mode: prod or dev")
	reset := flag.Bool("reset", false, "truncate tables before seeding (only allowed with -mode=dev)")
	flag.Parse()

	if *mode != "prod" && *mode != "dev" {
		log.Fatalf("invalid -mode %q: must be prod or dev", *mode)
	}
	if *reset && *mode != "dev" {
		log.Fatal("-reset is only allowed with -mode=dev")
	}

	_ = godotenv.Load()

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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := waitForDB(ctx, db, 5, time.Second); err != nil {
		log.Fatalf("database not ready: %s", err)
	}

	if *reset {
		if err := seeder.Reset(ctx, db); err != nil {
			log.Fatalf("reset error: %s", err)
		}
		log.Println("Database reset completed")
	}

	dbQueries := database.New(db)

	switch *mode {
	case "prod":
		data, err := seeder.SeedBase(ctx, dbQueries, adminEmail, adminPassword)
		if err != nil {
			log.Fatalf("seeding error: %s", err)
		}
		log.Printf("Seeding (prod) completed: admin=%s ingredients=%d", data.Admin.Email, len(data.Ingredients))
	case "dev":
		data, err := seeder.SeedTestData(ctx, dbQueries, adminEmail, adminPassword)
		if err != nil {
			log.Fatalf("seeding error: %s", err)
		}
		log.Printf(
			"Seeding (dev) completed: admin=%s users=%d ingredients=%d recipes=%d",
			data.Admin.Email,
			len(data.Users),
			len(data.Ingredients),
			len(data.Recipes),
		)
	}
}

func waitForDB(ctx context.Context, db *sql.DB, attempts int, delay time.Duration) error {
	var err error
	for i := 1; i <= attempts; i++ {
		if err = db.PingContext(ctx); err == nil {
			return nil
		}
		if i < attempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	return fmt.Errorf("ping failed after %d attempts: %w", attempts, err)
}
