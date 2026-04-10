package dto

// 用户注册请求
type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Token    string `json:"token" binding:"required"`
}

// 用户注册响应
type RegisterResponse struct {
	Result bool   `json:"result"`
	Msg    string `json:"msg"`
	ID     int64  `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Error  string `json:"error,omitempty"`
}

// 用户登录请求
type LoginRequest struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// 用户登录响应
type LoginResponse struct {
	Result       bool   `json:"result"`
	Msg          string `json:"msg"`
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
	Name         string `json:"name,omitempty"`
	ID           int64  `json:"id,omitempty"`
}

type RefreshResponse = LoginResponse

// 用户重置密码
type ResetPasswordRequest struct {
	Token       string `json:"token"` // 邮件收到的验证码，不是登录token
	NewPassword string `json:"newPwd"`
}

type ResetPasswordResponse struct {
	Result bool   `json:"result"`
	Msg    string `json:"msg"`
	Error  string `json:"error,omitempty"`
}

// 用户登出响应
type LogoutResponse struct {
	Result bool   `json:"result"`
	Msg    string `json:"msg"`
}
