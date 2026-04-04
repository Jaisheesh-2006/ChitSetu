package auction

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const repoOpTimeout = 8 * time.Second

var (
	ErrAuctionNotLive              = errors.New("auction is not live")
	ErrAuctionAlreadyLive          = errors.New("auction is already live")
	ErrAuctionStartDenied          = errors.New("only fund creator can start auction")
	ErrAuctionParticipantsNotReady = errors.New("all active members must join the auction room before bidding can start")
	ErrNotFundMember               = errors.New("user is not an active fund member")
	ErrContributionUnpaid          = errors.New("user contribution for this cycle is not paid")
	ErrUserAlreadyWon              = errors.New("user already won in this fund")
	ErrInvalidIncrement            = errors.New("increment must be one of 10, 100, 200")
	ErrNoEligibleWinner            = errors.New("no eligible winner found")
	ErrAuctionNotFinalized         = errors.New("auction finalization preconditions not met")
	ErrConsecutiveBid              = errors.New("you are currently leading — wait until someone surpasses your best bid")
)

type Repository struct {
	db                 *mongo.Database
	fundsCol           *mongo.Collection
	membersCol         *mongo.Collection
	contribCol         *mongo.Collection
	bidsCol            *mongo.Collection
	auctionSessionsCol *mongo.Collection
	auctionResultsCol  *mongo.Collection
	usersCol           *mongo.Collection
	payoutsCol         *mongo.Collection
	walletsCol         *mongo.Collection
}

type AuctionSession struct {
	ID               string     `bson:"_id" json:"_id"`
	FundID           string     `bson:"fund_id" json:"fund_id"`
	CycleNumber      int        `bson:"cycle_number" json:"cycle_number"`
	Status           string     `bson:"status" json:"status"`
	CurrentPrice     float64    `bson:"current_price" json:"current_price"`
	LastBidUserID    *string    `bson:"last_bid_user_id,omitempty" json:"last_bid_user_id,omitempty"`
	LastBidAt        *time.Time `bson:"last_bid_at,omitempty" json:"last_bid_at,omitempty"`
	BiddingStartedAt *time.Time `bson:"bidding_started_at,omitempty" json:"bidding_started_at,omitempty"`
	CreatedAt        time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time  `bson:"updated_at" json:"updated_at"`
}

type Bid struct {
	ID             string    `bson:"_id" json:"_id"`
	FundID         string    `bson:"fund_id" json:"fund_id"`
	CycleNumber    int       `bson:"cycle_number" json:"cycle_number"`
	UserID         string    `bson:"user_id" json:"user_id"`
	Increment      float64   `bson:"increment" json:"increment"`
	ResultingPrice float64   `bson:"resulting_price" json:"resulting_price"`
	CreatedAt      time.Time `bson:"created_at" json:"created_at"`
}

type AuctionResult struct {
	ID           string    `bson:"_id" json:"_id"`
	FundID       string    `bson:"fund_id" json:"fund_id"`
	CycleNumber  int       `bson:"cycle_number" json:"cycle_number"`
	WinnerUserID string    `bson:"winner_user_id" json:"winner_user_id"`
	WinningPrice float64   `bson:"winning_price" json:"winning_price"`
	PayoutAmount float64   `bson:"payout_amount" json:"payout_amount"`
	CreatedAt    time.Time `bson:"created_at" json:"created_at"`
}

