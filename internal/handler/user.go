package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/chenzanhong/zlog"
	"github.com/gin-gonic/gin"

	"github.com/chenzanhong/formallanglab-auth/internal/domain/dto"
	myErrors "github.com/chenzanhong/formallanglab-auth/internal/errors"
	"github.com/chenzanhong/formallanglab-auth/internal/middleware/metrics"
	userSvc "github.com/chenzanhong/formallanglab-auth/internal/service/user_s"
)

type UserHandler struct {
	userService userSvc.UserService
}

func NewUserHandler(userService userSvc.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// Register 用户注册
func (h *UserHandler) Register(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.ObserveOperationDuration("user", "register", time.Since(start).Seconds())
	}()
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		metrics.IncOperation("user", "register", "failure: parameter parsing error")
		zlog.Warnw("请求数据格式错误", "detail", err.Error())
		c.JSON(http.StatusBadRequest, dto.RegisterResponse{
			Result: false,
			Msg:    "请求数据格式错误",
		})

		return
	}

	newUser, err := h.userService.Register(c.Request.Context(), req.Name, req.Email, req.Password, req.Token)
	if err != nil {
		switch {
		case errors.Is(err, myErrors.ErrPasswordHashFailed):
			metrics.IncOperation("user", "register", "failure: password encryption error")
			zlog.Errorw("密码加密失败")
			c.JSON(http.StatusInternalServerError, dto.RegisterResponse{Result: false, Msg: "密码加密失败"})

			return
		case errors.Is(err, myErrors.ErrUserCreationFailed):
			metrics.IncOperation("user", "register", "failure: user creation error")
			zlog.Errorw("用户创建失败")
			c.JSON(http.StatusInternalServerError, dto.RegisterResponse{Result: false, Msg: "用户创建失败"})

			return
		default:
			metrics.IncOperation("user", "register", "failure: unknown error")
			zlog.Warnw("注册失败", "detail", err.Error())
			c.JSON(http.StatusInternalServerError, dto.RegisterResponse{Result: false, Msg: "注册失败", Error: err.Error()})

			return
		}
	}

	metrics.IncOperation("user", "register", "success")
	zlog.Infow("注册成功", "user_id", newUser.ID, "username", newUser.Name)
	c.JSON(http.StatusOK, dto.RegisterResponse{
		Result: true,
		Msg:    "注册成功",
		ID:     newUser.ID,
		Name:   newUser.Name,
	})
}

// Login 处理用户登录请求
func (h *UserHandler) Login(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.ObserveOperationDuration("user", "login", time.Since(start).Seconds())
	}()

	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		metrics.IncOperation("user", "login", "failure: parameter parsing error")
		zlog.Warnw("登录数据解析失败", "detail", err.Error())
		c.JSON(http.StatusBadRequest, dto.LoginResponse{
			Result: false,
			Msg:    "登录数据解析失败",
		})

		return
	}

	// 调用服务层登录逻辑
	resp, err := h.userService.Login(c.Request.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, myErrors.ErrUserNotFound):
			metrics.IncOperation("user", "login", "failure: user not found")
			zlog.Warnw("用户名或密码错误", "username", req.Name)
			c.JSON(http.StatusUnauthorized, dto.LoginResponse{Result: false, Msg: "用户名或密码错误"})

			return
		case errors.Is(err, myErrors.ErrInvalidCredentials):
			metrics.IncOperation("user", "login", "failure: invalid password")
			zlog.Warnw("用户名或密码错误", "username", req.Name)
			c.JSON(http.StatusUnauthorized, dto.LoginResponse{Result: false, Msg: "用户名或密码错误"})

			return
		default:
			metrics.IncOperation("user", "login", "failure: unknown error")
			zlog.Warnw("登录失败", "detail", err.Error())
			c.JSON(http.StatusInternalServerError, dto.LoginResponse{Result: false, Msg: "登录失败"})

			return
		}
	}

	// 设置刷新令牌到 HttpOnly Cookie 中
	// HTTP 环境下 Secure=false，HTTPS 环境下 Secure=true
	secure := isRequestHTTPS(c)
	c.SetCookie("refreshToken", resp.RefreshToken, 7*24*60*60, "/", "", secure, true) // 7 天过期

	metrics.IncOperation("user", "login", "success")
	zlog.Infow("登录成功", "user_id", resp.ID, "username", resp.Name)

	response := dto.LoginResponse{
		Result:       true,
		Msg:          "登录成功",
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		Name:         resp.Name,
		ID:           resp.ID,
	}

	c.JSON(http.StatusOK, response)
}

// Logout 处理用户登出请求
func (h *UserHandler) Logout(c *gin.Context) {
	// 从 Cookie 中获取刷新令牌（HTTP 和 HTTPS 都支持）
	refreshToken, err := c.Cookie("refreshToken")
	if err != nil || refreshToken == "" {
		// 如果没有刷新令牌，仍然返回成功登出响应
		c.JSON(http.StatusOK, dto.LogoutResponse{
			Result: true,
			Msg:    "登出成功",
		})

		return
	}

	if err = h.userService.RevokeRefreshToken(c.Request.Context(), refreshToken); err != nil {
		// 即使撤销失败也继续清除 Cookie，避免用户侧残留令牌
		zlog.Warnw("撤销刷新令牌失败", "detail", err.Error())
	}

	// 清除 Cookie 中的刷新令牌
	secure := isRequestHTTPS(c)
	c.SetCookie("refreshToken", "", -1, "/", "", secure, true)

	c.JSON(http.StatusOK, dto.LogoutResponse{
		Result: true,
		Msg:    "登出成功",
	})
}

