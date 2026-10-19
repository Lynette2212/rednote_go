package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"rednote/internal/models"
	"rednote/internal/pkg/cache"
	"rednote/internal/pkg/config"
	"rednote/internal/pkg/es"
	"rednote/internal/pkg/sqls"
	"rednote/internal/server"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// 集成测试跑在真实 PostgreSQL 上：每个用例创建一个独立 schema，
// 结束时 DROP CASCADE，用例之间互不影响，也绝不碰应用数据库。
//
// 运行前设置（必须指向名字里带 test 的专用库）：
//
//	export TEST_DATABASE_URL='postgres://postgres:密码@127.0.0.1:5432/rednote_test?sslmode=disable'
//	go test -count=1 ./...

func TestMain(m *testing.M) {
	// 测试环境不依赖 rednote.yaml：上传目录指向临时目录，JWT 密钥用测试值
	uploadDir, err := os.MkdirTemp("", "rednote-test-uploads-")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("REDNOTE_UPLOADDIR", uploadDir)
	_ = os.Setenv("REDNOTE_JWT_SECRET", "integration-test-secret")
	// 限流阈值调高避免用例互扰；限流本身由 TestAuthRateLimit 单独验证
	_ = os.Setenv("REDNOTE_RATELIMIT_AUTHPERMIN", "100000")
	_ = os.Setenv("REDNOTE_RATELIMIT_APIPERMIN", "100000")
	if err := config.Init(); err != nil {
		fmt.Fprintln(os.Stderr, "初始化测试配置失败:", err)
		os.Exit(1)
	}
	// 提供了 REDNOTE_REDIS_ADDR 才启用缓存/限流；否则相关用例自动跳过
	if addr := os.Getenv("REDNOTE_REDIS_ADDR"); addr != "" {
		if err := cache.Init(addr, os.Getenv("REDNOTE_REDIS_PASSWORD"), 0); err != nil {
			fmt.Fprintln(os.Stderr, "Redis 不可用，缓存/限流相关用例将跳过:", err)
		}
	}
	// 提供了 REDNOTE_ES_ADDR 才启用 ES 检索（独立测试索引，跑前清空）；
	// 未提供时搜索用例在数据库模糊匹配的降级模式下运行
	if addr := os.Getenv("REDNOTE_ES_ADDR"); addr != "" {
		if err := es.Init(addr, "rednote_notes_test"); err != nil {
			fmt.Fprintln(os.Stderr, "Elasticsearch 不可用，搜索用例将走降级模式:", err)
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			if err := es.DeleteIndex(ctx); err != nil {
				fmt.Fprintln(os.Stderr, "清理测试索引失败:", err)
			}
			if err := es.EnsureIndex(ctx); err != nil {
				fmt.Fprintln(os.Stderr, "创建测试索引失败:", err)
			}
			cancel()
		}
	}
	code := m.Run()
	_ = os.RemoveAll(uploadDir)
	os.Exit(code)
}

type testEnv struct {
	t      *testing.T
	router *gin.Engine
	db     *gorm.DB
}

type apiResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Success bool            `json:"success"`
}

func (r apiResp) into(t *testing.T, out interface{}) {
	t.Helper()
	if err := json.Unmarshal(r.Data, out); err != nil {
		t.Fatalf("解析 data 失败: %v (原始: %s)", err, string(r.Data))
	}
}

type noteDTO struct {
	Id            int64    `json:"id"`
	Title         string   `json:"title"`
	Images        []string `json:"images"`
	LikeCount     int64    `json:"likeCount"`
	FavoriteCount int64    `json:"favoriteCount"`
	CommentCount  int64    `json:"commentCount"`
	Liked         bool     `json:"liked"`
	Favorited     bool     `json:"favorited"`
	Author        struct {
		Id       int64  `json:"id"`
		Nickname string `json:"nickname"`
	} `json:"author"`
}

