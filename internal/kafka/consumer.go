package kafka

import (
	"context"
	"encoding/json"
	"log/slog"

	kafkago "github.com/segmentio/kafka-go"

	"go-rest-homework/internal/config"
)

type Consumer struct {
	reader *kafkago.Reader
	logger *slog.Logger
}

func NewConsumer(cfg config.Config, logger *slog.Logger) *Consumer {
	return &Consumer{
		logger: logger,
		reader: kafkago.NewReader(kafkago.ReaderConfig{
			Brokers:     []string{cfg.KafkaBootstrap},
			Topic:       cfg.KafkaTopic,
			GroupID:     cfg.KafkaConsumerGroup,
			StartOffset: kafkago.FirstOffset,
		}),
	}
}

func (c *Consumer) Run(ctx context.Context) {
	c.logger.Info("kafka_consumer_started")
	processed := map[string]struct{}{}

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.Error("consumer_fetch_error", "error", err)
			continue
		}

		var event UserRegisteredEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			c.logger.Error("invalid_json_event", "error", err, "raw", string(msg.Value))
			_ = c.reader.CommitMessages(ctx, msg)
			continue
		}

		if event.EventID != "" {
			if _, ok := processed[event.EventID]; ok {
				c.logger.Info("duplicate_event_skipped", "event_id", event.EventID, "partition", msg.Partition, "offset", msg.Offset)
				_ = c.reader.CommitMessages(ctx, msg)
				continue
			}
			processed[event.EventID] = struct{}{}
		}

		c.logger.Info("consumed_event", "value", event, "partition", msg.Partition, "offset", msg.Offset)
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Error("offset_commit_failed", "error", err, "partition", msg.Partition, "offset", msg.Offset)
			continue
		}
		c.logger.Info("offset_committed", "partition", msg.Partition, "offset", msg.Offset)
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
