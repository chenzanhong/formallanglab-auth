package handler

import (
	"github.com/chenzanhong/goutil/jwtx"
	"github.com/gin-gonic/gin"

	"github.com/chenzanhong/formallanglab-auth/internal/middleware/cors"
	"github.com/chenzanhong/formallanglab-auth/internal/middleware/metrics"
	"github.com/chenzanhong/formallanglab-auth/internal/middleware/rate"
	"github.com/chenzanhong/formallanglab-auth/internal/middleware/requestid"
)

func SetupRouter(userHandler *UserHandler, emailHandler *EmailHandler) *gin.Engine {
	router := gin.Default()

	// 1. 恢复中间件 - 最先使用，捕获所有panic
	// router.Use(gin.Recovery())

	// 2. 请求ID中间件 - 尽早设置，让后续中间件都能使用
	router.Use(requestid.RequestID())
	// 3. 全局速率限制 - 在处理请求初期进行限制，避免资源浪费，
	// 但为了与UserRateLimitMiddleware不重复，只在后面的公共路由组添加
	// router.Use(rate.GlobalRateLimitMiddleware())
	// 4. CORS中间件 - 尽早处理跨域请求，避免不必要的后续处理
	router.Use(cors.CORSMiddleware())
	// 5. 指标收集 - 收集所有处理过程的指标
	router.Use(metrics.HTTPMiddleware())

	auth := router.Group("/gdesign/auth")
	setupPublicRoutes(auth, userHandler, emailHandler) // 注册公开路由
	// setupAuthRoutes(router, userHandler, aiHandler, learnHandler) // 注册需要认证的路由
	auth.GET("/me", jwtx.GinJWTAuthMiddleware(), userHandler.CheckMe)

	return router
}

func setupPublicRoutes(router *gin.RouterGroup, userHandler *UserHandler, emailHandler *EmailHandler) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})
	router.HEAD("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	}) // 不需要限速；prometheus.yml中加上 metrics_path: /metrics
	router.POST("/register", rate.GlobalRateLimitMiddleware(), userHandler.Register)                            // 注册
	router.POST("/login", rate.GlobalRateLimitMiddleware(), userHandler.Login)                                  // 登入
	router.POST("/logout", rate.GlobalRateLimitMiddleware(), userHandler.Logout)                                // 登出
	router.POST("/refresh", rate.GlobalRateLimitMiddleware(), userHandler.Refresh)                              // 刷新 access token 和 refresh token
	router.POST("/reset-pwd", rate.GlobalRateLimitMiddleware(), userHandler.ResetPassword)                      // 重置密码
	router.POST("/register/code", rate.GlobalRateLimitMiddleware(), emailHandler.SendRegisterVerificationCode)  // 发送注册验证码
	router.POST("/reset-pwd/code", rate.GlobalRateLimitMiddleware(), emailHandler.SendResetPwdVerificationCode) // 发送请求重置密码的验证码
}
