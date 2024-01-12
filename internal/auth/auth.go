package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const TokenExp = time.Hour * 3
const SecretKey = "super_secret_key"
const AuthHeader = "Authorization"

type contextKey string

const TokenClaimsContextFieldName contextKey = "tokenClaims"

type TokenClaims struct {
	jwt.RegisteredClaims
	UserID int64
}

func CreateJwtToken(id int64) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
		},

		UserID: id,
	})

	tokenString, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func getTokenClaims(token string) (*TokenClaims, error) {
	tokenClaims := &TokenClaims{}

	parsedToken, err := jwt.ParseWithClaims(token, tokenClaims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(SecretKey), nil
		})
	if err != nil {
		return nil, err
	}

	if !parsedToken.Valid {
		fmt.Println("Token is not valid")
		return nil, err
	}

	return tokenClaims, nil
}

func WithAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		token := strings.Replace(req.Header.Get(AuthHeader), "Bearer ", "", 1)

		tokenClaims, tokenClaimsErr := getTokenClaims(token)
		if tokenClaimsErr != nil {
			http.Error(res, "authorization error", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(req.Context(), TokenClaimsContextFieldName, tokenClaims)

		h.ServeHTTP(res, req.WithContext(ctx))
	}
}
