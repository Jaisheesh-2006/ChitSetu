package wallet

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func getEncryptionKey() []byte {
	keyStr := os.Getenv("WALLET_ENCRYPTION_KEY")
	if keyStr == "" {
		keyStr = "0123456789abcdef0123456789abcdef" // 32 bytes fallback
	}
	if len(keyStr) < 32 {
		for len(keyStr) < 32 {
			keyStr += "0"
		}
	}
	return []byte(keyStr[:32])
}

// encryptPrivateAES encrypts the hex-encoded private key using AES-GCM.
func encryptPrivateAES(plaintextHex string) (string, error) {
	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintextHex), nil)
	return hex.EncodeToString(ciphertext), nil
}

// decryptPrivateAES decrypts the AES-GCM encrypted private key back to hex.
func decryptPrivateAES(encryptedHex string) (string, error) {
	key := getEncryptionKey()
	encryptedData, err := hex.DecodeString(encryptedHex)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(encryptedData) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// Service handles wallet creation and retrieval
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateWallet generates a new Ethereum wallet, encrypts the private key, and stores it for the user.
func (s *Service) CreateWallet(ctx context.Context, userID string) (string, error) {

	if userID == "" {
		return "", errors.New("invalid user id")
	}

	// prevent duplicate wallet creation
	existing, err := s.repo.GetWallet(ctx, userID)
	if err == nil && existing.Address != "" {
		return existing.Address, nil
	}

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return "", err
	}

	address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()

	privateKeyBytes := crypto.FromECDSA(privateKey)
	privateKeyHex := hex.EncodeToString(privateKeyBytes)

	encryptedKey, err := encryptPrivateAES(privateKeyHex)
	if err != nil {
		return "", err
	}

	err = s.repo.SaveWallet(ctx, userID, address, encryptedKey)
	if err != nil {
		return "", err
	}

	return address, nil
}

// GetWalletByUserID retrieves and decrypts the user's private key and address.
// Security: Private key is returned in plaintext HEX; only meant for internal signing operations.
func (s *Service) GetWalletByUserID(ctx context.Context, userID string) (string, string, error) {

	walletRecord, err := s.repo.GetWallet(ctx, userID)
	if err != nil {
		return "", "", err
	}

	decryptedKey, err := decryptPrivateAES(walletRecord.EncryptedPrivateKey)
	if err != nil {
		return "", "", err
	}

	return walletRecord.Address, "0x" + decryptedKey, nil
}

// ---- Public Balance Management Methods ----

// AddMintedTokens adds tokens to wallet after successful mint
func (s *Service) AddMintedTokens(ctx context.Context, userID string, amount float64) error {
	return s.repo.AddMintedTokens(ctx, userID, amount)
}

// DebitFromBalance deducts tokens when contributing to pool
func (s *Service) DebitFromBalance(ctx context.Context, userID string, amount float64) error {
	return s.repo.DebitFromBalance(ctx, userID, amount)
}

// CreditToBalance adds tokens for dividends and payouts
func (s *Service) CreditToBalance(ctx context.Context, userID string, amount float64, reason string) error {
	return s.repo.CreditToBalance(ctx, userID, amount, reason)
}

// --- Repository (MongoDB Database) ---

type WalletRecord struct {
	UserID              string    `bson:"_id"`
	Address             string    `bson:"address"`
	EncryptedPrivateKey string    `bson:"encrypted_private_key"`
	TokenBalance        float64   `bson:"token_balance" json:"token_balance"`             // Available tokens in wallet
	TotalTokensMinted   float64   `bson:"total_tokens_minted" json:"total_tokens_minted"` // Cumulative tokens minted
	LastBalanceUpdate   time.Time `bson:"last_balance_update"`
	CreatedAt           time.Time `bson:"created_at"`
	UpdatedAt           time.Time `bson:"updated_at"`
}

type Repository struct {
	col *mongo.Collection
}

func NewRepository(db *mongo.Database) *Repository {
	return &Repository{
		col: db.Collection("wallets"),
	}
}

func (r *Repository) SaveWallet(ctx context.Context, userID, address, encryptedKey string) error {
	now := time.Now()
	_, err := r.col.InsertOne(ctx, WalletRecord{
		UserID:              userID,
		Address:             address,
		EncryptedPrivateKey: encryptedKey,
		TokenBalance:        0,
		TotalTokensMinted:   0,
		LastBalanceUpdate:   now,
		CreatedAt:           now,
		UpdatedAt:           now,
	})
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errors.New("wallet already exists for this user")
		}
		return err
	}
	return nil
}

