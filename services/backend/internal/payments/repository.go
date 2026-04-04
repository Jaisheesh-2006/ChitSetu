package payments

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository struct {
	db             *mongo.Database
	contribCol     *mongo.Collection
	fundsCol       *mongo.Collection
	membershipsCol *mongo.Collection
	usersCol       *mongo.Collection
	sessionsCol    *mongo.Collection
	ordersCol      *mongo.Collection
}

type ContributionDue struct {
	ContributionID string    `bson:"_id"`
	FundID         string    `bson:"fund_id"`
	UserID         string    `bson:"user_id"`
	CycleNumber    int       `bson:"cycle_number"`
	AmountDue      float64   `bson:"amount_due"`
	DueDate        time.Time `bson:"due_date"`
	Status         string    `bson:"status"`
	Email          string
}

type PaymentSession struct {
	ID             string    `bson:"_id"`
	ContributionID string    `bson:"contribution_id"`
	UserID         string    `bson:"user_id"`
	AmountDue      float64   `bson:"amount_due"`
	Status         string    `bson:"status"`
	ExpiresAt      time.Time `bson:"expires_at"`
	FundID         string
	CycleNumber    int
	DueDate        time.Time
	ContributionSt string
}

type PaidContribution struct {
	ID               string  `bson:"_id"`
	FundID           string  `bson:"fund_id"`
	UserID           string  `bson:"user_id"`
	CycleNumber      int     `bson:"cycle_number"`
	AmountDue        float64 `bson:"amount_due"`
	Status           string  `bson:"status"`
	PaymentRef       string  `bson:"payment_ref,omitempty"`
	BlockchainTxHash string  `bson:"blockchain_tx_hash,omitempty"`
	BlockchainStatus string  `bson:"blockchain_status,omitempty"`
}

type fundDocument struct {
	ID                  string    `bson:"_id"`
	DurationMonths      int       `bson:"duration_months"`
	MonthlyContribution float64   `bson:"monthly_contribution"`
	StartDate           time.Time `bson:"start_date"`
}

func NewRepository(db *mongo.Database) *Repository {
	return &Repository{
		db:             db,
		contribCol:     db.Collection("contributions"),
		fundsCol:       db.Collection("funds"),
		membershipsCol: db.Collection("fund_members"),
		usersCol:       db.Collection("users"),
		sessionsCol:    db.Collection("payment_sessions"),
		ordersCol:      db.Collection("payment_orders"),
	}
}

func (r *Repository) EnsureContributionsForActiveFunds(ctx context.Context) error {
	cur, err := r.fundsCol.Find(ctx, bson.M{"status": "active"})
	if err != nil {
		return fmt.Errorf("list active funds: %w", err)
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var fund fundDocument
		if err := cur.Decode(&fund); err != nil {
			return fmt.Errorf("decode active fund: %w", err)
		}

		membersCur, err := r.membershipsCol.Find(ctx, bson.M{"fund_id": fund.ID, "status": "active"})
		if err != nil {
			return fmt.Errorf("list active memberships: %w", err)
		}

		for membersCur.Next(ctx) {
			var membership struct {
				UserID string `bson:"user_id"`
			}
			if err := membersCur.Decode(&membership); err != nil {
				membersCur.Close(ctx)
				return fmt.Errorf("decode membership: %w", err)
			}

			for cycleNumber := 1; cycleNumber <= fund.DurationMonths; cycleNumber++ {
				dueDate := fund.StartDate.AddDate(0, cycleNumber-1, 0)
				_, err := r.contribCol.UpdateOne(
					ctx,
					bson.M{"fund_id": fund.ID, "user_id": membership.UserID, "cycle_number": cycleNumber},
					bson.M{"$setOnInsert": bson.M{
						"_id":          uuid.NewString(),
						"fund_id":      fund.ID,
						"user_id":      membership.UserID,
						"cycle_number": cycleNumber,
						"amount_due":   fund.MonthlyContribution,
						"due_date":     dueDate,
						"status":       "pending",
						"created_at":   time.Now(),
					}},
					options.Update().SetUpsert(true),
				)
				if err != nil {
					membersCur.Close(ctx)
					return fmt.Errorf("ensure contribution for fund %s user %s cycle %d: %w", fund.ID, membership.UserID, cycleNumber, err)
				}
			}
		}
		if err := membersCur.Close(ctx); err != nil {
			return fmt.Errorf("close memberships cursor: %w", err)
		}
	}
	if err := cur.Err(); err != nil {
		return fmt.Errorf("iterate active funds: %w", err)
	}
	return nil
}

