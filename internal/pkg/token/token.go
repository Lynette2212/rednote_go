package token

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"rednote/internal/pkg/cache"
	"rednote/internal/pkg/config"
	"rednote/internal/pkg/errs"

	"github.com/golang-jwt/jwt/v4"
)

const tokenExpireHours = 24 * 7

type Claims struct {
	UserId int64 `json:"userId"`
	jwt.RegisteredClaims
}

func Generate(userId int64) (string, error) {
	claims := Claims{
		UserId: userId,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        newTokenID(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExpireHours * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(config.Instance.JwtSecret))
}

func Parse(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		// 锁定签名算法，防止 alg=none / 算法混淆
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errs.NotLogin()
		}
		return []byte(config.Instance.JwtSecret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return nil, errs.NotLogin()
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errs.NotLogin()
	}
	return claims, nil
}

// Revoke 将 token 加入黑名单直到其自然过期（Redis 未启用时无效果）
func Revoke(ctx context.Context, tokenString string, ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	cache.Set(ctx, revocationKey(tokenString), "1", ttl)
}

func IsRevoked(ctx context.Context, tokenString string) bool {
	_, ok := cache.Get(ctx, revocationKey(tokenString))
	return ok
}

func revocationKey(tokenString string) string {
	sum := sha256.Sum256([]byte(tokenString))
	return "rednote:token:revoked:" + hex.EncodeToString(sum[:])
}

func newTokenID() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
