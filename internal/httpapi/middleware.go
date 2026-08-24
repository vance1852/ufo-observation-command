package httpapi

import (
	"context"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
)

type contextKey string

const requestIDKey contextKey = "request_id"

func requestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		w.Header().Set("X-Request-ID", id)
		start := time.Now()
		defer func() {
			if recovered := recover(); recovered != nil {
				_ = debug.Stack()
				writeJSON(w, http.StatusInternalServerError, map[string]string{"code": "panic", "message": "internal server error", "request_id": id})
			}
			_ = start
		}()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}
