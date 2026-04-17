// internal/service/user_s/user_service.go
package user_s

import (
	"context"

	"github.com/chenzanhong/zlog"

	"github.com/chenzanhong/formallanglab-auth/configs"
	"github.com/chenzanhong/formallanglab-auth/internal/domain/dto"
	"github.com/chenzanhong/formallanglab-auth/internal/domain/model"
	"github.com/chenzanhong/formallanglab-auth/internal/repository"
)

type UserService interface {
	Register(ctx context.Context, name, email, password, token string) (*model.User, error)
	Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*dto.LoginResponse, error)
	RevokeRefreshToken(ctx context.Context, refreshToken string) error
	GetUserNameAndIDByRefreshToken(ctx context.Context, refreshToken string) (string, int64, error)
	ResetPassword(ctx context.Context, token, newPassword string) error
	GetUser(ctx context.Context, userID int64) (*model.User, error)
}

type UserServiceImpl struct {
	userRepo  repository.UserRepository
	emailRepo repository.EmailRepository
	JwtCfg    configs.JWTConfig
}

func NewUserService(userRepo repository.UserRepository, emailRepo repository.EmailRepository, jwtCfg configs.JWTConfig) UserService {
	zlog.Infow("JwtCfg", "AccessTokenExpireTime", jwtCfg.AccessTokenExpireTime, "RefreshTokenExpireTime", jwtCfg.RefreshTokenExpireTime)
	return &UserServiceImpl{userRepo: userRepo, emailRepo: emailRepo, JwtCfg: jwtCfg}
}
