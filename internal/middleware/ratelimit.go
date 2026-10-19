package middleware

import (
	"fmt"
	"time"

	"rednote/internal/pkg/cache"
	"rednote/internal/pkg/errs"
	"rednote/internal/pkg/ginx"

	"github.com/gin-gonic/gin"
)

// RateLimit 按客户端 IP 做固定窗口限流（Redis INCR + EXPIRE）。
// scope 区分不同档位（auth/api），limitPerMin 每次请求现读配置；
// Redis 未启用或读取出错时直接放行——限流是保护层，不是功能依赖。
func RateLimit(scope string, limitPerMin func() int) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !cache.Enabled() {
			ctx.Next()
			return
		}
		limit := limitPerMin()
		if limit <= 0 {
			ctx.Next()
			return
		}
		// 固定窗口：分钟级 key，首次自增时设置过期
		key := fmt.Sprintf("rednote:ratelimit:%s:%s:%s", scope, ctx.ClientIP(), time.Now().Format("200601021504"))
		n, err := cache.Incr(ctx.Request.Context(), key, 2*time.Minute)
		if err != nil {
			ctx.Next()
			return
		}
		if n > int64(limit) {
			ginx.WriteJSON(ctx, errs.TooManyRequests())
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}
