package kafka

import (
	"auth/internal/domain/model"
	"context"
)

type KafkaEmailConsumerService interface {
	ReadEmail(ctx context.Context) (*model.KafkaEmailEvent, error)
}
