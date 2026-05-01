FROM golang:1.21-alpine AS builder

WORKDIR /src

# 安装 git
RUN apk add --no-cache git

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# 编译
RUN CGO_ENABLED=0 go build -o /bin/mem-service ./cmd/mem-service/

# 运行时镜像
FROM alpine:latest

RUN apk add --no-cache git ca-certificates

WORKDIR /opt/feishu-agent-mem

# 复制编译好的二进制
COPY --from=builder /bin/mem-service ./bin/mem-service

# 创建数据目录
RUN mkdir -p ./data/decisions ./config

# 暴露 MCP 端口
EXPOSE 37777

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:37777/health || exit 1

# 启动服务
CMD ["./bin/mem-service"]
