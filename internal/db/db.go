package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Subscription struct {
	ID            int
	ChatID        int64
	DealType      string
	Region        int
	MetroStations []string
	MinPrice      int
	MaxPrice      int
	MinArea       float64
	MaxArea       float64
	Rooms         []int32
	MinScore      int

	// Extended filters (zero-valued when not set, meaning "no filter").
	MinUndergroundPlace int
	MinKitchenArea      float64
	MinFloor            int
	MaxFloor            int
	MinCeilingHeight    float64
	ChildrenRequired    bool
	PetsRequired        bool
	DishwasherRequired  bool
	ConditionerRequired bool
	MinRenovation       string
	BalconyRequired     bool
	BathroomType        string
}

type DB struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*DB, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("creating pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}
	return &DB{pool: pool}, nil
}

func (db *DB) Close() {
	db.pool.Close()
}

func (db *DB) GetActiveSubscriptions(ctx context.Context) ([]Subscription, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, chat_id, deal_type, region, min_price, max_price, min_area, max_area, rooms, min_score,
		        min_underground_place, min_kitchen_area, min_floor, max_floor, min_ceiling_height,
		        children_required, pets_required, dishwasher_required, conditioner_required,
		        min_renovation, balcony_required, bathroom_type, metro_stations
		 FROM user_subscriptions
		 WHERE is_active = TRUE`)
	if err != nil {
		return nil, fmt.Errorf("querying subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var s Subscription
		if err := rows.Scan(&s.ID, &s.ChatID, &s.DealType, &s.Region, &s.MinPrice, &s.MaxPrice,
			&s.MinArea, &s.MaxArea, &s.Rooms, &s.MinScore,
			&s.MinUndergroundPlace, &s.MinKitchenArea, &s.MinFloor, &s.MaxFloor, &s.MinCeilingHeight,
			&s.ChildrenRequired, &s.PetsRequired, &s.DishwasherRequired, &s.ConditionerRequired,
			&s.MinRenovation, &s.BalconyRequired, &s.BathroomType, &s.MetroStations); err != nil {
			return nil, fmt.Errorf("scanning subscription: %w", err)
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

// HasMultipleActiveRegions reports whether chatID currently has active
// subscriptions spanning more than one region, in which case notifications
// should mention which region a flat belongs to.
func (db *DB) HasMultipleActiveRegions(ctx context.Context, chatID int64) (bool, error) {
	var count int
	err := db.pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT region) FROM user_subscriptions
		 WHERE chat_id = $1 AND is_active = TRUE`,
		chatID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("counting distinct regions: %w", err)
	}
	return count > 1, nil
}

// HasMultipleActiveDealTypes reports whether chatID currently has active
// subscriptions spanning more than one deal type (rent and sale), in which
// case notifications should mention which deal type a flat belongs to.
func (db *DB) HasMultipleActiveDealTypes(ctx context.Context, chatID int64) (bool, error) {
	var count int
	err := db.pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT deal_type) FROM user_subscriptions
		 WHERE chat_id = $1 AND is_active = TRUE`,
		chatID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("counting distinct deal types: %w", err)
	}
	return count > 1, nil
}

// ScoringParams holds the 18 customizable scoring multipliers a subscriber
// overrode, mirroring subscription_scoring_params.
type ScoringParams struct {
	AllArea            float64
	KitchenArea        float64
	Pets               float64
	Dishwasher         float64
	Conditioner        float64
	Apartments         float64
	TwoRoom            float64
	ThreeRoom          float64
	FourRoom           float64
	AdditionalRooms    float64
	WindowsYard        float64
	WindowsStreet      float64
	WindowsBoth        float64
	RenovationDesign   float64
	RenovationEuro     float64
	RenovationCosmetic float64
	BathroomSeparated  float64
	Balcony            float64
	Loggia             float64
	Underground        float64
}

// GetScoringParams returns the custom scoring params for subscriptionID, or
// nil if the subscription uses default scoring (no row present).
func (db *DB) GetScoringParams(ctx context.Context, subscriptionID int) (*ScoringParams, error) {
	var p ScoringParams
	err := db.pool.QueryRow(ctx,
		`SELECT all_area_multiplier, kitchen_area_multiplier, pets_multiplier,
		        dishwasher_multiplier, conditioner_multiplier,
		        apartments_multiplier, two_room_multiplier, three_room_multiplier, four_room_multiplier,
		        additional_rooms_multiplier, windows_yard_multiplier, windows_street_multiplier,
		        windows_both_multiplier, renovation_design_mult, renovation_euro_mult,
		        renovation_cosmetic_mult, bathroom_separated_mult, balcony_multiplier,
		        loggia_multiplier, underground_score_mult
		 FROM subscription_scoring_params
		 WHERE subscription_id = $1`,
		subscriptionID,
	).Scan(&p.AllArea, &p.KitchenArea, &p.Pets,
		&p.Dishwasher, &p.Conditioner,
		&p.Apartments, &p.TwoRoom, &p.ThreeRoom, &p.FourRoom,
		&p.AdditionalRooms, &p.WindowsYard, &p.WindowsStreet,
		&p.WindowsBoth, &p.RenovationDesign, &p.RenovationEuro,
		&p.RenovationCosmetic, &p.BathroomSeparated, &p.Balcony,
		&p.Loggia, &p.Underground)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying scoring params: %w", err)
	}
	return &p, nil
}

func (db *DB) GetFlatIDByLink(ctx context.Context, link string) (int, error) {
	var id int
	err := db.pool.QueryRow(ctx,
		`SELECT id FROM flats_history WHERE link = $1`, link).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("getting flat id: %w", err)
	}
	return id, nil
}

func (db *DB) IsAlreadySent(ctx context.Context, subscriptionID, flatID int) (bool, error) {
	var exists bool
	err := db.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM user_sent_messages
			WHERE subscription_id = $1 AND flat_id = $2
		)`, subscriptionID, flatID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking sent message: %w", err)
	}
	return exists, nil
}

func (db *DB) InsertSentMessage(ctx context.Context, subscriptionID, flatID int) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO user_sent_messages (subscription_id, flat_id)
		 VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`,
		subscriptionID, flatID)
	if err != nil {
		return fmt.Errorf("inserting sent message: %w", err)
	}
	return nil
}
