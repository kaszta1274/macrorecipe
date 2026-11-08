package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kaszta1274/macrorecipe/internal/api/response"
	"github.com/kaszta1274/macrorecipe/internal/database"
)

type Ingredient struct {
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
		response.RespondWithError(w, http.StatusBadRequest, "invalid request payload", err)
		return
	}

	if params.Name == "" || params.Protein == nil || params.Fat == nil || params.Carbohydrates == nil {
		response.RespondWithError(w, http.StatusBadRequest, "all macro fields are required", nil)
		return
	}

	ingredient, err := h.DB.CreateIngredient(r.Context(), database.CreateIngredientParams{
		Name:          params.Name,
		Protein:       *params.Protein,
		Fat:           *params.Fat,
		Carbohydrates: *params.Carbohydrates,
	})
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "could not create ingredient", err)
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
		response.RespondWithError(w, http.StatusBadRequest, "invalid request", nil)
		return
	}

	ingredientId, err := uuid.Parse(id)
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "could not parse id", err)
		return
	}

	ingredient, err := h.DB.GetIngredientById(r.Context(), ingredientId)
	if err != nil {
		response.RespondWithError(w, http.StatusNotFound, "could not get ingredient", err)
		return
	}

	response.RespondWithJSON(
		w,
		http.StatusOK,
		databaseIngredientToResponseIngredient(ingredient),
	)
}

func databaseIngredientToResponseIngredient(ingredient database.Ingredient) Ingredient {
	return Ingredient{
		Name:          ingredient.Name,
		Protein:       ingredient.Protein,
		Fat:           ingredient.Fat,
		Carbohydrates: ingredient.Carbohydrates,
		CreatedAt:     ingredient.CreatedAt,
		UpdatedAt:     ingredient.UpdatedAt,
	}
}