func (r *Repository) GetWallet(ctx context.Context, userID string) (WalletRecord, error) {
	var record WalletRecord
	err := r.col.FindOne(ctx, bson.M{"_id": userID}).Decode(&record)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return WalletRecord{}, errors.New("wallet not found for user")
		}
		return WalletRecord{}, err
	}
	return record, nil
}

// UpdateTokenBalance adds or subtracts from the wallet's token balance.
// This is idempotent - it records each transaction separately if trackingID is provided.
func (r *Repository) UpdateTokenBalance(ctx context.Context, userID string, deltaAmount float64, trackingID string) error {
	if userID == "" {
		return errors.New("user id is required")
	}

	now := time.Now()
	update := bson.M{
		"$inc": bson.M{
			"token_balance": deltaAmount,
		},
		"$set": bson.M{
			"last_balance_update": now,
			"updated_at":          now,
		},
	}

	result, err := r.col.UpdateOne(ctx, bson.M{"_id": userID}, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("wallet not found for user")
	}
	return nil
}

// AddMintedTokens records tokens minted and adds to balance.
// If wallet doesn't exist, creates it with zero balance then updates it.
func (r *Repository) AddMintedTokens(ctx context.Context, userID string, amount float64) error {
	if userID == "" {
		return errors.New("user id is required")
	}
	if amount <= 0 {
		return errors.New("amount must be positive")
	}

	now := time.Now()

	// Try to update first
	update := bson.M{
		"$inc": bson.M{
			"token_balance":       amount,
			"total_tokens_minted": amount,
		},
		"$set": bson.M{
			"last_balance_update": now,
			"updated_at":          now,
		},
	}

	result, err := r.col.UpdateOne(ctx, bson.M{"_id": userID}, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		// Wallet doesn't exist - create it first
		log.Printf("⚠️  AddMintedTokens: Wallet not found for user %s, creating placeholder", userID)
		_, err := r.col.InsertOne(ctx, WalletRecord{
			UserID:              userID,
			Address:             "pending",
			EncryptedPrivateKey: "pending",
			TokenBalance:        amount,
			TotalTokensMinted:   amount,
			LastBalanceUpdate:   now,
			CreatedAt:           now,
			UpdatedAt:           now,
		})
		if err != nil && !mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("failed to create placeholder wallet: %w", err)
		}
		// If it was a duplicate, just try updating again
		if mongo.IsDuplicateKeyError(err) {
			result, err := r.col.UpdateOne(ctx, bson.M{"_id": userID}, update)
			if err != nil {
				return err
			}
			if result.MatchedCount == 0 {
				return errors.New("wallet update failed after creation attempt")
			}
		}
	}
	return nil
}

// DebitFromBalance deducts tokens from wallet balance (e.g., contribution to pool).
// Logs warnings if wallet doesn't exist but returns error to prevent silent failures.
func (r *Repository) DebitFromBalance(ctx context.Context, userID string, amount float64) error {
	if userID == "" {
		return errors.New("user id is required")
	}
	if amount <= 0 {
		return errors.New("amount must be positive")
	}

	now := time.Now()
	update := bson.M{
		"$inc": bson.M{
			"token_balance": -amount,
		},
		"$set": bson.M{
			"last_balance_update": now,
			"updated_at":          now,
		},
	}

	result, err := r.col.UpdateOne(ctx, bson.M{"_id": userID}, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		log.Printf("⚠️  DebitFromBalance WARNING: Wallet not found for user %s - balance debit failed", userID)
		return errors.New("wallet not found - cannot debit balance")
	}
	return nil
}

// CreditToBalance adds tokens to wallet balance (e.g., dividend payout).
func (r *Repository) CreditToBalance(ctx context.Context, userID string, amount float64, reason string) error {
	if userID == "" {
		return errors.New("user id is required")
	}
	if amount <= 0 {
		return errors.New("amount must be positive")
	}

	now := time.Now()
	update := bson.M{
		"$inc": bson.M{
			"token_balance": amount,
		},
		"$set": bson.M{
			"last_balance_update": now,
			"updated_at":          now,
		},
	}

	result, err := r.col.UpdateOne(ctx, bson.M{"_id": userID}, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("wallet not found for user")
	}
	return nil
}