type PayoutRecord struct {
	ID            string     `bson:"_id" json:"_id"`
	FundID        string     `bson:"fund_id" json:"fund_id"`
	CycleNumber   int        `bson:"cycle_number" json:"cycle_number"`
	WinnerUserID  string     `bson:"winner_user_id" json:"winner_user_id"`
	Amount        float64    `bson:"amount" json:"amount"`
	Status        string     `bson:"status" json:"status"`
	Provider      string     `bson:"provider" json:"provider"`
	ReferenceID   string     `bson:"reference_id,omitempty" json:"reference_id,omitempty"`
	LastError     string     `bson:"last_error,omitempty" json:"last_error,omitempty"`
	RetryCount    int        `bson:"retry_count" json:"retry_count"`
	InitiatedAt   time.Time  `bson:"initiated_at" json:"initiated_at"`
	CompletedAt   *time.Time `bson:"completed_at,omitempty" json:"completed_at,omitempty"`
	LastAttemptAt time.Time  `bson:"last_attempt_at" json:"last_attempt_at"`
	CreatedAt     time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time  `bson:"updated_at" json:"updated_at"`
}

type MemberProfileInfo struct {
	UserID        string  `json:"user_id"`
	FullName      string  `json:"full_name"`
	WalletAddress string  `json:"wallet_address"`
	IsWinner      bool    `json:"is_winner"`
	Dividend      float64 `json:"dividend,omitempty"`
}

type AuctionSnapshot struct {
	Session              *AuctionSession     `json:"session,omitempty"`
	Result               *AuctionResult      `json:"result,omitempty"`
	Bids                 []Bid               `json:"bids"`
	LiveCountdownSeconds *int64              `json:"live_countdown_seconds,omitempty"`
	MembersInfo          []MemberProfileInfo `json:"members_info,omitempty"`
}

type fundProjection struct {
	ID                  string    `bson:"_id"`
	CreatorID           string    `bson:"creator_id"`
	MonthlyContribution float64   `bson:"monthly_contribution"`
	DurationMonths      int       `bson:"duration_months"`
	MaxMembers          int       `bson:"max_members"`
	ContractAddress     string    `bson:"contract_address"`
	StartDate           time.Time `bson:"start_date"`
}

func (r *Repository) GetFundByID(ctx context.Context, fundID string) (*fundProjection, error) {
	timedCtx, cancel := context.WithTimeout(ctx, repoOpTimeout)
	defer cancel()

	var fund fundProjection
	err := r.fundsCol.FindOne(timedCtx, bson.M{"_id": fundID}).Decode(&fund)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("fund not found")
		}
		return nil, fmt.Errorf("get fund by id: %w", err)
	}
	return &fund, nil
}

func NewRepository(db *mongo.Database) *Repository {
	return &Repository{
		db:                 db,
		fundsCol:           db.Collection("funds"),
		membersCol:         db.Collection("fund_members"),
		contribCol:         db.Collection("contributions"),
		bidsCol:            db.Collection("bids"),
		auctionSessionsCol: db.Collection("auction_sessions"),
		auctionResultsCol:  db.Collection("auction_results"),
		usersCol:           db.Collection("users"),
		payoutsCol:         db.Collection("payouts"),
		walletsCol:         db.Collection("wallets"),
	}
}

func (r *Repository) GetMembersProfileInfo(ctx context.Context, fundID string) ([]MemberProfileInfo, error) {
	timedCtx, cancel := context.WithTimeout(ctx, repoOpTimeout)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"fund_id": fundID, "status": "active"}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "user_profiles",
			"localField":   "user_id",
			"foreignField": "user_id",
			"as":           "profile",
		}}},
		{{Key: "$unwind", Value: bson.M{"path": "$profile", "preserveNullAndEmptyArrays": true}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "wallets",
			"localField":   "user_id",
			"foreignField": "_id",
			"as":           "wallet",
		}}},
		{{Key: "$unwind", Value: bson.M{"path": "$wallet", "preserveNullAndEmptyArrays": true}}},
		{{Key: "$project", Value: bson.M{
			"user_id":        1,
			"full_name":      "$profile.full_name",
			"wallet_address": "$wallet.address",
		}}},
	}

	cursor, err := r.membersCol.Aggregate(timedCtx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate members profile: %w", err)
	}
	defer cursor.Close(timedCtx)

	var results []MemberProfileInfo
	if err := cursor.All(timedCtx, &results); err != nil {
		return nil, fmt.Errorf("decode members profile: %w", err)
	}

	return results, nil
}

