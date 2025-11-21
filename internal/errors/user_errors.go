package errors

import "errors"

var (
	ErrUserNotFound       = errors.New("用户不存在")
	ErrUserAlreadyExists  = errors.New("用户名已存在")
	ErrEmailAlreadyExists = errors.New("邮箱已被注册")
	ErrInvalidToken       = errors.New("验证码错误或已过期")
	ErrPasswordHashFailed = errors.New("密码加密失败")
	ErrUserCreationFailed = errors.New("用户创建失败")
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrInternal           = errors.New("内部服务错误")
	ErrTokenGenerationFailed = errors.New("token生成失败")
	ErrTokenSaveFailed    = errors.New("token保存失败")
	ErrTokenRevokeFailed  = errors.New("token撤销失败")
)
