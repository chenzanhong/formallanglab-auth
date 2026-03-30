// auth/internal/api/email/email.go
package api

import (
	"net/http"
	"time"

	"github.com/chenzanhong/formallanglab-auth/internal/domain/dto"
	"github.com/chenzanhong/formallanglab-auth/internal/metrics"
	email "github.com/chenzanhong/formallanglab-auth/internal/service/email_s"
	"github.com/chenzanhong/zlog"
	"github.com/gin-gonic/gin"
)

type EmailHandler struct {
	emailService email.EmailService
}

func NewEmailHandler(emailService email.EmailService) *EmailHandler {
	return &EmailHandler{emailService: emailService}
}

// 注册，发送验证码
func (h *EmailHandler) SendRegisterVerificationCode(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.ObserveOperationDuration("email", "send_register_code", time.Since(start).Seconds())
	}()

	var request dto.SendRegisterVerificationCodeRequest

	if err := c.BindJSON(&request); err != nil {
		metrics.IncOperation("email", "send_register_code", "failure: parameter parsing error")
		zlog.Warnw("发送注册验证码失败", "detail", "解析请求失败，请检查请求格式是否正确")
		c.JSON(http.StatusBadRequest, dto.VerificationCodeResponse{Msg: "请求数据格式错误", Result: false})

		return
	}

	// 检查邮箱格式
	if request.Email == "" {
		metrics.IncOperation("email", "send_register_code", "failure: empty email")
		zlog.Warnw("发送注册验证码失败", "detail", "邮箱为空")
		c.JSON(http.StatusBadRequest, dto.VerificationCodeResponse{Msg: "请输入邮箱地址", Result: false})

		return
	}

	err := h.emailService.SendRegisterVerificationCode(c.Request.Context(), request.Email)
	if err != nil {
		metrics.IncOperation("email", "send_register_code", "failure: send code error")
		zlog.Errorw("发送注册验证码失败", "detail", err.Error(), "email", request.Email)
		c.JSON(http.StatusInternalServerError, dto.VerificationCodeResponse{Msg: "验证码发送失败", Result: false})

		return
	}

	// 这里暂时返回成功消息
	metrics.IncOperation("email", "send_register_code", "success")
	zlog.Infow("发送注册验证码成功", "email", request.Email)
	c.JSON(http.StatusOK, dto.VerificationCodeResponse{
		Msg:    "验证码已发送，请检查邮箱",
		Result: true,
	})
}

// 发送重置密码的验证码
func (h *EmailHandler) SendResetPwdVerificationCode(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.ObserveOperationDuration("email", "send_reset_password_code", time.Since(start).Seconds())
	}()

	// 实现请求重置密码的逻辑
	var request dto.SendResetPasswordVerificationCodeRequest

	if err := c.BindJSON(&request); err != nil {
		metrics.IncOperation("email", "send_reset_password_code", "failure: parameter parsing error")
		zlog.Warnw("发送重置密码的验证码失败", "detail", "解析请求失败，请检查请求格式是否正确", "username")
		c.JSON(http.StatusBadRequest, dto.VerificationCodeResponse{Msg: "解析请求失败，请检查请求格式是否正确", Result: false})

		return
	}

	err := h.emailService.SendResetPwdVerificationCode(c.Request.Context(), request.Email)
	if err != nil {
		metrics.IncOperation("email", "send_reset_password_code", "failure: send code error")
		zlog.Errorw("发送重置密码的验证码失败", "detail", err.Error(), "email", request.Email)
		c.JSON(http.StatusInternalServerError, dto.VerificationCodeResponse{Msg: "发送重置密码的验证码失败", Result: false})

		return
	}

	metrics.IncOperation("email", "send_reset_password_code", "success")
	zlog.Infow("发送重置密码的验证码成功", "email", request.Email)
	c.JSON(http.StatusOK, dto.VerificationCodeResponse{
		Msg:    "重置密码请求已发送，请检查邮箱",
		Result: true,
	})
}
