# 小红薯 RedNote

小红书风格的图文 / 视频社区，包含 Web 前端与 Go 后端。

| 目录 | 技术栈 | 说明 |
|---|---|---|
| `web/` | React 18 + TypeScript + Vite | Web 前端（图文 + 视频） |
| `internal/` | Go + Gin + GORM | 后端 API（分层参考 bbs-go） |
| `docker-compose.yml` | PostgreSQL / Redis / Elasticsearch | 基础设施 |

## 功能

- 信息流、笔记详情、发布、评论、点赞 / 收藏 / 关注、搜索
- 图文与视频笔记：详情页使用原生 `<video>` 播放，信息流展示视频封面与标识
- 上传：图片 ≤ 20MB，视频 mp4/webm ≤ 200MB（保存在 `data/uploads`）
- Redis：信息流缓存（30s）、接口限流
- Elasticsearch：笔记全文检索索引（发布/删除自动同步，启动时回填）

## 目录结构

```text
rednote_go/
├── main.go                  # 服务入口（与 bbs-go 一致）
├── internal/
│   ├── handlers/api/        # HTTP 处理器
│   ├── services/            # 业务逻辑
│   ├── repositories/        # 数据访问
│   ├── models/
│   │   ├── constants/       # 状态等常量
│   │   ├── req/             # 请求 DTO
│   │   └── resp/            # 响应 DTO
│   ├── middleware/
│   ├── server/              # 初始化与路由
│   └── pkg/                 # cache / es / token 等基础设施
├── web/
│   ├── public/covers/       # 演示封面（Unsplash，见 CREDITS.md）
│   ├── public/videos/       # 内置演示视频与封面
│   └── src/                 # React 页面与组件
├── docker-compose.yml
└── rednote.example.yaml
```

运行时上传文件放在 `data/`，不提交到仓库。

## 依赖

- Go 1.26、Node.js 22+、PostgreSQL 18
- Redis / Elasticsearch：`docker compose up -d redis elasticsearch`
- 无需 FFmpeg / Electron；视频播放使用浏览器原生能力

## 启动

### 1. 基础设施

```bash
docker compose up -d redis elasticsearch
```

PostgreSQL 使用本机服务（默认数据库 `rednote`，连接参数见 `rednote.example.yaml`）。

### 2. 后端

```bash
cp rednote.example.yaml rednote.yaml
go build -o rednote.exe .
./rednote.exe
```

默认监听 `http://127.0.0.1:8082`。

运行集成测试（需要测试库 `rednote_test`，Redis / ES 可选）：

```bash
TEST_DATABASE_URL="postgres://postgres:密码@127.0.0.1:5432/rednote_test?sslmode=disable" \
REDNOTE_REDIS_ADDR="127.0.0.1:6379" REDNOTE_ES_ADDR="http://127.0.0.1:9200" \
go test ./...
```

### 3. Web

```bash
cd web
npm install
npm run dev
```

打开 http://127.0.0.1:5173

测试账号：`demo` / `demo123`（昵称“小薯同学”）。
登录后推荐流会自动排除自己的笔记，自己的内容在右上角「我的」里查看。

生产构建时可用 `VITE_API_BASE` 指向后端地址；开发模式由 Vite 代理 `/api` 与 `/uploads`。

## 视频笔记

- 演示视频在 `web/public/videos/`（樱夜 / 海边黄昏 / 霓虹雨夜），封面在 `posters/`
- 发布页支持上传 mp4 / webm，后端保存到 `data/uploads/videos/` 并通过 `/uploads/videos/...` 访问
- 笔记字段：`mediaType`（image/video）、`videoUrl`；视频笔记可用 `images[0]` 作为封面

## 工程化

- HTTP 状态码：按业务 code 映射 400/401/403/404/429/500，保持 JSON 信封
- 路由级鉴权：`OptionalAuth`（公开接口解析用户）与 `RequireAuth`（写接口强制登录）
- JWT：锁定 HS256、7 天有效期、登出写入 Redis 黑名单；封禁用户拒绝登录态
- CORS：显式白名单（http://127.0.0.1:5173、http://localhost:5173），不使用通配符 + credentials
- 错误隐藏：内部异常只记录日志，客户端统一收到“服务器内部错误”
- 上传：请求体与文件流双重大小限制；笔记媒体只允许引用白名单目录
- 关注/粉丝使用冗余计数，Follow/Unfollow 在事务内完成，重复操作幂等
- 评论删除：一级评论级联隐藏回复，并按实际条数扣减计数
- 笔记详情缓存使用 `singleflight` 防击穿；限流按登录用户优先、未登录按 IP（Lua 原子 INCR+EXPIRE）
- 健康检查：`/healthz`（进程存活）、`/readyz`（PostgreSQL，可选 Redis/ES）
- 用户响应统一走 `models/resp.UserResponse`，不直接暴露 GORM 模型
- CI：Go vet / build / test（PostgreSQL service）+ Web 构建

## 说明

- 仓库不包含 `node_modules`、构建产物、数据库密码配置与运行时上传文件。
- 当前视频上传落本地磁盘；生产部署建议替换为对象存储 + CDN，并补充病毒扫描/转码。
