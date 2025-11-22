# 第一阶段：构建阶段
FROM golang:1.24-alpine AS builder

# 设置工作目录
WORKDIR /app

# 设置国内Go代理镜像源，加速依赖下载
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=off

# 复制go.mod和go.sum文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制所有源代码
COPY . .

# 构建应用，设置CGO_ENABLED=0以创建静态二进制文件
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o auth-server cmd/main.go

# 第二阶段：运行阶段
FROM alpine:3.20

# 添加安全标签
LABEL maintainer="GDesign Team"
LABEL version="1.0"
LABEL description="Auth Service for GDesign Project"

# 设置工作目录
WORKDIR /app

# 合并RUN指令，减少镜像层，同时创建非root用户
RUN apk --no-cache add ca-certificates tzdata wget && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    addgroup -g 1000 authuser && \
    adduser -u 1000 -G authuser -h /app -D authuser && \
    mkdir -p /app/logs /app/configs /app/migrations && \
    chown -R authuser:authuser /app

# 切换到非root用户运行应用
USER authuser

# 从构建阶段复制二进制文件
COPY --from=builder /app/auth-server /app/

# 复制配置文件
COPY configs/config.yaml /app/configs/

# 复制迁移文件
COPY migrations/ /app/migrations/

# 创建环境变量示例文件
COPY --chown=authuser:authuser .env.example /app/

# 依赖 docker-compose 的 volume 挂载或环境变量传入真实配置。
# COPY --chown=authuser:authuser .env /app/

# 暴露服务端口
EXPOSE 8080 4000 6060

# 运行应用
CMD ["/app/auth-server"]

# ======================= 使用说明 =======================
# 1. 构建镜像：
#    docker build -t gdesign-auth .
#
# 2. 准备环境：
#    - 创建.env文件（从.env.example复制并填写实际密钥）
#    - 确保PostgreSQL和Redis服务正在运行
#
# 3. 运行容器（方式1：使用环境变量传递敏感信息）：
#    docker run -d \
#      --name gdesign-auth \
#      -p 8080:8080 \
#      -e DB_HOST=host.docker.internal \
#      -e DB_PASSWORD=your_db_password \
#      -e JWT_KEY=your_jwt_key \
#      -e REDIS_HOST=host.docker.internal \
#      gdesign-auth
#
# 4. 运行容器（方式2：使用卷挂载配置文件）：
#    docker run -d \
#      --name gdesign-auth \
#      -p 8080:8080 \
#      -v $(pwd)/.env:/app/.env \
#      -v $(pwd)/configs:/app/configs \
#      -v $(pwd)/logs:/app/logs \
#      -v $(pwd)/migrations:/app/migrations \
#      gdesign-auth