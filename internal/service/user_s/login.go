package user_s

import (
	"auth/internal/domain/dto"
	"auth/internal/errors"
	"auth/internal/middleware"
	"auth/pkg/cryptoutil"
	"context"
)

func (s *UserServiceImpl) Login(ctx context.Context, req *dto.LoginRequest) (resp *dto.LoginResponse, err error) {
	name := req.Name
	password := req.Password

	// 检查用户名是否存在
	exists, err := s.userRepo.ExistsByName(ctx, name)
	if err != nil {
		return nil, errors.ErrInternal
	}
	if !exists {
		return nil, errors.ErrUserNotFound
	}
	// 检查密码是否正确
	user, err := s.userRepo.GetUserByName(ctx, name)
	if err != nil {
		return nil, errors.ErrInternal
	}

	if !cryptoutil.CheckPasswordHash(password, user.Password) {
		return nil, errors.ErrInvalidCredentials
	}

	// 生成访问令牌 (JWT)
	accessToken, err := middleware.GenerateAccessToken(user.Name, int(user.ID))
	if err != nil {
		return nil, errors.ErrTokenGenerationFailed
	}

	// 生成刷新令牌 (随机字符串)
	refreshToken, err := middleware.GenerateRandomRefreshToken()
	if err != nil {
		return nil, errors.ErrTokenGenerationFailed
	}

	// 保存刷新令牌到Redis
	if err := s.userRepo.SaveRefreshToken(ctx, int64(user.ID), refreshToken); err != nil {
		return nil, errors.ErrTokenSaveFailed
	}

	return &dto.LoginResponse{
		Result:       true,
		Msg:          "登录成功",
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Name:         user.Name,
		ID:           user.ID,
	}, nil
}

// RevokeRefreshToken 撤销用户的刷新令牌
// 注意：由于我们现在的刷新令牌是随机字符串，我们无法直接从中提取用户ID
// 因此我们需要客户端在登出时同时提供用户ID和刷新令牌
func (s *UserServiceImpl) RevokeRefreshToken(ctx context.Context, userID int64, refreshToken string) error {
	// 直接从Redis中删除刷新令牌
	// 在新的实现中，ValidateRefreshToken已经移到了repository层
	// 这里我们只需要调用repository的RevokeRefreshToken方法即可
	if err := s.userRepo.RevokeRefreshToken(ctx, userID, refreshToken); err != nil {
		return errors.ErrTokenRevokeFailed
	}

	return nil
}

// GetUserIDByRefreshToken 根据刷新令牌获取用户ID
func (s *UserServiceImpl) GetUserIDByRefreshToken(ctx context.Context, refreshToken string) (int64, error) {
	return s.userRepo.GetUserIDByRefreshToken(ctx, refreshToken)
}
