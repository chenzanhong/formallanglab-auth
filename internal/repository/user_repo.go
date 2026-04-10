package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/chenzanhong/formallanglab-auth/internal/domain/model"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *model.User) error

	ExistsByID(ctx context.Context, id int64) (bool, error)
	ExistsByName(ctx context.Context, name string) (bool, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)

	GetAllUsers(ctx context.Context) ([]*model.User, error)
	GetUserByID(ctx context.Context, id int64) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	GetUserByName(ctx context.Context, name string) (*model.User, error)

	UpdateUser(ctx context.Context, user *model.User) error
	UpdatePasswordByUsername(ctx context.Context, username, password string) error
	UpdatePasswordByEmail(ctx context.Context, email, password string) error

	DeleteUser(ctx context.Context, user *model.User) error
	DeleteUserByID(ctx context.Context, id int64) error
	DeleteUserByEmail(ctx context.Context, email string) error
	DeleteUserByName(ctx context.Context, name string) error

	// Refresh token 相关方法
	SaveRefreshToken(ctx context.Context, refreshToken string, username string, userID int64, expireSeconds int) error
	ValidateRefreshToken(ctx context.Context, refreshToken string) (bool, error)
	RevokeRefreshToken(ctx context.Context, refreshToken string) error
	GetUserNameAndIDByRefreshToken(ctx context.Context, refreshToken string) (string, int64, error)
}

type UserRepositoryImpl struct {
	DB    *gorm.DB
	Redis *redis.Client
}

func NewUserRepository(db *gorm.DB, redis *redis.Client) UserRepository {
	return &UserRepositoryImpl{DB: db, Redis: redis}
}

func (r *UserRepositoryImpl) CreateUser(ctx context.Context, user *model.User) error {
	return r.DB.WithContext(ctx).Create(user).Error
}

func (r *UserRepositoryImpl) ExistsByID(ctx context.Context, id int64) (bool, error) {
	var count int64
	if err := r.DB.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *UserRepositoryImpl) ExistsByName(ctx context.Context, name string) (bool, error) {
	var count int64
	if err := r.DB.WithContext(ctx).Model(&model.User{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *UserRepositoryImpl) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	if err := r.DB.WithContext(ctx).Model(&model.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *UserRepositoryImpl) GetAllUsers(ctx context.Context) ([]*model.User, error) {
	var users []*model.User
	if err := r.DB.WithContext(ctx).Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepositoryImpl) GetUserByID(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	if err := r.DB.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepositoryImpl) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := r.DB.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepositoryImpl) GetUserByName(ctx context.Context, name string) (*model.User, error) {
	var user model.User
	if err := r.DB.WithContext(ctx).Where("name = ?", name).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepositoryImpl) UpdateUser(ctx context.Context, user *model.User) error {
	return r.DB.WithContext(ctx).Save(user).Error
}

func (r *UserRepositoryImpl) UpdatePasswordByUsername(ctx context.Context, username, password string) error {
	return r.DB.WithContext(ctx).Model(&model.User{}).Where("name = ?", username).Update("password", password).Error
}

func (r *UserRepositoryImpl) UpdatePasswordByEmail(ctx context.Context, email, password string) error {
	return r.DB.WithContext(ctx).Model(&model.User{}).Where("email = ?", email).Update("password", password).Error
}

func (r *UserRepositoryImpl) DeleteUser(ctx context.Context, user *model.User) error {
	return r.DB.WithContext(ctx).Delete(user).Error
}

func (r *UserRepositoryImpl) DeleteUserByID(ctx context.Context, id int64) error {
	return r.DB.WithContext(ctx).Delete(&model.User{}, id).Error
}

func (r *UserRepositoryImpl) DeleteUserByEmail(ctx context.Context, email string) error {
	return r.DB.WithContext(ctx).Delete(&model.User{}, "email = ?", email).Error
}

func (r *UserRepositoryImpl) DeleteUserByName(ctx context.Context, name string) error {
	return r.DB.WithContext(ctx).Delete(&model.User{}, "name = ?", name).Error
}

// buildRefreshTokenKey 构建 refresh token 的 Redis key
func buildRefreshTokenKey(refreshToken string) string {
	return fmt.Sprintf("refresh_token:%s", refreshToken)
}

// SaveRefreshToken 保存 refresh token 到 Redis
// 新机制：使用 refresh token 作为 key，用户 ID 作为 value
func (r *UserRepositoryImpl) SaveRefreshToken(ctx context.Context, refreshToken string, username string, userID int64, expireSeconds int) error {
	key := buildRefreshTokenKey(refreshToken)
	// 保存 refresh token，设置指定的过期时间（单位：秒）
	value := fmt.Sprintf("%s:%d", username, userID)

	return r.Redis.Set(ctx, key, value, time.Duration(expireSeconds)*time.Second).Err()
}

// ValidateRefreshToken 验证 refresh token 是否有效
func (r *UserRepositoryImpl) ValidateRefreshToken(ctx context.Context, refreshToken string) (bool, error) {
	key := buildRefreshTokenKey(refreshToken)
	_, err := r.Redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// RevokeRefreshToken 删除 refresh token
func (r *UserRepositoryImpl) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	key := buildRefreshTokenKey(refreshToken)
	return r.Redis.Del(ctx, key).Err()
}

// GetUserIDByRefreshToken 根据 refresh token 获取用户 ID
func (r *UserRepositoryImpl) GetUserNameAndIDByRefreshToken(ctx context.Context, refreshToken string) (string, int64, error) {
	key := buildRefreshTokenKey(refreshToken)
	userStr, err := r.Redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", 0, fmt.Errorf("refresh token not found")
	}
	if err != nil {
		return "", 0, err
	}
	var username string
	var userID int64
	_, err = fmt.Sscanf(userStr, "%s:%d", &username, &userID)
	if err != nil {
		return "", 0, fmt.Errorf("invalid user ID format")
	}

	return username, userID, nil
}
