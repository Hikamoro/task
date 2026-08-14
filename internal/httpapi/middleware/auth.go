package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"task/internal/auth"
)

func RequireAuth(m *auth.Manager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeUnauthorized(w)
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := m.Parse(token)
		if err != nil {
			writeUnauthorized(w)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithUserID(r.Context(), claims.UserID)))
	})
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": "unauthorized", "message": "authentication required"},
	})
}
