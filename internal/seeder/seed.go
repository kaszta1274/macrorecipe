package seeder

import (
	"context"
	"database/sql"
	"log"

	"github.com/kaszta1274/macrorecipe/internal/auth"
	"github.com/kaszta1274/macrorecipe/internal/database"
)

func RunSeeders(db *database.Queries, adminEmail, adminPassword string) error {
	ctx := context.Background()

	_, err := db.GetUserByEmail(ctx, adminEmail)
	if err == sql.ErrNoRows {
		hashedPassword, err := auth.HashPassword(adminPassword)
		if err != nil {
			return err
		}

		_, err = db.CreateUser(ctx, database.CreateUserParams{
			Email:        adminEmail,
			Username:     "Admin",
			PasswordHash: hashedPassword,
			Role:         database.UserRoleAdmin,
		})
		if err != nil {
			return err
		}
		log.Println("Seeding: Admin acount created successfully")
	} else if err != nil {
		return err
	} else {
		log.Println("Seeding: Admin acount already exists")
	}

	return nil
}
