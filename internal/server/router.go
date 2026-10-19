package server

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
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
	slog.Info("服务启动", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		os.Exit(-1)
	}
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
	registerAPIRoutes(app.Group("/api", middleware.AuthMiddleware, apiLimit), authLimit)

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
	group.GET("/user/info", api.UserInfo)
	group.POST("/user/update", api.UserUpdateProfile)
	group.GET("/user/profile/:id", api.UserProfile)

	noteGroup := group.Group("/note")
	noteGroup.POST("/create", api.NoteCreate)
	noteGroup.GET("/feed", api.NoteFeed)
	noteGroup.GET("/follow_feed", api.NoteFollowFeed)
	noteGroup.GET("/user_notes", api.NoteUserNotes)
	noteGroup.GET("/user_likes", api.NoteUserLikes)
	noteGroup.GET("/user_favorites", api.NoteUserFavorites)
	noteGroup.GET("/search", api.NoteSearch)
	noteGroup.POST("/delete/:id", api.NoteDelete)
	noteGroup.GET("/:id", api.NoteDetail)

	commentGroup := group.Group("/comment")
	commentGroup.POST("/create", api.CommentCreate)
	commentGroup.GET("/list", api.CommentList)
	commentGroup.POST("/delete/:id", api.CommentDelete)

	group.POST("/like/add", api.LikeAdd)
	group.POST("/like/remove", api.LikeRemove)
	group.POST("/favorite/add", api.FavoriteAdd)
	group.POST("/favorite/remove", api.FavoriteRemove)
	group.POST("/follow/add", api.FollowAdd)
	group.POST("/follow/remove", api.FollowRemove)

	group.POST("/upload", api.Upload)
}
