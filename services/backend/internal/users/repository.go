package users

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const dbTimeout = 5 * time.Second

type UserProfileDocument struct {
	ID        string         `bson:"_id" json:"_id"`
	Email     string         `bson:"email" json:"email"`
	FullName  string         `bson:"full_name,omitempty" json:"full_name,omitempty"`
	AvatarURL string         `bson:"avatar_url,omitempty" json:"avatar_url,omitempty"`
	Profile   *ProfileFields `bson:"profile,omitempty" json:"profile,omitempty"`
	KYC       *KYCFields     `bson:"kyc,omitempty" json:"kyc,omitempty"`
	Credit    *CreditFields  `bson:"credit,omitempty" json:"credit,omitempty"`
}

type ProfileFields struct {
	PAN           string  `bson:"pan,omitempty" json:"pan,omitempty"`
	DOB           string  `bson:"dob,omitempty" json:"dob,omitempty"`
	MonthlyIncome float64 `bson:"monthly_income,omitempty" json:"monthly_income,omitempty"`
	Occupation    string  `bson:"occupation,omitempty" json:"occupation,omitempty"`
	Address       string  `bson:"address,omitempty" json:"address,omitempty"`
	City          string  `bson:"city,omitempty" json:"city,omitempty"`
	State         string  `bson:"state,omitempty" json:"state,omitempty"`
	Pincode       string  `bson:"pincode,omitempty" json:"pincode,omitempty"`
}

type KYCFields struct {
	Status      string     `bson:"status,omitempty" json:"status,omitempty"`
	VerifiedAt  *time.Time `bson:"verified_at,omitempty" json:"verified_at,omitempty"`
	BankAccount string     `bson:"bank_account,omitempty" json:"-"`
	IFSCCode    string     `bson:"ifsc_code,omitempty" json:"-"`
}

type CreditFields struct {
	Score              int         `bson:"score,omitempty" json:"score,omitempty"`
	RiskCategory       string      `bson:"risk_category,omitempty" json:"risk_category,omitempty"`
	CheckedAt          time.Time   `bson:"checked_at,omitempty" json:"checked_at,omitempty"`
	CibilScore         *int        `bson:"cibil_score,omitempty" json:"cibil_score,omitempty"`
	DefaultProbability float64     `bson:"default_probability,omitempty" json:"default_probability,omitempty"`
	SyntheticHistory   interface{} `bson:"synthetic_history,omitempty" json:"synthetic_history,omitempty"`
}

// KYCData is a read-only projection combining user_profiles and users collections.
type KYCData struct {
	KYCStatus          string
	PAN                string
	Age                int
	Income             float64
	EmploymentYears    int
	CibilScore         *int
	SyntheticHistory   interface{}
	TrustScore         int
	RiskBand           string
	DefaultProbability float64
	CheckedAt          time.Time
}

type Repository struct {
	profilesCol *mongo.Collection
	usersCol    *mongo.Collection
	fundsCol    *mongo.Collection
	membersCol  *mongo.Collection
	contribCol  *mongo.Collection
}

type UserFundMembership struct {
	FundID   string     `bson:"fund_id"`
	Status   string     `bson:"status"`
	JoinedAt *time.Time `bson:"joined_at,omitempty"`
}

type UserFund struct {
	ID                  string    `bson:"_id"`
	Name                string    `bson:"name"`
	TotalAmount         float64   `bson:"total_amount"`
	MonthlyContribution float64   `bson:"monthly_contribution"`
	DurationMonths      int       `bson:"duration_months"`
	Status              string    `bson:"status"`
	StartDate           time.Time `bson:"start_date"`
	CreatorID           string    `bson:"creator_id"`
}

type Contribution struct {
	FundID      string    `bson:"fund_id"`
	UserID      string    `bson:"user_id"`
	CycleNumber int       `bson:"cycle_number"`
	AmountDue   float64   `bson:"amount_due"`
	DueDate     time.Time `bson:"due_date"`
	Status      string    `bson:"status"`
	PaidAt      time.Time `bson:"paid_at,omitempty"`
	CreatedAt   time.Time `bson:"created_at"`
}

