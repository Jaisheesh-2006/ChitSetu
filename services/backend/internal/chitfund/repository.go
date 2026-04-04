package chitfund

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const opTimeout = 5 * time.Second

type Repository struct {
	db                *mongo.Database
	fundsCol          *mongo.Collection
	membersCol        *mongo.Collection
	usersCol          *mongo.Collection
	contribCol        *mongo.Collection
	bidsCol           *mongo.Collection
	auctionSessionCol *mongo.Collection
	auctionResultsCol *mongo.Collection
	payoutsCol        *mongo.Collection
	chatMessagesCol   *mongo.Collection
}

type Fund struct {
	ID                  string    `bson:"_id" json:"_id"`
	Name                string    `bson:"name" json:"name"`
	Description         string    `bson:"description" json:"description"`
	TotalAmount         float64   `bson:"total_amount" json:"total_amount"`
	MonthlyContribution float64   `bson:"monthly_contribution" json:"monthly_contribution"`
	DurationMonths      int       `bson:"duration_months" json:"duration_months"`
	MaxMembers          int       `bson:"max_members" json:"max_members"`
	ContractAddress     string    `bson:"contract_address,omitempty" json:"contract_address,omitempty"`
	Status              string    `bson:"status" json:"status"`
	StartDate           time.Time `bson:"start_date" json:"start_date"`
	CreatorID           string    `bson:"creator_id" json:"creator_id"`
	CreatedAt           time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt           time.Time `bson:"updated_at" json:"updated_at"`
}

type FundWithCount struct {
	Fund
	CurrentMemberCount int64 `json:"current_member_count"`
}

type FundMember struct {
	ID        string     `bson:"_id"`
	FundID    string     `bson:"fund_id"`
	UserID    string     `bson:"user_id"`
	Status    string     `bson:"status"`
	JoinedAt  *time.Time `bson:"joined_at,omitempty" json:"joined_at,omitempty"`
	CreatedAt time.Time  `bson:"created_at"`
}

type UserKYC struct {
	Status string `bson:"status"`
}

type UserCredit struct {
	Score              int       `bson:"score"`
	RiskCategory       string    `bson:"risk_category"`
	CheckedAt          time.Time `bson:"checked_at"`
	DefaultProbability float64   `bson:"default_probability"`
}

type FundUser struct {
	ID       string      `bson:"_id"`
	Email    string      `bson:"email"`
	FullName string      `bson:"full_name"`
	KYC      *UserKYC    `bson:"kyc,omitempty"`
	Credit   *UserCredit `bson:"credit,omitempty"`
}

type MemberView struct {
	UserID             string     `json:"user_id"`
	FullName           string     `json:"full_name"`
	Email              string     `json:"email"`
	Status             string     `json:"status"`
	JoinedAt           *time.Time `json:"joined_at,omitempty"`
	TrustScore         int        `json:"trust_score"`
	RiskBand           string     `json:"risk_band"`
	DefaultProbability float64    `json:"default_probability"`
}

type FundContribution struct {
	FundID      string    `bson:"fund_id" json:"fund_id"`
	UserID      string    `bson:"user_id" json:"user_id"`
	CycleNumber int       `bson:"cycle_number" json:"cycle_number"`
	AmountDue   float64   `bson:"amount_due" json:"amount_due"`
	DueDate     time.Time `bson:"due_date" json:"due_date"`
	Status      string    `bson:"status" json:"status"`
}

func NewRepository(db *mongo.Database) *Repository {
	return &Repository{
		db:                db,
		fundsCol:          db.Collection("funds"),
		membersCol:        db.Collection("fund_members"),
		usersCol:          db.Collection("users"),
		contribCol:        db.Collection("contributions"),
		bidsCol:           db.Collection("bids"),
		auctionSessionCol: db.Collection("auction_sessions"),
		auctionResultsCol: db.Collection("auction_results"),
		payoutsCol:        db.Collection("payouts"),
		chatMessagesCol:   db.Collection("chat_messages"),
	}
}

func (r *Repository) CreateFund(ctx context.Context, fund Fund) (*Fund, error) {
	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	_, err := r.fundsCol.InsertOne(timedCtx, fund)
	if err != nil {
		return nil, fmt.Errorf("insert fund: %w", err)
	}
	return &fund, nil
}

