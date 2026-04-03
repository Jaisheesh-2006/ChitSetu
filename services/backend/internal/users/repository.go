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
	Status     string     `bson:"status,omitempty" json:"status,omitempty"`
	VerifiedAt *time.Time `bson:"verified_at,omitempty" json:"verified_at,omitempty"`
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
}

func NewRepository(db *mongo.Database) *Repository {
	return &Repository{
		profilesCol: db.Collection("user_profiles"),
		usersCol:    db.Collection("users"),
	}
}
func (r *Repository) UpsertProfile(ctx context.Context, profile ProfileInput) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	now := time.Now()
	_, err := r.profilesCol.UpdateOne(
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
				{Key: "kyc_status", Value: "pending"},
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
	return nil
}

func (r *Repository) StoreSyntheticHistory(ctx context.Context, userID string, history interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	result, err := r.usersCol.UpdateOne(
		ctx,
		bson.D{{Key: "_id", Value: userID}, {Key: "kyc.status", Value: "pan_verified"}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "kyc.status", Value: "credit_fetched"},
			{Key: "credit.synthetic_history", Value: history},
		}}},
	)
	if err != nil {
		return fmt.Errorf("store synthetic history: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("kyc transition failed: expected status=pan_verified")
	}
	return nil
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
	return nil
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
