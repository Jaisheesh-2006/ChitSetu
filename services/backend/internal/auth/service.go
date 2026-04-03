package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/argon2"
)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type Service struct {
	usersCol    *mongo.Collection
	sessionCol  *mongo.Collection
	jwtSecret  []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	appBaseURL string
	httpClient *http.Client
}
type userDocument struct {
	ID               string    `bson:"_id"`
	Email            string    `bson:"email"`
	PasswordHash     string    `bson:"password_hash"`
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

func NewService(db *mongo.Database) *Service {
	if db == nil {
		return nil
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-insecure-jwt-secret-change-me"
	}

	baseURL := strings.TrimSpace(os.Getenv("APP_BASE_URL"))
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}

	return &Service{
		usersCol:   db.Collection("users"),
		sessionCol: db.Collection("auth_sessions"),
		jwtSecret:  []byte(secret),
		accessTTL:  15 * time.Minute,
		refreshTTL: 7 * 24 * time.Hour,
		appBaseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
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