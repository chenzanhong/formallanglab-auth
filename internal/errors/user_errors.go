package errors

import "fmt"

var (
	ErrUserNotFound          = fmt.Errorf("用户不存在")
	ErrUserAlreadyExists     = fmt.Errorf("用户名已存在")
	ErrEmailAlreadyExists    = fmt.Errorf("邮箱已被注册")
	ErrInvalidToken          = fmt.Errorf("验证码错误或已过期")
	ErrPasswordHashFailed    = fmt.Errorf("密码加密失败")
	ErrUserCreationFailed    = fmt.Errorf("用户创建失败")
	ErrInvalidCredentials    = fmt.Errorf("用户名或密码错误")
	ErrInternal              = fmt.Errorf("内部服务错误")
	ErrTokenGenerationFailed = fmt.Errorf("token生成失败")
	ErrTokenSaveFailed       = fmt.Errorf("token保存失败")
	ErrTokenRevokeFailed     = fmt.Errorf("token撤销失败")
)
