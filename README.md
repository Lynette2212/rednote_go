# 小红薯 RedNote

小红书风格的图文 / 视频社区，包含 Web、桌面端与 Go 后端。

| 目录 | 技术栈 | 说明 |
|---|---|---|
| `web/` | React 18 + TypeScript + Vite | Web 前端 |
| `desktop/` | Electron 44 + React 18 + TypeScript | 桌面端；自研 FFmpeg 8.1 播放内核 |
| `internal/` | Go + Gin + GORM | 后端 API |
| `docker-compose.yml` | PostgreSQL / Redis / Elasticsearch | 基础设施 |

## 功能

- 竖向信息流、滚轮 / 方向键切换
- 桌面端播放器：FFmpeg 解封装 + 软解、miniaudio 音频输出、音频主时钟音画同步、暂停、精确 seek
- 内置 3 段二次元测试视频（樱夜 / 海边黄昏 / 霓虹雨夜），支持上一段 / 下一段切换
- Redis：信息流缓存（30s）与接口限流
- Elasticsearch：笔记全文检索索引（发布/删除自动同步，启动时回填）

## 依赖下载

- FFmpeg 8.1 shared dev（Windows x64）
  - 官方：https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-n8.1-latest-win64-gpl-shared-8.1.zip
  - 国内镜像：https://gh-proxy.com/https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-n8.1-latest-win64-gpl-shared-8.1.zip
  - 解压后把 `include/`、`lib/`、`bin/` 放到 `desktop/native/third_party/ffmpeg/`
- Electron 44 头文件：https://npmmirror.com/mirrors/electron/v44.9.0/node-v44.9.0-headers.tar.gz
  - 解压到 `desktop/native/third_party/electron/`
- miniaudio：https://raw.githubusercontent.com/mackron/miniaudio/master/miniaudio.h
  - 放到 `desktop/native/third_party/miniaudio.h`

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

### 3. Web

```bash
cd web
npm install
npm run dev
```

打开 http://127.0.0.1:5173

### 4. 桌面端

原生模块需要 Electron 的导入库（由 `electron.exe` 导出生成，见下方脚本），然后：

```bash
cd desktop
npm install
npm run build:native
npm run build
npm start
```

生成 `electron.lib`（PowerShell / Git Bash，需 VS 的 dumpbin 与 lib 在 PATH）：

```bash
cd desktop/native
dumpbin -exports ../node_modules/electron/dist/electron.exe | grep -o 'napi_[A-Za-z0-9_]*' | sort -u > third_party/electron/napi_exports.txt
{ echo "LIBRARY electron.exe"; echo "EXPORTS"; cat third_party/electron/napi_exports.txt; } > third_party/electron/electron.def
lib -def:third_party/electron/electron.def -machine:x64 -out:third_party/electron/electron.lib
```

## 说明

- 仓库不包含 `node_modules`、构建产物、FFmpeg DLL（`avcodec-62.dll` 单文件 114MB，超过 GitHub 限制）以及数据库密码配置。
- 桌面端点信息流封面即可打开播放器，默认播放 `desktop/assets/videos/` 下的二次元测试片。
- 测试视频由 `tools/gen_anime.py` 程序化生成（Pillow 绘制 + 合成配乐，720×1280 / 24fps / H.264 + AAC），重新生成：

```bash
python tools/gen_anime.py
```