func (r *Repository) ListOpenFunds(ctx context.Context) ([]Fund, error) {
	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	cursor, err := r.fundsCol.Find(
		timedCtx,
		bson.D{{Key: "status", Value: "open"}},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find funds: %w", err)
	}
	defer cursor.Close(timedCtx)

	result := make([]Fund, 0)
	for cursor.Next(timedCtx) {
		var fund Fund
		if err := cursor.Decode(&fund); err != nil {
			return nil, fmt.Errorf("decode fund: %w", err)
		}
		result = append(result, fund)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate funds: %w", err)
	}
	return result, nil
}

func (r *Repository) GetFundByID(ctx context.Context, fundID string) (*Fund, error) {
	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	var fund Fund
	err := r.fundsCol.FindOne(timedCtx, bson.D{{Key: "_id", Value: fundID}}).Decode(&fund)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("find fund by id: %w", err)
	}
	return &fund, nil
}

func (r *Repository) CountActiveMembers(ctx context.Context, fundID string) (int64, error) {
	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	count, err := r.membersCol.CountDocuments(
		timedCtx,
		bson.D{{Key: "fund_id", Value: fundID}, {Key: "status", Value: "active"}},
	)
	if err != nil {
		return 0, fmt.Errorf("count active members: %w", err)
	}
	return count, nil
}

func (r *Repository) GetFundUser(ctx context.Context, userID string) (*FundUser, error) {
	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	var user FundUser
	err := r.usersCol.FindOne(
		timedCtx,
		bson.D{{Key: "_id", Value: userID}},
		options.FindOne().SetProjection(
			bson.D{
				{Key: "email", Value: 1},
				{Key: "full_name", Value: 1},
				{Key: "kyc", Value: 1},
				{Key: "credit", Value: 1},
			},
		),
	).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("find user for fund flow: %w", err)
	}
	return &user, nil
}

func (r *Repository) HasMembershipRecord(ctx context.Context, fundID, userID string) (bool, error) {
	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	count, err := r.membersCol.CountDocuments(
		timedCtx,
		bson.D{{Key: "fund_id", Value: fundID}, {Key: "user_id", Value: userID}},
	)
	if err != nil {
		return false, fmt.Errorf("check membership existence: %w", err)
	}
	return count > 0, nil
}

func (r *Repository) GetMembershipStatus(ctx context.Context, fundID, userID string) (string, error) {
	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	var member FundMember
	err := r.membersCol.FindOne(
		timedCtx,
		bson.D{{Key: "fund_id", Value: fundID}, {Key: "user_id", Value: userID}},
		options.FindOne().SetProjection(bson.D{{Key: "status", Value: 1}}),
	).Decode(&member)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "none", nil
		}
		return "", fmt.Errorf("get membership status: %w", err)
	}

	if member.Status == "" {
		return "none", nil
	}

	return member.Status, nil
}

func (r *Repository) HasActiveMembership(ctx context.Context, fundID, userID string) (bool, error) {
	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	count, err := r.membersCol.CountDocuments(
		timedCtx,
		bson.D{{Key: "fund_id", Value: fundID}, {Key: "user_id", Value: userID}, {Key: "status", Value: "active"}},
	)
	if err != nil {
		return false, fmt.Errorf("check active membership: %w", err)
	}
	return count > 0, nil
}

func (r *Repository) CreatePendingApplication(ctx context.Context, fundID, userID string) error {
	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	_, err := r.membersCol.InsertOne(
		timedCtx,
		FundMember{
			ID:        uuid.NewString(),
			FundID:    fundID,
			UserID:    userID,
			Status:    "pending",
			CreatedAt: time.Now(),
		},
	)
	if err != nil {
		if isDuplicateKey(err) {
			return ErrMembershipAlreadyExists
		}
		return fmt.Errorf("insert pending application: %w", err)
	}
	return nil
}

func (r *Repository) ApprovePendingMember(ctx context.Context, fundID, userID string) (bool, error) {
	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	now := time.Now()
	result, err := r.membersCol.UpdateOne(
		timedCtx,
		bson.D{{Key: "fund_id", Value: fundID}, {Key: "user_id", Value: userID}, {Key: "status", Value: "pending"}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: "active"}, {Key: "joined_at", Value: now}}}},
	)
	if err != nil {
		return false, fmt.Errorf("approve pending member: %w", err)
	}
	return result.MatchedCount > 0, nil
}

