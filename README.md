# Auth 服务

## 模块定位

Auth 服务是 FormalLangLab 项目的身份认证中心（Identity Provider），负责用户账户管理、安全认证和令牌签发。作为整个系统的安全入口，Auth 服务为其他业务服务提供统一的身份验证和授权机制。

## 功能特性

- **用户注册与登录**：支持邮箱注册、用户名密码登录
- **JWT 令牌管理**：签发 Access Token 和 Refresh Token，支持令牌刷新
- **密码管理**：支持密码重置功能
- **验证码机制**：配合 Email 服务完成注册验证码发送
- **用户信息管理**：支持用户信息查询和更新
- **安全登出**：令牌注销和会话清理
- **Kafka 事件驱动**：通过消息队列实现异步事件处理

## 技术栈

| 类别 | 技术 |
|------|------|
| 开发语言 | Go 1.23+ |
| Web 框架 | Gin |
| 数据库 | PostgreSQL |
| 消息队列 | Kafka |
| 令牌机制 | JWT（JSON Web Token） |
| 密码加密 | bcrypt |
| 日志系统 | 结构化日志（zlog） |
| 配置管理 | YAML 配置文件 |
| 监控指标 | Prometheus + Grafana |

## 项目结构

```
backend/auth/
├── cmd/                        # 程序入口
│   └── main.go                 # 主服务入口
├── configs/                    # 配置管理
│   ├── config.go               # 配置结构定义与加载
│   └── config.yaml.example     # 配置文件示例
├── internal/                   # 核心业务代码
│   ├── domain/                 # 领域模型
│   │   ├── dto/                # 数据传输对象
│   │   │   ├── base.go         # 基础 DTO
│   │   │   ├── user.go         # 用户相关 DTO
│   │   │   └── email.go        # 邮件相关 DTO
│   │   └── model/              # 领域实体模型
│   │       ├── user.go         # 用户模型
│   │       └── email.go        # 邮件模型
│   ├── handler/                # HTTP 接口层
│   │   ├── user.go             # 用户处理器
│   │   ├── email.go            # 邮件处理器
│   │   └── router.go           # 路由定义
│   ├── service/                # 业务逻辑层
│   │   ├── user_s/             # 用户服务
│   │   │   ├── user_service.go # 用户服务接口定义
│   │   │   ├── register.go     # 注册逻辑
│   │   │   ├── login.go        # 登录逻辑
│   │   │   ├── refresh.go      # 刷新令牌逻辑
│   │   │   ├── reset_pwd.go    # 重置密码逻辑
│   │   │   └── info.go         # 用户信息逻辑
│   │   ├── email_s/            # 邮件服务
│   │   │   └── email.go        # 邮件服务实现
│   │   └── kafka_s/            # Kafka 服务
│   │       ├── producer.go     # Kafka 生产者
│   │       └── consumer.go     # Kafka 消费者
│   ├── repository/             # 数据访问层
│   │   ├── user_repo.go        # 用户数据访问
│   │   ├── email_repo.go       # 邮件数据访问
│   │   └── init.go             # 存储库初始化
│   ├── middleware/             # 中间件
│   │   ├── cors/               # 跨域中间件
│   │   ├── jwt/                # JWT 认证中间件
│   │   ├── metrics/            # 指标采集中间件
│   │   ├── rate/               # 速率限制中间件
│   │   └── requestid/          # 请求 ID 中间件
│   └── errors/                 # 错误定义
│       ├── user_errors.go      # 用户相关错误
│       └── email_errors.go     # 邮件相关错误
├── pkg/                        # 公共包
│   ├── cryptoutil/             # 加密工具包
│   │   └── crypto.go           # 密码加密工具
│   └── token/                  # 令牌工具包
│       └── token.go            # JWT 令牌生成与验证
├── migrations/                 # 数据库迁移脚本
├── logs/                       # 日志文件目录
├── .gitignore                  # Git 忽略配置
├── .golangci.yml               # Go 代码检查配置
├── Dockerfile                  # Docker 镜像构建文件
├── Makefile                    # Make 命令配置
├── go.mod                      # Go 模块依赖
└── go.sum                      # Go 依赖校验文件
```

