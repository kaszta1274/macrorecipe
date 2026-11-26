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

type Recipe struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Instructions string    `json:"instructions"`
	Portions     int       `json:"portions"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RecipeSummary struct {
	ID               uuid.UUID   `json:"id"`
	Name             string      `json:"name"`
	Description      string      `json:"description"`
	Portions         int         `json:"portions"`
	Author           UserSummary `json:"author"`
	MacrosPerPortion Macros      `json:"macros_per_portion"`
	LikesCount       int64       `json:"likes_count"`
	LikedByMe        bool        `json:"liked_by_me"`
	CommentsCount    int64       `json:"comments_count"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

type RecipeDetail struct {
	RecipeSummary
	Instructions string                      `json:"instructions"`
	Ingredients  []RecipeIngredientsByRecipe `json:"ingredients"`
	MacrosTotal  Macros                      `json:"macros_total"`
}

type RecipeRequest struct {
	Name         string `json:"name" validate:"required,min=2,max=100"`
	Description  string `json:"description" validate:"required,min=2,max=1000"`
	Instructions string `json:"instructions" validate:"required,min=2,max=2000"`
	Portions     int    `json:"portions" validate:"required,gte=1,lte=100"`
}

type RecipeHandler struct {
	DB *database.Queries
}

func NewRecipeHandler(db *database.Queries) *RecipeHandler {
	return &RecipeHandler{
		DB: db,
	}
}

func (h *RecipeHandler) CreateRecipe(w http.ResponseWriter, r *http.Request) {
	params := RecipeRequest{}
	if err := request.DecodeJSON(w, r, &params); err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	userId, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get user id", nil)
		return
	}

	recipe, err := h.DB.CreateRecipe(r.Context(), database.CreateRecipeParams{
		UserID:       userId,
		Name:         params.Name,
		Description:  params.Description,
		Instructions: params.Instructions,
		Portions:     int16(params.Portions),
	})
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not create recipe", err)
		return
	}

	response.RespondWithJSON(w, http.StatusCreated, toRecipe(recipe))
}

func (h *RecipeHandler) GetRecipeById(w http.ResponseWriter, r *http.Request) {
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

	recipeId, err := uuid.Parse(id)
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid recipe id format", err)
		return
	}

	recipe, err := h.DB.GetRecipeDetailById(r.Context(), database.GetRecipeDetailByIdParams{
		ViewerID: viewerID,
		ID:       recipeId,
	})
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusNotFound, "recipe not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not get recipe", err)
		return
	}

	ingredients, err := h.DB.ListRecipeIngredientsByRecipeId(r.Context(), recipe.ID)
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not get recipe ingredients", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, toRecipeDetail(recipe, ingredients))
}

func (h *RecipeHandler) ListFeedRecipes(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get user id", nil)
		return
	}

	limit, offset, _ := request.ParsePagination(r)

	recipes, err := h.DB.ListFeedRecipes(r.Context(), database.ListFeedRecipesParams{
		ViewerID: viewerID,
		Offset:   int32(offset),
		Limit:    int32(limit),
	})
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not list feed recipes", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, mapSlice(recipes, toFeedRecipeSummary))
}

func (h *RecipeHandler) ListRecipesByName(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get user id", nil)
		return
	}

	ingredientIDs, err := parseIngredientIDs(r)
	if err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid ingredient_ids format", err)
		return
	}

	limit, offset, query := request.ParsePagination(r)

	recipes, err := h.DB.ListRecipeSummaries(r.Context(), database.ListRecipeSummariesParams{
		SearchQuery:   query,
		IngredientIds: ingredientIDs,
		Limit:         int32(limit),
		Offset:        int32(offset),
		ViewerID:      viewerID,
	})
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not list recipes", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, mapSlice(recipes, toRecipeSummary))
}

func (h *RecipeHandler) ListMyRecipes(w http.ResponseWriter, r *http.Request) {
	userId, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get user id", nil)
		return
	}

	limit, offset, query := request.ParsePagination(r)

	recipes, err := h.DB.ListRecipeSummaries(r.Context(), database.ListRecipeSummariesParams{
		AuthorID:      uuid.NullUUID{UUID: userId, Valid: true},
		SearchQuery:   query,
		IngredientIds: []uuid.UUID{},
		Limit:         int32(limit),
		Offset:        int32(offset),
		ViewerID:      userId,
	})
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not list recipes", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, mapSlice(recipes, toRecipeSummary))
}

func (h *RecipeHandler) ListRecipesByUserId(w http.ResponseWriter, r *http.Request) {
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

	limit, offset, query := request.ParsePagination(r)

	recipes, err := h.DB.ListRecipeSummaries(r.Context(), database.ListRecipeSummariesParams{
		AuthorID:      uuid.NullUUID{UUID: userId, Valid: true},
		SearchQuery:   query,
		IngredientIds: []uuid.UUID{},
		Limit:         int32(limit),
		Offset:        int32(offset),
		ViewerID:      viewerID,
	})
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not list recipes", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, mapSlice(recipes, toRecipeSummary))
}

