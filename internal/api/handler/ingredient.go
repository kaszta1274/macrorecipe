package handler

import "github.com/kaszta1274/macrorecipe/internal/database"

type IngredientHandler struct {
	DB          *database.Queries
	TokenSecret string
}

func NewIngredientHandler(db *database.Queries, tokenSecret string) *UserHandler {
	return &UserHandler{
		DB:          db,
		TokenSecret: tokenSecret,
	}
}
