package consumer

import (
	"context"
	"encoding/json"
	"time"

	kafka "github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/asmisnik/flats-analyzer/internal/db"
	"github.com/asmisnik/flats-analyzer/internal/formatter"
	"github.com/asmisnik/flats-analyzer/internal/metrics"
	"github.com/asmisnik/flats-analyzer/internal/model"
	"github.com/asmisnik/flats-analyzer/internal/notifier"
)

type Consumer struct {
	reader   *kafka.Reader
	db       *db.DB
	notifier *notifier.Client
	logger   *zap.Logger
}

func New(brokers []string, topic, groupID string, database *db.DB, notifierClient *notifier.Client, logger *zap.Logger) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			MinBytes:       1,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
		}),
		db:       database,
		notifier: notifierClient,
		logger:   logger,
	}
}

func (c *Consumer) Run(ctx context.Context) {
	c.logger.Info("kafka consumer started")
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.Error("kafka fetch error", zap.Error(err))
			continue
		}

		c.processMessage(ctx, msg)

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Warn("kafka commit error", zap.Error(err))
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

func (c *Consumer) processMessage(ctx context.Context, msg kafka.Message) {
	start := time.Now()

	var flat model.FlatInfo
	if err := json.Unmarshal(msg.Value, &flat); err != nil {
		c.logger.Warn("failed to unmarshal flat", zap.Error(err))
		return
	}
	metrics.FlatsConsumed.Inc()
	c.logger.Debug("flat consumed", zap.String("link", flat.Link))

	flatID, err := c.db.GetFlatIDByLink(ctx, flat.Link)
	if err != nil {
		c.logger.Warn("flat not found in db", zap.String("link", flat.Link), zap.Error(err))
		return
	}

	subs, err := c.db.GetActiveSubscriptions(ctx)
	if err != nil {
		c.logger.Error("failed to get subscriptions", zap.Error(err))
		return
	}

	for _, sub := range subs {
		if !matchesSubscription(&flat, &sub) {
			c.logger.Debug("flat does not match subscription",
				zap.String("link", flat.Link), zap.Int("sub_id", sub.ID))
			continue
		}
		metrics.SubscriptionsMatched.Inc()

		sent, err := c.db.IsAlreadySent(ctx, sub.ID, flatID)
		if err != nil {
			c.logger.Warn("sent check failed", zap.Int("sub_id", sub.ID), zap.Error(err))
			continue
		}
		if sent {
			c.logger.Debug("flat already sent", zap.String("link", flat.Link), zap.Int("sub_id", sub.ID))
			continue
		}

		showRegion, err := c.db.HasMultipleActiveRegions(ctx, sub.ChatID)
		if err != nil {
			c.logger.Warn("checking multiple regions failed", zap.Int64("chat_id", sub.ChatID), zap.Error(err))
			showRegion = false
		}

		text := formatter.FormatFlat(&flat, showRegion)
		if err := c.notifier.Send(ctx, sub.ChatID, text); err != nil {
			metrics.MessagesFailed.Inc()
			c.logger.Warn("send failed",
				zap.Int64("chat_id", sub.ChatID),
				zap.String("link", flat.Link),
				zap.Error(err))
			continue
		}
		metrics.MessagesSent.Inc()

		if err := c.db.InsertSentMessage(ctx, sub.ID, flatID); err != nil {
			c.logger.Warn("insert sent message failed", zap.Int("sub_id", sub.ID), zap.Error(err))
		}

		c.logger.Info("notification sent",
			zap.Int64("chat_id", sub.ChatID),
			zap.String("link", flat.Link),
		)
	}

	metrics.ProcessDuration.Observe(time.Since(start).Seconds())
}

func matchesSubscription(f *model.FlatInfo, s *db.Subscription) bool {
	if s.Region > 0 && f.Region != s.Region {
		return false
	}
	if s.MinPrice > 0 && f.Price < s.MinPrice {
		return false
	}
	if s.MaxPrice > 0 && f.Price > s.MaxPrice {
		return false
	}
	if s.MinArea > 0 && f.TotalArea < s.MinArea {
		return false
	}
	if s.MaxArea > 0 && f.TotalArea > s.MaxArea {
		return false
	}
	if s.MinScore > 0 && f.FlatScore < s.MinScore {
		return false
	}
	if len(s.Rooms) > 0 {
		matched := false
		for _, r := range s.Rooms {
			if int(r) == f.RoomNumber {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}
