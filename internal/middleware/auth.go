package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"task_manager/internal/service"
)

type contextKey string

const userIDKey contextKey = "user_id"

type AuthMiddleware struct {
	jwtManager *service.JWTManager
}

func NewAuthMiddleware(jwtManager *service.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
	}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			writeJSONError(
				w,
				http.StatusUnauthorized,
				"unauthorized",
				"unauthorized",
			)
			return
		}

		const prefix = "Bearer "

		if !strings.HasPrefix(authHeader, prefix) {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, prefix)

		claims, err := m.jwtManager.Validate(tokenString)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDKey).(int64)
	return userID, ok
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func writeJSONError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(errorResponse{
		Error:   code,
		Message: message,
	})
}