type pagingDTO struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("需要设置 TEST_DATABASE_URL，例如 postgres://postgres:密码@127.0.0.1:5432/rednote_test?sslmode=disable")
	}
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		t.Fatal("TEST_DATABASE_URL 必须是 postgres:// 或 postgresql:// 连接串")
	}
	if !strings.Contains(strings.ToLower(u.Path), "test") {
		t.Fatalf("拒绝在非测试库上运行（库名必须含 test）: %s", u.Path)
	}

	admin, err := gorm.Open(postgres.Open(dsn), silentGorm())
	if err != nil {
		t.Fatalf("连接测试库失败: %v", err)
	}
	schema := fmt.Sprintf("rednote_test_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatalf("建测试 schema 失败: %v", err)
	}
	t.Cleanup(func() {
		if err := admin.Exec("DROP SCHEMA " + schema + " CASCADE").Error; err != nil {
			t.Errorf("清理测试 schema 失败: %v", err)
		}
		if adminDB, err := admin.DB(); err == nil {
			adminDB.Close()
		}
	})

	query := u.Query()
	query.Set("search_path", schema)
	u.RawQuery = query.Encode()
	db, err := gorm.Open(postgres.Open(u.String()), silentGorm())
	if err != nil {
		t.Fatalf("连接测试 schema 失败: %v", err)
	}
	if err := db.AutoMigrate(models.Models...); err != nil {
		t.Fatalf("测试库建表失败: %v", err)
	}
	sqls.Init(db)
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})

	return &testEnv{t: t, router: server.NewRouter(), db: db}
}

func silentGorm() *gorm.Config {
	return &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)}
}

func (e *testEnv) do(method, path, token string, body interface{}) apiResp {
	e.t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			e.t.Fatal(err)
		}
		reader = bytes.NewReader(data)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)

	var resp apiResp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		e.t.Fatalf("%s %s 返回无法解析: %s", method, path, w.Body.String())
	}
	return resp
}

func (e *testEnv) mustOK(resp apiResp, what string) {
	e.t.Helper()
	if !resp.Success {
		e.t.Fatalf("%s 失败: code=%d message=%s", what, resp.Code, resp.Message)
	}
}

// register 注册一个用户，返回 token 与用户 id
func (e *testEnv) register(username string) (string, int64) {
	e.t.Helper()
	resp := e.do(http.MethodPost, "/api/register", "", map[string]interface{}{
		"username": username,
		"password": "password123",
	})
	e.mustOK(resp, "注册 "+username)
	var data struct {
		Token string `json:"token"`
		User  struct {
			Id int64 `json:"id"`
		} `json:"user"`
	}
	resp.into(e.t, &data)
	if data.Token == "" || data.User.Id == 0 {
		e.t.Fatalf("注册返回缺少 token/user: %s", string(resp.Data))
	}
	return data.Token, data.User.Id
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// uploadFile 以 multipart 形式上传文件，返回接口响应
func (e *testEnv) uploadFile(token, filename string, content []byte) apiResp {
	e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		e.t.Fatal(err)
	}
	if _, err := fw.Write(content); err != nil {
		e.t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		e.t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)

	var resp apiResp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		e.t.Fatalf("上传返回无法解析: %s", w.Body.String())
	}
	return resp
}

func (e *testEnv) uploadPNG(token string) string {
	e.t.Helper()
	resp := e.uploadFile(token, "cover.png", pngBytes(e.t))
	e.mustOK(resp, "上传图片")
	var data struct {
		Url string `json:"url"`
	}
	resp.into(e.t, &data)
	if data.Url == "" {
		e.t.Fatal("上传成功但没有返回 url")
	}
	return data.Url
}