// GetFundMembers returns simple member info (just UserID) for a fund.
func (r *Repository) GetFundMembers(ctx context.Context, fundID string) ([]MemberProfileInfo, error) {
	timedCtx, cancel := context.WithTimeout(ctx, repoOpTimeout)
	defer cancel()

	query := bson.M{"fund_id": fundID, "status": "active"}
	cursor, err := r.membersCol.Find(timedCtx, query)
	if err != nil {
		return nil, fmt.Errorf("find fund members: %w", err)
	}
	defer cursor.Close(timedCtx)

	var members []struct {
		UserID string `bson:"user_id"`
	}
	if err := cursor.All(timedCtx, &members); err != nil {
		return nil, fmt.Errorf("decode fund members: %w", err)
	}

	// Convert to MemberProfileInfo format
	results := make([]MemberProfileInfo, len(members))
	for i, m := range members {
		results[i] = MemberProfileInfo{
			UserID: m.UserID,
		}
	}

	return results, nil
}

func (r *Repository) StartAuction(ctx context.Context, fundID, requesterUserID string, baseAmount float64) (*AuctionSession, error) {
	timedCtx, cancel := context.WithTimeout(ctx, repoOpTimeout)
	defer cancel()

	now := time.Now().UTC()
	session := &AuctionSession{}

	txnSession, err := r.db.Client().StartSession()
	if err != nil {
		return nil, fmt.Errorf("start auction transaction session: %w", err)
	}
	defer txnSession.EndSession(timedCtx)

	_, err = txnSession.WithTransaction(timedCtx, func(sc mongo.SessionContext) (interface{}, error) {
		var fund fundProjection
		err := r.fundsCol.FindOne(
			sc,
			bson.M{"_id": fundID},
			options.FindOne().SetProjection(bson.M{"creator_id": 1, "duration_months": 1, "start_date": 1}),
		).Decode(&fund)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, fmt.Errorf("fund not found")
			}
			return nil, fmt.Errorf("find fund for auction start: %w", err)
		}
		if fund.CreatorID != requesterUserID {
			return nil, ErrAuctionStartDenied
		}

		var existing AuctionSession
		err = r.auctionSessionsCol.FindOne(sc, bson.M{"fund_id": fundID, "status": bson.M{"$in": []string{"live", "waiting"}}}).Decode(&existing)
		if err == nil {
			return nil, ErrAuctionAlreadyLive
		}
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("check existing active auction: %w", err)
		}

		cycleNumber := currentCycleNumber(fund.StartDate, fund.DurationMonths, now)

		activeMembersCount, err := r.membersCol.CountDocuments(sc, bson.M{"fund_id": fundID, "status": "active"})
		if err != nil {
			return nil, fmt.Errorf("count active members: %w", err)
		}
		paidContribsCount, err := r.contribCol.CountDocuments(sc, bson.M{"fund_id": fundID, "cycle_number": cycleNumber, "status": "paid"})
		if err != nil {
			return nil, fmt.Errorf("count paid contributions: %w", err)
		}
		if paidContribsCount < activeMembersCount {
			return nil, errors.New("all active members must pay their contribution for this cycle before the auction can start")
		}
		*session = AuctionSession{
			ID:           uuid.NewString(),
			FundID:       fundID,
			CycleNumber:  cycleNumber,
			Status:       "waiting",
			CurrentPrice: baseAmount,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		_, err = r.auctionSessionsCol.InsertOne(sc, session)
		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				return nil, ErrAuctionAlreadyLive
			}
			return nil, fmt.Errorf("insert auction session: %w", err)
		}

		return nil, nil
	})
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (r *Repository) ActivateAuction(ctx context.Context, fundID string) error {
	timedCtx, cancel := context.WithTimeout(ctx, repoOpTimeout)
	defer cancel()

	now := time.Now().UTC()
	res, err := r.auctionSessionsCol.UpdateOne(
		timedCtx,
		bson.M{"fund_id": fundID, "status": "waiting"},
		bson.M{"$set": bson.M{
			"status":             "live",
			"bidding_started_at": now,
			"updated_at":         now,
		}},
	)
	if err != nil {
		return fmt.Errorf("activate auction: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("no waiting auction found to activate")
	}
	return nil
}

func (r *Repository) PlaceIncrementBid(ctx context.Context, fundID, userID string, increment float64) (*Bid, *AuctionSession, error) {
	timedCtx, cancel := context.WithTimeout(ctx, repoOpTimeout)
	defer cancel()

	now := time.Now().UTC()
	createdBid := &Bid{}
	updatedSession := &AuctionSession{}

	txnSession, err := r.db.Client().StartSession()
	if err != nil {
		return nil, nil, fmt.Errorf("start bid transaction session: %w", err)
	}
	defer txnSession.EndSession(timedCtx)

	_, err = txnSession.WithTransaction(timedCtx, func(sc mongo.SessionContext) (interface{}, error) {
		var live AuctionSession
		err := r.auctionSessionsCol.FindOne(sc, bson.M{"fund_id": fundID, "status": "live"}).Decode(&live)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, ErrAuctionNotLive
			}
			return nil, fmt.Errorf("find live auction session: %w", err)
		}

		memberCount, err := r.membersCol.CountDocuments(sc, bson.M{"fund_id": fundID, "user_id": userID, "status": "active"})
		if err != nil {
			return nil, fmt.Errorf("check active member: %w", err)
		}
		if memberCount == 0 {
			return nil, ErrNotFundMember
		}

		paidCount, err := r.contribCol.CountDocuments(sc, bson.M{
			"fund_id":      fundID,
			"user_id":      userID,
			"cycle_number": live.CycleNumber,
			"status":       "paid",
		})
		if err != nil {
			return nil, fmt.Errorf("check paid contribution: %w", err)
		}
		if paidCount == 0 {
			return nil, ErrContributionUnpaid
		}

		winsCount, err := r.auctionResultsCol.CountDocuments(sc, bson.M{"fund_id": fundID, "winner_user_id": userID})
		if err != nil {
			return nil, fmt.Errorf("check prior wins: %w", err)
		}
		if winsCount > 0 {
			return nil, ErrUserAlreadyWon
		}

		// Prevent consecutive bids by the same user
		if live.LastBidUserID != nil && *live.LastBidUserID == userID {
			return nil, ErrConsecutiveBid
		}

		// Aggregate this user's existing discount total for this auction cycle
		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: bson.M{
				"fund_id":      fundID,
				"cycle_number": live.CycleNumber,
				"user_id":      userID,
			}}},
			{{Key: "$group", Value: bson.M{
				"_id":   nil,
				"total": bson.M{"$sum": "$increment"},
			}}},
		}
		cur, err := r.bidsCol.Aggregate(sc, pipeline)
		if err != nil {
			return nil, fmt.Errorf("aggregate user bid total: %w", err)
		}
		var aggResult []struct {
			Total float64 `bson:"total"`
		}
		if err := cur.All(sc, &aggResult); err != nil {
			return nil, fmt.Errorf("decode user bid total: %w", err)
		}
		var existingUserTotal float64
		if len(aggResult) > 0 {
			existingUserTotal = aggResult[0].Total
		}
		newUserTotal := existingUserTotal + increment

		// Only update the session's current_price if this user is now the new leader
		auctionPrice := live.CurrentPrice
		if newUserTotal > live.CurrentPrice {
			auctionPrice = newUserTotal
			err = r.auctionSessionsCol.FindOneAndUpdate(
				sc,
				bson.M{"fund_id": fundID, "cycle_number": live.CycleNumber, "status": "live"},
				bson.M{"$set": bson.M{
					"current_price":    newUserTotal,
					"last_bid_user_id": userID,
					"last_bid_at":      now,
					"updated_at":       now,
				}},
				options.FindOneAndUpdate().SetReturnDocument(options.After),
			).Decode(updatedSession)
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					return nil, ErrAuctionNotLive
				}
				return nil, fmt.Errorf("atomic auction leader update: %w", err)
			}
		} else {
			// Keep the existing leader, but refresh last_bid_at so the 20s window resets on every accepted bid.
			err = r.auctionSessionsCol.FindOneAndUpdate(
				sc,
				bson.M{"fund_id": fundID, "cycle_number": live.CycleNumber, "status": "live"},
				bson.M{"$set": bson.M{
					"last_bid_at": now,
					"updated_at":  now,
				}},
				options.FindOneAndUpdate().SetReturnDocument(options.After),
			).Decode(updatedSession)
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					return nil, ErrAuctionNotLive
				}
				return nil, fmt.Errorf("refresh auction idle timer: %w", err)
			}
			auctionPrice = updatedSession.CurrentPrice
		}

		// Record the bid event
		*createdBid = Bid{
			ID:             uuid.NewString(),
			FundID:         fundID,
			CycleNumber:    live.CycleNumber,
			UserID:         userID,
			Increment:      increment,
			ResultingPrice: auctionPrice, // stores the auction-wide leading discount
			CreatedAt:      now,
		}
		if _, err := r.bidsCol.InsertOne(sc, createdBid); err != nil {
			return nil, fmt.Errorf("insert bid event log: %w", err)
		}

		return nil, nil
	})
	if err != nil {
		return nil, nil, err
	}

	return createdBid, updatedSession, nil
}

