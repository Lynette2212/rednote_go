package api

import (
	"context"
	"net/http"
	"time"

	"rednote/internal/pkg/cache"
	"rednote/internal/pkg/es"
	"rednote/internal/pkg/sqls"

	"github.com/gin-gonic/gin"
)

// Healthz 仅表示进程存活，不访问任何依赖
func Healthz(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readyz 检查数据库；Redis / ES 启用时一并检查，失败返回 503
func Readyz(ctx *gin.Context) {
	checkCtx, cancel := context.WithTimeout(ctx.Request.Context(), 2*time.Second)
	defer cancel()

	if db, err := sqls.DB().DB(); err != nil || db.PingContext(checkCtx) != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "dependency": "postgres"})
		return
	}
	if cache.Enabled() && !cache.Ping(checkCtx) {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "dependency": "redis"})
		return
	}
	if es.Enabled() && !es.Ping(checkCtx) {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "dependency": "elasticsearch"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "ready"})
}