func (e *testEnv) createNote(token, title string) noteDTO {
	e.t.Helper()
	imgURL := e.uploadPNG(token)
	resp := e.do(http.MethodPost, "/api/note/create", token, map[string]interface{}{
		"title":   title,
		"content": "正文内容",
		"images":  []string{imgURL},
		"tags":    []string{"测试"},
	})
	e.mustOK(resp, "发笔记")
	var note noteDTO
	resp.into(e.t, &note)
	return note
}

func (e *testEnv) noteDetail(noteId int64, token string) noteDTO {
	e.t.Helper()
	resp := e.do(http.MethodGet, fmt.Sprintf("/api/note/%d", noteId), token, nil)
	e.mustOK(resp, "笔记详情")
	var note noteDTO
	resp.into(e.t, &note)
	return note
}

// ---------- 认证 ----------

func TestRegisterLoginAndAuth(t *testing.T) {
	e := newTestEnv(t)

	token, userId := e.register("alice_001")
	if userId == 0 {
		t.Fatal("注册未返回用户 id")
	}

	// 登录返回新 token，可访问 /api/user/info
	login := e.do(http.MethodPost, "/api/login", "", map[string]interface{}{
		"username": "alice_001", "password": "password123",
	})
	e.mustOK(login, "登录")
	var loginData struct {
		Token string `json:"token"`
	}
	login.into(t, &loginData)
	if loginData.Token == "" {
		t.Fatal("登录未返回 token")
	}

	info := e.do(http.MethodGet, "/api/user/info", loginData.Token, nil)
	e.mustOK(info, "获取当前用户")
	var user struct {
		Username string `json:"username"`
	}
	info.into(t, &user)
	if user.Username != "alice_001" {
		t.Fatalf("当前用户不符: %s", user.Username)
	}

	// 密码错误 / 用户名重复
	if resp := e.do(http.MethodPost, "/api/login", "", map[string]interface{}{
		"username": "alice_001", "password": "wrong-password",
	}); resp.Success {
		t.Fatal("错误密码竟然登录成功")
	}
	if resp := e.do(http.MethodPost, "/api/register", "", map[string]interface{}{
		"username": "alice_001", "password": "password123",
	}); resp.Success {
		t.Fatal("重复用户名竟然注册成功")
	}

	// 未登录状态访问需登录接口
	noAuth := e.do(http.MethodPost, "/api/note/create", "", map[string]interface{}{
		"title": "x", "images": []string{"a"},
	})
	if noAuth.Success || noAuth.Code != 401 {
		t.Fatalf("未登录发笔记应返回 401，实际 code=%d success=%v", noAuth.Code, noAuth.Success)
	}
	if token == "" {
		t.Fatal("注册返回的 token 为空")
	}
}

// ---------- 发笔记 / 瀑布流 / 详情 ----------

func TestPublishFeedAndDetail(t *testing.T) {
	e := newTestEnv(t)
	token, _ := e.register("author_001")

	note := e.createNote(token, "我的第一篇笔记")
	if note.Id == 0 || note.Title != "我的第一篇笔记" {
		t.Fatalf("发布笔记返回值异常: %+v", note)
	}
	if len(note.Images) != 1 || !strings.HasPrefix(note.Images[0], "/uploads/") {
		t.Fatalf("图片 url 异常: %v", note.Images)
	}
	if note.Author.Nickname == "" {
		t.Fatal("详情缺少作者信息")
	}

	// 未登录也能看推荐流
	feed := e.do(http.MethodGet, "/api/note/feed?page=1&limit=10", "", nil)
	e.mustOK(feed, "获取推荐流")
	var feedData struct {
		Results []noteDTO `json:"results"`
		Paging  pagingDTO `json:"paging"`
	}
	feed.into(t, &feedData)
	if len(feedData.Results) != 1 || feedData.Paging.Total != 1 {
		t.Fatalf("推荐流结果不符: results=%d total=%d", len(feedData.Results), feedData.Paging.Total)
	}

	// 没有图片不允许发布
	noImage := e.do(http.MethodPost, "/api/note/create", token, map[string]interface{}{
		"title": "没有图片",
	})
	if noImage.Success {
		t.Fatal("无图片的笔记竟然发布成功")
	}
}

