package middleware

import (
	"net/http"
	"strings"

	"rednote/internal/pkg/common"
	"rednote/internal/pkg/errs"
	"rednote/internal/pkg/token"
	"rednote/internal/services"

	"github.com/gin-gonic/gin"
)

// OptionalAuth 解析 JWT，把当前用户放入上下文（不强制登录）
func OptionalAuth(ctx *gin.Context) {
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

// RequireAuth 要求已登录；未登录直接中止请求，避免新接口漏写判空逻辑
func RequireAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if common.GetCurrentUser(ctx) == nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    errs.CodeNotLogin,
				"message": "请先登录",
				"success": false,
			})
			return
		}
		ctx.Next()
	}
}
