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
)

type Recipe struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Instructions string    `json:"instructions"`
	Portions     int       `json:"portions"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RecipeRequest struct {
	Name         string `json:"name" validate:"required,min=2,max=100"`
	Description  string `json:"description" validate:"required,min=2,max=1000"`
	Instructions string `json:"instructions" validate:"required,min=2,max=2000"`
	Portions     int    `json:"portions" validate:"required,gte=1,lte=100"`
}

type RecipeHandler struct {
	DB *database.Queries
}

func NewRecipeHandler(db *database.Queries) *RecipeHandler {
	return &RecipeHandler{
		DB: db,
	}
}

func (h *RecipeHandler) CreateRecipe(w http.ResponseWriter, r *http.Request) {
	params := RecipeRequest{}
	if err := request.DecodeJSON(w, r, &params); err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	userId, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get user id", nil)
		return
	}

	recipe, err := h.DB.CreateRecipe(r.Context(), database.CreateRecipeParams{
		UserID:       userId,
		Name:         params.Name,
		Description:  params.Description,
		Instructions: params.Instructions,
		Portions:     int16(params.Portions),
	})
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not create recipe", err)
		return
	}

	response.RespondWithJSON(w, http.StatusCreated, toRecipe(recipe))
}

func (h *RecipeHandler) GetRecipeById(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid request", nil)
		return
	}

	recipeId, err := uuid.Parse(id)
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

	response.RespondWithJSON(w, http.StatusOK, toRecipe(recipe))
}

func (h *RecipeHandler) ListRecipesByName(w http.ResponseWriter, r *http.Request) {
	limit, offset, query := request.ParsePagination(r)

	recipes, err := h.DB.ListRecipesByName(r.Context(), database.ListRecipesByNameParams{
		Limit:       int32(limit),
		Offset:      int32(offset),
		SearchQuery: query,
	})
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not list recipes", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, mapSlice(recipes, toRecipe))
}

func (h *RecipeHandler) ListMyRecipes(w http.ResponseWriter, r *http.Request) {
	userId, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get user id", nil)
		return
	}

	limit, offset, query := request.ParsePagination(r)

	recipes, err := h.DB.ListRecipesByUserId(r.Context(), database.ListRecipesByUserIdParams{
		UserID:      userId,
		Limit:       int32(limit),
		Offset:      int32(offset),
		SearchQuery: query,
	})
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not list recipes", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, mapSlice(recipes, toRecipe))
}

func (h *RecipeHandler) ListRecipesByUserId(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid request", nil)
		return
	}

	userId, err := uuid.Parse(id)
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid user id format", err)
		return
	}

	limit, offset, query := request.ParsePagination(r)

	recipes, err := h.DB.ListRecipesByUserId(r.Context(), database.ListRecipesByUserIdParams{
		UserID:      userId,
		Limit:       int32(limit),
		Offset:      int32(offset),
		SearchQuery: query,
	})
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not list recipes", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, mapSlice(recipes, toRecipe))
}

func (h *RecipeHandler) UpdateRecipe(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := auth.FromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get auth ctx", nil)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid request", nil)
		return
	}

	recipeId, err := uuid.Parse(id)
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
		response.RespondWithError(w, r, http.StatusForbidden, "you do not have permission to modify this recipe", nil)
		return
	}

	params := RecipeRequest{}
	if err := request.DecodeJSON(w, r, &params); err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	updatedRecipe, err := h.DB.UpdateRecipe(r.Context(), database.UpdateRecipeParams{
		ID:           recipe.ID,
		Name:         params.Name,
		Description:  params.Description,
		Instructions: params.Instructions,
		Portions:     int16(params.Portions),
	})
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusNotFound, "recipe not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not update recipe", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, toRecipe(updatedRecipe))
}

func (h *RecipeHandler) DeleteRecipe(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := auth.FromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get auth ctx", nil)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid request", nil)
		return
	}

	recipeId, err := uuid.Parse(id)
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
		response.RespondWithError(w, r, http.StatusForbidden, "you do not have permission to delete this recipe", nil)
		return
	}

	err = h.DB.DeleteRecipe(r.Context(), recipe.ID)
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not delete recipe", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toRecipe(recipe database.Recipe) Recipe {
	return Recipe{
		ID:           recipe.ID,
		UserID:       recipe.UserID,
		Name:         recipe.Name,
		Description:  recipe.Description,
		Instructions: recipe.Instructions,
		Portions:     int(recipe.Portions),
		CreatedAt:    recipe.CreatedAt,
		UpdatedAt:    recipe.UpdatedAt,
	}
}
