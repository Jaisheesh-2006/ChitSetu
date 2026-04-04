package web3

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type ContractService struct {
	contract *Contract // default contract from CHIT_CONTRACT_ADDRESS
	token    *TokenContract
	factory  *FactoryContract
	client   *Client
}

func (s *ContractService) Client() *Client {
	return s.client
}

// Constructor
func NewContractService(client *Client) (*ContractService, error) {
	if client == nil {
		return nil, fmt.Errorf("web3 client is nil")
	}

	factoryAddress := os.Getenv("FACTORY_CONTRACT_ADDRESS")
	var factory *FactoryContract
	if factoryAddress != "" {
		if err := ensureContractCode(client, factoryAddress, "factory"); err != nil {
			return nil, err
		}
		var err error
		factory, err = NewFactoryContract(client, factoryAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to init factory contract: %w", err)
		}
	}

	tokenAddress := os.Getenv("TOKEN_CONTRACT_ADDRESS")
	if tokenAddress == "" {
		return nil, fmt.Errorf("TOKEN_CONTRACT_ADDRESS not set")
	}
	if err := ensureContractCode(client, tokenAddress, "token"); err != nil {
		return nil, err
	}
	token, err := NewTokenContract(client, tokenAddress, "")
	if err != nil {
		return nil, fmt.Errorf("failed to init token contract: %w", err)
	}

	contractAddress := os.Getenv("CHIT_CONTRACT_ADDRESS")
	var contract *Contract
	if contractAddress != "" {
		if err := ensureContractCode(client, contractAddress, "default chit"); err != nil {
			return nil, err
		}
		contract, err = NewContract(client, contractAddress, "")
		if err != nil {
			return nil, fmt.Errorf("failed to init default contract: %w", err)
		}
	}

	return &ContractService{
		contract: contract,
		token:    token,
		factory:  factory,
		client:   client,
	}, nil
}

func ensureContractCode(client *Client, address, label string) error {
	if !common.IsHexAddress(strings.TrimSpace(address)) {
		return fmt.Errorf("invalid %s contract address: %s", label, address)
	}

	code, err := client.EthClient.CodeAt(context.Background(), common.HexToAddress(strings.TrimSpace(address)), nil)
	if err != nil {
		return fmt.Errorf("read code for %s contract: %w", label, err)
	}
	if len(code) == 0 {
		return fmt.Errorf("%s contract has no code at address %s on chain %s", label, address, client.ChainID.String())
	}

	return nil
}

func (s *ContractService) MintTokens(ctx context.Context, userAddress string, amount *big.Int) (string, error) {
	fmt.Printf("Minting %s tokens to user wallet: %s\n", amount.String(), userAddress)

	receipt, err := s.SendTransactionAndWait(ctx, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return s.token.MintWithOpts(opts, userAddress, amount)
	})
	if err != nil {
		return "", fmt.Errorf("failed to mint tokens: %w", err)
	}

	return receipt.TxHash.Hex(), nil
}

// ApproveFundForUser sets a specific user's allowance for a fund contract.
func (s *ContractService) ApproveFundForUser(ctx context.Context, privateKey string, fundAddress string) (string, error) {
	if fundAddress == "" {
		return "", fmt.Errorf("fund address is required")
	}

	// 2^256 - 1 (Infinite)
	maxUint256 := new(big.Int).Sub(
		new(big.Int).Lsh(big.NewInt(1), 256),
		big.NewInt(1),
	)

	tx, err := s.token.ApproveWithKey(ctx, privateKey, fundAddress, maxUint256)
	if err != nil {
		return "", fmt.Errorf("user approve failed for fund %s: %w", fundAddress, err)
	}

	return tx.Hash().Hex(), nil
}

// ApproveInfinite sets the manager's allowance for a fund contract.
func (s *ContractService) ApproveInfinite(fundAddress string) error {
	if fundAddress == "" {
		return fmt.Errorf("no fund address provided for manager approval")
	}

	// 2^256 - 1
	maxUint256 := new(big.Int).Sub(
		new(big.Int).Lsh(big.NewInt(1), 256),
		big.NewInt(1),
	)

	_, err := s.token.Approve(context.Background(), fundAddress, maxUint256)
	if err != nil {
		return fmt.Errorf("infinite approve for manager failed for fund %s: %w", fundAddress, err)
	}

	fmt.Printf("Infinite manager allowance granted to ChitFund: %s\n", fundAddress)
	return nil
}

// ApproveTokenWithKey signs an ERC20 Approve transaction from a custom private key.
func (s *ContractService) ApproveTokenWithKey(ctx context.Context, privateKeyStr, spenderAddress string, amount *big.Int) (string, error) {
	if s.token == nil {
		return "", fmt.Errorf("token contract not initialized")
	}

	tx, err := s.token.ApproveWithKey(ctx, privateKeyStr, spenderAddress, amount)
	if err != nil {
		return "", fmt.Errorf("token approve failed: %w", err)
	}

	return tx.Hash().Hex(), nil
}

