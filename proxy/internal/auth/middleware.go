package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

// Require checks the Authorization header matches the expected key.
// Returns a middleware that wraps any handler.
func Require(expectedKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := extractBearer(r.Header.Get("Authorization"))
			if key == "" || key != expectedKey {
				slog.Warn("unauthorized request", "path", r.URL.Path, "ip", r.RemoteAddr)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "invalid or missing API key",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// extractBearer pulls the token from "Bearer <token>"
func extractBearer(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
