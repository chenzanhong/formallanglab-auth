package user_s

import (
	"auth/internal/domain/model"
	"context"
)

func (s *UserServiceImpl) GetUser(ctx context.Context, userID int64) (*model.User, error) {
	return s.userRepo.GetUserByID(ctx, userID)
}