func (r *Repository) ExpireSessions(ctx context.Context) error {
	_, err := r.sessionsCol.UpdateMany(
		ctx,
		bson.M{"status": "created", "expires_at": bson.M{"$lte": time.Now()}},
		bson.M{"$set": bson.M{"status": "expired", "updated_at": time.Now()}},
	)
	if err != nil {
		return fmt.Errorf("expire sessions: %w", err)
	}
	return nil
}

func (r *Repository) ListUpcomingPendingContributions(ctx context.Context) ([]ContributionDue, error) {
	today := time.Now().Truncate(24 * time.Hour)
	end := today.AddDate(0, 0, 3)
	cur, err := r.contribCol.Find(ctx, bson.M{
		"status":   "pending",
		"due_date": bson.M{"$gte": today, "$lt": end},
	})
	if err != nil {
		return nil, fmt.Errorf("list pending contributions: %w", err)
	}
	defer cur.Close(ctx)

	result := make([]ContributionDue, 0)
	for cur.Next(ctx) {
		var item ContributionDue
		if err := cur.Decode(&item); err != nil {
			return nil, fmt.Errorf("decode pending contribution: %w", err)
		}

		var user struct {
			Email string `bson:"email"`
		}
		err := r.usersCol.FindOne(ctx, bson.M{"_id": item.UserID}, options.FindOne().SetProjection(bson.M{"email": 1})).Decode(&user)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				continue
			}
			return nil, fmt.Errorf("lookup user email: %w", err)
		}
		item.Email = user.Email
		result = append(result, item)
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending contributions: %w", err)
	}
	return result, nil
}

func (r *Repository) GetPendingContributionInfo(ctx context.Context, fundID, userID string, cycleNumber int) (string, float64, time.Time, string, error) {
	var contribution struct {
		ID        string    `bson:"_id"`
		AmountDue float64   `bson:"amount_due"`
		DueDate   time.Time `bson:"due_date"`
		Status    string    `bson:"status"`
	}
	err := r.contribCol.FindOne(ctx, bson.M{
		"fund_id":      fundID,
		"user_id":      userID,
		"cycle_number": cycleNumber,
	}).Decode(&contribution)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", 0, time.Time{}, "", fmt.Errorf("contribution not found")
		}
		return "", 0, time.Time{}, "", fmt.Errorf("lookup contribution: %w", err)
	}

	return contribution.ID, contribution.AmountDue, contribution.DueDate, contribution.Status, nil
}

func (r *Repository) GetActiveSessionForContribution(ctx context.Context, contributionID string) (*PaymentSession, error) {
	var session PaymentSession
	err := r.sessionsCol.FindOne(
		ctx,
		bson.M{"contribution_id": contributionID, "status": "created", "expires_at": bson.M{"$gt": time.Now()}},
		options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("get active session: %w", err)
	}
	return &session, nil
}

