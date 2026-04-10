package user_s

import (
	"context"

	"github.com/chenzanhong/formallanglab-auth/internal/domain/dto"
	"github.com/chenzanhong/formallanglab-auth/internal/errors"
	"github.com/chenzanhong/formallanglab-auth/internal/middleware/jwt"
)

// Refresh 使用刷新令牌获取新的访问令牌
func (s *UserServiceImpl) Refresh(ctx context.Context, refreshToken string) (resp *dto.LoginResponse, err error) {
	// 根据刷新令牌获取用户ID
	username, userID, err := s.GetUserNameAndIDByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, errors.ErrInvalidToken
	}

	// 验证刷新令牌是否有效
	valid, err := s.userRepo.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, errors.ErrInternal
	}
	if !valid {
		return nil, errors.ErrInvalidToken
	}

	// 生成新的访问令牌
	accessToken, err := jwt.GenerateAccessToken(username, userID, s.JwtCfg.AccessTokenExpireTime)
	if err != nil {
		return nil, errors.ErrTokenGenerationFailed
	}

	// 生成新的刷新令牌
	newRefreshToken, err := jwt.GenerateRandomRefreshToken()
	if err != nil {
		return nil, errors.ErrTokenGenerationFailed
	}

	// 删除旧的刷新令牌
	if err := s.userRepo.RevokeRefreshToken(ctx, refreshToken); err != nil {
		return nil, errors.ErrTokenRevokeFailed
	}

	// 保存新的刷新令牌，使用配置文件中的过期时间
	if err := s.userRepo.SaveRefreshToken(ctx, newRefreshToken, username, userID, s.JwtCfg.RefreshTokenExpireTime); err != nil {
		return nil, errors.ErrTokenSaveFailed
	}

	return &dto.LoginResponse{
		Result:       true,
		Msg:          "令牌刷新成功",
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		Name:         username,
		ID:           userID,
	}, nil
}