## 中间件

| 中间件 | 文件路径 | 功能描述 |
|--------|---------|---------|
| RequestID | `middleware/requestid/requestid.go` | 为每个请求生成唯一 ID，便于日志追踪和调试 |
| CORS | `middleware/cors/cors.go` | 处理跨域请求，支持预检请求和凭证传递 |
| JWT | `middleware/jwt/jwt.go` | JWT Token 认证，验证请求头中的 Token 有效性 |
| UserRateLimit | `middleware/rate/rate.go` | 用户级速率限制，防止恶意请求和暴力破解 |
| Metrics | `middleware/metrics/metrics.go` | Prometheus 指标采集，监控请求延迟、QPS 和错误率 |

## 核心服务

### 用户服务（User Service）

**文件**：`internal/service/user_s/`

**功能模块**：
- **注册（Register）**：处理用户注册请求，验证邮箱唯一性，触发验证码发送
- **登录（Login）**：验证用户名密码，签发 JWT 令牌（Access Token + Refresh Token）
- **刷新令牌（Refresh）**：使用 Refresh Token 获取新的 Access Token
- **重置密码（Reset Password）**：通过邮箱验证码重置用户密码
- **用户信息（Info）**：查询和更新用户基本信息
- **登出（Logout）**：注销令牌，清理会话状态

### 邮件服务（Email Service）

**文件**：`internal/service/email_s/email.go`

**功能**：
- 发送注册验证码
- 发送密码重置验证码
- 验证码生成与验证
- 请求频率限制（1 分钟内只能发送 1 次）

### Kafka 服务（Kafka Service）

**文件**：`internal/service/kafka_s/`

**功能**：
- **生产者（Producer）**：向 Kafka 发送用户操作事件（如注册、登录）
- **消费者（Consumer）**：消费异步事件，触发验证码发送等操作

## 数据访问层

### 用户存储库（User Repository）

**文件**：`internal/repository/user_repo.go`

**功能**：
- 用户 CRUD 操作
- 用户名和邮箱唯一性检查
- 用户密码验证
- 用户信息查询和更新

### 邮件存储库（Email Repository）

**文件**：`internal/repository/email_repo.go`

**功能**：
- 验证码记录管理
- 验证码有效性验证
- 验证码发送频率控制

## 领域模型

### 数据传输对象（DTO）

**文件**：`internal/domain/dto/`

**主要 DTO**：
- `RegisterRequest`：注册请求体
- `LoginRequest`：登录请求体
- `RefreshTokenRequest`：刷新令牌请求体
- `ResetPasswordRequest`：重置密码请求体
- `UserInfoResponse`：用户信息响应体
- `SendVerificationCodeRequest`：发送验证码请求体

### 领域实体（Model）

**文件**：`internal/domain/model/`

**主要模型**：
- `User`：用户实体（ID、用户名、邮箱、密码哈希、创建时间等）
- `VerificationCode`：验证码实体（邮箱、验证码、过期时间等）

## 令牌机制

### JWT 配置

**文件**：`pkg/token/token.go`

**令牌类型**：
- **Access Token**：短期有效（默认 30 分钟），用于 API 请求认证
- **Refresh Token**：长期有效（默认 7 天），用于刷新 Access Token

**令牌内容**：
- `user_id`：用户 ID
- `username`：用户名
- `email`：用户邮箱
- `exp`：过期时间
- `iat`：签发时间
- `iss`：签发者

### 密码加密

**文件**：`pkg/cryptoutil/crypto.go`

**功能**：
- 使用 bcrypt 算法对用户密码进行哈希加密
- 支持密码哈希验证
- 可配置加密成本因子（cost factor）

## API 接口

### 用户认证接口

#### 用户注册