func (r *Repository) CreatePaymentSession(ctx context.Context, contributionID, userID string, amountDue float64, expiresAt time.Time) (*PaymentSession, error) {
	now := time.Now()
	session := PaymentSession{
		ID:             uuid.NewString(),
		ContributionID: contributionID,
		UserID:         userID,
		AmountDue:      amountDue,
		Status:         "created",
		ExpiresAt:      expiresAt,
	}
	_, err := r.sessionsCol.InsertOne(ctx, bson.M{
		"_id":             session.ID,
		"contribution_id": session.ContributionID,
		"user_id":         session.UserID,
		"amount_due":      session.AmountDue,
		"status":          session.Status,
		"expires_at":      session.ExpiresAt,
		"created_at":      now,
		"updated_at":      now,
	})
	if err != nil {
		return nil, fmt.Errorf("create payment session: %w", err)
	}
	return &session, nil
}

func (r *Repository) GetSessionForUser(ctx context.Context, sessionID, userID string) (*PaymentSession, error) {
	var session PaymentSession
	err := r.sessionsCol.FindOne(ctx, bson.M{"_id": sessionID, "user_id": userID}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("get session for user: %w", err)
	}

	var contribution struct {
		FundID      string    `bson:"fund_id"`
		CycleNumber int       `bson:"cycle_number"`
		DueDate     time.Time `bson:"due_date"`
		Status      string    `bson:"status"`
	}
	err = r.contribCol.FindOne(ctx, bson.M{"_id": session.ContributionID}).Decode(&contribution)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("contribution not found")
		}
		return nil, fmt.Errorf("load contribution for session: %w", err)
	}

	session.FundID = contribution.FundID
	session.CycleNumber = contribution.CycleNumber
	session.DueDate = contribution.DueDate
	session.ContributionSt = contribution.Status
	return &session, nil
}

func (r *Repository) UpsertOrderForSession(ctx context.Context, sessionID, orderID string) error {
	now := time.Now()
	_, err := r.ordersCol.UpdateOne(
		ctx,
		bson.M{"session_id": sessionID},
		bson.M{"$set": bson.M{"order_id": orderID, "updated_at": now}, "$setOnInsert": bson.M{"_id": uuid.NewString(), "session_id": sessionID, "created_at": now}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("upsert order for session: %w", err)
	}
	return nil
}

func (r *Repository) GetOrderForSession(ctx context.Context, sessionID string) (string, error) {
	var order struct {
		OrderID string `bson:"order_id"`
	}
	err := r.ordersCol.FindOne(ctx, bson.M{"session_id": sessionID}).Decode(&order)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", fmt.Errorf("order mapping not found")
		}
		return "", fmt.Errorf("get order for session: %w", err)
	}
	return order.OrderID, nil
}