func (r *Repository) GetAuctionSnapshot(ctx context.Context, fundID string) (*AuctionSnapshot, error) {
	timedCtx, cancel := context.WithTimeout(ctx, repoOpTimeout)
	defer cancel()

	snapshot := &AuctionSnapshot{Bids: make([]Bid, 0)}

	var session AuctionSession
	err := r.auctionSessionsCol.FindOne(
		timedCtx,
		bson.M{"fund_id": fundID},
		options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	).Decode(&session)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return snapshot, nil
		}
		return nil, fmt.Errorf("find latest auction session: %w", err)
	}
	snapshot.Session = &session

	cursor, err := r.bidsCol.Find(
		timedCtx,
		bson.M{"fund_id": fundID, "cycle_number": session.CycleNumber},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(50),
	)
	if err != nil {
		return nil, fmt.Errorf("find auction bids: %w", err)
	}
	defer cursor.Close(timedCtx)

	for cursor.Next(timedCtx) {
		var bid Bid
		if err := cursor.Decode(&bid); err != nil {
			return nil, fmt.Errorf("decode auction bid: %w", err)
		}
		snapshot.Bids = append(snapshot.Bids, bid)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate auction bids: %w", err)
	}

	var result AuctionResult
	err = r.auctionResultsCol.FindOne(
		timedCtx,
		bson.M{"fund_id": fundID, "cycle_number": session.CycleNumber},
	).Decode(&result)
	if err == nil {
		snapshot.Result = &result
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, fmt.Errorf("find auction result: %w", err)
	}

	return snapshot, nil
}

