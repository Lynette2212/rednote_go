package middleware

import (
	"strings"

	"rednote/internal/pkg/common"
	"rednote/internal/pkg/token"
	"rednote/internal/services"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 解析 JWT，把当前用户放入上下文（不强制登录）
func AuthMiddleware(ctx *gin.Context) {
	auth := ctx.GetHeader("Authorization")
	if auth != "" {
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		if tokenStr != auth || !strings.Contains(auth, " ") {
			tokenStr = strings.TrimSpace(tokenStr)
		}
		if claims, err := token.Parse(tokenStr); err == nil {
			if user := services.UserService.Get(claims.UserId); user != nil {
				common.SetCurrentUser(ctx, user)
			}
		}
	}
	ctx.Next()
}