func (r *Repository) GetPaidContribution(ctx context.Context, id string) (*PaidContribution, error) {
	var res PaidContribution
	err := r.contribCol.FindOne(ctx, bson.M{"_id": id, "status": "paid"}).Decode(&res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *Repository) MarkPaymentVerified(ctx context.Context, userID, sessionID, paymentID string) (bool, *PaidContribution, error) {
	session, err := r.db.Client().StartSession()
	if err != nil {
		return false, nil, fmt.Errorf("start verify transaction session: %w", err)
	}
	defer session.EndSession(ctx)

	var alreadyPaid bool
	var paidContribution PaidContribution
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		var paymentSession struct {
			ID             string    `bson:"_id"`
			ContributionID string    `bson:"contribution_id"`
			ExpiresAt      time.Time `bson:"expires_at"`
			Status         string    `bson:"status"`
		}
		err := r.sessionsCol.FindOne(sc, bson.M{"_id": sessionID, "user_id": userID}).Decode(&paymentSession)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, fmt.Errorf("session not found")
			}
			return nil, fmt.Errorf("load payment session: %w", err)
		}

		var contribution struct {
			ID               string  `bson:"_id"`
			FundID           string  `bson:"fund_id"`
			UserID           string  `bson:"user_id"`
			CycleNumber      int     `bson:"cycle_number"`
			AmountDue        float64 `bson:"amount_due"`
			Status           string  `bson:"status"`
			PaymentRef       string  `bson:"payment_ref,omitempty"`
			BlockchainTxHash string  `bson:"blockchain_tx_hash,omitempty"`
			BlockchainStatus string  `bson:"blockchain_status,omitempty"`
		}
		err = r.contribCol.FindOne(sc, bson.M{"_id": paymentSession.ContributionID}).Decode(&contribution)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, fmt.Errorf("contribution not found")
			}
			return nil, fmt.Errorf("load contribution: %w", err)
		}

		if contribution.Status == "paid" {
			if _, err := r.sessionsCol.UpdateOne(sc, bson.M{"_id": sessionID}, bson.M{"$set": bson.M{"status": "used", "updated_at": time.Now()}}); err != nil {
				return nil, fmt.Errorf("mark used on already paid contribution: %w", err)
			}
			alreadyPaid = true
			paidContribution = PaidContribution{
				ID:               contribution.ID,
				FundID:           contribution.FundID,
				UserID:           contribution.UserID,
				CycleNumber:      contribution.CycleNumber,
				AmountDue:        contribution.AmountDue,
				Status:           contribution.Status,
				PaymentRef:       contribution.PaymentRef,
				BlockchainTxHash: contribution.BlockchainTxHash,
				BlockchainStatus: contribution.BlockchainStatus,
			}
			return nil, nil
		}

		if paymentSession.ExpiresAt.Before(time.Now()) {
			if _, err := r.sessionsCol.UpdateOne(sc, bson.M{"_id": sessionID}, bson.M{"$set": bson.M{"status": "expired", "updated_at": time.Now()}}); err != nil {
				return nil, fmt.Errorf("expire outdated session: %w", err)
			}
			return nil, fmt.Errorf("session expired")
		}
		if paymentSession.Status != "created" {
			return nil, fmt.Errorf("session is not payable")
		}

		upd, err := r.contribCol.UpdateOne(sc, bson.M{"_id": paymentSession.ContributionID}, bson.M{"$set": bson.M{"status": "paid", "payment_ref": paymentID, "paid_at": time.Now(), "updated_at": time.Now()}})
		if err != nil {
			return nil, fmt.Errorf("mark contribution paid: %w", err)
		}
		if upd.MatchedCount == 0 {
			return nil, fmt.Errorf("mark contribution paid affected 0 rows")
		}

		if _, err := r.sessionsCol.UpdateOne(sc, bson.M{"_id": sessionID}, bson.M{"$set": bson.M{"status": "used", "updated_at": time.Now()}}); err != nil {
			return nil, fmt.Errorf("mark session used: %w", err)
		}
		if _, err := r.ordersCol.UpdateOne(sc, bson.M{"session_id": sessionID}, bson.M{"$set": bson.M{"payment_id": paymentID, "verified_at": time.Now(), "updated_at": time.Now()}}); err != nil {
			return nil, fmt.Errorf("update payment order verification: %w", err)
		}

		alreadyPaid = false
		paidContribution = PaidContribution{
			ID:               contribution.ID,
			FundID:           contribution.FundID,
			UserID:           contribution.UserID,
			CycleNumber:      contribution.CycleNumber,
			AmountDue:        contribution.AmountDue,
			Status:           "paid",
			PaymentRef:       paymentID,
			BlockchainTxHash: contribution.BlockchainTxHash,
			BlockchainStatus: contribution.BlockchainStatus,
		}
		return nil, nil
	})
	if err != nil {
		return false, nil, err
	}

	return alreadyPaid, &paidContribution, nil
}

func (r *Repository) SetContributionBlockchainStatus(ctx context.Context, contributionID, status string) error {
	_, err := r.contribCol.UpdateOne(
		ctx,
		bson.M{"_id": contributionID},
		bson.M{"$set": bson.M{
			"blockchain_status": status,
			"updated_at":        time.Now(),
		}},
	)
	return err
}