func (r *Repository) IsFundParticipant(ctx context.Context, fundID, userID string) (bool, error) {
	timedCtx, cancel := context.WithTimeout(ctx, repoOpTimeout)
	defer cancel()

	var fund fundProjection
	err := r.fundsCol.FindOne(
		timedCtx,
		bson.M{"_id": fundID},
		options.FindOne().SetProjection(bson.M{"creator_id": 1}),
	).Decode(&fund)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, fmt.Errorf("find fund participant details: %w", err)
	}
	if fund.CreatorID == userID {
		return true, nil
	}

	count, err := r.membersCol.CountDocuments(timedCtx, bson.M{"fund_id": fundID, "user_id": userID, "status": "active"})
	if err != nil {
		return false, fmt.Errorf("count active membership: %w", err)
	}
	return count > 0, nil
}

func (r *Repository) CountActiveMembers(ctx context.Context, fundID string) (int, error) {
	timedCtx, cancel := context.WithTimeout(ctx, repoOpTimeout)
	defer cancel()

	count, err := r.membersCol.CountDocuments(timedCtx, bson.M{"fund_id": fundID, "status": "active"})
	if err != nil {
		return 0, fmt.Errorf("count active members: %w", err)
	}

	return int(count), nil
}

func (r *Repository) ListLiveAuctions(ctx context.Context, limit int64) ([]AuctionSession, error) {
	timedCtx, cancel := context.WithTimeout(ctx, repoOpTimeout)
	defer cancel()

	if limit <= 0 {
		limit = 100
	}

	cursor, err := r.auctionSessionsCol.Find(
		timedCtx,
		bson.M{"status": "live"},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}).SetLimit(limit),
	)
	if err != nil {
		return nil, fmt.Errorf("find live auctions: %w", err)
	}
	defer cursor.Close(timedCtx)

	result := make([]AuctionSession, 0)
	for cursor.Next(timedCtx) {
		var item AuctionSession
		if err := cursor.Decode(&item); err != nil {
			return nil, fmt.Errorf("decode live auction: %w", err)
		}
		result = append(result, item)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate live auctions: %w", err)
	}

	return result, nil
}