func TestFeedPagination(t *testing.T) {
	e := newTestEnv(t)
	token, _ := e.register("author_page")
	for i := 1; i <= 3; i++ {
		e.createNote(token, fmt.Sprintf("第%d篇", i))
	}

	first := e.do(http.MethodGet, "/api/note/feed?page=1&limit=2", "", nil)
	e.mustOK(first, "第一页")
	var page1 struct {
		Results []noteDTO `json:"results"`
		Paging  pagingDTO `json:"paging"`
	}
	first.into(t, &page1)
	if len(page1.Results) != 2 || page1.Paging.Total != 3 {
		t.Fatalf("第一页不符: len=%d total=%d", len(page1.Results), page1.Paging.Total)
	}

	second := e.do(http.MethodGet, "/api/note/feed?page=2&limit=2", "", nil)
	e.mustOK(second, "第二页")
	var page2 struct {
		Results []noteDTO `json:"results"`
		Paging  pagingDTO `json:"paging"`
	}
	second.into(t, &page2)
	if len(page2.Results) != 1 {
		t.Fatalf("第二页应有 1 条，实际 %d", len(page2.Results))
	}
	if page1.Results[0].Id == page2.Results[0].Id {
		t.Fatal("分页出现重复数据")
	}
}

// ---------- 关注 ----------

func TestFollowCountsAndIdempotency(t *testing.T) {
	e := newTestEnv(t)
	tokenA, userA := e.register("follow_a")
	_, userB := e.register("follow_b")

	// 未登录不能关注
	if resp := e.do(http.MethodPost, "/api/follow/add", "", map[string]interface{}{
		"userId": userB,
	}); resp.Code != 401 {
		t.Fatalf("未登录关注应返回 401，实际 code=%d", resp.Code)
	}

	follow := func(token string, target int64) apiResp {
		return e.do(http.MethodPost, "/api/follow/add", token, map[string]interface{}{"userId": target})
	}
	unfollow := func(token string, target int64) apiResp {
		return e.do(http.MethodPost, "/api/follow/remove", token, map[string]interface{}{"userId": target})
	}

	e.mustOK(follow(tokenA, userB), "关注")
	// 重复关注必须幂等，计数不能加两次
	e.mustOK(follow(tokenA, userB), "重复关注")
	// 不能关注自己
	if resp := follow(tokenA, userA); resp.Success {
		t.Fatal("关注自己竟然成功")
	}

	var profile struct {
		User struct {
			FollowingCount int64 `json:"followingCount"`
			FansCount      int64 `json:"fansCount"`
		} `json:"user"`
		FollowingCount int64 `json:"followingCount"`
		FansCount      int64 `json:"fansCount"`
		Following      bool  `json:"following"`
	}

	resp := e.do(http.MethodGet, fmt.Sprintf("/api/user/profile/%d", userB), tokenA, nil)
	e.mustOK(resp, "被关注者主页")
	resp.into(t, &profile)
	if profile.FansCount != 1 || profile.User.FansCount != 1 || !profile.Following {
		t.Fatalf("被关注者粉丝数异常: top=%d user=%d following=%v", profile.FansCount, profile.User.FansCount, profile.Following)
	}

	resp = e.do(http.MethodGet, fmt.Sprintf("/api/user/profile/%d", userA), tokenA, nil)
	e.mustOK(resp, "关注者主页")
	resp.into(t, &profile)
	if profile.FollowingCount != 1 || profile.User.FollowingCount != 1 {
		t.Fatalf("关注者关注数异常: top=%d user=%d", profile.FollowingCount, profile.User.FollowingCount)
	}

	e.mustOK(unfollow(tokenA, userB), "取关")
	e.mustOK(unfollow(tokenA, userB), "重复取关")

	resp = e.do(http.MethodGet, fmt.Sprintf("/api/user/profile/%d", userB), tokenA, nil)
	e.mustOK(resp, "取关后被关注者主页")
	resp.into(t, &profile)
	if profile.FansCount != 0 || profile.Following {
		t.Fatalf("取关后粉丝数异常: fans=%d following=%v", profile.FansCount, profile.Following)
	}
}

