package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Subscription struct {
	ID       int
	ChatID   int64
	MinPrice int
	MaxPrice int
	MinArea  float64
	MaxArea  float64
	Rooms    []int32
	MinScore int
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
		`SELECT id, chat_id, min_price, max_price, min_area, max_area, rooms, min_score
		 FROM user_subscriptions
		 WHERE is_active = TRUE`)
	if err != nil {
		return nil, fmt.Errorf("querying subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var s Subscription
		if err := rows.Scan(&s.ID, &s.ChatID, &s.MinPrice, &s.MaxPrice,
			&s.MinArea, &s.MaxArea, &s.Rooms, &s.MinScore); err != nil {
			return nil, fmt.Errorf("scanning subscription: %w", err)
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
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