func NewRepository(db *mongo.Database) *Repository {
	return &Repository{
		profilesCol: db.Collection("user_profiles"),
		usersCol:    db.Collection("users"),
		fundsCol:    db.Collection("funds"),
		membersCol:  db.Collection("fund_members"),
		contribCol:  db.Collection("contributions"),
	}
}

func (r *Repository) currentKYCStatus(ctx context.Context, userID string) (string, error) {
	var userDoc struct {
		KYC *KYCFields `bson:"kyc,omitempty"`
	}

	err := r.usersCol.FindOne(
		ctx,
		bson.D{{Key: "_id", Value: userID}},
		options.FindOne().SetProjection(bson.D{{Key: "kyc", Value: 1}}),
	).Decode(&userDoc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", nil
		}
		return "", fmt.Errorf("get current kyc status: %w", err)
	}
	if userDoc.KYC == nil {
		return "", nil
	}
	return userDoc.KYC.Status, nil
}

func (r *Repository) syncProfileKYCStatus(ctx context.Context, userID string, status string) error {
	result, err := r.profilesCol.UpdateOne(
		ctx,
		bson.D{{Key: "user_id", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "kyc_status", Value: status},
			{Key: "updated_at", Value: time.Now()},
		}}},
	)
	if err != nil {
		return fmt.Errorf("sync profile kyc status: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("sync profile kyc status affected 0 rows")
	}
	return nil
}

func (r *Repository) UpsertProfile(ctx context.Context, profile ProfileInput) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	now := time.Now()
	kycStatus, err := r.currentKYCStatus(ctx, profile.UserID)
	if err != nil {
		return err
	}
	if kycStatus == "" {
		kycStatus = "pending"
	}
	_, err = r.profilesCol.UpdateOne(
		ctx,
		bson.D{{Key: "user_id", Value: profile.UserID}},
		bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "full_name", Value: profile.FullName},
				{Key: "age", Value: profile.Age},
				{Key: "phone_number", Value: profile.PhoneNumber},
				{Key: "pan_number", Value: profile.PANNumber},
				{Key: "monthly_income", Value: profile.MonthlyIncome},
				{Key: "employment_years", Value: profile.EmploymentYears},
				{Key: "kyc_status", Value: kycStatus},
				{Key: "updated_at", Value: now},
			}},
			{Key: "$setOnInsert", Value: bson.D{{Key: "user_id", Value: profile.UserID}, {Key: "created_at", Value: now}}},
		},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("upsert user profile: %w", err)
	}

	updateResult, err := r.usersCol.UpdateOne(
		ctx,
		bson.D{{Key: "_id", Value: profile.UserID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "profile_completed", Value: true},
			{Key: "updated_at", Value: now},
			{Key: "profile.pan", Value: profile.PANNumber},
			{Key: "profile.monthly_income", Value: profile.MonthlyIncome},
			{Key: "profile.occupation", Value: ""},
			{Key: "profile.address", Value: ""},
			{Key: "profile.city", Value: ""},
			{Key: "profile.state", Value: ""},
			{Key: "profile.pincode", Value: ""},
			{Key: "full_name", Value: profile.FullName},
		}}},
	)
	if err != nil {
		return fmt.Errorf("mark profile completed: %w", err)
	}
	if updateResult.MatchedCount == 0 {
		return fmt.Errorf("mark profile completed affected 0 rows")
	}

	// Initialise kyc.status to "pending" only if it has not been set yet.
	// This preserves any existing KYC progress when a user updates their profile.
	_, _ = r.usersCol.UpdateOne(
		ctx,
		bson.D{
			{Key: "_id", Value: profile.UserID},
			{Key: "kyc.status", Value: bson.D{{Key: "$exists", Value: false}}},
		},
		bson.D{{Key: "$set", Value: bson.D{{Key: "kyc.status", Value: "pending"}}}},
	)

	return nil
}

