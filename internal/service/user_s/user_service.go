// internal/service/user_s/user_service.go
package user_s

import (
	"auth/internal/domain/dto"
	"auth/internal/domain/model"
	"auth/internal/repository"
	"context"
)

type UserService interface {
	Register(ctx context.Context, name, email, password, token string) (*model.User, error)
	// Login 处理用户登录，返回访问令牌和刷新令牌
	Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error)
	// Refresh 使用刷新令牌获取新的访问令牌
	Refresh(ctx context.Context, refreshToken string) (*dto.LoginResponse, error)
	// RevokeRefreshToken 撤销用户的刷新令牌
	RevokeRefreshToken(ctx context.Context, refreshToken string) error
	// GetUserIDByRefreshToken 根据刷新令牌获取用户ID
	GetUserNameAndIDByRefreshToken(ctx context.Context, refreshToken string) (string, int64, error)
	ResetPassword(ctx context.Context, token, newPassword string) error
}

type UserServiceImpl struct {
	userRepo  repository.UserRepository
	emailRepo repository.EmailRepository
}

func NewUserService(userRepo repository.UserRepository, emailRepo repository.EmailRepository) UserService {
	return &UserServiceImpl{userRepo: userRepo, emailRepo: emailRepo}
}
