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

type Follow struct {
	FollowerID  uuid.UUID `json:"follower_id"`
	FollowingID uuid.UUID `json:"following_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type UserListItem struct {
	UserSummary
	FollowedAt   time.Time `json:"followed_at"`
	FollowedByMe bool      `json:"followed_by_me"`
}

type FollowHandler struct {
	DB *database.Queries
}

func NewFollowHandler(db *database.Queries) *FollowHandler {
	return &FollowHandler{
		DB: db,
	}
}

func (h *FollowHandler) CreateFollow(w http.ResponseWriter, r *http.Request) {
	followerID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get user id", nil)
		return
	}

	targetID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid user id format", err)
		return
	}
	if followerID == targetID {
		response.RespondWithError(w, r, http.StatusBadRequest, "cannot follow yourself", nil)
		return
	}

	target, err := h.DB.GetUserById(r.Context(), targetID)
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusNotFound, "user not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not get user", err)
		return
	}

	follow, err := h.DB.CreateFollow(r.Context(), database.CreateFollowParams{
		FollowerID:  followerID,
		FollowingID: target.ID,
	})
	if pqErr := pq.As(err, pqerror.UniqueViolation); pqErr != nil {
		response.RespondWithError(w, r, http.StatusConflict, "already following this user", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not create follow", err)
		return
	}

	response.RespondWithJSON(w, http.StatusCreated, toFollow(follow))
}

func (h *FollowHandler) DeleteFollow(w http.ResponseWriter, r *http.Request) {
	followerID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get user id", nil)
		return
	}

	targetID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid user id format", err)
		return
	}

	err = h.DB.DeleteFollow(r.Context(), database.DeleteFollowParams{
		FollowerID:  followerID,
		FollowingID: targetID,
	})
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not delete follow", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *FollowHandler) ListFollowersByUserId(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get user id", nil)
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid user id format", err)
		return
	}

	if _, err := h.DB.GetUserById(r.Context(), userID); errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusNotFound, "user not found", nil)
		return
	} else if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not get user", err)
		return
	}

	limit, offset, _ := request.ParsePagination(r)

	followers, err := h.DB.ListFollowersByUserId(r.Context(), database.ListFollowersByUserIdParams{
		ViewerID: viewerID,
		UserID:   userID,
		Offset:   int32(offset),
		Limit:    int32(limit),
	})
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not list followers", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, mapSlice(followers, toFollowerItem))
}

func (h *FollowHandler) ListFollowingsByUserId(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get user id", nil)
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid user id format", err)
		return
	}

	if _, err := h.DB.GetUserById(r.Context(), userID); errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusNotFound, "user not found", nil)
		return
	} else if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not get user", err)
		return
	}

	limit, offset, _ := request.ParsePagination(r)

	followings, err := h.DB.ListFollowingsByUserId(r.Context(), database.ListFollowingsByUserIdParams{
		ViewerID: viewerID,
		UserID:   userID,
		Offset:   int32(offset),
		Limit:    int32(limit),
	})
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not list following", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, mapSlice(followings, toFollowingItem))
}

func toFollow(f database.Follow) Follow {
	return Follow{
		FollowerID:  f.FollowerID,
		FollowingID: f.FollowingID,
		CreatedAt:   f.CreatedAt,
	}
}

func toFollowerItem(r database.ListFollowersByUserIdRow) UserListItem {
	summary := UserSummary{ID: r.ID, Username: r.Username, AvatarURL: nullStringPtr(r.AvatarUrl)}
	return UserListItem{
		UserSummary:  summary,
		FollowedAt:   r.FollowedAt,
		FollowedByMe: r.FollowedByMe,
	}
}

func toFollowingItem(r database.ListFollowingsByUserIdRow) UserListItem {
	summary := UserSummary{ID: r.ID, Username: r.Username, AvatarURL: nullStringPtr(r.AvatarUrl)}
	return UserListItem{
		UserSummary:  summary,
		FollowedAt:   r.FollowedAt,
		FollowedByMe: r.FollowedByMe,
	}
}