func (r *Repository) FinalizeAuction(ctx context.Context, fundID string, cycleNumber int) (*AuctionResult, bool, error) {
	timedCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	now := time.Now().UTC()
	finalized := &AuctionResult{}
	created := false

	txnSession, err := r.db.Client().StartSession()
	if err != nil {
		return nil, false, fmt.Errorf("start finalize transaction session: %w", err)
	}
	defer txnSession.EndSession(timedCtx)

	_, err = txnSession.WithTransaction(timedCtx, func(sc mongo.SessionContext) (interface{}, error) {
		var live AuctionSession
		err := r.auctionSessionsCol.FindOne(
			sc,
			bson.M{"fund_id": fundID, "cycle_number": cycleNumber, "status": "live"},
		).Decode(&live)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, ErrAuctionNotFinalized
			}
			return nil, fmt.Errorf("find live session for finalize: %w", err)
		}

		var existing AuctionResult
		err = r.auctionResultsCol.FindOne(sc, bson.M{"fund_id": fundID, "cycle_number": cycleNumber}).Decode(&existing)
		if err == nil {
			if _, updateErr := r.auctionSessionsCol.UpdateOne(sc, bson.M{"_id": live.ID, "status": "live"}, bson.M{"$set": bson.M{"status": "ended", "updated_at": now}}); updateErr != nil {
				return nil, fmt.Errorf("mark ended after existing result: %w", updateErr)
			}
			*finalized = existing
			created = false
			return nil, nil
		}
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("check existing auction result: %w", err)
		}

		winnerUserID := ""
		winningPrice := 0.0
		if live.LastBidUserID != nil && *live.LastBidUserID != "" {
			winnerUserID = *live.LastBidUserID
			winningPrice = live.CurrentPrice
		} else {
			winnerUserID, err = r.pickFallbackWinner(sc, fundID, cycleNumber)
			if err != nil {
				return nil, err
			}
		}

		var fund fundProjection
		err = r.fundsCol.FindOne(sc, bson.M{"_id": fundID}, options.FindOne().SetProjection(bson.M{"monthly_contribution": 1})).Decode(&fund)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, fmt.Errorf("fund not found during finalize")
			}
			return nil, fmt.Errorf("load fund details during finalize: %w", err)
		}

		memberCount, err := r.membersCol.CountDocuments(sc, bson.M{"fund_id": fundID, "status": "active"})
		if err != nil {
			return nil, fmt.Errorf("count active members during finalize: %w", err)
		}
		totalPool := float64(memberCount) * fund.MonthlyContribution
		payoutAmount := totalPool - winningPrice
		if payoutAmount < 0 {
			payoutAmount = 0
		}

		*finalized = AuctionResult{
			ID:           uuid.NewString(),
			FundID:       fundID,
			CycleNumber:  cycleNumber,
			WinnerUserID: winnerUserID,
			WinningPrice: winningPrice,
			PayoutAmount: payoutAmount,
			CreatedAt:    now,
		}

		_, err = r.auctionResultsCol.InsertOne(sc, finalized)
		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				created = false
				return nil, nil
			}
			return nil, fmt.Errorf("insert auction result: %w", err)
		}

		updateResult, err := r.auctionSessionsCol.UpdateOne(
			sc,
			bson.M{"_id": live.ID, "status": "live"},
			bson.M{"$set": bson.M{"status": "ended", "updated_at": now}},
		)
		if err != nil {
			return nil, fmt.Errorf("mark auction session ended: %w", err)
		}
		if updateResult.MatchedCount == 0 {
			return nil, fmt.Errorf("mark auction session ended affected 0 rows")
		}

		created = true
		return nil, nil
	})
	if err != nil {
		if errors.Is(err, ErrAuctionNotFinalized) {
			return nil, false, ErrAuctionNotFinalized
		}
		return nil, false, err
	}

	return finalized, created, nil
}