func (r *Repository) GetKYCData(ctx context.Context, userID string) (*KYCData, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	// Read profile fields
	var profileDoc struct {
		PAN             string  `bson:"pan_number"`
		Age             int     `bson:"age"`
		Income          float64 `bson:"monthly_income"`
		EmploymentYears int     `bson:"employment_years"`
	}
	err := r.profilesCol.FindOne(
		ctx,
		bson.D{{Key: "user_id", Value: userID}},
	).Decode(&profileDoc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("get kyc profile: %w", err)
	}

	// Read KYC and credit state
	var userDoc struct {
		KYC    *KYCFields    `bson:"kyc"`
		Credit *CreditFields `bson:"credit"`
	}
	err = r.usersCol.FindOne(
		ctx,
		bson.D{{Key: "_id", Value: userID}},
		options.FindOne().SetProjection(bson.D{
			{Key: "kyc", Value: 1},
			{Key: "credit", Value: 1},
		}),
	).Decode(&userDoc)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, fmt.Errorf("get kyc from users: %w", err)
	}

	data := &KYCData{
		PAN:             profileDoc.PAN,
		Age:             profileDoc.Age,
		Income:          profileDoc.Income,
		EmploymentYears: profileDoc.EmploymentYears,
	}
	if userDoc.KYC != nil {
		data.KYCStatus = userDoc.KYC.Status
	}
	if userDoc.Credit != nil {
		data.CibilScore = userDoc.Credit.CibilScore
		data.TrustScore = userDoc.Credit.Score
		data.RiskBand = userDoc.Credit.RiskCategory
		data.DefaultProbability = userDoc.Credit.DefaultProbability
		data.CheckedAt = userDoc.Credit.CheckedAt
		data.SyntheticHistory = userDoc.Credit.SyntheticHistory
	}
	return data, nil
}

func (r *Repository) StorePANVerified(ctx context.Context, userID string, cibilScore *int) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	now := time.Now().UTC()
	setFields := bson.D{
		{Key: "kyc.status", Value: "pan_verified"},
		{Key: "kyc.verified_at", Value: now},
	}
	if cibilScore != nil {
		setFields = append(setFields, bson.E{Key: "credit.cibil_score", Value: *cibilScore})
	}

	result, err := r.usersCol.UpdateOne(
		ctx,
		bson.D{{Key: "_id", Value: userID}, {Key: "kyc.status", Value: "pending"}},
		bson.D{{Key: "$set", Value: setFields}},
	)
	if err != nil {
		return fmt.Errorf("store pan verified: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("kyc transition failed: expected status=pending")
	}
	return r.syncProfileKYCStatus(ctx, userID, "pan_verified")
}

func (r *Repository) StoreSyntheticHistory(ctx context.Context, userID string, history interface{}, bankAccount string, ifscCode string) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	result, err := r.usersCol.UpdateOne(
		ctx,
		bson.D{{Key: "_id", Value: userID}, {Key: "kyc.status", Value: "pan_verified"}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "kyc.status", Value: "credit_fetched"},
			{Key: "kyc.bank_account", Value: bankAccount},
			{Key: "kyc.ifsc_code", Value: ifscCode},
			{Key: "credit.synthetic_history", Value: history},
		}}},
	)
	if err != nil {
		return fmt.Errorf("store synthetic history: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("kyc transition failed: expected status=pan_verified")
	}
	return r.syncProfileKYCStatus(ctx, userID, "credit_fetched")
}

// StoreTrustScore stores the ML-produced trust score and transitions
// kyc.status: credit_fetched → verified.
func (r *Repository) StoreTrustScore(
	ctx context.Context,
	userID string,
	score int,
	riskBand string,
	defaultProb float64,
) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	now := time.Now().UTC()
	result, err := r.usersCol.UpdateOne(
		ctx,
		bson.D{{Key: "_id", Value: userID}, {Key: "kyc.status", Value: "credit_fetched"}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "kyc.status", Value: "verified"},
			{Key: "credit.score", Value: score},
			{Key: "credit.risk_category", Value: riskBand},
			{Key: "credit.default_probability", Value: defaultProb},
			{Key: "credit.checked_at", Value: now},
		}}},
	)
	if err != nil {
		return fmt.Errorf("store trust score: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("kyc transition failed: expected status=credit_fetched")
	}
	return r.syncProfileKYCStatus(ctx, userID, "verified")
}

