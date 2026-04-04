package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/crypto/argon2"
)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

const resendFromAddress = "Acme <onboarding@resend.dev>"

type Service struct {
	usersCol      *mongo.Collection
	sessionCol    *mongo.Collection
	jwtSecret     []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	resetTokenCol *mongo.Collection
	appBaseURL    string
	resendAPIKey  string
	httpClient    *http.Client
}
type userDocument struct {
	ID               string    `bson:"_id"`
	Email            string    `bson:"email"`
	PasswordHash     string    `bson:"password_hash"`
	GoogleSub        string    `bson:"google_sub,omitempty"`
	AuthProvider     string    `bson:"auth_provider"`
	ProfileCompleted bool      `bson:"profile_completed"`
	CreatedAt        time.Time `bson:"created_at"`
	UpdatedAt        time.Time `bson:"updated_at"`
}

type authSessionDocument struct {
	ID               string     `bson:"_id"`
	UserID           string     `bson:"user_id"`
	RefreshTokenHash string     `bson:"refresh_token_hash"`
	ExpiresAt        time.Time  `bson:"expires_at"`
	RevokedAt        *time.Time `bson:"revoked_at,omitempty"`
	CreatedAt        time.Time  `bson:"created_at"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresInSec int64  `json:"expires_in_sec"`
}

type passwordResetTokenDocument struct {
	UserID    string    `bson:"user_id"`
	TokenHash string    `bson:"token_hash"`
	ExpiresAt time.Time `bson:"expires_at"`
	CreatedAt time.Time `bson:"created_at"`
}

func NewService(db *mongo.Database) *Service {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-insecure-jwt-secret-change-me"
	}

	baseURL := strings.TrimSpace(os.Getenv("APP_BASE_URL"))
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}

	return &Service{
		usersCol:      db.Collection("users"),
		sessionCol:    db.Collection("auth_sessions"),
		resetTokenCol: db.Collection("password_reset_tokens"),
		jwtSecret:     []byte(secret),
		accessTTL:     15 * time.Minute,
		refreshTTL:    7 * 24 * time.Hour,
		appBaseURL:    strings.TrimRight(baseURL, "/"),
		resendAPIKey:  strings.TrimSpace(os.Getenv("RESEND_API_KEY")),
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}
func (s *Service) JWTSecret() []byte {
	out := make([]byte, len(s.jwtSecret))
	copy(out, s.jwtSecret)
	return out
}

func (s *Service) Register(ctx context.Context, email, password string) (*TokenPair, error) {
	if s == nil || s.usersCol == nil || s.sessionCol == nil {
		return nil, errors.New("auth service not initialized")
	}

	email = strings.TrimSpace(strings.ToLower(email))
	if err := validateCredentials(email, password); err != nil {
		return nil, err
	}

	hash, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	userID := uuid.NewString()
	_, err = s.usersCol.InsertOne(ctx, userDocument{
		ID:               userID,
		Email:            email,
		PasswordHash:     hash,
		ProfileCompleted: false,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	})
	if err != nil {
		if isDuplicateKeyError(err) {
			return nil, errors.New("user already exists")
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return s.IssueTokenPair(ctx, userID)

}

func validateCredentials(email, password string) error {
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	iterations := uint32(2)
	memory := uint32(64 * 1024)
	parallelism := uint8(2)
	keyLength := uint32(32)

	hash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLength)

	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		memory,
		iterations,
		parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func (s *Service) IssueTokenPair(ctx context.Context, userID string) (*TokenPair, error) {
	if s == nil || s.sessionCol == nil {
		return nil, errors.New("auth service not initialized")
	}

	now := time.Now()
	accessExp := now.Add(s.accessTTL)

	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(accessExp),
		Issuer:    "chitsetu-backend",
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := jwtToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	rawRefresh, err := randomToken(48)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	_, err = s.sessionCol.InsertOne(ctx, authSessionDocument{
		ID:               uuid.NewString(),
		UserID:           userID,
		RefreshTokenHash: tokenHash(rawRefresh),
		ExpiresAt:        now.Add(s.refreshTTL),
		CreatedAt:        now,
	})
	if err != nil {
		return nil, fmt.Errorf("persist refresh session: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		TokenType:    "Bearer",
		ExpiresInSec: int64(s.accessTTL.Seconds()),
	}, nil
}

func randomToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func isDuplicateKeyError(err error) bool {
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

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, errors.New("refresh token is required")
	}

	hash := tokenHash(refreshToken)

	var session authSessionDocument
	err := s.sessionCol.FindOne(ctx, bson.M{
		"refresh_token_hash": hash,
		"revoked_at":         bson.M{"$exists": false},
		"expires_at":         bson.M{"$gt": time.Now()},
	}).Decode(&session)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("invalid refresh token")
		}
		return nil, fmt.Errorf("lookup session: %w", err)
	}

	now := time.Now()
	if _, err := s.sessionCol.UpdateOne(ctx, bson.M{"_id": session.ID}, bson.M{"$set": bson.M{"revoked_at": now}}); err != nil {
		return nil, fmt.Errorf("revoke old session: %w", err)
	}

	return s.IssueTokenPair(ctx, session.UserID)
}

func (s *Service) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if !emailRegex.MatchString(email) {
		return nil, errors.New("invalid email format")
	}

	var user userDocument
	err := s.usersCol.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("invalid email or password")
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	ok, err := verifyPassword(password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return nil, errors.New("invalid email or password")
	}

	return s.IssueTokenPair(ctx, user.ID)
}

func verifyPassword(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, errors.New("invalid encoded hash format")
	}

	var memory uint32
	var iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, fmt.Errorf("parse hash params: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("decode salt: %w", err)
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("decode hash: %w", err)
	}

	recomputed := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(decodedHash)))
	return subtleCompare(decodedHash, recomputed), nil
}

func subtleCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := range a {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

func (s *Service) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	var user userDocument
	err := s.usersCol.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Don't reveal account existence, but also don't return a token for non-users
			return "", nil
		}
		return "", fmt.Errorf("find user: %w", err)
	}

	token, err := randomToken(32)
	if err != nil {
		return "", fmt.Errorf("generate reset token: %w", err)
	}

	_, err = s.resetTokenCol.UpdateOne(
		ctx,
		bson.M{"user_id": user.ID},
		bson.M{
			"$set": bson.M{
				"token_hash": tokenHash(token),
				"expires_at": time.Now().Add(1 * time.Hour),
				"created_at": time.Now(),
			},
		},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		return "", fmt.Errorf("store reset token: %w", err)
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.appBaseURL, token)
	log.Printf("Password reset link for %s: %s", email, resetLink)

	if s.resendAPIKey != "" {
		_ = s.sendPasswordResetEmail(email, resetLink)
	}

	return token, nil
}

func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	if len(newPassword) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	hash := tokenHash(token)
	var resetDoc passwordResetTokenDocument
	err := s.resetTokenCol.FindOne(ctx, bson.M{
		"token_hash": hash,
		"expires_at": bson.M{"$gt": time.Now()},
	}).Decode(&resetDoc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("invalid or expired reset token")
		}
		return fmt.Errorf("find reset token: %w", err)
	}

	newHash, err := hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	_, err = s.usersCol.UpdateOne(ctx, bson.M{"_id": resetDoc.UserID}, bson.M{
		"$set": bson.M{"password_hash": newHash, "updated_at": time.Now()},
	})
	if err != nil {
		return fmt.Errorf("update user password: %w", err)
	}

	_, _ = s.resetTokenCol.DeleteMany(ctx, bson.M{
		"$or": []bson.M{
			{"user_id": resetDoc.UserID},
			{"token_hash": hash},
		},
	})

	return nil
}

func (s *Service) sendPasswordResetEmail(toEmail, resetLink string) error {
	payload := map[string]any{
		"from":    resendFromAddress,
		"to":      []string{toEmail},
		"subject": "Reset Your ChitSetu Password",
		"html":    fmt.Sprintf("<p>You requested a password reset.</p><p>Click the link below to set a new password:</p><p><a href=\"%s\">Reset Password</a></p><p>This link will expire in 1 hour.</p>", resetLink),
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.resendAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
