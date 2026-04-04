package web3

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"math"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type ContributionPayload struct {
	FundID string `json:"fund_id"`
	UserID string `json:"user_id"`
	Cycle  uint64 `json:"cycle"`
	Amount uint64 `json:"amount"`
}

type AuctionPayload struct {
	FundID   string `json:"fund_id"`
	Cycle    uint64 `json:"cycle"`
	Winner   string `json:"winner"`
	Discount uint64 `json:"discount"`
}

type PayoutPayload struct {
	FundID string `json:"fund_id"`
	Cycle  uint64 `json:"cycle"`
	Winner string `json:"winner"`
	Amount uint64 `json:"amount"`
}

type Service struct {
	client   *Client
	contract *Contract
}

func NewServiceFromEnv() (*Service, error) {
	rpcURL := strings.TrimSpace(os.Getenv("WEB3_RPC_URL"))
	privateKey := strings.TrimSpace(os.Getenv("WEB3_PRIVATE_KEY"))
	contractAddress := strings.TrimSpace(os.Getenv("CONTRACT_ADDRESS"))
	abiJSON := strings.TrimSpace(os.Getenv("CONTRACT_ABI_JSON"))

	client, err := NewClient(rpcURL, privateKey)
	if err != nil {
		return nil, err
	}

	contract, err := NewContract(client, contractAddress, abiJSON)
	if err != nil {
		return nil, err
	}

	return &Service{client: client, contract: contract}, nil
}

// Removed outdated Record wrappers
func (s *Service) WaitForReceipt(ctx context.Context, txHash string) (*types.Receipt, error) {
	if !common.IsHexHash(strings.TrimSpace(txHash)) {
		return nil, fmt.Errorf("invalid tx hash")
	}

	hash := common.HexToHash(strings.TrimSpace(txHash))
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		receipt, err := s.client.EthClient.TransactionReceipt(ctx, hash)
		if err == nil {
			return receipt, nil
		}
		if !errors.Is(err, ethereum.NotFound) {
			return nil, fmt.Errorf("load transaction receipt: %w", err)
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait receipt timeout: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func (s *Service) WaitForReceiptWithTimeout(txHash string, timeout time.Duration) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.WaitForReceipt(ctx, txHash)
}

func INRToWei(amountINR float64) *big.Int {
	// 1 INR = 10^18 Wei
	// amountINR can be float (e.g. 100.50)
	// We convert to minor units (Paise) first to keep precision
	paise := int64(math.Round(amountINR * 100))

	// 1 INR = 100 Paise
	// 1 INR = 10^18 Wei
	// 1 Paise = 10^16 Wei
	multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(16), nil)
	return new(big.Int).Mul(big.NewInt(paise), multiplier)
}
