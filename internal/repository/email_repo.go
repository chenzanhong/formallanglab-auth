package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type EmailRepository interface {
	// 注册相关
	SaveRegisterVerificationToken(ctx context.Context, email, token string) error             // 保存注册验证码
	HasRegisterVerificationToken(ctx context.Context, email string) (bool, error)             // 检查注册验证码是否存在
	ValidateRegisterVerificationToken(ctx context.Context, email, token string) (bool, error) // 验证注册验证码
	DeleteRegisterVerificationToken(ctx context.Context, email string) error                  // 删除注册验证码

	// 找回密码相关
	SaveResetPwdToken(ctx context.Context, token, email string) error          // 保存重置密码验证码
	HasResetPwdToken(ctx context.Context, token string) (bool, error)          // 检查重置密码验证码是否存在
	GetEmailByResetPwdToken(ctx context.Context, token string) (string, error) // 根据重置密码验证码获取邮箱
	DeleteResetPwdToken(ctx context.Context, token string) error               // 删除重置密码验证码

	// 用户查询（也可以移到 UserRepo，但这里为了简化先放这）
	// UserExistsFromRedisByEmail(ctx context.Context, email string) (bool, error)
	// GetUsernameFromRedisByEmail(ctx context.Context, email string) (string, error)
}

type EmailRepositoryImpl struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewEmailRepository(db *gorm.DB, redis *redis.Client) EmailRepository {
	return &EmailRepositoryImpl{db: db, redis: redis}
}

// --- Register Token ---

func (r *EmailRepositoryImpl) SaveRegisterVerificationToken(ctx context.Context, email, token string) error {
	key := fmt.Sprintf("register_verification_token:%s", email)
	return r.redis.Set(ctx, key, token, 1*time.Minute).Err()
}

func (r *EmailRepositoryImpl) HasRegisterVerificationToken(ctx context.Context, email string) (bool, error) {
	key := fmt.Sprintf("register_verification_token:%s", email)
	_, err := r.redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	return err == nil, err
}

func (r *EmailRepositoryImpl) ValidateRegisterVerificationToken(ctx context.Context, email, token string) (bool, error) {
	key := fmt.Sprintf("register_verification_token:%s", email)
	stored, err := r.redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return stored == token, nil
}

func (r *EmailRepositoryImpl) DeleteRegisterVerificationToken(ctx context.Context, email string) error {
	key := fmt.Sprintf("register_verification_token:%s", email)
	return r.redis.Del(ctx, key).Err()
}

// --- Reset Password Token ---

func (r *EmailRepositoryImpl) SaveResetPwdToken(ctx context.Context, token, email string) error {
	key := fmt.Sprintf("reset_pwd_token:%s", token)
	return r.redis.Set(ctx, key, email, 1*time.Minute).Err()
}

func (r *EmailRepositoryImpl) HasResetPwdToken(ctx context.Context, token string) (bool, error) {
	key := fmt.Sprintf("reset_pwd_token:%s", token)
	_, err := r.redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	return err == nil, err
}

func (r *EmailRepositoryImpl) GetEmailByResetPwdToken(ctx context.Context, token string) (string, error) {
	key := fmt.Sprintf("reset_pwd_token:%s", token)
	email, err := r.redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", errors.New("token 不存在或已过期")
	}
	return email, err
}

func (r *EmailRepositoryImpl) DeleteResetPwdToken(ctx context.Context, token string) error {
	key := fmt.Sprintf("reset_pwd_token:%s", token)
	return r.redis.Del(ctx, key).Err()
}