func (r *Repository) RejectPendingMember(ctx context.Context, fundID, userID string) (bool, error) {
	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	result, err := r.membersCol.UpdateOne(
		timedCtx,
		bson.D{{Key: "fund_id", Value: fundID}, {Key: "user_id", Value: userID}, {Key: "status", Value: "pending"}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: "rejected"}}}},
	)
	if err != nil {
		return false, fmt.Errorf("reject pending member: %w", err)
	}

	return result.MatchedCount > 0, nil
}

func (r *Repository) ApprovePendingMemberAndCreateContributions(ctx context.Context, fund Fund, userID string) (bool, error) {
	session, err := r.db.Client().StartSession()
	if err != nil {
		return false, fmt.Errorf("start transaction session: %w", err)
	}
	defer session.EndSession(ctx)

	updated := false
	_, err = session.WithTransaction(ctx, func(sc context.Context) (interface{}, error) {
		now := time.Now()
		updateResult, err := r.membersCol.UpdateOne(
			sc,
			bson.D{{Key: "fund_id", Value: fund.ID}, {Key: "user_id", Value: userID}, {Key: "status", Value: "pending"}},
			bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: "active"}, {Key: "joined_at", Value: now}}}},
		)
		if err != nil {
			return nil, fmt.Errorf("approve pending member: %w", err)
		}
		if updateResult.MatchedCount == 0 {
			return nil, nil
		}

		for cycleNumber := 1; cycleNumber <= fund.DurationMonths; cycleNumber++ {
			dueDate := fund.StartDate.AddDate(0, cycleNumber-1, 0)
			_, err := r.contribCol.UpdateOne(
				sc,
				bson.D{{Key: "fund_id", Value: fund.ID}, {Key: "user_id", Value: userID}, {Key: "cycle_number", Value: cycleNumber}},
				bson.D{{Key: "$setOnInsert", Value: bson.D{
					{Key: "_id", Value: uuid.NewString()},
					{Key: "fund_id", Value: fund.ID},
					{Key: "user_id", Value: userID},
					{Key: "cycle_number", Value: cycleNumber},
					{Key: "amount_due", Value: fund.MonthlyContribution},
					{Key: "due_date", Value: dueDate},
					{Key: "status", Value: "pending"},
					{Key: "created_at", Value: now},
				}}},
				options.UpdateOne().SetUpsert(true),
			)
			if err != nil {
				return nil, fmt.Errorf("create contribution schedule for cycle %d: %w", cycleNumber, err)
			}
		}

		updated = true
		return nil, nil
	})
	if err != nil {
		return false, err
	}

	return updated, nil
}

func (r *Repository) IsActiveMember(ctx context.Context, fundID, userID string) (bool, error) {
	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	count, err := r.membersCol.CountDocuments(
		timedCtx,
		bson.D{{Key: "fund_id", Value: fundID}, {Key: "user_id", Value: userID}, {Key: "status", Value: "active"}},
	)
	if err != nil {
		return false, fmt.Errorf("check active membership: %w", err)
	}

	return count > 0, nil
}

