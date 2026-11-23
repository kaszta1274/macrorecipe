package seeder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/kaszta1274/macrorecipe/internal/auth"
	"github.com/kaszta1274/macrorecipe/internal/database"
)

type SeedData struct {
	Admin       database.User
	Users       []database.User
	Ingredients map[string]database.Ingredient
	Recipes     []database.Recipe
	Likes       []database.Like
}

func SeedBase(ctx context.Context, db *database.Queries, adminEmail, adminPassword string) (SeedData, error) {
	data := SeedData{
		Ingredients: make(map[string]database.Ingredient, len(starterIngredients)),
	}

	admin, err := ensureUser(ctx, db, adminEmail, "Admin", adminPassword, database.UserRoleAdmin)
	if err != nil {
		return SeedData{}, fmt.Errorf("seed admin: %w", err)
	}
	data.Admin = admin

	for _, ingParams := range starterIngredients {
		ing, err := ensureIngredient(ctx, db, ingParams)
		if err != nil {
			return SeedData{}, fmt.Errorf("seed ingredient %q: %w", ingParams.Name, err)
		}
		data.Ingredients[ing.Name] = ing
	}

	return data, nil
}

func SeedTestData(ctx context.Context, db *database.Queries, adminEmail, adminPassword string) (SeedData, error) {
	data, err := SeedBase(ctx, db, adminEmail, adminPassword)
	if err != nil {
		return SeedData{}, err
	}

	data.Users = make([]database.User, 0, len(testUsers))
	for _, u := range testUsers {
		user, err := ensureUser(ctx, db, u.email, u.username, u.password, database.UserRoleUser)
		if err != nil {
			return SeedData{}, fmt.Errorf("seed user %q: %w", u.email, err)
		}
		data.Users = append(data.Users, user)
	}

	// shortcut: assumes seeded test users have <=100 recipes; add a dedicated query if that grows.
	existingByOwner := make(map[uuid.UUID]map[string]database.Recipe, len(data.Users))
	for _, owner := range data.Users {
		recipes, err := db.ListRecipesByUserId(ctx, database.ListRecipesByUserIdParams{
			UserID:      owner.ID,
			Limit:       100,
			Offset:      0,
			SearchQuery: "",
		})
		if err != nil {
			return SeedData{}, fmt.Errorf("list recipes for %q: %w", owner.Email, err)
		}
		byName := make(map[string]database.Recipe, len(recipes))
		for _, r := range recipes {
			byName[r.Name] = r
		}
		existingByOwner[owner.ID] = byName
	}

	data.Recipes = make([]database.Recipe, 0, len(testRecipes))
	for _, sr := range testRecipes {
		owner := data.Users[sr.ownerIndex]
		recipe, exists := existingByOwner[owner.ID][sr.name]
		if !exists {
			recipe, err = db.CreateRecipe(ctx, database.CreateRecipeParams{
				UserID:       owner.ID,
				Name:         sr.name,
				Description:  sr.description,
				Instructions: sr.instructions,
				Portions:     sr.portions,
			})
			if err != nil {
				return SeedData{}, fmt.Errorf("seed recipe %q: %w", sr.name, err)
			}
			existingByOwner[owner.ID][recipe.Name] = recipe
		}
		data.Recipes = append(data.Recipes, recipe)

		for _, sri := range sr.ingredients {
			ing, ok := data.Ingredients[sri.ingredientName]
			if !ok {
				return SeedData{}, fmt.Errorf("seed recipe %q: unknown ingredient %q", sr.name, sri.ingredientName)
			}
			_, err := db.GetRecipeIngredientById(ctx, database.GetRecipeIngredientByIdParams{
				RecipeID:     recipe.ID,
				IngredientID: ing.ID,
			})
			if errors.Is(err, sql.ErrNoRows) {
				_, err = db.CreateRecipeIngredient(ctx, database.CreateRecipeIngredientParams{
					RecipeID:     recipe.ID,
					IngredientID: ing.ID,
					Amount:       sri.amount,
				})
			}
			if err != nil {
				return SeedData{}, fmt.Errorf("seed recipe ingredient (%q, %q): %w", sr.name, sri.ingredientName, err)
			}
		}
	}

	data.Likes = make([]database.Like, 0, len(testLikes))
	for _, sl := range testLikes {
		user := data.Users[sl.userIndex]
		recipe := data.Recipes[sl.recipeIndex]
		like, err := db.GetLikeById(ctx, database.GetLikeByIdParams{
			UserID:   user.ID,
			RecipeID: recipe.ID,
		})
		if errors.Is(err, sql.ErrNoRows) {
			like, err = db.CreateLike(ctx, database.CreateLikeParams{
				UserID:   user.ID,
				RecipeID: recipe.ID,
			})
		}
		if err != nil {
			return SeedData{}, fmt.Errorf("seed like (%q, %q): %w", user.Email, recipe.Name, err)
		}
		data.Likes = append(data.Likes, like)
	}

	return data, nil
}

func Reset(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "TRUNCATE TABLE users, ingredients CASCADE")
	return err
}

func ensureUser(
	ctx context.Context,
	db *database.Queries,
	email, username, password string,
	role database.UserRole,
) (database.User, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))

	existing, err := db.GetUserByEmail(ctx, normalizedEmail)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return database.User{}, err
	}

	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return database.User{}, err
	}

	created, err := db.CreateUser(ctx, database.CreateUserParams{
		Email:        normalizedEmail,
		Username:     username,
		PasswordHash: hashedPassword,
		Role:         role,
	})
	if err != nil {
		return database.User{}, err
	}

	return database.User{
		ID:           created.ID,
		Email:        created.Email,
		Username:     created.Username,
		PasswordHash: hashedPassword,
		Role:         created.Role,
		CreatedAt:    created.CreatedAt,
		UpdatedAt:    created.UpdatedAt,
	}, nil
}

func ensureIngredient(
	ctx context.Context,
	db *database.Queries,
	params database.CreateIngredientParams,
) (database.Ingredient, error) {
	existing, err := db.GetIngredientByName(ctx, params.Name)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return database.Ingredient{}, err
	}

	return db.CreateIngredient(ctx, params)
}
