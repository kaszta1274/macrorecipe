package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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

type UserCreateRequest struct {
	Password string `json:"password"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type UserLoginRequest struct {
	Password string `json:"password"`
	Email    string `json:"email"`
}

type UserUpdateRequest struct {
	Username string `json:"username"`
}

type UserUpdatePasswordRequest struct {
	Password    string `json:"password"`
	NewPassword string `json:"new_password"`
}

type UserDeleteRequest struct {
	Password string `json:"password"`
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
	decoder := json.NewDecoder(r.Body)
	params := UserCreateRequest{}
	err := decoder.Decode(&params)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "invalid request payload", err)
		return
	}

	if params.Password == "" || params.Email == "" || params.Username == "" {
		response.RespondWithError(w, http.StatusBadRequest, "email, username and password are required", nil)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "could not hash password", err)
		return
	}

	user, err := h.DB.CreateUser(r.Context(), database.CreateUserParams{
		Email:        params.Email,
		Username:     params.Username,
		PasswordHash: hashedPassword,
		Role:         database.UserRoleUser,
	})
	if pqErr := pq.As(err, pqerror.UniqueViolation); pqErr != nil {
		if pqErr.Constraint == "email" {
			response.RespondWithError(w, http.StatusConflict, "email already registered", nil)
			return
		}
		if pqErr.Constraint == "username" {
			response.RespondWithError(w, http.StatusConflict, "username already taken", nil)
			return
		}
		response.RespondWithError(w, http.StatusConflict, "user already exists", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "could not create user", err)
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
	decoder := json.NewDecoder(r.Body)
	params := UserLoginRequest{}
	err := decoder.Decode(&params)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "invalid request payload", err)
		return
	}

	if params.Password == "" || params.Email == "" {
		response.RespondWithError(w, http.StatusBadRequest, "email and password are required", nil)
		return
	}

	user, err := h.DB.GetUserByEmail(r.Context(), params.Email)
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, http.StatusUnauthorized, "incorrect email or password", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "could not process login", err)
		return
	}

	match, err := auth.VerifyPasswordHash(params.Password, user.PasswordHash)
	if err != nil || !match {
		response.RespondWithError(w, http.StatusUnauthorized, "incorrect email or password", err)
		return
	}

	accessToken, err := auth.MakeJWTWithRole(user.ID, user.Role, h.TokenSecret, time.Hour)
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "could not create access token", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, UserLoginResponse{
		User:  databaseUserToResponseUser(user),
		Token: accessToken,
	})
}

func (h *UserHandler) GetUserById(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.RespondWithError(w, http.StatusBadRequest, "invalid request", nil)
		return
	}

	userId, err := uuid.Parse(id)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "invalid user id format", err)
		return
	}

	user, err := h.DB.GetUserById(r.Context(), userId)
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, http.StatusNotFound, "user not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "could not get user", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, databaseUserToResponseUser(user))
}

func (h *UserHandler) UpdateUsername(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	params := UserUpdateRequest{}
	err := decoder.Decode(&params)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "invalid request payload", err)
		return
	}

	if params.Username == "" {
		response.RespondWithError(w, http.StatusBadRequest, "username is required", nil)
		return
	}

	userId, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, http.StatusUnauthorized, "could not get user id", nil)
		return
	}

	updatedUser, err := h.DB.UpdateUser(r.Context(), database.UpdateUserParams{
		ID:       userId,
		Username: params.Username,
	})
	if pqErr := pq.As(err, pqerror.UniqueViolation); pqErr != nil {
		response.RespondWithError(w, http.StatusConflict, "user with this username exists", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "could not update user", err)
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
	decoder := json.NewDecoder(r.Body)
	params := UserUpdatePasswordRequest{}
	err := decoder.Decode(&params)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "invalid request payload", err)
		return
	}

	if params.Password == "" || params.NewPassword == "" {
		response.RespondWithError(w, http.StatusBadRequest, "current password and new password are required", nil)
		return
	}

	userId, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, http.StatusUnauthorized, "could not get user id", nil)
		return
	}

	user, err := h.DB.GetUserById(r.Context(), userId)
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, http.StatusNotFound, "user not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "could not get user", err)
		return
	}

	match, err := auth.VerifyPasswordHash(params.Password, user.PasswordHash)
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "could not verify password", err)
		return
	}
	if !match {
		response.RespondWithError(w, http.StatusUnauthorized, "invalid credentials", nil)
		return
	}

	hashedNewPassword, err := auth.HashPassword(params.NewPassword)
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "could not hash password", err)
		return
	}

	updatedUser, err := h.DB.UpdateUserPassword(r.Context(), database.UpdateUserPasswordParams{
		ID:           user.ID,
		PasswordHash: hashedNewPassword,
	})
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "could not update user", err)
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
	decoder := json.NewDecoder(r.Body)
	params := UserDeleteRequest{}
	err := decoder.Decode(&params)
	if err != nil {
		response.RespondWithError(w, http.StatusBadRequest, "invalid request payload", err)
		return
	}

	if params.Password == "" {
		response.RespondWithError(w, http.StatusBadRequest, "password is required", nil)
		return
	}

	userId, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, http.StatusUnauthorized, "could not get user id", nil)
		return
	}

	user, err := h.DB.GetUserById(r.Context(), userId)
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, http.StatusNotFound, "user not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "could not get user", err)
		return
	}

	match, err := auth.VerifyPasswordHash(params.Password, user.PasswordHash)
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "could not verify password", err)
		return
	}
	if !match {
		response.RespondWithError(w, http.StatusUnauthorized, "invalid credentials", nil)
		return
	}

	err = h.DB.DeleteUser(r.Context(), userId)
	if err != nil {
		response.RespondWithError(w, http.StatusInternalServerError, "could not delete user", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func databaseUserToResponseUser(user database.User) User {
	return User{
		ID:        user.ID,
		Email:     user.Email,
		Username:  user.Username,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
