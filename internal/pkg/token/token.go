package token

import (
	"time"

	"rednote/internal/pkg/config"
	"rednote/internal/pkg/errs"

	"github.com/golang-jwt/jwt/v4"
)

const tokenExpireDays = 30

type Claims struct {
	UserId int64 `json:"userId"`
	jwt.RegisteredClaims
}

func Generate(userId int64) (string, error) {
	claims := Claims{
		UserId: userId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExpireDays * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(config.Instance.JwtSecret))
}

func Parse(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(config.Instance.JwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, errs.NotLogin()
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errs.NotLogin()
	}
	return claims, nil
}
