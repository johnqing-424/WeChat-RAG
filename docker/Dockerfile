FROM golang:1.24 AS build-stage

WORKDIR /app
ENV GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod/ go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod/ --mount=type=cache,target=/root/.cache/go-build/ \
    CGO_ENABLED=0 GOOS=linux go build -o /wechat-rag ./cmd/main.go

FROM alpine:latest

# 安装必要工具
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk --no-cache add ca-certificates tzdata curl bash bind-tools iputils

# 设置时区
ENV TZ=Asia/Shanghai

# 创建app目录
WORKDIR /app
COPY --from=build-stage /wechat-rag /app/

# 复制配置文件 - 修改为从项目根目录复制
COPY config.yml /app/

# 添加智能启动脚本
COPY scripts/smart-entrypoint.sh /app/
RUN chmod +x /app/smart-entrypoint.sh

EXPOSE 80

ENTRYPOINT ["/app/smart-entrypoint.sh"]
