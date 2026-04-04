package chat

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const opTimeout = 8 * time.Second

type Message struct {
	ID          string    `bson:"_id" json:"_id"`
	FundID      string    `bson:"fund_id" json:"fund_id"`
	UserID      string    `bson:"user_id" json:"user_id"`
	FullName    string    `bson:"full_name" json:"full_name"`
	MessageText string    `bson:"message" json:"message"`
	ChatType    string    `bson:"chat_type" json:"chat_type"` // "fund" or "auction"
	CycleNumber int       `bson:"cycle_number,omitempty" json:"cycle_number,omitempty"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
}

type Repository struct {
	messagesCol *mongo.Collection
	membersCol  *mongo.Collection
	usersCol    *mongo.Collection
}

func NewRepository(db *mongo.Database) *Repository {
	return &Repository{
		messagesCol: db.Collection("chat_messages"),
		membersCol:  db.Collection("fund_members"),
		usersCol:    db.Collection("users"),
	}
}

func (r *Repository) SaveMessage(ctx context.Context, msg *Message) error {
	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	msg.ID = uuid.New().String()
	msg.CreatedAt = time.Now().UTC()

	_, err := r.messagesCol.InsertOne(timedCtx, msg)
	if err != nil {
		return fmt.Errorf("insert chat message: %w", err)
	}
	return nil
}

func (r *Repository) GetMessages(ctx context.Context, fundID, chatType string, cycleNumber, limit int, before *time.Time) ([]Message, error) {
	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	filter := bson.M{
		"fund_id":   fundID,
		"chat_type": chatType,
	}
	if chatType == "auction" && cycleNumber > 0 {
		filter["cycle_number"] = cycleNumber
	}
	if before != nil {
		filter["created_at"] = bson.M{"$lt": *before}
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.messagesCol.Find(timedCtx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find chat messages: %w", err)
	}
	defer cursor.Close(timedCtx)

	var messages []Message
	if err := cursor.All(timedCtx, &messages); err != nil {
		return nil, fmt.Errorf("decode chat messages: %w", err)
	}
	return messages, nil
}

func (r *Repository) IsActiveMember(ctx context.Context, fundID, userID string) (bool, error) {
	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	count, err := r.membersCol.CountDocuments(timedCtx, bson.M{
		"fund_id": fundID,
		"user_id": userID,
		"status":  "active",
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *Repository) GetUserFullName(ctx context.Context, userID string) string {
	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	var user struct {
		FullName string `bson:"full_name"`
	}
	err := r.usersCol.FindOne(timedCtx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return "Anonymous"
	}
	if user.FullName == "" {
		return "Anonymous"
	}
	return user.FullName
}