func (r *Repository) ListContributionsByFundAndCycle(ctx context.Context, fundID string, cycleNumber int) ([]FundContribution, error) {
	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	cursor, err := r.contribCol.Find(
		timedCtx,
		bson.D{{Key: "fund_id", Value: fundID}, {Key: "cycle_number", Value: cycleNumber}},
		options.Find().SetSort(bson.D{{Key: "due_date", Value: 1}, {Key: "user_id", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find contributions by fund and cycle: %w", err)
	}
	defer cursor.Close(timedCtx)

	contributions := make([]FundContribution, 0)
	for cursor.Next(timedCtx) {
		var contribution FundContribution
		if err := cursor.Decode(&contribution); err != nil {
			return nil, fmt.Errorf("decode fund contribution: %w", err)
		}
		contributions = append(contributions, contribution)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate fund contributions: %w", err)
	}

	return contributions, nil
}

func (r *Repository) GetFundByContractAddress(ctx context.Context, contractAddress string) (*Fund, error) {
	if contractAddress == "" {
		return nil, nil
	}

	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	var fund Fund
	err := r.fundsCol.FindOne(timedCtx, bson.D{{Key: "contract_address", Value: contractAddress}}).Decode(&fund)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("find fund by contract address: %w", err)
	}

	return &fund, nil
}

func (r *Repository) GetContributionByAmount(ctx context.Context, fundID, userID string, amount float64) (*FundContribution, error) {
	if fundID == "" || userID == "" {
		return nil, nil
	}

	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	const epsilon = 0.000001
	filter := bson.D{
		{Key: "fund_id", Value: fundID},
		{Key: "user_id", Value: userID},
		{Key: "amount_due", Value: bson.D{{Key: "$gte", Value: amount - epsilon}, {Key: "$lte", Value: amount + epsilon}}},
	}

	var contribution FundContribution
	err := r.contribCol.FindOne(
		timedCtx,
		filter,
		options.FindOne().SetSort(bson.D{{Key: "cycle_number", Value: -1}}),
	).Decode(&contribution)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("find contribution by amount: %w", err)
	}

	return &contribution, nil
}

func (r *Repository) ListMembersByFund(ctx context.Context, fundID string) ([]FundMember, error) {
	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	cursor, err := r.membersCol.Find(
		timedCtx,
		bson.D{{Key: "fund_id", Value: fundID}},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find fund members: %w", err)
	}
	defer cursor.Close(timedCtx)

	result := make([]FundMember, 0)
	for cursor.Next(timedCtx) {
		var member FundMember
		if err := cursor.Decode(&member); err != nil {
			return nil, fmt.Errorf("decode fund member: %w", err)
		}
		result = append(result, member)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate fund members: %w", err)
	}
	return result, nil
}

func (r *Repository) ListUsersByIDs(ctx context.Context, userIDs []string) (map[string]FundUser, error) {
	if len(userIDs) == 0 {
		return map[string]FundUser{}, nil
	}

	values := make(bson.A, 0, len(userIDs))
	for _, userID := range userIDs {
		values = append(values, userID)
	}

	timedCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()

	cursor, err := r.usersCol.Find(
		timedCtx,
		bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: values}}}},
		options.Find().SetProjection(bson.D{
			{Key: "email", Value: 1},
			{Key: "full_name", Value: 1},
			{Key: "credit", Value: 1},
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("find users by ids: %w", err)
	}
	defer cursor.Close(timedCtx)

	result := make(map[string]FundUser, len(userIDs))
	for cursor.Next(timedCtx) {
		var user FundUser
		if err := cursor.Decode(&user); err != nil {
			return nil, fmt.Errorf("decode user in list users by ids: %w", err)
		}
		result[user.ID] = user
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate users by ids: %w", err)
	}
	return result, nil
}

func (r *Repository) DeleteFundCascade(ctx context.Context, fundID string) error {
	session, err := r.db.Client().StartSession()
	if err != nil {
		return fmt.Errorf("start fund delete session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sc context.Context) (interface{}, error) {
		filter := bson.D{{Key: "fund_id", Value: fundID}}

		if _, err := r.contribCol.DeleteMany(sc, filter); err != nil {
			return nil, fmt.Errorf("delete contributions: %w", err)
		}
		if _, err := r.membersCol.DeleteMany(sc, filter); err != nil {
			return nil, fmt.Errorf("delete fund members: %w", err)
		}
		if _, err := r.bidsCol.DeleteMany(sc, filter); err != nil {
			return nil, fmt.Errorf("delete bids: %w", err)
		}
		if _, err := r.auctionSessionCol.DeleteMany(sc, filter); err != nil {
			return nil, fmt.Errorf("delete auction sessions: %w", err)
		}
		if _, err := r.auctionResultsCol.DeleteMany(sc, filter); err != nil {
			return nil, fmt.Errorf("delete auction results: %w", err)
		}
		if _, err := r.payoutsCol.DeleteMany(sc, filter); err != nil {
			return nil, fmt.Errorf("delete payouts: %w", err)
		}
		if _, err := r.chatMessagesCol.DeleteMany(sc, filter); err != nil {
			return nil, fmt.Errorf("delete chat messages: %w", err)
		}

		result, err := r.fundsCol.DeleteOne(sc, bson.D{{Key: "_id", Value: fundID}})
		if err != nil {
			return nil, fmt.Errorf("delete fund: %w", err)
		}
		if result.DeletedCount == 0 {
			return nil, fmt.Errorf("fund not found")
		}

		return nil, nil
	})
	if err != nil {
		return err
	}

	return nil
}

var ErrMembershipAlreadyExists = fmt.Errorf("membership already exists")

func isDuplicateKey(err error) bool {
	var writeErr mongo.WriteException
	if errors.As(err, &writeErr) {
		for _, we := range writeErr.WriteErrors {
			if we.Code == 11000 {
				return true
			}
		}
	}
	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) {
		return cmdErr.Code == 11000
	}
	return false
}
