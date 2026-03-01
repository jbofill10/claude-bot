package api

import (
	"context"
	"net/http"
	"strconv"
)

type contextKey string

const userIDKey contextKey = "userID"

func UserCookieMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("user_id")
		if err != nil || cookie.Value == "" {
			http.Error(w, `{"error":"no user selected"}`, http.StatusUnauthorized)
			return
		}
		id, err := strconv.ParseInt(cookie.Value, 10, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid user_id cookie"}`, http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserID(r *http.Request) int64 {
	id, _ := r.Context().Value(userIDKey).(int64)
	return id
}
