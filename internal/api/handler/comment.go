package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kaszta1274/macrorecipe/internal/api/request"
	"github.com/kaszta1274/macrorecipe/internal/api/response"
	"github.com/kaszta1274/macrorecipe/internal/auth"
	"github.com/kaszta1274/macrorecipe/internal/database"
)

type Comment struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	RecipeID  uuid.UUID `json:"recipe_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CommentRequest struct {
	Body string `json:"body" validate:"required,min=1,max=1000"`
}

type CommentsByRecipeId struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Username  string    `json:"username"`
	RecipeID  uuid.UUID `json:"recipe_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CommentHandler struct {
	DB *database.Queries
}

func NewCommentHandler(db *database.Queries) *CommentHandler {
	return &CommentHandler{
		DB: db,
	}
}

func (h *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
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

	params := CommentRequest{}
	if err := request.DecodeJSON(w, r, &params); err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}
	params.Body = strings.TrimSpace(params.Body)
	if params.Body == "" {
		response.RespondWithError(w, r, http.StatusBadRequest, "comment body cannot be empty", nil)
		return
	}

	comment, err := h.DB.CreateComment(r.Context(), database.CreateCommentParams{
		UserID:   userId,
		RecipeID: recipe.ID,
		Body:     params.Body,
	})
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not create comment", err)
		return
	}

	response.RespondWithJSON(w, http.StatusCreated, toComment(comment))
}

func (h *CommentHandler) ListCommentsByRecipeId(w http.ResponseWriter, r *http.Request) {
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

	limit, offset, _ := request.ParsePagination(r)

	comments, err := h.DB.ListCommentsByRecipeId(r.Context(), database.ListCommentsByRecipeIdParams{
		RecipeID: recipe.ID,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not list comments", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, mapSlice(comments, toRecipeComment))
}

func (h *CommentHandler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := auth.FromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get auth ctx", nil)
		return
	}

	commentId, err := uuid.Parse(chi.URLParam(r, "comment_id"))
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid comment id format", err)
		return
	}

	comment, err := h.DB.GetCommentById(r.Context(), commentId)
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusNotFound, "comment not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not get comment", err)
		return
	}
	if comment.UserID != authCtx.UserID {
		response.RespondWithError(w, r, http.StatusForbidden, "you do not have permission to update this comment", nil)
		return
	}

	params := CommentRequest{}
	if err := request.DecodeJSON(w, r, &params); err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}
	params.Body = strings.TrimSpace(params.Body)
	if params.Body == "" {
		response.RespondWithError(w, r, http.StatusBadRequest, "comment body cannot be empty", nil)
		return
	}

	updatedComment, err := h.DB.UpdateComment(r.Context(), database.UpdateCommentParams{
		ID:   comment.ID,
		Body: params.Body,
	})
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusNotFound, "comment not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not update comment", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, toComment(updatedComment))
}

func (h *CommentHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := auth.FromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get auth ctx", nil)
		return
	}

	commentId, err := uuid.Parse(chi.URLParam(r, "comment_id"))
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid comment id format", err)
		return
	}

	comment, err := h.DB.GetCommentById(r.Context(), commentId)
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusNotFound, "comment not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not get comment", err)
		return
	}
	if authCtx.Role != database.UserRoleAdmin && comment.UserID != authCtx.UserID {
		response.RespondWithError(w, r, http.StatusForbidden, "you do not have permission to delete this comment", nil)
		return
	}

	err = h.DB.DeleteComment(r.Context(), comment.ID)
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not delete comment", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toComment(comment database.Comment) Comment {
	return Comment{
		ID:        comment.ID,
		UserID:    comment.UserID,
		RecipeID:  comment.RecipeID,
		Body:      comment.Body,
		CreatedAt: comment.CreatedAt,
		UpdatedAt: comment.UpdatedAt,
	}
}

func toRecipeComment(comment database.ListCommentsByRecipeIdRow) CommentsByRecipeId {
	return CommentsByRecipeId{
		ID:        comment.ID,
		UserID:    comment.UserID,
		Username:  comment.Username,
		RecipeID:  comment.RecipeID,
		Body:      comment.Body,
		CreatedAt: comment.CreatedAt,
		UpdatedAt: comment.UpdatedAt,
	}
}
