package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime/trace"
	"syscall"
	"time"

	"github.com/chenzanhong/formallanglab-auth/configs"
	"github.com/chenzanhong/formallanglab-auth/internal/api"
	mtr "github.com/chenzanhong/formallanglab-auth/internal/metrics"
	"github.com/chenzanhong/formallanglab-auth/internal/middleware"
	rep "github.com/chenzanhong/formallanglab-auth/internal/repository"
	emailSvc "github.com/chenzanhong/formallanglab-auth/internal/service/email_s"
	kafka_s "github.com/chenzanhong/formallanglab-auth/internal/service/kafka_s"
	userSvc "github.com/chenzanhong/formallanglab-auth/internal/service/user_s"
	"github.com/chenzanhong/goutil/jwtx"
	"github.com/chenzanhong/zlog"
	"github.com/gin-gonic/gin"
)

func init() {
	mtr.PrometheusRegister() // 初始化 Prometheus
}

func main() {
	// 1. 加载配置
	config, err := configs.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败：%v", err.Error())
	}
	// 2. 设置环境变量
	configs.SyncConfigToEnv(*config)
	// 3. 初始化JWT
	jwtx.InitWithHS256(
		os.Getenv("JWT_KEY"),
		&middleware.AccessTokenClaims{},
		jwtx.WithAutoInject(true),
	)

	// 4. 初始化日志
	zlog.InitLogger(config.Log)

	// 5. 初始化数据库
	repo, err := rep.Init()
	if err != nil {
		zlog.Fatalf("Failed to initialize database: %v", err)
	}

	// 6. 组装服务
	userRepo := rep.NewUserRepository(repo.DB, repo.Redis)
	emailRepo := rep.NewEmailRepository(repo.DB, repo.Redis)
	kafkaProducer := kafka_s.NewDefaultKafkaProducerService()
	userService := userSvc.NewUserService(userRepo, emailRepo, config.JWT)
	emailService := emailSvc.NewEmailService(emailRepo, userRepo, kafkaProducer)

	// 7. 初始化处理器
	userHandler := api.NewUserHandler(userService)
	emailHandler := api.NewEmailHandler(emailService)

	// 8. 注册路由
	r := api.SetupRouter(userHandler, emailHandler)

	// 9. 创建 HTTP 服务实例
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", os.Getenv("SERVER_PORT")),
		Handler: r,
	}

	// 10. 创建 context 监听系统信号
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 应用 trace（通过 ENABLE_TRACE 环境变量控制）
	go func() {
		if v, ok := os.LookupEnv("ENABLE_TRACE"); ok && v == "true" {
			f, _ := os.Create("server.trace")
			defer f.Close()
			trace.Start(f)
			// go tool trace server.trace
			defer trace.Stop()
		}
	}()

	// 11. 启动pprof http服务（通过 PPROF_PORT 环境变量控制，默认为 6060）
	go func() {
		if v, ok := os.LookupEnv("PPROF_PORT"); ok && v != "" && v != "0" {
			zlog.Infof("Starting pprof on localhost:%d", config.Server.PprofPort)
			http.ListenAndServe(fmt.Sprintf("localhost:%d", config.Server.PprofPort), nil)
		}
	}()

	// 12. 启动独立的 metrics 服务（通过 METRICS_PORT 环境变量控制）
	go func() {
		if metricsPort := os.Getenv("METRICS_PORT"); metricsPort != "0" && metricsPort != "" {
			r := gin.New()
			r.Use(gin.Recovery())
			zlog.Infow("Starting metrics on localhost:", metricsPort)
			r.GET("/gdesign/auth/metrics", mtr.MetricsHandler())
			r.Run(fmt.Sprintf(":%s", metricsPort))
		}
	}()

	// 13. 启动主 HTTP 服务
	// 根据 SERVER_ENV 环境变量决定启动 HTTP 或 HTTPS 服务
	if v, ok := os.LookupEnv("SERVER_ENV"); ok && v == "pro" {
		certFile := "localhost.crt"
		keyFile := "localhost.key"

		// 检查证书是否存在
		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			zlog.Fatalf("TLS certificate not found. Please generate localhost.crt and localhost.key")
		}
		// 启动 HTTPS 服务
		go func() {
			// 启动 HTTPS 服务器
			zlog.Infof("Starting HTTPS server on https://localhost:%s", os.Getenv("SERVER_PORT"))
			if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				zlog.Fatalf("HTTP server ListenAndServe error: %v", err)
			}
		}()
	} else {
		// 启动 HTTP 服务
		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				zlog.Fatalf("HTTP server ListenAndServe error: %v", err)
			}
		}()
	}

	zlog.Info("Server started on :8080")

	// 等待中断信号（SIGINT, SIGTERM）
	<-ctx.Done()

	zlog.Info("Shutting down server...")

	// 14. 优雅关闭 HTTP 服务
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		zlog.Errorf("HTTP server Shutdown error: %v", err)
	} else {
		zlog.Info("HTTP server gracefully stopped")
	}

	// 15. 关闭 PostgreSQL 数据库连接
	sqlDB, gormErr := repo.DB.DB()
	if gormErr == nil {
		if err := sqlDB.Close(); err != nil {
			zlog.Errorf("PostgreSQL GORM DB Close error: %v", err)
		} else {
			zlog.Info("PostgreSQL GORM DB closed")
		}
	} else {
		zlog.Error("Failed to get underlying SQL DB from GORM")
	}

	// 16. 关闭 Redis 连接
	if err := repo.Redis.Close(); err != nil {
		zlog.Errorf("Redis Close error: %v", err)
	} else {
		zlog.Info("Redis connection closed")
	}

	// 17. 关闭 Kafka 生产者
	kafkaProducer.Close()

	zlog.Info("Server exited")
}
