package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kaszta1274/macrorecipe/internal/api/handler"
	appmw "github.com/kaszta1274/macrorecipe/internal/api/middleware"
	"github.com/kaszta1274/macrorecipe/internal/database"
)

type ApiConfig struct {
	DB          *database.Queries
	TokenSecret string
}

func (cfg *ApiConfig) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIPFromRemoteAddr)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	userHandler := handler.NewUserHandler(cfg.DB, cfg.TokenSecret)
	ingredientHandler := handler.NewIngredientHandler(cfg.DB, cfg.TokenSecret)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", userHandler.CreateUser)
		r.Post("/login", userHandler.LoginUser)
	})

	r.Route("/api", func(r chi.Router) {
		r.Use(appmw.AuthMiddleware(cfg.TokenSecret))

		r.Get("/users/{id}", userHandler.GetUserById)
		r.Patch("/users/profile", userHandler.UpdateUsername)
		r.Patch("/users/password", userHandler.UpdatePassword)
		r.Delete("/users", userHandler.DeleteUser)

		r.Get("/ingredients/{id}", ingredientHandler.GetIngredientById)
		r.Get("/ingredients", ingredientHandler.ListIngredients)
	})

	r.Route("/api/admin", func(r chi.Router) {
		r.Use(appmw.AuthMiddleware(cfg.TokenSecret))
		r.Use(appmw.RequireAdmin)

		r.Post("/ingredients", ingredientHandler.CreateIngredient)
		r.Put("/ingredients/{id}", ingredientHandler.UpdateIngredient)
		r.Delete("/ingredients/{id}", ingredientHandler.DeleteIngredient)
	})

	return r
}
