package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/kaszta1274/macrorecipe/internal/api"
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

type UserCreateBody struct {
	Password string `json:"password"`
	Email    string `json:"email"`
	Username string `json:"username"`
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
	params := UserCreateBody{}
	err := decoder.Decode(&params)
	if err != nil {
		api.RespondWithError(w, http.StatusBadGateway, "invalid request payload", err)
		return
	}

	if params.Password == "" || params.Email == "" || params.Username == "" {
		api.RespondWithError(w, http.StatusBadRequest, "email, username and password are required", nil)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "could not hash password", err)
		return
	}

	user, err := h.DB.CreateUser(r.Context(), database.CreateUserParams{
		Email:        params.Email,
		Username:     params.Username,
		PasswordHash: hashedPassword,
		Role:         database.UserRoleUser,
	})
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "could not create user", err)
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, User{
		ID:        user.ID,
		Email:     user.Email,
		Username:  user.Username,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
}
