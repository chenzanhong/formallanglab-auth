package user_s

import (
	"auth/internal/domain/model"
	"auth/internal/errors"
	"auth/pkg/cryptoutil"
	"context"
)

// internal/service/user_s/register.go
func (s *UserServiceImpl) Register(ctx context.Context, name, email, password, token string) (*model.User, error) {
	// 检查用户名
	if exists, _ := s.userRepo.ExistsByName(ctx, name); exists {
		return nil, errors.ErrUserAlreadyExists // ← 返回标准错误
	}

	// 检查邮箱
	if exists, _ := s.userRepo.ExistsByEmail(ctx, email); exists {
		return nil, errors.ErrEmailAlreadyExists
	}

	// 验证注册验证码
	if valid, _ := s.emailRepo.ValidateRegisterVerificationToken(ctx, email, token); !valid {
		return nil, errors.ErrInvalidToken
	}

	// 密码

	hashed, err := cryptoutil.HashPassword(password)
	if err != nil {
		return nil, errors.ErrPasswordHashFailed
	}

	user := &model.User{Name: name, Email: email, Password: hashed}
	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, errors.ErrUserCreationFailed
	}

	s.emailRepo.DeleteRegisterVerificationToken(ctx, email)
	return user, nil
}
