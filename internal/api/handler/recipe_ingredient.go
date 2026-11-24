package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kaszta1274/macrorecipe/internal/api/request"
	"github.com/kaszta1274/macrorecipe/internal/api/response"
	"github.com/kaszta1274/macrorecipe/internal/auth"
	"github.com/kaszta1274/macrorecipe/internal/database"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

type RecipeIngredient struct {
	RecipeID     uuid.UUID `json:"recipe_id"`
	IngredientID uuid.UUID `json:"ingredient_id"`
	Amount       float32   `json:"amount"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RecipeIngredientsByRecipe struct {
	RecipeID       uuid.UUID `json:"recipe_id"`
	IngredientID   uuid.UUID `json:"ingredient_id"`
	IngredientName string    `json:"ingredient_name"`
	Amount         float32   `json:"amount"`
	Protein        float32   `json:"protein"`
	Fat            float32   `json:"fat"`
	Carbohydrates  float32   `json:"carbohydrates"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type RecipeIngredientCreateRequest struct {
	IngredientID uuid.UUID `json:"ingredient_id" validate:"required,uuid"`
	Amount       float32   `json:"amount" validate:"gt=0"`
}

type RecipeIngredientUpdateRequest struct {
	Amount float32 `json:"amount" validate:"gt=0"`
}

type RecipeIngredientHandler struct {
	DB *database.Queries
}

func NewRecipeIngredientHandler(db *database.Queries) *RecipeIngredientHandler {
	return &RecipeIngredientHandler{
		DB: db,
	}
}

func (h *RecipeIngredientHandler) CreateRecipeIngredient(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := auth.FromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get auth ctx", nil)
		return
	}

	recipeId, err := uuid.Parse(chi.URLParam(r, "recipe_id"))
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid recipe id format", err)
		return
	}

	recipe, err := h.DB.GetRecipeById(r.Context(), recipeId)
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusNotFound, "recipe not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not get recipe", err)
		return
	}
	if authCtx.Role != database.UserRoleAdmin && recipe.UserID != authCtx.UserID {
		response.RespondWithError(w, r, http.StatusForbidden, "you do not have permission to add ingredients to this recipe", nil)
		return
	}

	params := RecipeIngredientCreateRequest{}
	if err := request.DecodeJSON(w, r, &params); err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	ingredient, err := h.DB.GetIngredientById(r.Context(), params.IngredientID)
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusNotFound, "ingredient not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not get ingredient", err)
		return
	}

	recipeIngredient, err := h.DB.CreateRecipeIngredient(r.Context(), database.CreateRecipeIngredientParams{
		RecipeID:     recipe.ID,
		IngredientID: ingredient.ID,
		Amount:       params.Amount,
	})
	pqErr := pq.As(err, pqerror.UniqueViolation)
	if pqErr != nil {
		response.RespondWithError(w, r, http.StatusConflict, "ingredient already exists in this recipe", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not create recipe ingredient", err)
		return
	}

	response.RespondWithJSON(
		w,
		http.StatusCreated,
		toRecipeIngredient(recipeIngredient),
	)
}

func (h *RecipeIngredientHandler) ListRecipeIngredientsByRecipeId(w http.ResponseWriter, r *http.Request) {
	recipeId, err := uuid.Parse(chi.URLParam(r, "recipe_id"))
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid recipe id format", err)
		return
	}

	recipeIngredients, err := h.DB.ListRecipeIngredientsByRecipeId(r.Context(), recipeId)
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not get ingredients for recipe", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, mapSlice(recipeIngredients, toRecipeIngredientDetail))
}

func (h *RecipeIngredientHandler) UpdateRecipeIngredient(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := auth.FromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get auth ctx", nil)
		return
	}

	ingredientId, err := uuid.Parse(chi.URLParam(r, "ingredient_id"))
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid ingredient id format", err)
		return
	}

	recipeId, err := uuid.Parse(chi.URLParam(r, "recipe_id"))
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid recipe id format", err)
		return
	}

	recipe, err := h.DB.GetRecipeById(r.Context(), recipeId)
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusNotFound, "recipe not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not get recipe", err)
		return
	}
	if authCtx.Role != database.UserRoleAdmin && recipe.UserID != authCtx.UserID {
		response.RespondWithError(w, r, http.StatusForbidden, "you do not have permission to update ingredients in this recipe", nil)
		return
	}

	params := RecipeIngredientUpdateRequest{}
	if err := request.DecodeJSON(w, r, &params); err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	updatedRecipeIngredient, err := h.DB.UpdateRecipeIngredient(
		r.Context(),
		database.UpdateRecipeIngredientParams{
			RecipeID:     recipe.ID,
			IngredientID: ingredientId,
			Amount:       params.Amount,
		},
	)
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusNotFound, "recipe ingredient not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not update recipe ingredient", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, toRecipeIngredient(updatedRecipeIngredient))
}

func (h *RecipeIngredientHandler) DeleteRecipeIngredient(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := auth.FromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get auth ctx", nil)
		return
	}

	ingredientId, err := uuid.Parse(chi.URLParam(r, "ingredient_id"))
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid ingredient id format", err)
		return
	}

	recipeId, err := uuid.Parse(chi.URLParam(r, "recipe_id"))
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid recipe id format", err)
		return
	}

	recipe, err := h.DB.GetRecipeById(r.Context(), recipeId)
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusNotFound, "recipe not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not get recipe", err)
		return
	}
	if authCtx.Role != database.UserRoleAdmin && recipe.UserID != authCtx.UserID {
		response.RespondWithError(w, r, http.StatusForbidden, "you do not have permission to delete ingredients in this recipe", nil)
		return
	}

	err = h.DB.DeleteRecipeIngredient(r.Context(), database.DeleteRecipeIngredientParams{
		RecipeID:     recipe.ID,
		IngredientID: ingredientId,
	})
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not delete recipe ingredient", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toRecipeIngredient(recipeIngredient database.RecipeIngredient) RecipeIngredient {
	return RecipeIngredient{
		RecipeID:     recipeIngredient.RecipeID,
		IngredientID: recipeIngredient.IngredientID,
		Amount:       recipeIngredient.Amount,
		CreatedAt:    recipeIngredient.CreatedAt,
		UpdatedAt:    recipeIngredient.UpdatedAt,
	}
}

func toRecipeIngredientDetail(ri database.ListRecipeIngredientsByRecipeIdRow) RecipeIngredientsByRecipe {
	return RecipeIngredientsByRecipe{
		RecipeID:       ri.RecipeID,
		IngredientID:   ri.IngredientID,
		IngredientName: ri.IngredientName,
		Amount:         ri.Amount,
		Protein:        ri.Protein,
		Fat:            ri.Fat,
		Carbohydrates:  ri.Carbohydrates,
		CreatedAt:      ri.CreatedAt,
		UpdatedAt:      ri.UpdatedAt,
	}
}
