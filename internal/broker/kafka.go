package broker

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

// KafkaWriter wraps a segmentio kafka.Writer and topic information.
type KafkaWriter struct {
	writer *kafka.Writer
	topic  string
}

// NewKafkaWriter initializes a KafkaWriter for the given brokers and topic.
func NewKafkaWriter(brokers []string, topic string) *KafkaWriter {
	w := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	return &KafkaWriter{writer: w, topic: topic}
}

// PublishTransactionEvent marshals eventPayload to JSON and publishes it to Kafka.
func (k *KafkaWriter) PublishTransactionEvent(ctx context.Context, eventPayload interface{}) error {
	if k == nil || k.writer == nil {
		return nil
	}

	b, err := json.Marshal(eventPayload)
	if err != nil {
		slog.Error("failed to marshal event payload", "error", err)
		return err
	}

	msg := kafka.Message{
		Key:   nil,
		Value: b,
	}

	if err := k.writer.WriteMessages(ctx, msg); err != nil {
		slog.Error("failed to write message to kafka", "error", err)
		return err
	}

	slog.Info("published event to kafka", "topic", k.topic)
	return nil
}

// Close closes the underlying writer.
func (k *KafkaWriter) Close() error {
	if k == nil || k.writer == nil {
		return nil
	}
	return k.writer.Close()
}