// Refresh 处理刷新令牌请求
func (h *UserHandler) Refresh(c *gin.Context) {
	var refreshToken string
	var err error

	// 优先从 cookie 获取（HTTP 和 HTTPS 都支持）
	refreshToken, err = c.Cookie("refreshToken")

	// 如果 cookie 没有，且不是 HTTPS，尝试从请求体获取
	if err != nil || refreshToken == "" {
		zlog.Debugw("Refresh token not found in cookie",
			"clientIP", c.ClientIP(),
			"error", err)

		if isRequestHTTPS(c) {
			// HTTPS 情况下没有 cookie，直接返回错误
			zlog.Warnw("Refresh token missing in HTTPS request",
				"clientIP", c.ClientIP(),
				"userAgent", c.Request.UserAgent())

			c.JSON(http.StatusUnauthorized, dto.LoginResponse{
				Result: false,
				Msg:    "缺少刷新令牌",
			})

			return
		}

		// HTTP 情况下，记录警告日志并从请求体获取 refreshToken
		zlog.Warnw("Refresh token request over HTTP (not HTTPS)",
			"clientIP", c.ClientIP(),
			"userAgent", c.Request.UserAgent())

		var req struct {
			RefreshToken string `json:"refreshToken"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
			zlog.Warnw("Refresh token missing in request body",
				"clientIP", c.ClientIP(),
				"error", err)
			c.JSON(http.StatusUnauthorized, dto.LoginResponse{
				Result: false,
				Msg:    "缺少刷新令牌",
			})

			return
		}
		refreshToken = req.RefreshToken
	}

	resp, err := h.userService.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		zlog.Errorw("Refresh token service failed",
			"clientIP", c.ClientIP(),
			"error", err.Error())
		c.JSON(http.StatusUnauthorized, dto.LoginResponse{Result: false, Msg: err.Error()})

		return
	}

	// 设置新的 refreshToken 到 cookie
	secure := isRequestHTTPS(c)
	c.SetCookie("refreshToken", resp.RefreshToken, 7*24*60*60, "/", "", secure, true)

	c.JSON(http.StatusOK, dto.RefreshResponse{
		Result:       true,
		Msg:          "令牌刷新成功",
		Name:         resp.Name,
		ID:           resp.ID,
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
	})
}

func isRequestHTTPS(c *gin.Context) bool {
	// 1. 检查 X-Forwarded-Proto (Nginx 常用)
	if c.GetHeader("X-Forwarded-Proto") == "https" {
		return true
	}

	// 2. 检查 X-Forwarded-Ssl (某些云厂商负载均衡常用)
	if c.GetHeader("X-Forwarded-Ssl") == "on" {
		return true
	}

	return false
}

// ResetPassword 重置密码
func (h *UserHandler) ResetPassword(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.ObserveOperationDuration("email", "reset_password", time.Since(start).Seconds())
	}()
	// 实现重置密码的逻辑
	var request dto.ResetPasswordRequest

	if err := c.BindJSON(&request); err != nil {
		metrics.IncOperation("email", "reset_password", "failure: parameter parsing error")
		zlog.Warnw("重置密码失败", "detail", "解析请求数据失败")
		c.JSON(http.StatusBadRequest, dto.ResetPasswordResponse{Msg: "请求数据格式错误", Result: false})

		return
	}

	if request.NewPassword == "" {
		metrics.IncOperation("email", "reset_password", "failure: empty password")
		zlog.Warnw("重置密码失败", "detail", "新密码为空")
		c.JSON(http.StatusBadRequest, dto.ResetPasswordResponse{Msg: "新密码不能为空", Result: false})

		return
	}

	err := h.userService.ResetPassword(c.Request.Context(), request.Token, request.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, myErrors.ErrInvalidToken):
			metrics.IncOperation("email", "reset_password", "failure: invalid token")
			zlog.Warnw("重置密码失败", "detail", "验证码错误或已过期")
			c.JSON(http.StatusUnauthorized, dto.ResetPasswordResponse{Msg: "验证码错误或已过期", Result: false})

			return
		case errors.Is(err, myErrors.ErrPasswordHashFailed):
			metrics.IncOperation("email", "reset_password", "failure: password encryption error")
			zlog.Errorw("重置密码失败", "detail", "密码加密失败")
			c.JSON(http.StatusInternalServerError, dto.ResetPasswordResponse{Msg: "密码加密失败", Result: false})

			return
		case errors.Is(err, myErrors.ErrUserNotFound):
			metrics.IncOperation("email", "reset_password", "failure: user not found")
			zlog.Warnw("重置密码失败", "detail", "用户不存在")
			c.JSON(http.StatusUnauthorized, dto.ResetPasswordResponse{Msg: "用户不存在", Result: false})

			return
		default:
			metrics.IncOperation("email", "reset_password", "failure: reset password error")
			zlog.Warnw("重置密码失败", "detail", err.Error())
			c.JSON(http.StatusInternalServerError, dto.ResetPasswordResponse{Msg: "重置密码失败", Result: false})

			return
		}
	}

	metrics.IncOperation("email", "reset_password", "success")
	zlog.Infow("重置密码成功")
	c.JSON(http.StatusOK, dto.ResetPasswordResponse{
		Msg:    "重置密码成功",
		Result: true,
	})
}

func (h *UserHandler) CheckMe(c *gin.Context) {
	// 不做任何处理，只是借助 JWT 判断 token 是否还有效
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
