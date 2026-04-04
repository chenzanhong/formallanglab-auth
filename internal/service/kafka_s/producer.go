// auth/internal/service/kafka_s/producer.go
package kafka

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/segmentio/kafka-go"

	"github.com/chenzanhong/formallanglab-auth/internal/domain/model"
)

type KafkaProducerService interface {
	SendEmailEvent(ctx context.Context, event *model.KafkaEmailEvent) error
}

type KafkaServiceImpl struct {
	producer *kafka.Writer
}

func NewKafkaProducerService(producer *kafka.Writer) *KafkaServiceImpl {
	return &KafkaServiceImpl{
		producer: producer,
	}
}

func NewDefaultKafkaProducerService() *KafkaServiceImpl {
	topic := os.Getenv("KAFKA_TOPIC")
	brokers := strings.Split(strings.TrimSpace(os.Getenv("KAFKA_BROKERS")), ",")

	return &KafkaServiceImpl{
		producer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireAll, // 根据需要调整
		},
	}
}

func (k *KafkaServiceImpl) Close() {
	if k.producer != nil {
		k.producer.Close()
	}
}

func (k *KafkaServiceImpl) SendEmailEvent(ctx context.Context, event *model.KafkaEmailEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return k.producer.WriteMessages(ctx, kafka.Message{
		Value: data,
	})
}