// ---------- 上传校验（① 的 MIME 嗅探） ----------

func TestUploadRejectsFakeImage(t *testing.T) {
	e := newTestEnv(t)
	token, _ := e.register("uploader_001")

	// 把文本改名成 .jpg：扩展名合法、内容不是图片，必须被拒
	resp := e.uploadFile(token, "evil.jpg", []byte("这不是图片，只是纯文本内容"))
	if resp.Success {
		t.Fatal("内容为文本的 .jpg 竟然上传成功")
	}
	if resp.Code != 400 {
		t.Fatalf("应返回参数错误 400，实际 %d", resp.Code)
	}

	// 扩展名不在白名单：直接拒
	if resp := e.uploadFile(token, "note.txt", []byte("hello")); resp.Success {
		t.Fatal(".txt 竟然上传成功")
	}

	// 真图片能通过
	if url := e.uploadPNG(token); url == "" {
		t.Fatal("真图片上传失败")
	}
}

// ---------- 点赞 / 收藏（① 的事务一致性） ----------

func TestLikeAndFavoriteCounts(t *testing.T) {
	e := newTestEnv(t)
	authorToken, _ := e.register("author_like")
	userToken, _ := e.register("user_like")
	note := e.createNote(authorToken, "点赞测试")

	like := func(action string) {
		resp := e.do(http.MethodPost, "/api/like/"+action, userToken, map[string]interface{}{"noteId": note.Id})
		e.mustOK(resp, "点赞操作 "+action)
	}

	like("add")
	if got := e.noteDetail(note.Id, userToken); got.LikeCount != 1 || !got.Liked {
		t.Fatalf("点赞后应为 1/已赞，实际 %d/%v", got.LikeCount, got.Liked)
	}
	like("add") // 重复点赞：幂等，不重复计数
	if got := e.noteDetail(note.Id, userToken); got.LikeCount != 1 {
		t.Fatalf("重复点赞后仍应为 1，实际 %d", got.LikeCount)
	}
	like("remove") // 重复取消：幂等
	like("remove")
	if got := e.noteDetail(note.Id, userToken); got.LikeCount != 0 || got.Liked {
		t.Fatalf("取消点赞后应为 0/未赞，实际 %d/%v", got.LikeCount, got.Liked)
	}

	fav := func(action string) {
		resp := e.do(http.MethodPost, "/api/favorite/"+action, userToken, map[string]interface{}{"noteId": note.Id})
		e.mustOK(resp, "收藏操作 "+action)
	}
	fav("add")
	fav("add")
	if got := e.noteDetail(note.Id, userToken); got.FavoriteCount != 1 || !got.Favorited {
		t.Fatalf("收藏后应为 1/已藏，实际 %d/%v", got.FavoriteCount, got.Favorited)
	}
	fav("remove")
	fav("remove")
	if got := e.noteDetail(note.Id, userToken); got.FavoriteCount != 0 || got.Favorited {
		t.Fatalf("取消收藏后应为 0/未藏，实际 %d/%v", got.FavoriteCount, got.Favorited)
	}
}

