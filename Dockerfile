# 构建阶段 - 使用轻量Go镜像
FROM golang:1.22.2-alpine AS builder

# 设置构建参数
ARG TARGETARCH

# 安装最小化依赖（仅保留构建必需工具）
RUN apk add --no-cache git gcc musl-dev && \
    rm -rf /var/cache/apk/*  # 清理缓存

WORKDIR /app

# 复制依赖文件并下载（利用缓存层）
COPY go.mod go.sum ./
RUN go mod download && \
    go mod verify  # 验证依赖完整性

# 复制源代码
COPY . .

# 构建静态链接的二进制（去除调试信息）
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -ldflags "-s -w -X 'main.GoVersion=$(go version)' -X main.GitCommit=$(git rev-parse HEAD) -X 'main.BuildTime=$(date +"%Y-%m-%d %H:%M:%S")'" \
    -o China_Telecom_Monitor ./cmd/main.go

# 运行阶段 - 使用极小的Alpine镜像（约5MB）
FROM alpine:3.19.1

# 安装必要的基础工具（仅保留CA证书用于HTTPS）
RUN apk add --no-cache ca-certificates tzdata && \
    rm -rf /var/cache/apk/*  # 清理安装缓存

# 创建非root用户（增强安全性同时减少权限）
RUN adduser -D -H -h /app appuser

WORKDIR /app

# 从构建阶段复制二进制文件（仅复制必要文件）
COPY --from=builder /app/China_Telecom_Monitor .

# 创建数据目录并设置权限
RUN mkdir -p /app/data/log /app/data/tokens && \
    chown -R appuser:appuser /app

# 切换到非root用户
USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/China_Telecom_Monitor"]