// CreateFund deploys a new child ChitFund via the factory.
// Returns the tx hash and the new child contract address.
func (s *ContractService) CreateFund(
	ctx context.Context,
	tokenAddress string,
	members uint64,
	contribution *big.Int,
	name string,
) (txHash string, fundAddress string, err error) {
	if s.factory == nil {
		return "", "", fmt.Errorf("factory contract not initialized — set FACTORY_CONTRACT_ADDRESS in .env")
	}

	receipt, err := s.SendTransactionAndWait(ctx, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return s.factory.CreateFundWithOpts(opts, tokenAddress, members, contribution, name)
	})
	if err != nil {
		return "", "", fmt.Errorf("factory deployment failed: %w", err)
	}

	fundAddress, err = s.factory.ExtractFundAddressFromReceipt(receipt)
	if err != nil {
		return receipt.TxHash.Hex(), "", err
	}

	return receipt.TxHash.Hex(), fundAddress, nil
}

// contractFor returns a Contract binding for the given fund address.
// Falls back to the default contract if fundAddress is empty.
func (s *ContractService) contractFor(fundAddress string) (*Contract, error) {
	fundAddress = strings.TrimSpace(fundAddress)
	if fundAddress == "" || strings.HasPrefix(fundAddress, "pending:") {
		// If explicitly pending, we shouldn't fall back to a default that might be wrong (like the factory)
		if strings.HasPrefix(fundAddress, "pending:") {
			return nil, fmt.Errorf("fund deployment is still pending")
		}
		return s.contract, nil
	}
	return NewContract(s.client, fundAddress, "")
}

func (s *ContractService) RegisterMember(ctx context.Context, fundAddress, member string) (string, error) {
	c, err := s.contractFor(fundAddress)
	if err != nil {
		return "", fmt.Errorf("bind contract: %w", err)
	}

	receipt, err := s.SendTransactionAndWait(ctx, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return c.JoinFundWithOpts(opts, member)
	})
	if err != nil {
		return "", fmt.Errorf("RegisterMember failed: %w", err)
	}
	return receipt.TxHash.Hex(), nil
}

func (s *ContractService) JoinFund(ctx context.Context, fundAddress, member string) (string, error) {
	// Join the fund on behalf of the user (signed by manager)
	// Because of our "Auto-Approve" contract change, we no longer need the user to approve manually.
	return s.RegisterMember(ctx, fundAddress, member)
}

func (s *ContractService) DepositContribution(ctx context.Context, fundAddress, member string) (string, error) {
	c, err := s.contractFor(fundAddress)
	if err != nil {
		return "", fmt.Errorf("bind contract: %w", err)
	}

	receipt, err := s.SendTransactionAndWait(ctx, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return c.DepositContributionWithOpts(opts, member)
	})
	if err != nil {
		return "", fmt.Errorf("DepositContribution failed: %w", err)
	}
	return receipt.TxHash.Hex(), nil
}

func (s *ContractService) FinalizeAuction(ctx context.Context, fundAddress, winner string, discount *big.Int) (string, error) {
	c, err := s.contractFor(fundAddress)
	if err != nil {
		return "", fmt.Errorf("bind contract: %w", err)
	}

	receipt, err := s.SendTransactionAndWait(ctx, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return c.FinalizeAuctionWithOpts(opts, winner, discount)
	})
	if err != nil {
		return "", fmt.Errorf("FinalizeAuction failed: %w", err)
	}
	return receipt.TxHash.Hex(), nil
}

// SendTransactionAndWait submits a transaction and waits for it to be mined.
func (s *ContractService) SendTransactionAndWait(ctx context.Context, txFunc func(opts *bind.TransactOpts) (*types.Transaction, error)) (*types.Receipt, error) {
	tx, err := s.client.SendTransaction(ctx, txFunc)
	if err != nil {
		return nil, err
	}

	receipt, err := s.WaitForReceiptWithTimeout(tx.Hash().Hex(), 5*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("waiting for tx %s: %w", tx.Hash().Hex(), err)
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return receipt, fmt.Errorf("transaction %s reverted", tx.Hash().Hex())
	}

	return receipt, nil
}

func (s *ContractService) SimulateCall(ctx context.Context, fundAddress string, from common.Address, method string, args ...interface{}) (string, error) {
	c, err := s.contractFor(fundAddress)
	if err != nil {
		return "", err
	}
	data, err := c.abiDef.Pack(method, args...)
	if err != nil {
		return "", fmt.Errorf("abi pack failed: %w", err)
	}

	msg := ethereum.CallMsg{
		To:   &c.address,
		From: from,
		Data: data,
	}
	return s.client.SimulateCall(ctx, msg)
}