// TestConcurrentLikeConsistency 并发点赞：计数必须与关系表行数一致
func TestConcurrentLikeConsistency(t *testing.T) {
	e := newTestEnv(t)
	authorToken, _ := e.register("author_race")
	note := e.createNote(authorToken, "并发点赞")

	const users = 8
	tokens := make([]string, 0, users)
	for i := 0; i < users; i++ {
		token, _ := e.register(fmt.Sprintf("racer_%02d", i))
		tokens = append(tokens, token)
	}

	var wg sync.WaitGroup
	for _, token := range tokens {
		wg.Add(1)
		go func(tk string) {
			defer wg.Done()
			var buf bytes.Buffer
			_ = json.NewEncoder(&buf).Encode(map[string]interface{}{"noteId": note.Id})
			req := httptest.NewRequest(http.MethodPost, "/api/like/add", &buf)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+tk)
			w := httptest.NewRecorder()
			e.router.ServeHTTP(w, req)
			var resp apiResp
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err == nil && !resp.Success {
				t.Errorf("并发点赞请求失败: %s", resp.Message)
			}
		}(token)
	}
	wg.Wait()

	if got := e.noteDetail(note.Id, authorToken); got.LikeCount != users {
		t.Fatalf("并发点赞后计数应为 %d，实际 %d", users, got.LikeCount)
	}
	var relations int64
	if err := e.db.Model(&models.UserLike{}).Where("note_id = ?", note.Id).Count(&relations).Error; err != nil {
		t.Fatal(err)
	}
	if relations != users {
		t.Fatalf("关系表应有 %d 行，实际 %d", users, relations)
	}

	// 同一用户并发点 5 次：计数只加 1，且所有请求都成功（行锁保证幂等）
	single := tokens[0]
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			_ = json.NewEncoder(&buf).Encode(map[string]interface{}{"noteId": note.Id})
			req := httptest.NewRequest(http.MethodPost, "/api/like/remove", &buf)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+single)
			w := httptest.NewRecorder()
			e.router.ServeHTTP(w, req)
			var resp apiResp
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err == nil && !resp.Success {
				t.Errorf("并发取消点赞失败: %s", resp.Message)
			}
		}()
	}
	wg.Wait()
	if got := e.noteDetail(note.Id, authorToken); got.LikeCount != users-1 {
		t.Fatalf("同一用户取消一次后应为 %d，实际 %d", users-1, got.LikeCount)
	}
}

// ---------- 评论 ----------

func TestCommentCountsAndPermission(t *testing.T) {
	e := newTestEnv(t)
	authorToken, _ := e.register("author_cmt")
	userToken, _ := e.register("user_cmt")
	otherToken, _ := e.register("other_cmt")
	note := e.createNote(authorToken, "评论测试")

	root := e.do(http.MethodPost, "/api/comment/create", userToken, map[string]interface{}{
		"noteId": note.Id, "content": "第一条评论",
	})
	e.mustOK(root, "发一级评论")
	var rootComment struct {
		Id       int64 `json:"id"`
		ParentId int64 `json:"parentId"`
	}
	root.into(t, &rootComment)
	if rootComment.ParentId != 0 {
		t.Fatalf("一级评论 parentId 应为 0，实际 %d", rootComment.ParentId)
	}

	// 回复一级评论
	reply := e.do(http.MethodPost, "/api/comment/create", otherToken, map[string]interface{}{
		"noteId": note.Id, "parentId": rootComment.Id, "content": "回复一下",
	})
	e.mustOK(reply, "回复评论")
	var replyComment struct {
		Id       int64 `json:"id"`
		ParentId int64 `json:"parentId"`
	}
	reply.into(t, &replyComment)

	// 回复"回复"：应扁平化挂到一级评论下（最多两级）
	nested := e.do(http.MethodPost, "/api/comment/create", userToken, map[string]interface{}{
		"noteId": note.Id, "parentId": replyComment.Id, "content": "回复的回复",
	})
	e.mustOK(nested, "回复二级评论")
	var nestedComment struct {
		ParentId int64 `json:"parentId"`
	}
	nested.into(t, &nestedComment)
	if nestedComment.ParentId != rootComment.Id {
		t.Fatalf("三级回复应挂到一级评论 %d，实际 %d", rootComment.Id, nestedComment.ParentId)
	}

	if got := e.noteDetail(note.Id, authorToken); got.CommentCount != 3 {
		t.Fatalf("评论数应为 3，实际 %d", got.CommentCount)
	}

	// 列表：1 条一级评论 + 2 条回复
	list := e.do(http.MethodGet, fmt.Sprintf("/api/comment/list?noteId=%d&page=1&limit=20", note.Id), "", nil)
	e.mustOK(list, "评论列表")
	var listData struct {
		Results []struct {
			Id       int64 `json:"id"`
			Children []struct {
				Id int64 `json:"id"`
			} `json:"children"`
		} `json:"results"`
		Paging pagingDTO `json:"paging"`
	}
	list.into(t, &listData)
	if len(listData.Results) != 1 || len(listData.Results[0].Children) != 2 {
		t.Fatalf("评论列表结构不符: roots=%d children=%d", len(listData.Results), len(listData.Results[0].Children))
	}

	// 删除他人评论：无权限
	if resp := e.do(http.MethodPost, fmt.Sprintf("/api/comment/delete/%d", rootComment.Id), otherToken, nil); resp.Code != 403 {
		t.Fatalf("删他人评论应返回 403，实际 code=%d", resp.Code)
	}
	// 删除自己的评论：计数同步减 1
	del := e.do(http.MethodPost, fmt.Sprintf("/api/comment/delete/%d", rootComment.Id), userToken, nil)
	e.mustOK(del, "删除自己的评论")
	if got := e.noteDetail(note.Id, authorToken); got.CommentCount != 2 {
		t.Fatalf("删除后评论数应为 2，实际 %d", got.CommentCount)
	}
}

