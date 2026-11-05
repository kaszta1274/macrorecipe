package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/kaszta1274/macrorecipe/internal/api/response"
	"github.com/kaszta1274/macrorecipe/internal/auth"
	"github.com/kaszta1274/macrorecipe/internal/database"
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
	if err != nil {
		response.RespondWithError(w, http.StatusUnauthorized, "incorrect email or password", err)
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