- **路由**：`POST /gdesign/auth/register`
- **功能**：创建新用户账户
- **流程**：验证邮箱 → 检查唯一性 → 密码加密 → 创建用户 → 返回令牌

#### 用户登录

- **路由**：`POST /gdesign/auth/login`
- **功能**：用户身份验证
- **响应**：Access Token + Refresh Token（通过 Cookie 返回）

#### 刷新令牌

- **路由**：`POST /gdesign/auth/refresh`
- **功能**：使用 Refresh Token 获取新的 Access Token
- **机制**：验证 Refresh Token → 签发新令牌对

#### 用户登出

- **路由**：`POST /gdesign/auth/logout`
- **功能**：注销当前用户的令牌
- **操作**：清除 Refresh Token 记录，前端清空本地状态

### 用户管理接口

#### 获取用户信息

- **路由**：`GET /gdesign/auth/user/info`
- **认证**：JWT Token
- **功能**：获取当前登录用户的基本信息

#### 更新用户信息

- **路由**：`PUT /gdesign/auth/user/info`
- **认证**：JWT Token
- **功能**：更新用户基本信息（昵称、头像等）

### 验证码接口

#### 发送注册验证码

- **路由**：`POST /gdesign/auth/email/send-register-code`
- **功能**：向指定邮箱发送注册验证码
- **限制**：1 分钟内只能发送 1 次

#### 发送重置密码验证码

- **路由**：`POST /gdesign/auth/email/send-reset-code`
- **功能**：向指定邮箱发送密码重置验证码
- **限制**：1 分钟内只能发送 1 次

### 系统接口

#### 健康检查

- **路由**：`GET /api/health`
- **功能**：服务健康状态检查

## 配置说明

**文件**：`configs/config.yaml.example`

**主要配置项**：
- 服务端口配置
- PostgreSQL 数据库配置
- JWT 配置（密钥、Access Token 过期时间、Refresh Token 过期时间）
- Kafka 配置（生产者、消费者）
- 日志配置
- 速率限制配置
- bcrypt 加密成本因子

## 安全机制

### 密码安全

- 使用 bcrypt 算法进行密码哈希
- 可配置加密成本因子（默认 10）
- 密码长度和复杂度验证

### 令牌安全

- Access Token 短期有效，降低泄露风险
- Refresh Token 长期有效，支持无感刷新
- 令牌包含用户关键信息，签名防篡改
- 登出时主动注销令牌

### 速率限制

- 注册接口：防止批量注册恶意账户
- 登录接口：防止暴力破解密码
- 验证码发送：1 分钟内只能发送 1 次

### 事件驱动

- 通过 Kafka 实现异步事件处理
- 验证码发送与注册流程解耦
- 支持事件日志记录和审计

## Docker 与 Makefile 使用说明

### Dockerfile 说明

Dockerfile 采用多阶段构建，分为构建阶段和运行阶段：

**构建阶段**：
- 使用 Go 1.24 Alpine 镜像作为构建环境
- 配置国内 Go 代理镜像源加速依赖下载
- 先下载依赖再复制源代码，优化 Docker 缓存层
- 构建静态二进制文件（CGO_ENABLED=0）

**运行阶段**：
- 使用 Alpine 3.20 精简镜像
- 创建非 root 用户（authuser）运行应用，提升安全性
- 配置 Asia/Shanghai 时区
- 复制二进制文件和配置文件
- 暴露服务端口（默认 8082）

### Makefile 命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `make lint` | 运行代码质量检查 | `make lint` |
| `make lint-fix` | 运行代码质量检查并自动修复 | `make lint-fix` |
| `make build` | 构建并推送 Docker 镜像 | `make build` |
| `make build T=false` | 仅构建镜像，不推送 | `make build T=false` |
| `make deploy` | 构建镜像并部署服务 | `make deploy` |

**快速开始**：
```bash
# 1. 代码质量检查
make lint

# 2. 构建镜像（本地测试，不推送）
make build T=false

# 3. 构建并推送镜像
make build

# 4. 部署服务
make deploy
```