func (r *Repository) SetContributionBlockchainPending(ctx context.Context, contributionID, txHash string) error {
	result, err := r.contribCol.UpdateOne(
		ctx,
		bson.M{"_id": contributionID},
		bson.M{"$set": bson.M{
			"blockchain_tx_hash":         txHash,
			"blockchain_status":          "pending",
			"blockchain_error":           "",
			"blockchain_last_attempt_at": time.Now(),
			"updated_at":                 time.Now(),
		}},
	)
	if err != nil {
		return fmt.Errorf("set blockchain pending: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("set blockchain pending affected 0 rows")
	}
	return nil
}

func (r *Repository) SetContributionBlockchainFailed(ctx context.Context, contributionID string, failure error) error {
	message := "web3 transaction failed"
	if failure != nil {
		message = failure.Error()
	}

	result, err := r.contribCol.UpdateOne(
		ctx,
		bson.M{"_id": contributionID},
		bson.M{
			"$set": bson.M{
				"blockchain_status":          "failed",
				"blockchain_error":           message,
				"blockchain_last_attempt_at": time.Now(),
				"updated_at":                 time.Now(),
			},
			"$inc": bson.M{"blockchain_retry_count": 1},
		},
	)
	if err != nil {
		return fmt.Errorf("set blockchain failed: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("set blockchain failed affected 0 rows")
	}
	return nil
}

func (r *Repository) SetContributionBlockchainConfirmed(ctx context.Context, contributionID string) error {
	result, err := r.contribCol.UpdateOne(
		ctx,
		bson.M{"_id": contributionID},
		bson.M{"$set": bson.M{
			"blockchain_status":          "confirmed",
			"blockchain_error":           "",
			"blockchain_confirmed_at":    time.Now(),
			"blockchain_last_attempt_at": time.Now(),
			"updated_at":                 time.Now(),
		}},
	)
	if err != nil {
		return fmt.Errorf("set blockchain confirmed: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("set blockchain confirmed affected 0 rows")
	}
	return nil
}

func (r *Repository) ListContributionsForBlockchainRetry(ctx context.Context, limit int64) ([]PaidContribution, error) {
	if limit <= 0 {
		limit = 100
	}

	cursor, err := r.contribCol.Find(
		ctx,
		bson.M{
			"status": "paid",
			"$or": []bson.M{
				{"blockchain_status": ""},
				{"blockchain_status": "failed"},
				{"blockchain_status": bson.M{"$exists": false}},
			},
		},
		options.Find().SetSort(bson.D{{Key: "updated_at", Value: 1}}).SetLimit(limit),
	)
	if err != nil {
		return nil, fmt.Errorf("list failed blockchain contributions: %w", err)
	}
	defer cursor.Close(ctx)

	results := make([]PaidContribution, 0)
	for cursor.Next(ctx) {
		var contribution PaidContribution
		if err := cursor.Decode(&contribution); err != nil {
			return nil, fmt.Errorf("decode failed blockchain contribution: %w", err)
		}
		results = append(results, contribution)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate failed blockchain contributions: %w", err)
	}

	return results, nil
}
func (r *Repository) GetFundContractAddress(ctx context.Context, fundID string) (string, error) {
	var fund struct {
		ContractAddress string `bson:"contract_address"`
	}
	err := r.fundsCol.FindOne(ctx, bson.M{"_id": fundID}).Decode(&fund)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", fmt.Errorf("fund not found: %s", fundID)
		}
		return "", fmt.Errorf("lookup fund contract address: %w", err)
	}
	return fund.ContractAddress, nil
}
func (r *Repository) ListContributionsByFundAndCycle(ctx context.Context, fundID string, cycle int) ([]PaidContribution, error) {
	cursor, err := r.contribCol.Find(ctx, bson.M{"fund_id": fundID, "cycle_number": cycle})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	results := make([]PaidContribution, 0)
	for cursor.Next(ctx) {
		var c PaidContribution
		if err := cursor.Decode(&c); err != nil {
			return nil, err
		}
		results = append(results, c)
	}
	return results, nil
}
