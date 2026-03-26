package user_s

import (
	"auth/internal/domain/dto"
	"auth/internal/errors"
	"auth/internal/middleware"
	"context"
)

// Refresh 使用刷新令牌获取新的访问令牌
func (s *UserServiceImpl) Refresh(ctx context.Context, refreshToken string) (resp *dto.LoginResponse, err error) {
	// 根据刷新令牌获取用户ID
	username, userID, err := s.GetUserNameAndIDByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, errors.ErrInvalidToken
	}

	// 验证刷新令牌是否有效
	valid, err := s.userRepo.ValidateRefreshToken(ctx,  refreshToken)
	if err != nil {
		return nil, errors.ErrInternal
	}
	if !valid {
		return nil, errors.ErrInvalidToken
	}

	// 获取用户信息
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, errors.ErrInternal
	}

	// 生成新的访问令牌
	accessToken, err := middleware.GenerateAccessToken(user.Name, user.ID, s.JwtCfg.AccessTokenExpireTime)
	if err != nil {
		return nil, errors.ErrTokenGenerationFailed
	}

	// 生成新的刷新令牌
	newRefreshToken, err := middleware.GenerateRandomRefreshToken()
	if err != nil {
		return nil, errors.ErrTokenGenerationFailed
	}

	// 删除旧的刷新令牌
	if err := s.userRepo.RevokeRefreshToken(ctx, refreshToken); err != nil {
		return nil, errors.ErrTokenRevokeFailed
	}

	// 保存新的刷新令牌
	if err := s.userRepo.SaveRefreshToken(ctx, newRefreshToken, username, userID); err != nil {
		return nil, errors.ErrTokenSaveFailed
	}

	return &dto.LoginResponse{
		Result:       true,
		Msg:          "令牌刷新成功",
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		Name:         user.Name,
		ID:           user.ID,
	}, nil
}