// ---------- HTTP 加固（① 的 body 限制与 404） ----------

func TestBodyLimitAndNotFound(t *testing.T) {
	e := newTestEnv(t)

	huge := strings.Repeat("a", 2<<20) // 2MB，超过 1MB 上限
	resp := e.do(http.MethodPost, "/api/register", "", map[string]interface{}{
		"username": "big_001", "password": huge,
	})
	if resp.Success {
		t.Fatal("超限请求体竟然成功")
	}

	missing := e.do(http.MethodGet, "/api/does-not-exist", "", nil)
	if missing.Code != 404 {
		t.Fatalf("未知接口应返回 404，实际 %d", missing.Code)
	}
}

// ---------- Redis 缓存（需 REDNOTE_REDIS_ADDR，未配置自动跳过） ----------

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	if !cache.Enabled() {
		t.Skip("未配置 REDNOTE_REDIS_ADDR，跳过 Redis 相关用例")
	}
	rdb := redis.NewClient(&redis.Options{Addr: os.Getenv("REDNOTE_REDIS_ADDR")})
	t.Cleanup(func() { rdb.Close() })
	return rdb
}

func TestNoteDetailCacheAndInvalidation(t *testing.T) {
	e := newTestEnv(t)
	rdb := newTestRedis(t)
	ctx := context.Background()

	authorToken, _ := e.register("author_cache")
	userToken, _ := e.register("user_cache")
	note := e.createNote(authorToken, "缓存测试")
	key := cache.NoteKey(note.Id)
	rdb.Del(ctx, key)

	// 第一次读取：回源数据库并写缓存
	e.noteDetail(note.Id, "")
	if n, _ := rdb.Exists(ctx, key).Result(); n != 1 {
		t.Fatal("详情读取后应写入缓存")
	}

	// 点赞改变计数：缓存必须被删除
	e.mustOK(e.do(http.MethodPost, "/api/like/add", userToken, map[string]interface{}{"noteId": note.Id}), "点赞")
	if n, _ := rdb.Exists(ctx, key).Result(); n != 0 {
		t.Fatal("点赞后缓存应被删除")
	}

	// 重新读取：拿到最新计数并再次写回缓存
	got := e.noteDetail(note.Id, userToken)
	if got.LikeCount != 1 {
		t.Fatalf("缓存失效后应读到最新计数 1，实际 %d", got.LikeCount)
	}
	if n, _ := rdb.Exists(ctx, key).Result(); n != 1 {
		t.Fatal("重新读取后应再次写回缓存")
	}
}

