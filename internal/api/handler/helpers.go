package handler

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"math"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/kaszta1274/macrorecipe/internal/api/response"
	"github.com/kaszta1274/macrorecipe/internal/storage"
)

type Macros struct {
	Calories      float32 `json:"calories"`
	Protein       float32 `json:"protein"`
	Fat           float32 `json:"fat"`
	Carbohydrates float32 `json:"carbohydrates"`
}

func calculateMacros(protein, fat, carbs float64) Macros {
	return calculateMacrosPerPortion(protein, fat, carbs, 1)
}

func calculateMacrosPerPortion(protein, fat, carbs float64, portions int) Macros {
	if portions < 1 {
		portions = 1
	}
	n := float64(portions)
	// Strip float32 storage noise (REAL columns) before dividing so list and detail views agree.
	p := (math.Round(protein*100) / 100) / n
	f := (math.Round(fat*100) / 100) / n
	c := (math.Round(carbs*100) / 100) / n
	return Macros{
		Calories:      round1(4*p + 9*f + 4*c),
		Protein:       round1(p),
		Fat:           round1(f),
		Carbohydrates: round1(c),
	}
}

func round1(v float64) float32 {
	return float32(math.Round(v*10) / 10)
}

func nullStringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

func saveImageOrRespond(w http.ResponseWriter, r *http.Request, store *storage.LocalStorage, subdir string) (string, bool) {
	urlPath, err := store.SaveImage(w, r, "image", subdir)
	if err == nil {
		return urlPath, true
	}
	switch {
	case errors.Is(err, storage.ErrFileTooLarge):
		response.RespondWithError(w, r, http.StatusRequestEntityTooLarge, err.Error(), err)
	case errors.Is(err, storage.ErrInvalidImageType), errors.Is(err, storage.ErrMissingFile):
		response.RespondWithError(w, r, http.StatusBadRequest, err.Error(), err)
	default:
		response.RespondWithError(w, r, http.StatusInternalServerError, "could not save image", err)
	}
	return "", false
}

func deleteImageBestEffort(ctx context.Context, store *storage.LocalStorage, urlPath string) {
	if err := store.Delete(urlPath); err != nil {
		slog.WarnContext(ctx, "could not delete image file",
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.String("path", urlPath),
			slog.Any("error", err),
		)
	}
}

func mapSlice[In, Out any](in []In, fn func(In) Out) []Out {
	out := make([]Out, len(in)) // non-nil when empty so JSON marshals as [] instead of null
	for i, v := range in {
		out[i] = fn(v)
	}
	return out
}
