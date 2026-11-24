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

type Like struct {
	UserID    uuid.UUID `json:"user_id"`
	RecipeID  uuid.UUID `json:"recipe_id"`
	CreatedAt time.Time `json:"created_at"`
}

type LikeByUserId struct {
	UserID            uuid.UUID `json:"user_id"`
	RecipeID          uuid.UUID `json:"recipe_id"`
	RecipeName        string    `json:"recipe_name"`
	RecipeDescription string    `json:"recipe_description"`
	CreatedAt         time.Time `json:"created_at"`
}

type LikeHandler struct {
	DB *database.Queries
}

func NewLikeHandler(db *database.Queries) *LikeHandler {
	return &LikeHandler{
		DB: db,
	}
}

func (h *LikeHandler) CreateLike(w http.ResponseWriter, r *http.Request) {
	userId, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get user id", nil)
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

	like, err := h.DB.CreateLike(r.Context(), database.CreateLikeParams{
		UserID:   userId,
		RecipeID: recipe.ID,
	})
	pqErr := pq.As(err, pqerror.UniqueViolation)
	if pqErr != nil {
		response.RespondWithError(w, r, http.StatusConflict, "recipe already liked", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not create like", err)
		return
	}

	response.RespondWithJSON(w, http.StatusCreated, toLike(like))
}

func (h *LikeHandler) ListLikesByUserId(w http.ResponseWriter, r *http.Request) {
	userId, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid user id format", err)
		return
	}

	limit, offset, _ := request.ParsePagination(r)

	likes, err := h.DB.ListLikesByUserId(r.Context(), database.ListLikesByUserIdParams{
		UserID: userId,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not get likes", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, mapSlice(likes, toUserLike))
}

func (h *LikeHandler) DeleteLike(w http.ResponseWriter, r *http.Request) {
	userId, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get user id", nil)
		return
	}

	recipeId, err := uuid.Parse(chi.URLParam(r, "recipe_id"))
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid recipe id format", err)
		return
	}

	err = h.DB.DeleteLike(r.Context(), database.DeleteLikeParams{
		UserID:   userId,
		RecipeID: recipeId,
	})
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not delete like", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toLike(like database.Like) Like {
	return Like{
		UserID:    like.UserID,
		RecipeID:  like.RecipeID,
		CreatedAt: like.CreatedAt,
	}
}

func toUserLike(like database.ListLikesByUserIdRow) LikeByUserId {
	return LikeByUserId{
		UserID:            like.UserID,
		RecipeID:          like.RecipeID,
		RecipeName:        like.RecipeName,
		RecipeDescription: like.RecipeDescription,
		CreatedAt:         like.CreatedAt,
	}
}
