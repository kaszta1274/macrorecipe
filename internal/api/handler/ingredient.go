package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kaszta1274/macrorecipe/internal/api/response"
	"github.com/kaszta1274/macrorecipe/internal/database"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

type Ingredient struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Protein       float32   `json:"protein"`
	Fat           float32   `json:"fat"`
	Carbohydrates float32   `json:"carbohydrates"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type IngredientCreateRequest struct {
	Name          string   `json:"name"`
	Protein       *float32 `json:"protein"`
	Fat           *float32 `json:"fat"`
	Carbohydrates *float32 `json:"carbohydrates"`
}

type IngredientUpdateRequest struct {
	Name          string   `json:"name"`
	Protein       *float32 `json:"protein"`
	Fat           *float32 `json:"fat"`
	Carbohydrates *float32 `json:"carbohydrates"`
}

type IngredientHandler struct {
	DB          *database.Queries
	TokenSecret string
}

func NewIngredientHandler(db *database.Queries, tokenSecret string) *IngredientHandler {
	return &IngredientHandler{
		DB:          db,
		TokenSecret: tokenSecret,
	}
}

func (h *IngredientHandler) CreateIngredient(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	params := IngredientCreateRequest{}
	err := decoder.Decode(&params)
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid request payload", err)
		return
	}

	if params.Name == "" || params.Protein == nil || params.Fat == nil || params.Carbohydrates == nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "name and all macro fields are required", nil)
		return
	}

	ingredient, err := h.DB.CreateIngredient(r.Context(), database.CreateIngredientParams{
		Name:          params.Name,
		Protein:       *params.Protein,
		Fat:           *params.Fat,
		Carbohydrates: *params.Carbohydrates,
	})
	pqErr := pq.As(err, pqerror.UniqueViolation)
	if pqErr != nil {
		response.RespondWithError(w, r, http.StatusConflict, "ingredient with this name exists", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not create ingredient", err)
		return
	}

	response.RespondWithJSON(
		w,
		http.StatusCreated,
		databaseIngredientToResponseIngredient(ingredient),
	)
}

func (h *IngredientHandler) GetIngredientById(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid request", nil)
		return
	}

	ingredientId, err := uuid.Parse(id)
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid ingredient id format", err)
		return
	}

	ingredient, err := h.DB.GetIngredientById(r.Context(), ingredientId)
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusNotFound, "ingredient not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not get ingredient", err)
		return
	}

	response.RespondWithJSON(
		w,
		http.StatusOK,
		databaseIngredientToResponseIngredient(ingredient),
	)
}

func (h *IngredientHandler) ListIngredients(w http.ResponseWriter, r *http.Request) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 1 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	query := r.URL.Query().Get("query")

	ingredients, err := h.DB.ListIngredients(r.Context(), database.ListIngredientsParams{
		Limit:       int32(limit),
		Offset:      int32(offset),
		SearchQuery: query,
	})
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not list ingredients", err)
		return
	}

	respondWithIngredientsJSON(w, ingredients)
}

func (h *IngredientHandler) UpdateIngredient(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid request", nil)
		return
	}

	ingredientId, err := uuid.Parse(id)
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid ingredient id format", err)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := IngredientUpdateRequest{}
	err = decoder.Decode(&params)
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid request payload", err)
		return
	}

	if params.Name == "" || params.Protein == nil || params.Fat == nil || params.Carbohydrates == nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "name and all macro fields are required", nil)
		return
	}

	updatedIngredient, err := h.DB.UpdateIngredient(r.Context(), database.UpdateIngredientParams{
		ID:            ingredientId,
		Name:          params.Name,
		Protein:       *params.Protein,
		Fat:           *params.Fat,
		Carbohydrates: *params.Carbohydrates,
	})
	pqErr := pq.As(err, pqerror.UniqueViolation)
	if pqErr != nil {
		response.RespondWithError(w, r, http.StatusConflict, "ingredient with this name exists", nil)
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusNotFound, "ingredient not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not update ingredient", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, databaseIngredientToResponseIngredient(updatedIngredient))
}

func (h *IngredientHandler) DeleteIngredient(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid request", nil)
		return
	}

	ingredientId, err := uuid.Parse(id)
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid ingredient id format", err)
		return
	}

	err = h.DB.DeleteIngredient(r.Context(), ingredientId)
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not delete ingredient", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func databaseIngredientToResponseIngredient(ingredient database.Ingredient) Ingredient {
	return Ingredient{
		ID:            ingredient.ID,
		Name:          ingredient.Name,
		Protein:       ingredient.Protein,
		Fat:           ingredient.Fat,
		Carbohydrates: ingredient.Carbohydrates,
		CreatedAt:     ingredient.CreatedAt,
		UpdatedAt:     ingredient.UpdatedAt,
	}
}

func respondWithIngredientsJSON(w http.ResponseWriter, ingredients []database.Ingredient) {
	resp := []Ingredient{}

	for _, ingredient := range ingredients {
		resp = append(resp, Ingredient{
			ID:            ingredient.ID,
			Name:          ingredient.Name,
			Protein:       ingredient.Protein,
			Fat:           ingredient.Fat,
			Carbohydrates: ingredient.Carbohydrates,
			CreatedAt:     ingredient.CreatedAt,
			UpdatedAt:     ingredient.UpdatedAt,
		})
	}
	response.RespondWithJSON(w, http.StatusOK, resp)
}
