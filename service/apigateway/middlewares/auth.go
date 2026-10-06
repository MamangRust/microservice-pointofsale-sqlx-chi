package middlewares

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/MamangRust/microservice-point-of-sale-apigateway/httpx"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

var whiteListPaths = []string{
	"/api/auth/login",
	"/api/auth/register",
	"/api/auth/hello",
	"/api/auth/refresh-token",
	"/api/auth/forgot-password",
	"/api/auth/reset-password",
	"/api/auth/verify-code",
	"/docs/",
	"/docs",
	"/swagger",
}

// JWTAuth validates the Bearer access token and stores the token subject in
// the request context under "userID", matching the request-scoped value that
// echojwt previously set on echo.Context.
func JWTAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skipAuth(r) {
				next.ServeHTTP(w, r)
				return
			}

			tokenString, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok || tokenString == "" {
				unauthorized(w)
				return
			}

			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(viper.GetString("SECRET_KEY")), nil
			})
			if err != nil || !token.Valid {
				unauthorized(w)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				unauthorized(w)
				return
			}
			r = r.WithContext(httpx.SetValue(r.Context(), "userID", claims["sub"]))

			next.ServeHTTP(w, r)
		})
	}
}

func skipAuth(r *http.Request) bool {
	path := r.URL.Path

	for _, p := range whiteListPaths {
		if path == p || strings.HasPrefix(path, "/swagger") {
			return true
		}
	}

	return false
}

func unauthorized(w http.ResponseWriter) {
	_ = httpx.JSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
}
