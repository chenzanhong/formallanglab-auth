package user_s

import (
	"context"

	"github.com/chenzanhong/formallanglab-auth/internal/domain/model"
)

func (s *UserServiceImpl) GetUser(ctx context.Context, userID int64) (*model.User, error) {
	return s.userRepo.GetUserByID(ctx, userID)
}
