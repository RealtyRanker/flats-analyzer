package consumer

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	kafka "github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/asmisnik/flats-analyzer/internal/db"
	"github.com/asmisnik/flats-analyzer/internal/formatter"
	"github.com/asmisnik/flats-analyzer/internal/metrics"
	"github.com/asmisnik/flats-analyzer/internal/model"
	"github.com/asmisnik/flats-analyzer/internal/notifier"
	"github.com/asmisnik/flats-analyzer/internal/scoring"
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
		score := flat.FlatScore
		if sub.MinScore > 0 {
			params, err := c.db.GetScoringParams(ctx, sub.ID)
			if err != nil {
				c.logger.Warn("fetching scoring params failed", zap.Int("sub_id", sub.ID), zap.Error(err))
			} else if params != nil {
				score = scoring.Score(&flat, toCustomParams(params))
			}
		}

		if !matchesSubscription(&flat, &sub, score) {
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

		showDealType, err := c.db.HasMultipleActiveDealTypes(ctx, sub.ChatID)
		if err != nil {
			c.logger.Warn("checking multiple deal types failed", zap.Int64("chat_id", sub.ChatID), zap.Error(err))
			showDealType = false
		}

		text := formatter.FormatFlat(&flat, showRegion, showDealType)
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

func matchesSubscription(f *model.FlatInfo, s *db.Subscription, score int) bool {
	if s.DealType != "" && f.DealType != s.DealType {
		return false
	}
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
	if s.MinScore > 0 && score < s.MinScore {
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

	if s.MinUndergroundPlace > 0 {
		place := undergroundPlaceForSubscription(f, s)
		if place == 0 || place > s.MinUndergroundPlace {
			return false
		}
	}
	if len(s.MetroStations) > 0 && !stationsIntersect(f.UndergroundStations, s.MetroStations) {
		return false
	}
	if s.MinKitchenArea > 0 && f.KitchenArea < s.MinKitchenArea {
		return false
	}
	if s.MinFloor > 0 && f.Floor < s.MinFloor {
		return false
	}
	if s.MaxFloor > 0 && f.Floor > s.MaxFloor {
		return false
	}
	if s.MinCeilingHeight > 0 && f.CeilingHeight < s.MinCeilingHeight {
		return false
	}
	if s.ChildrenRequired && !f.ChildrenAllowed {
		return false
	}
	if s.PetsRequired && !f.PetsAllowed {
		return false
	}
	if s.DishwasherRequired && !f.HasDishwasher {
		return false
	}
	if s.ConditionerRequired && !f.HasConditioner {
		return false
	}
	if s.MinRenovation != "" && renovationRank(f.Renovation) < renovationRank(s.MinRenovation) {
		return false
	}
	if s.BalconyRequired && f.BalconyCount == 0 && f.LoggiaCount == 0 {
		return false
	}
	switch s.BathroomType {
	case "separated":
		if f.SeparatedBathroomCount == 0 {
			return false
		}
	case "combined":
		if f.CombinedBathroomCount == 0 {
			return false
		}
	}

	return true
}

// toCustomParams converts the DB-shaped scoring params into scoring.CustomParams.
func toCustomParams(p *db.ScoringParams) *scoring.CustomParams {
	return &scoring.CustomParams{
		AllArea:            p.AllArea,
		KitchenArea:        p.KitchenArea,
		Pets:               p.Pets,
		Dishwasher:         p.Dishwasher,
		Conditioner:        p.Conditioner,
		Apartments:         p.Apartments,
		TwoRoom:            p.TwoRoom,
		ThreeRoom:          p.ThreeRoom,
		FourRoom:           p.FourRoom,
		AdditionalRooms:    p.AdditionalRooms,
		WindowsYard:        p.WindowsYard,
		WindowsStreet:      p.WindowsStreet,
		WindowsBoth:        p.WindowsBoth,
		RenovationDesign:   p.RenovationDesign,
		RenovationEuro:     p.RenovationEuro,
		RenovationCosmetic: p.RenovationCosmetic,
		BathroomSeparated:  p.BathroomSeparated,
		Balcony:            p.Balcony,
		Loggia:             p.Loggia,
		Underground:        p.Underground,
	}
}

// stationsIntersect reports whether any of a flat's underground stations
// matches (case-insensitively) any station in the subscription's filter set.
func stationsIntersect(flatStations, filterStations []string) bool {
	for _, fs := range flatStations {
		for _, ss := range filterStations {
			if strings.EqualFold(fs, ss) {
				return true
			}
		}
	}
	return false
}

// undergroundPlaceForSubscription returns the underground place to check
// against s.MinUndergroundPlace: the flat's default place (from the static,
// unweighted metro ranking) unless the subscriber named priority stations,
// in which case it's the flat's best place among its underground stations in
// that subscriber's priority-boosted station ranking (precomputed once by
// subscription-handler at subscription-creation time and stored in
// s.PriorityStationNames, best-first) — so two subscribers with different
// priorities can see a different place for the same flat, with no per-flat
// recomputation of the ranking algorithm here. Returns 0 (undefined) if no
// ranked station matches any of the flat's underground stations.
func undergroundPlaceForSubscription(f *model.FlatInfo, s *db.Subscription) int {
	if len(s.PriorityStationNames) == 0 {
		return f.UndergroundPlace
	}

	best := 0
	for _, station := range f.UndergroundStations {
		for i, ranked := range s.PriorityStationNames {
			if !strings.EqualFold(normalizeStationName(station), normalizeStationName(ranked)) {
				continue
			}
			place := i + 1
			if best == 0 || place < best {
				best = place
			}
			break
		}
	}
	return best
}

// normalizeStationName folds ё/Ё to е/Е, matching subscription-handler's
// metro.NormalizeStationName so a flat's station names compare correctly
// against a subscriber's priority-boosted ranking regardless of that
// difference; case is additionally ignored via strings.EqualFold above.
func normalizeStationName(s string) string {
	s = strings.ReplaceAll(s, "ё", "е")
	return strings.ReplaceAll(s, "Ё", "Е")
}

// renovationRank ranks renovation levels design > euro > cosmetic > (any
// other value, including no renovation info), matching subscription-handler's
// session.Renovation* ordering.
func renovationRank(level string) int {
	switch level {
	case "design":
		return 3
	case "euro":
		return 2
	case "cosmetic":
		return 1
	default:
		return 0
	}
}
