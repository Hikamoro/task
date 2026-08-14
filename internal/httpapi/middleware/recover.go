package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func Recover(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered", "value", rec)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]string{"code": "internal", "message": "internal server error"},
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