func (r *Repository) GetUserProfile(ctx context.Context, userID string) (*UserProfileDocument, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	var user UserProfileDocument
	err := r.usersCol.FindOne(
		ctx,
		bson.D{{Key: "_id", Value: userID}},
		options.FindOne().SetProjection(
			bson.D{
				{Key: "email", Value: 1},
				{Key: "full_name", Value: 1},
				{Key: "avatar_url", Value: 1},
				{Key: "profile", Value: 1},
				{Key: "kyc", Value: 1},
				{Key: "credit", Value: 1},
			},
		),
	).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("get user profile: %w", err)
	}

	return &user, nil
}

func (r *Repository) GetUserCredit(ctx context.Context, userID string) (*CreditFields, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	var user struct {
		Credit *CreditFields `bson:"credit,omitempty"`
	}
	err := r.usersCol.FindOne(
		ctx,
		bson.D{{Key: "_id", Value: userID}},
		options.FindOne().SetProjection(bson.D{{Key: "credit", Value: 1}}),
	).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("get user credit: %w", err)
	}

	return user.Credit, nil
}

func (r *Repository) ListActiveMembershipsByUser(ctx context.Context, userID string) ([]UserFundMembership, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	cursor, err := r.membersCol.Find(
		ctx,
		bson.D{{Key: "user_id", Value: userID}, {Key: "status", Value: "active"}},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find active memberships: %w", err)
	}
	defer cursor.Close(ctx)

	memberships := make([]UserFundMembership, 0)
	for cursor.Next(ctx) {
		var membership UserFundMembership
		if err := cursor.Decode(&membership); err != nil {
			return nil, fmt.Errorf("decode active membership: %w", err)
		}
		memberships = append(memberships, membership)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate active memberships: %w", err)
	}

	return memberships, nil
}

func (r *Repository) GetFundByID(ctx context.Context, fundID string) (*UserFund, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	var fund UserFund
	err := r.fundsCol.FindOne(
		ctx,
		bson.D{{Key: "_id", Value: fundID}},
		options.FindOne().SetProjection(bson.D{
			{Key: "name", Value: 1},
			{Key: "total_amount", Value: 1},
			{Key: "monthly_contribution", Value: 1},
			{Key: "duration_months", Value: 1},
			{Key: "status", Value: 1},
			{Key: "start_date", Value: 1},
			{Key: "creator_id", Value: 1},
		}),
	).Decode(&fund)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("find fund by id: %w", err)
	}

	return &fund, nil
}

func (r *Repository) CountActiveMembersByFund(ctx context.Context, fundID string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	count, err := r.membersCol.CountDocuments(
		ctx,
		bson.D{{Key: "fund_id", Value: fundID}, {Key: "status", Value: "active"}},
	)
	if err != nil {
		return 0, fmt.Errorf("count active members by fund: %w", err)
	}

	return count, nil
}

func (r *Repository) ListContributionsByUser(ctx context.Context, userID string) ([]Contribution, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	cursor, err := r.contribCol.Find(
		ctx,
		bson.D{{Key: "user_id", Value: userID}},
		options.Find().SetSort(bson.D{{Key: "due_date", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find contributions by user: %w", err)
	}
	defer cursor.Close(ctx)

	contributions := make([]Contribution, 0)
	for cursor.Next(ctx) {
		var contribution Contribution
		if err := cursor.Decode(&contribution); err != nil {
			return nil, fmt.Errorf("decode contribution by user: %w", err)
		}
		contributions = append(contributions, contribution)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate contributions by user: %w", err)
	}

	return contributions, nil
}