// ---------- 接口限流（需 REDNOTE_REDIS_ADDR，未配置自动跳过） ----------

func TestAuthRateLimit(t *testing.T) {
	e := newTestEnv(t)
	rdb := newTestRedis(t)
	ctx := context.Background()

	// 清掉本分钟的 auth 窗口，让计数从零开始
	keys, _ := rdb.Keys(ctx, "rednote:ratelimit:auth:*").Result()
	if len(keys) > 0 {
		rdb.Del(ctx, keys...)
	}

	old := config.Instance.RateLimit.AuthPerMin
	config.Instance.RateLimit.AuthPerMin = 3
	t.Cleanup(func() { config.Instance.RateLimit.AuthPerMin = old })

	var got429 bool
	for i := 0; i < 6; i++ {
		resp := e.do(http.MethodPost, "/api/login", "", map[string]interface{}{
			"username": "nobody", "password": "wrong",
		})
		if resp.Code == 429 {
			got429 = true
			break
		}
	}
	if !got429 {
		t.Fatal("超过阈值后应返回 429")
	}
}

// ---------- 搜索（有 ES 走 ES 相关性检索，未配置时降级 ILIKE，两种模式都必须通过） ----------

func TestSearchNotes(t *testing.T) {
	e := newTestEnv(t)
	token, _ := e.register("author_search")
	e.createNote(token, "海边捡贝壳的一天")
	note2 := e.createNote(token, "家门口的面包店")

	search := func(q string) []noteDTO {
		t.Helper()
		resp := e.do(http.MethodGet, "/api/note/search?q="+url.QueryEscape(q), "", nil)
		e.mustOK(resp, "搜索 "+q)
		var data struct {
			Results []noteDTO `json:"results"`
		}
		resp.into(t, &data)
		return data.Results
	}

	// 标题命中
	if got := search("贝壳"); len(got) != 1 || got[0].Title != "海边捡贝壳的一天" {
		t.Fatalf("标题搜索应命中 1 条，实际 %d 条", len(got))
	}

	// 正文命中：新发布的笔记必须立即可搜
	// （ES 侧靠写入时 refresh=wait_for；ILIKE 降级模式天然即时）
	created := e.do(http.MethodPost, "/api/note/create", token, map[string]interface{}{
		"title": "随手记", "content": "正文里藏着长颈鹿三个字",
		"images": []string{"/uploads/placeholder.png"}, "tags": []string{"随笔"},
	})
	e.mustOK(created, "发笔记")
	if got := search("长颈鹿"); len(got) != 1 {
		t.Fatalf("发布后正文搜索应命中 1 条，实际 %d 条", len(got))
	}

	// 标签命中
	if got := search("随笔"); len(got) != 1 {
		t.Fatalf("标签搜索应命中 1 条，实际 %d 条", len(got))
	}

	// 删除后不可再搜到
	e.mustOK(e.do(http.MethodPost, fmt.Sprintf("/api/note/delete/%d", note2.Id), token, nil), "删除笔记")
	if got := search("面包"); len(got) != 0 {
		t.Fatalf("删除后不应再搜到，实际 %d 条", len(got))
	}
}

// ---------- 数据隔离自检：每个用例都在全新 schema 里，计数从零开始 ----------

func TestSchemaIsolation(t *testing.T) {
	e := newTestEnv(t)
	authorToken, _ := e.register("author_rel")
	e.createNote(authorToken, "关系表检查")

	var total int64
	if err := e.db.Model(&models.Note{}).Count(&total).Error; err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("隔离失效：当前 schema 里应只有 1 条笔记，实际 %d", total)
	}
}