func (r *Repository) UpsertPayoutInitiated(ctx context.Context, fundID string, cycleNumber int, winnerUserID string, amount float64, provider string) (*PayoutRecord, error) {
	timedCtx, cancel := context.WithTimeout(ctx, repoOpTimeout)
	defer cancel()

	now := time.Now().UTC()
	filter := bson.M{"fund_id": fundID, "cycle_number": cycleNumber}
	update := bson.M{
		"$set": bson.M{
			"winner_user_id":  winnerUserID,
			"amount":          amount,
			"provider":        provider,
			"status":          "initiated",
			"last_attempt_at": now,
			"updated_at":      now,
		},
		"$setOnInsert": bson.M{
			"_id":          uuid.NewString(),
			"created_at":   now,
			"initiated_at": now,
			"retry_count":  0,
		},
	}

	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var payout PayoutRecord
	if err := r.payoutsCol.FindOneAndUpdate(timedCtx, filter, update, opts).Decode(&payout); err != nil {
		return nil, fmt.Errorf("upsert payout initiated: %w", err)
	}

	return &payout, nil
}

func (r *Repository) MarkPayoutCompleted(ctx context.Context, payoutID, referenceID string) error {
	timedCtx, cancel := context.WithTimeout(ctx, repoOpTimeout)
	defer cancel()

	now := time.Now().UTC()
	result, err := r.payoutsCol.UpdateOne(
		timedCtx,
		bson.M{"_id": payoutID},
		bson.M{"$set": bson.M{
			"status":          "completed",
			"reference_id":    referenceID,
			"last_error":      "",
			"completed_at":    now,
			"last_attempt_at": now,
			"updated_at":      now,
		}},
	)
	if err != nil {
		return fmt.Errorf("mark payout completed: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("mark payout completed affected 0 rows")
	}
	return nil
}

func (r *Repository) MarkPayoutFailed(ctx context.Context, payoutID string, failure error) error {
	timedCtx, cancel := context.WithTimeout(ctx, repoOpTimeout)
	defer cancel()

	message := "payout failed"
	if failure != nil {
		message = failure.Error()
	}
	result, err := r.payoutsCol.UpdateOne(
		timedCtx,
		bson.M{"_id": payoutID},
		bson.M{
			"$set": bson.M{
				"status":          "failed",
				"last_error":      message,
				"last_attempt_at": time.Now().UTC(),
				"updated_at":      time.Now().UTC(),
			},
			"$inc": bson.M{"retry_count": 1},
		},
	)
	if err != nil {
		return fmt.Errorf("mark payout failed: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("mark payout failed affected 0 rows")
	}
	return nil
}

func (r *Repository) ListFailedPayouts(ctx context.Context, limit int64) ([]PayoutRecord, error) {
	timedCtx, cancel := context.WithTimeout(ctx, repoOpTimeout)
	defer cancel()

	if limit <= 0 {
		limit = 50
	}

	cursor, err := r.payoutsCol.Find(
		timedCtx,
		bson.M{"status": "failed", "retry_count": bson.M{"$lt": 10}},
		options.Find().SetSort(bson.D{{Key: "updated_at", Value: 1}}).SetLimit(limit),
	)
	if err != nil {
		return nil, fmt.Errorf("find failed payouts: %w", err)
	}
	defer cursor.Close(timedCtx)

	result := make([]PayoutRecord, 0)
	for cursor.Next(timedCtx) {
		var item PayoutRecord
		if err := cursor.Decode(&item); err != nil {
			return nil, fmt.Errorf("decode failed payout: %w", err)
		}
		result = append(result, item)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate failed payouts: %w", err)
	}
	return result, nil
}

func (r *Repository) pickFallbackWinner(ctx context.Context, fundID string, cycleNumber int) (string, error) {
	cursor, err := r.membersCol.Find(ctx, bson.M{"fund_id": fundID, "status": "active"})
	if err != nil {
		return "", fmt.Errorf("find active members for fallback winner: %w", err)
	}
	defer cursor.Close(ctx)

	type candidate struct {
		UserID string
		Score  int
	}

	candidates := make([]candidate, 0)
	for cursor.Next(ctx) {
		var member struct {
			UserID string `bson:"user_id"`
		}
		if err := cursor.Decode(&member); err != nil {
			return "", fmt.Errorf("decode member for fallback winner: %w", err)
		}

		paidCount, err := r.contribCol.CountDocuments(ctx, bson.M{
			"fund_id":      fundID,
			"user_id":      member.UserID,
			"cycle_number": cycleNumber,
			"status":       "paid",
		})
		if err != nil {
			return "", fmt.Errorf("check paid contribution for fallback winner: %w", err)
		}
		if paidCount == 0 {
			continue
		}

		wonCount, err := r.auctionResultsCol.CountDocuments(ctx, bson.M{"fund_id": fundID, "winner_user_id": member.UserID})
		if err != nil {
			return "", fmt.Errorf("check prior winner for fallback winner: %w", err)
		}
		if wonCount > 0 {
			continue
		}

		score := 0
		var user struct {
			Credit struct {
				Score int `bson:"score"`
			} `bson:"credit"`
		}
		err = r.usersCol.FindOne(ctx, bson.M{"_id": member.UserID}, options.FindOne().SetProjection(bson.M{"credit.score": 1})).Decode(&user)
		if err == nil {
			score = user.Credit.Score
		}

		candidates = append(candidates, candidate{UserID: member.UserID, Score: score})
	}
	if err := cursor.Err(); err != nil {
		return "", fmt.Errorf("iterate members for fallback winner: %w", err)
	}
	if len(candidates) == 0 {
		return "", ErrNoEligibleWinner
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].UserID < candidates[j].UserID
		}
		return candidates[i].Score > candidates[j].Score
	})

	return candidates[0].UserID, nil
}

func currentCycleNumber(startDate time.Time, durationMonths int, now time.Time) int {
	if durationMonths <= 0 {
		return 1
	}
	if now.Before(startDate) {
		return 1
	}

	monthsElapsed := (now.Year()-startDate.Year())*12 + int(now.Month()-startDate.Month())
	cycle := monthsElapsed + 1
	if cycle < 1 {
		cycle = 1
	}
	if cycle > durationMonths {
		cycle = durationMonths
	}
	return cycle
}