func (h *RecipeHandler) UpdateRecipe(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := auth.FromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get auth ctx", nil)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid request", nil)
		return
	}

	recipeId, err := uuid.Parse(id)
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
	if authCtx.Role != database.UserRoleAdmin && recipe.UserID != authCtx.UserID {
		response.RespondWithError(w, r, http.StatusForbidden, "you do not have permission to modify this recipe", nil)
		return
	}

	params := RecipeRequest{}
	if err := request.DecodeJSON(w, r, &params); err != nil {
		response.RespondWithError(w, r, http.StatusBadRequest, err.Error(), err)
		return
	}

	updatedRecipe, err := h.DB.UpdateRecipe(r.Context(), database.UpdateRecipeParams{
		ID:           recipe.ID,
		Name:         params.Name,
		Description:  params.Description,
		Instructions: params.Instructions,
		Portions:     int16(params.Portions),
	})
	if errors.Is(err, sql.ErrNoRows) {
		response.RespondWithError(w, r, http.StatusNotFound, "recipe not found", nil)
		return
	}
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not update recipe", err)
		return
	}

	response.RespondWithJSON(w, http.StatusOK, toRecipe(updatedRecipe))
}

func (h *RecipeHandler) DeleteRecipe(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := auth.FromContext(r.Context())
	if !ok {
		response.RespondWithError(w, r, http.StatusUnauthorized, "could not get auth ctx", nil)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.RespondWithError(w, r, http.StatusBadRequest, "invalid request", nil)
		return
	}

	recipeId, err := uuid.Parse(id)
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
	if authCtx.Role != database.UserRoleAdmin && recipe.UserID != authCtx.UserID {
		response.RespondWithError(w, r, http.StatusForbidden, "you do not have permission to delete this recipe", nil)
		return
	}

	err = h.DB.DeleteRecipe(r.Context(), recipe.ID)
	if err != nil {
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not delete recipe", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseIngredientIDs(r *http.Request) ([]uuid.UUID, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("ingredient_ids"))
	if raw == "" {
		return []uuid.UUID{}, nil // non-nil so pq.Array sends '{}' (cardinality 0) instead of NULL
	}

	parts := strings.Split(raw, ",")
	ids := make([]uuid.UUID, 0, len(parts))
	seen := make(map[uuid.UUID]struct{}, len(parts))
	for _, part := range parts {
		id, err := uuid.Parse(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, nil
}

func toRecipe(recipe database.Recipe) Recipe {
	return Recipe{
		ID:           recipe.ID,
		UserID:       recipe.UserID,
		Name:         recipe.Name,
		Description:  recipe.Description,
		Instructions: recipe.Instructions,
		Portions:     int(recipe.Portions),
		CreatedAt:    recipe.CreatedAt,
		UpdatedAt:    recipe.UpdatedAt,
	}
}

func toRecipeDetail(recipe database.GetRecipeDetailByIdRow, ingredients []database.ListRecipeIngredientsByRecipeIdRow) RecipeDetail {
	var totalProtein, totalFat, totalCarbs float64
	for _, ing := range ingredients {
		totalProtein += float64(ing.Protein) * float64(ing.Amount) / 100.0
		totalFat += float64(ing.Fat) * float64(ing.Amount) / 100.0
		totalCarbs += float64(ing.Carbohydrates) * float64(ing.Amount) / 100.0
	}

	portions := int(recipe.Portions)
	summary := RecipeSummary{
		ID:               recipe.ID,
		Name:             recipe.Name,
		Description:      recipe.Description,
		Portions:         portions,
		Author:           UserSummary{ID: recipe.AuthorID, Username: recipe.AuthorUsername},
		MacrosPerPortion: calculateMacrosPerPortion(totalProtein, totalFat, totalCarbs, portions),
		LikesCount:       recipe.LikesCount,
		LikedByMe:        recipe.LikedByMe,
		CommentsCount:    recipe.CommentsCount,
		CreatedAt:        recipe.CreatedAt,
		UpdatedAt:        recipe.UpdatedAt,
	}

	return RecipeDetail{
		RecipeSummary: summary,
		Instructions:  recipe.Instructions,
		Ingredients:   mapSlice(ingredients, toRecipeIngredientDetail),
		MacrosTotal:   calculateMacros(totalProtein, totalFat, totalCarbs),
	}
}

func toRecipeSummary(r database.ListRecipeSummariesRow) RecipeSummary {
	portions := int(r.Portions)
	return RecipeSummary{
		ID:               r.ID,
		Name:             r.Name,
		Description:      r.Description,
		Portions:         portions,
		Author:           UserSummary{ID: r.AuthorID, Username: r.AuthorUsername},
		MacrosPerPortion: calculateMacrosPerPortion(float64(r.TotalProtein), float64(r.TotalFat), float64(r.TotalCarbs), portions),
		LikesCount:       r.LikesCount,
		LikedByMe:        r.LikedByMe,
		CommentsCount:    r.CommentsCount,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}

func toFeedRecipeSummary(r database.ListFeedRecipesRow) RecipeSummary {
	portions := int(r.Portions)
	return RecipeSummary{
		ID:               r.ID,
		Name:             r.Name,
		Description:      r.Description,
		Portions:         portions,
		Author:           UserSummary{ID: r.AuthorID, Username: r.AuthorUsername},
		MacrosPerPortion: calculateMacrosPerPortion(float64(r.TotalProtein), float64(r.TotalFat), float64(r.TotalCarbs), portions),
		LikesCount:       r.LikesCount,
		LikedByMe:        r.LikedByMe,
		CommentsCount:    r.CommentsCount,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}
