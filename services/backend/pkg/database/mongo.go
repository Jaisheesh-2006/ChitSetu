package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type Config struct {
	URI            string
	DatabaseName   string
	MaxPoolSize    uint64
	MinPoolSize    uint64
	ConnectTimeout time.Duration
}
type Store struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func LoadConfigFromEnv() Config {
	connectTimeoutMs := getenvOrDefaultInt("MONGO_CONNECT_TIMEOUT_MS", 5000)
	return Config{
		URI:            getenvOrDefault("MONGO_URI", "mongodb://localhost:27017"),
		DatabaseName:   getenvOrDefault("MONGO_DB_NAME", "chitsetu_db"),
		MaxPoolSize:    uint64(getenvOrDefaultInt("MONGO_MAX_POOL_SIZE", 50)),
		MinPoolSize:    uint64(getenvOrDefaultInt("MONGO_MIN_POOL_SIZE", 5)),
		ConnectTimeout: time.Duration(connectTimeoutMs) * time.Millisecond,
	}
}
func Connect(ctx context.Context, cfg Config) (*Store, error) {
	clientOptions := options.Client().
		ApplyURI(cfg.URI).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMinPoolSize(cfg.MinPoolSize).
		SetConnectTimeout(cfg.ConnectTimeout)

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("connect to mongo: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ping mongo primary: %w", err)
	}

	return &Store{
		Client:   client,
		Database: client.Database(cfg.DatabaseName),
	}, nil
}
func (s *Store) Ping(ctx context.Context) error {
	return s.Client.Ping(ctx, readpref.Primary())
}

func (s *Store) Close(ctx context.Context) error {
	if err := s.Client.Disconnect(ctx); err != nil {
		return fmt.Errorf("disconnect mongo client: %w", err)
	}
	return nil
}

func (s *Store) EnsureIndexes(ctx context.Context) error {
	indexSets := map[string][]mongo.IndexModel{
		"users": {
			{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_users_email")},
			{Keys: bson.D{{Key: "profile.pan", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true).SetName("uniq_users_profile_pan_sparse")},
		},
		"auth_sessions": {
			{Keys: bson.D{{Key: "refresh_token_hash", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_auth_sessions_refresh_hash")},
			{Keys: bson.D{{Key: "expires_at", Value: 1}}, Options: options.Index().SetName("idx_auth_sessions_expires_at")},
		},
		"user_profiles": {
			{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_user_profiles_user_id")},
		},
		"application_results": {
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "fund_id", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_application_results_user_fund").SetPartialFilterExpression(bson.M{"fund_id": bson.M{"$type": "string"}})},
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}, Options: options.Index().SetName("idx_application_results_user_created")},
		},
		"funds": {
			{Keys: bson.D{{Key: "status", Value: 1}}, Options: options.Index().SetName("idx_funds_status")},
			{Keys: bson.D{{Key: "creator_id", Value: 1}}, Options: options.Index().SetName("idx_funds_creator_id")},
		},
		"fund_members": {
			{Keys: bson.D{{Key: "fund_id", Value: 1}, {Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_fund_members_fund_user")},
			{Keys: bson.D{{Key: "fund_id", Value: 1}}, Options: options.Index().SetName("idx_fund_members_fund_id")},
			{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetName("idx_fund_members_user_id")},
		},
		"contributions": {
			{Keys: bson.D{{Key: "fund_id", Value: 1}, {Key: "user_id", Value: 1}, {Key: "cycle_number", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_contributions_fund_user_cycle_number")},
			{Keys: bson.D{{Key: "fund_id", Value: 1}, {Key: "user_id", Value: 1}}, Options: options.Index().SetName("idx_contributions_fund_user")},
			{Keys: bson.D{{Key: "fund_id", Value: 1}, {Key: "cycle_number", Value: 1}}, Options: options.Index().SetName("idx_contributions_fund_cycle_number")},
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "status", Value: 1}}, Options: options.Index().SetName("idx_contributions_user_status")},
			{Keys: bson.D{{Key: "status", Value: 1}, {Key: "due_date", Value: 1}}, Options: options.Index().SetName("idx_contributions_status_due")},
		},
		"payment_sessions": {
			{Keys: bson.D{{Key: "contribution_id", Value: 1}, {Key: "status", Value: 1}, {Key: "expires_at", Value: 1}}, Options: options.Index().SetName("idx_payment_sessions_contrib_status_exp")},
			{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetName("idx_payment_sessions_user")},
		},
		"payment_orders": {
			{Keys: bson.D{{Key: "session_id", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_payment_orders_session")},
			{Keys: bson.D{{Key: "order_id", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_payment_orders_order")},
		},
		"auction_sessions": {
			{Keys: bson.D{{Key: "fund_id", Value: 1}, {Key: "cycle_number", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_auction_sessions_fund_cycle")},
			{Keys: bson.D{{Key: "status", Value: 1}, {Key: "last_bid_at", Value: 1}, {Key: "created_at", Value: 1}}, Options: options.Index().SetName("idx_auction_sessions_live_idle")},
		},
		"auction_results": {
			{Keys: bson.D{{Key: "fund_id", Value: 1}, {Key: "cycle_number", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_auction_results_fund_cycle")},
			{Keys: bson.D{{Key: "fund_id", Value: 1}, {Key: "winner_user_id", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_auction_results_fund_winner")},
		},
		"payouts": {
			{Keys: bson.D{{Key: "fund_id", Value: 1}, {Key: "cycle_number", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_payouts_fund_cycle")},
			{Keys: bson.D{{Key: "status", Value: 1}, {Key: "updated_at", Value: 1}}, Options: options.Index().SetName("idx_payouts_status_updated")},
		},
	}

	for collectionName, indexes := range indexSets {
		if len(indexes) == 0 {
			continue
		}
		if _, err := s.Database.Collection(collectionName).Indexes().CreateMany(ctx, indexes); err != nil {
			return fmt.Errorf("create indexes for %s: %w", collectionName, err)
		}
	}

	// Cleanup obsolete bids unique index from pre-increment model to allow multiple bids per user.
	_ = s.Database.Collection("bids").Indexes().DropOne(ctx, "uniq_bids_fund_cycle_user")

	return nil
}

func getenvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getenvOrDefaultInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil {
		return defaultValue
	}
	return parsed
}
