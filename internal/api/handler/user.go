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
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

type User struct {
	ID        uuid.UUID         `json:"id"`
	Email     string            `json:"email"`
	Username  string            `json:"username"`
	Role      database.UserRole `json:"role"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type UserSummary struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
}

type UserPublicProfile struct {
	ID             uuid.UUID `json:"id"`
	Username       string    `json:"username"`
	RecipesCount   int64     `json:"recipes_count"`
	FollowersCount int64     `json:"followers_count"`
	FollowingCount int64     `json:"following_count"`
	FollowedByMe   bool      `json:"followed_by_me"`
	CreatedAt      time.Time `json:"created_at"`
}

type UserCreateRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,alphanum,min=3,max=30"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}

type UserLoginRequest struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type UserUpdateRequest struct {
	Username string `json:"username" validate:"required,alphanum,min=3,max=30"`
}

type UserUpdatePasswordRequest struct {
	Password    string `json:"password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required"`
}

type UserDeleteRequest struct {
	Password string `json:"password" validate:"required"`
}

type UserLoginResponse struct {
	User
	Token string `json:"token"`
}

type UserHandler struct {
	DB          *database.Queries
	TokenSecret string
}

func NewUserHandler(db *database.Queries, tokenSecret string) *UserHandler {
	return &UserHandler{
		DB:          db,
		TokenSecret: tokenSecret,
	}
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	params := UserCreateRequest{}
	if err := request.DecodeJSON(w, r, &params); err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not hash password", err)
		return
	}

	email := strings.ToLower(strings.TrimSpace(params.Email))

	user, err := h.DB.CreateUser(r.Context(), database.CreateUserParams{
		Email:        email,
		Username:     params.Username,
		PasswordHash: hashedPassword,
		Role:         database.UserRoleUser,
	})
	if pqErr := pq.As(err, pqerror.UniqueViolation); pqErr != nil {
		if pqErr.Constraint == "users_email_key" {
			response.RespondWithError(w, r, http.StatusConflict, "email already registered", nil)
			return
		}
		if pqErr.Constraint == "users_username_key" {
			response.RespondWithError(w, r, http.StatusConflict, "username already taken", nil)
			return
		}
		response.RespondWithError(w, r, http.StatusConflict, "user already exists", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not create user", err)
		return
	}

	response.RespondWithJSON(w, http.StatusCreated, User{
		ID:        user.ID,
		Email:     user.Email,
		Username:  user.Username,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
}

func (h *UserHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	params := UserLoginRequest{}
	if err := request.DecodeJSON(w, r, &params); err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	email := strings.ToLower(strings.TrimSpace(params.Email))

	user, err := h.DB.GetUserByEmail(r.Context(), email)
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusUnauthorized, "incorrect email or password", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not process login", err)
		return
	}

	match, err := auth.VerifyPasswordHash(params.Password, user.PasswordHash)
	if err != nil || !match {
		response.RespondWithError(w, r, http.StatusUnauthorized, "incorrect email or password", err)
		return
	}

	accessToken, err := auth.MakeJWTWithRole(user.ID, user.Role, h.TokenSecret, time.Hour)
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not create access token", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, UserLoginResponse{
		User:  toUser(user),
		Token: accessToken,
	})
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userId, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get user id", nil)
		return
	}

	user, err := h.DB.GetUserById(r.Context(), userId)
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusNotFound, "user not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not get user", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, toUser(user))
}

func (h *UserHandler) GetUserById(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get user id", nil)
		return
	}

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

	profile, err := h.DB.GetUserPublicProfileById(r.Context(), database.GetUserPublicProfileByIdParams{
		ViewerID: viewerID,
		ID:       userId,
	})
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusNotFound, "user not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not get user", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, toUserPublicProfile(profile))
}

func (h *UserHandler) UpdateUsername(w http.ResponseWriter, r *http.Request) {
	params := UserUpdateRequest{}
	if err := request.DecodeJSON(w, r, &params); err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	userId, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get user id", nil)
		return
	}

	updatedUser, err := h.DB.UpdateUser(r.Context(), database.UpdateUserParams{
		ID:       userId,
		Username: params.Username,
	})
	if pqErr := pq.As(err, pqerror.UniqueViolation); pqErr != nil {
		response.RespondWithError(w, r, http.StatusConflict, "user with this username exists", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not update user", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, User{
		ID:        updatedUser.ID,
		Email:     updatedUser.Email,
		Username:  updatedUser.Username,
		Role:      updatedUser.Role,
		CreatedAt: updatedUser.CreatedAt,
		UpdatedAt: updatedUser.UpdatedAt,
	})
}

func (h *UserHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	params := UserUpdatePasswordRequest{}
	if err := request.DecodeJSON(w, r, &params); err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	userId, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get user id", nil)
		return
	}

	user, err := h.DB.GetUserById(r.Context(), userId)
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusNotFound, "user not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not get user", err)
		return
	}

	match, err := auth.VerifyPasswordHash(params.Password, user.PasswordHash)
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not verify password", err)
		return
	}
	if !match {
		response.RespondWithError(w, r, http.StatusUnauthorized, "invalid credentials", nil)
		return
	}

	hashedNewPassword, err := auth.HashPassword(params.NewPassword)
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not hash password", err)
		return
	}

	updatedUser, err := h.DB.UpdateUserPassword(r.Context(), database.UpdateUserPasswordParams{
		ID:           user.ID,
		PasswordHash: hashedNewPassword,
	})
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not update user", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, User{
		ID:        updatedUser.ID,
		Email:     updatedUser.Email,
		Username:  updatedUser.Username,
		Role:      updatedUser.Role,
		CreatedAt: updatedUser.CreatedAt,
		UpdatedAt: updatedUser.UpdatedAt,
	})
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	params := UserDeleteRequest{}
	if err := request.DecodeJSON(w, r, &params); err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	userId, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get user id", nil)
		return
	}

	user, err := h.DB.GetUserById(r.Context(), userId)
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusNotFound, "user not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not get user", err)
		return
	}

	match, err := auth.VerifyPasswordHash(params.Password, user.PasswordHash)
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not verify password", err)
		return
	}
	if !match {
		response.RespondWithError(w, r, http.StatusUnauthorized, "invalid credentials", nil)
		return
	}

	err = h.DB.DeleteUser(r.Context(), userId)
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not delete user", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toUser(user database.User) User {
	return User{
		ID:        user.ID,
		Email:     user.Email,
		Username:  user.Username,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func toUserPublicProfile(p database.GetUserPublicProfileByIdRow) UserPublicProfile {
	return UserPublicProfile{
		ID:             p.ID,
		Username:       p.Username,
		RecipesCount:   p.RecipesCount,
		FollowersCount: p.FollowersCount,
		FollowingCount: p.FollowingCount,
		FollowedByMe:   p.FollowedByMe,
		CreatedAt:      p.CreatedAt,
	}
}
