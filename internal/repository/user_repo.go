package repository

import (
	"auth/internal/domain/model"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
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
	SaveRefreshToken(ctx context.Context, userID int64, refreshToken string) error
	ValidateRefreshToken(ctx context.Context, userID int64, refreshToken string) (bool, error)
	RevokeRefreshToken(ctx context.Context, userID int64, refreshToken string) error
	GetUserIDByRefreshToken(ctx context.Context, refreshToken string) (int64, error)
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

// SaveRefreshToken 保存refresh token到Redis
// 新机制：使用 refresh token 作为 key，用户 ID 作为 value
func (r *UserRepositoryImpl) SaveRefreshToken(ctx context.Context, userID int64, refreshToken string) error {
	key := fmt.Sprintf("refresh_token:%s", refreshToken)
	// 保存refresh token，设置30天过期时间
	return r.Redis.Set(ctx, key, userID, 30*24*time.Hour).Err()
}

// ValidateRefreshToken 验证refresh token是否有效
func (r *UserRepositoryImpl) ValidateRefreshToken(ctx context.Context, userID int64, refreshToken string) (bool, error) {
	key := fmt.Sprintf("refresh_token:%s", refreshToken)
	storedUserID, err := r.Redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// 检查存储的用户 ID 是否与提供的用户 ID 相匹配
	var storedID int64
	fmt.Sscanf(storedUserID, "%d", &storedID)
	return storedID == userID, nil
}

// RevokeRefreshToken 删除refresh token
func (r *UserRepositoryImpl) RevokeRefreshToken(ctx context.Context, userID int64, refreshToken string) error {
	key := fmt.Sprintf("refresh_token:%s", refreshToken)
	// 直接删除refresh token，不验证是否匹配
	return r.Redis.Del(ctx, key).Err()
}

// GetUserIDByRefreshToken 根据refresh token获取用户ID
func (r *UserRepositoryImpl) GetUserIDByRefreshToken(ctx context.Context, refreshToken string) (int64, error) {
	key := fmt.Sprintf("refresh_token:%s", refreshToken)
	userIDStr, err := r.Redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return 0, fmt.Errorf("refresh token not found")
	}
	if err != nil {
		return 0, err
	}

	var userID int64
	_, err = fmt.Sscanf(userIDStr, "%d", &userID)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID format")
	}

	return userID, nil
}
