# syntax=docker/dockerfile:1

# 1. 前端构建阶段
FROM node:22-alpine AS frontend-builder
WORKDIR /web
RUN corepack enable && corepack prepare pnpm@latest --activate
COPY wechat-official-account-admin-fe/package.json wechat-official-account-admin-fe/pnpm-lock.yaml* ./
RUN pnpm install --frozen-lockfile || pnpm install
COPY wechat-official-account-admin-fe/ ./
RUN pnpm build

# 2. 后端构建阶段
FROM golang:1.26-alpine AS backend-builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
# 将第一阶段前端构建好的产物复制过来，供 //go:embed 打包
COPY --from=frontend-builder /web/dist ./wechat-official-account-admin-fe/dist
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w -buildid=" -o /app/woaa main.go

# 3. 最终运行镜像
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata curl && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app
COPY --from=backend-builder /app/woaa /app/woaa
COPY config/ /app/config/
COPY atlas/ /app/atlas/
COPY .env.example /app/.env.example

# 挂载数据与媒体文件卷
VOLUME ["/app/data", "/app/files"]

EXPOSE 9305

CMD ["/app/woaa"]
