package dto

// 发送注册验证码请求
type SendRegisterVerificationCodeRequest struct {
	Email string `json:"email" binding:"required"`
}

type SendRegisterVerificationCodeResponse = BaseResponse

// 发送重置密码验证码请求
type SendResetPasswordVerificationCodeRequest struct {
	Email string `json:"email" binding:"required"`
}

type SendResetPasswordVerificationCodeResponse = BaseResponse


// 验证码响应
type VerificationCodeResponse = BaseResponse