func (s *ContractService) IsMemberPaid(ctx context.Context, fundAddress, member string) (bool, error) {
	c, err := s.contractFor(fundAddress)
	if err != nil {
		return false, fmt.Errorf("bind contract: %w", err)
	}
	return c.IsMemberPaid(ctx, member)
}

func (s *ContractService) GetTransactionStatus(txHash string) (status string, blockNumber uint64, err error) {
	hash := common.HexToHash(txHash)
	receipt, err := s.client.EthClient.TransactionReceipt(context.Background(), hash)
	if err != nil {
		if err.Error() == "not found" {
			return "pending", 0, nil
		}
		return "", 0, err
	}

	if receipt.Status == types.ReceiptStatusSuccessful {
		return "success", receipt.BlockNumber.Uint64(), nil
	}
	return "failed", receipt.BlockNumber.Uint64(), nil
}

func (s *ContractService) WaitForReceiptWithTimeout(txHash string, timeout time.Duration) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	hash := common.HexToHash(txHash)

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for receipt")
		default:
			receipt, err := s.client.EthClient.TransactionReceipt(context.Background(), hash)

			if err == nil && receipt != nil {
				fmt.Println("TX Confirmed:", txHash)
				return receipt, nil
			}

			time.Sleep(2 * time.Second)
		}
	}
}

type TokenTransfer struct {
	TxHash    string  `json:"tx_hash"`
	From      string  `json:"from"`
	To        string  `json:"to"`
	Value     float64 `json:"value"`
	Type      string  `json:"type"` // "credit" or "debit"
	Timestamp uint64  `json:"timestamp"`
}

func (s *ContractService) GetTokenTransfers(ctx context.Context, address string) ([]TokenTransfer, error) {
	if s.token == nil {
		return nil, fmt.Errorf("token contract not initialized")
	}

	addr := common.HexToAddress(address)
	topics := [][]common.Hash{
		{common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")}, // Transfer event
	}

	// Filter for 'From'
	queryFrom := ethereum.FilterQuery{
		Addresses: []common.Address{s.token.address},
		Topics:    append(topics, []common.Hash{common.BytesToHash(addr.Bytes())}),
		FromBlock: big.NewInt(0),
	}

	// Filter for 'To'
	queryTo := ethereum.FilterQuery{
		Addresses: []common.Address{s.token.address},
		Topics:    append(topics, nil, []common.Hash{common.BytesToHash(addr.Bytes())}),
		FromBlock: big.NewInt(0),
	}

	logsFrom, err := s.client.EthClient.FilterLogs(ctx, queryFrom)
	if err != nil {
		return nil, err
	}
	logsTo, err := s.client.EthClient.FilterLogs(ctx, queryTo)
	if err != nil {
		return nil, err
	}
	allLogs := append(logsFrom, logsTo...)
	transfers := make([]TokenTransfer, 0)
	divider := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))

	// Cache for block timestamps to avoid redundant RPC calls
	timestampCache := make(map[uint64]uint64)

	for _, vLog := range allLogs {
		// Safety check: Ensure this is a standard Transfer event with 3 topics
		if len(vLog.Topics) < 3 {
			continue
		}

		from := common.BytesToAddress(vLog.Topics[1].Bytes())
		to := common.BytesToAddress(vLog.Topics[2].Bytes())

		// ERC20 Transfer events store the value in the Data field if it's not indexed
		val := new(big.Int).SetBytes(vLog.Data)

		fval := new(big.Float).SetInt(val)
		humanVal, _ := new(big.Float).Quo(fval, divider).Float64()

		tType := "credit"
		if from == addr {
			tType = "debit"
		}

		ts, ok := timestampCache[vLog.BlockNumber]
		if !ok {
			header, err := s.client.EthClient.HeaderByNumber(ctx, big.NewInt(int64(vLog.BlockNumber)))
			if err == nil {
				ts = header.Time
				timestampCache[vLog.BlockNumber] = ts
			}
		}

		transfers = append(transfers, TokenTransfer{
			TxHash:    vLog.TxHash.Hex(),
			From:      from.Hex(),
			To:        to.Hex(),
			Value:     humanVal,
			Type:      tType,
			Timestamp: ts,
		})
	}

	// Sort by recent (timestamp descending)
	for i := 0; i < len(transfers); i++ {
		for j := i + 1; j < len(transfers); j++ {
			if transfers[i].Timestamp < transfers[j].Timestamp {
				transfers[i], transfers[j] = transfers[j], transfers[i]
			}
		}
	}

	return transfers, nil
}

func (s *ContractService) GetTokenBalance(ctx context.Context, address string) (*big.Int, error) {
	if s.token == nil {
		return nil, fmt.Errorf("token contract not initialized")
	}
	return s.token.BalanceOf(ctx, address)
}
