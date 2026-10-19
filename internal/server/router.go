package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"rednote/internal/handlers/api"
	"rednote/internal/middleware"
	"rednote/internal/pkg/config"
	"rednote/internal/pkg/errs"
	"rednote/internal/pkg/ginx"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// maxJSONBody 非 multipart 请求体上限，防止超大 JSON 打爆内存
const maxJSONBody = 1 << 20 // 1MB

func NewServer() {
	srv := &http.Server{
		Addr:              ":" + cast.ToString(config.Instance.Port),
		Handler:           NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("服务启动", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("服务异常退出", "err", err)
			os.Exit(-1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("收到退出信号，开始优雅关闭")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("优雅关闭超时", "err", err)
		return
	}
	slog.Info("服务已关闭")
}

// NewRouter 构建路由；集成测试可直接复用
func NewRouter() *gin.Engine {
	conf := config.Instance

	gin.SetMode(gin.ReleaseMode)
	app := gin.New()
	app.Use(gin.Recovery())
	app.Use(bodyLimit(maxJSONBody))

	if err := app.SetTrustedProxies(nil); err != nil {
		panic(err)
	}
	app.MaxMultipartMemory = 8 << 20

	corsConfig := cors.Config{
		AllowOrigins:     conf.AllowedOrigins,
		AllowCredentials: true,
		MaxAge:           600,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodHead},
		AllowHeaders:     []string{"*"},
	}
	if len(corsConfig.AllowOrigins) == 0 {
		corsConfig.AllowAllOrigins = true
		corsConfig.AllowCredentials = false
	}
	app.Use(cors.New(corsConfig))

	// 上传文件静态访问
	app.Static("/uploads", conf.UploadDir)

	// 全 API 宽松限流；登录/注册在路由上叠加更严格的一档
	apiLimit := middleware.RateLimit("api", func() int { return config.Instance.RateLimit.ApiPerMin })
	authLimit := middleware.RateLimit("auth", func() int { return config.Instance.RateLimit.AuthPerMin })
	registerAPIRoutes(app.Group("/api", middleware.OptionalAuth, apiLimit), authLimit)

	app.NoRoute(func(ctx *gin.Context) {
		ginx.WriteJSON(ctx, errs.NotFound("接口不存在"))
	})

	return app
}

// bodyLimit 给非 multipart 请求套上体积上限（图片上传走 multipart，另有单独校验）
func bodyLimit(limit int64) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		switch ctx.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			if !strings.HasPrefix(ctx.ContentType(), "multipart/") && ctx.Request.Body != nil {
				ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, limit)
			}
		}
		ctx.Next()
	}
}

func registerAPIRoutes(group *gin.RouterGroup, authLimit gin.HandlerFunc) {
	group.POST("/register", authLimit, api.Register)
	group.POST("/login", authLimit, api.Login)
	group.GET("/user/profile/:id", api.UserProfile)

	noteGroup := group.Group("/note")
	noteGroup.GET("/feed", api.NoteFeed)
	noteGroup.GET("/user_notes", api.NoteUserNotes)
	noteGroup.GET("/search", api.NoteSearch)
	noteGroup.GET("/:id", api.NoteDetail)

	commentGroup := group.Group("/comment")
	commentGroup.GET("/list", api.CommentList)

	// 以下接口必须登录；RequireAuth 在路由层统一兜底
	protected := group.Group("", middleware.RequireAuth())
	protected.GET("/user/info", api.UserInfo)
	protected.POST("/user/update", api.UserUpdateProfile)
	protected.POST("/note/create", api.NoteCreate)
	protected.GET("/note/follow_feed", api.NoteFollowFeed)
	protected.GET("/note/user_likes", api.NoteUserLikes)
	protected.GET("/note/user_favorites", api.NoteUserFavorites)
	protected.POST("/note/delete/:id", api.NoteDelete)
	protected.POST("/comment/create", api.CommentCreate)
	protected.POST("/comment/delete/:id", api.CommentDelete)
	protected.POST("/like/add", api.LikeAdd)
	protected.POST("/like/remove", api.LikeRemove)
	protected.POST("/favorite/add", api.FavoriteAdd)
	protected.POST("/favorite/remove", api.FavoriteRemove)
	protected.POST("/follow/add", api.FollowAdd)
	protected.POST("/follow/remove", api.FollowRemove)
	protected.POST("/upload", api.Upload)
}
