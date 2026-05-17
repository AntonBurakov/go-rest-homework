package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"go-rest-homework/internal/config"
	"go-rest-homework/internal/storage"
)

type Publisher struct {
	writer *kafkago.Writer
	logger *slog.Logger
}

type UserRegisteredEvent struct {
	EventID   string `json:"event_id"`
	EventName string `json:"event_name"`
	UserID    int64  `json:"user_id"`
	Email     string `json:"email"`
	TraceID   string `json:"trace_id,omitempty"`
}

func NewPublisher(cfg config.Config, logger *slog.Logger) *Publisher {
	return &Publisher{
		logger: logger,
		writer: &kafkago.Writer{
			Addr:                   kafkago.TCP(cfg.KafkaBootstrap),
			Topic:                  cfg.KafkaTopic,
			RequiredAcks:           kafkago.RequireAll,
			AllowAutoTopicCreation: true,
			BatchTimeout:           10 * time.Millisecond,
			WriteTimeout:           10 * time.Second,
		},
	}
}

func (p *Publisher) PublishUserRegistered(ctx context.Context, user storage.User, traceID string) bool {
	event := UserRegisteredEvent{
		EventID:   newEventID(),
		EventName: "user_registered",
		UserID:    user.ID,
		Email:     user.Email,
		TraceID:   traceID,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		p.logger.Error("marshal_user_registered_failed", "error", err, "user_id", user.ID)
		return false
	}

	for attempt := 1; attempt <= 3; attempt++ {
		writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err = p.writer.WriteMessages(writeCtx, kafkago.Message{
			Key:   []byte(strconv.FormatInt(user.ID, 10)),
			Value: payload,
		})
		cancel()

		if err == nil {
			p.logger.Info("published_user_registered", "kafka_event", event)
			return true
		}

		p.logger.Error("publish_failed", "error", err, "attempt", attempt, "kafka_event", event)
		select {
		case <-ctx.Done():
			return false
		case <-time.After(time.Duration(attempt) * time.Second):
		}
	}

	p.logger.Warn("giving_up_publishing_user_registered", "kafka_event", event)
	return false
}

func (p *Publisher) Close() error {
	return p.writer.Close()
}
