package kafka

import (
	"context"

	"github.com/chenzanhong/formallanglab-auth/internal/domain/model"
)

type KafkaEmailConsumerService interface {
	ReadEmail(ctx context.Context) (*model.KafkaEmailEvent, error)
}
