# 说明：本机直连 Docker Hub 会超时，因此基础镜像默认走国内可用的镜像源；
# 模块代理默认 goproxy.cn。可直连官方源的环境用构建参数覆盖即可：
#   docker build \
#     --build-arg BASE_GO=golang:1.26-alpine \
#     --build-arg BASE_ALPINE=alpine:3.20 \
#     --build-arg GOPROXY=https://proxy.golang.org,direct .
# 注意：供 FROM 使用的 ARG 必须声明在第一个 FROM 之前
ARG BASE_GO=docker.m.daocloud.io/library/golang:1.26-alpine
ARG BASE_ALPINE=docker.m.daocloud.io/library/alpine:3.20

# 构建：静态编译 Go 服务（依赖全是纯 Go，不需要 CGO）
FROM ${BASE_GO} AS build
WORKDIR /src
ENV CGO_ENABLED=0
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY internal ./internal
RUN go build -trimpath -ldflags="-s -w" -o /out/rednote .

# 运行：最小镜像 + 非 root 用户
FROM ${BASE_ALPINE}
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 rednote \
    && mkdir -p /data/uploads \
    && chown -R rednote:rednote /data
WORKDIR /app
COPY --from=build /out/rednote ./rednote
USER rednote
# 不依赖配置文件：全部用环境变量提供（见 docker-compose.yml）
ENV REDNOTE_PORT=8082 \
    REDNOTE_UPLOADDIR=/data/uploads \
    TZ=Asia/Shanghai
EXPOSE 8082
VOLUME /data
ENTRYPOINT ["./rednote"]
