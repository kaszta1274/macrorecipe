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

	r.Route("/api/users", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(appmw.AuthMiddleware(cfg.TokenSecret))
			r.Get("/{id}", userHandler.GetUserById)
		})
	})

	r.Route("/api/ingredients", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(appmw.AuthMiddleware(cfg.TokenSecret))
			r.Use(appmw.RequireAdmin)
			r.Post("/", ingredientHandler.CreateIngredient)
		})

		r.Group(func(r chi.Router) {
			r.Use(appmw.AuthMiddleware(cfg.TokenSecret))
			r.Get("/{id}", ingredientHandler.GetIngredientById)
		})
	})

	return r
}
