package response

import (
	"encoding/json"
	"log"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("error marshaling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(code)
	w.Write(data)
}

func RespondWithError(w http.ResponseWriter, r *http.Request, code int, msg string, err error) {
	reqID := middleware.GetReqID(r.Context())
	if code > 499 {
		slog.ErrorContext(r.Context(), msg,
			slog.String("request_id", reqID),
			slog.Int("status", code),
			slog.Any("error", err),
		)
	} else if err != nil {
		slog.WarnContext(r.Context(), msg,
			slog.String("request_id", reqID),
			slog.Int("status", code),
			slog.Any("error", err),
		)
	}

	clientMsg := msg
	if code > 499 {
		clientMsg = "internal server error"
	}
	type errorResponse struct {
		Error string `json:"error"`
	}
	RespondWithJSON(w, code, errorResponse{
		Error: clientMsg,
	})
}
