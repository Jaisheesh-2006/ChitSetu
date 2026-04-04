package web3

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Client struct {
	EthClient  *ethclient.Client
	Auth       *bind.TransactOpts
	ChainID    *big.Int
	privateKey *ecdsa.PrivateKey
	from       string
	mu         sync.Mutex
	lastNonce  uint64
	nonceInit  bool
}

func NewClient(rpcURL string, privateKey string) (*Client, error) {
	if rpcURL == "" {
		return nil, fmt.Errorf("web3 rpc url is required")
	}
	if privateKey == "" {
		return nil, fmt.Errorf("web3 private key is required")
	}

	ethClient, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial web3 rpc: %w", err)
	}

	chainID, err := ethClient.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("read chain id: %w", err)
	}

	privKey, err := crypto.HexToECDSA(trimHexPrefix(privateKey))
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("create transactor: %w", err)
	}
	auth.GasLimit = 0 // Let it estimate automatically

	from := crypto.PubkeyToAddress(privKey.PublicKey).Hex()

	return &Client{
		EthClient:  ethClient,
		Auth:       auth,
		ChainID:    chainID,
		privateKey: privKey,
		from:       from,
	}, nil
}

func (c *Client) PrepareTransactOpts(ctx context.Context) (*bind.TransactOpts, error) {
	return c.prepareTransactOptsWithKey(ctx, c.privateKey)
}

func (c *Client) PrepareTransactOptsFrom(ctx context.Context, privateKeyStr string) (*bind.TransactOpts, error) {
	privKey, err := crypto.HexToECDSA(trimHexPrefix(privateKeyStr))
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return c.prepareTransactOptsWithKey(ctx, privKey)
}

func (c *Client) prepareTransactOptsWithKey(ctx context.Context, privKey *ecdsa.PrivateKey) (*bind.TransactOpts, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.prepareTransactOptsWithKeyLocked(ctx, privKey)
}

func (c *Client) prepareTransactOptsWithKeyLocked(ctx context.Context, privKey *ecdsa.PrivateKey) (*bind.TransactOpts, error) {

	if c == nil || c.EthClient == nil || privKey == nil {
		return nil, fmt.Errorf("web3 client is not initialized or key is nil")
	}

	addr := crypto.PubkeyToAddress(privKey.PublicKey)

	var nonce uint64
	var err error

	// Only track nonce for the manager (client's default account)
	isManager := addr.Hex() == c.from

	if isManager {
		if !c.nonceInit {
			nonce, err = c.EthClient.PendingNonceAt(ctx, addr)
			if err != nil {
				return nil, fmt.Errorf("read pending nonce: %w", err)
			}
			c.lastNonce = nonce
			c.nonceInit = true
		} else {
			c.lastNonce++
			nonce = c.lastNonce
		}
	} else {
		// For others (users), just use network pending nonce
		nonce, err = c.EthClient.PendingNonceAt(ctx, addr)
		if err != nil {
			return nil, fmt.Errorf("read user pending nonce: %w", err)
		}
	}

	gasPrice, err := c.EthClient.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("suggest gas price: %w", err)
	}

	// 20% Gas Price buffer for higher transaction reliability
	gasPriceWithBuffer := new(big.Int).Mul(gasPrice, big.NewInt(120))
	gasPriceWithBuffer.Div(gasPriceWithBuffer, big.NewInt(100))

	auth, err := bind.NewKeyedTransactorWithChainID(privKey, c.ChainID)
	if err != nil {
		return nil, fmt.Errorf("create keyed transactor: %w", err)
	}

	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)
	auth.GasLimit = 0 // Auto-estimate
	auth.GasPrice = gasPriceWithBuffer

	return auth, nil
}

// SimulateCall attempts to run the transaction locally to see if it reverts.
func (c *Client) SimulateCall(ctx context.Context, msg ethereum.CallMsg) (string, error) {
	result, err := c.EthClient.CallContract(ctx, msg, nil)
	if err != nil {
		return "", fmt.Errorf("simulation failed: %w", err)
	}
	return string(result), nil
}

// SendTransaction wraps a contract call with a global mutex and nonce management.
// This is used for manager-signed transactions to prevent nonce collisions.
func (c *Client) SendTransaction(ctx context.Context, txFunc func(opts *bind.TransactOpts) (*types.Transaction, error)) (*types.Transaction, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	opts, err := c.prepareTransactOptsWithKeyLocked(ctx, c.privateKey)
	if err != nil {
		return nil, err
	}

	tx, err := txFunc(opts)
	if err != nil {
		// If transaction submission fails (e.g. invalid gas, underpriced),
		// we should reset the nonceInit to re-sync with network on next try.
		c.nonceInit = false
		return nil, err
	}

	// Double-check nonce sync if it's the manager
	if addr := crypto.PubkeyToAddress(c.privateKey.PublicKey); addr.Hex() == c.from {
		fmt.Printf("TX SUBMITTED [manager]: hash=%s nonce=%d\n", tx.Hash().Hex(), tx.Nonce())
	}

	return tx, nil
}

func trimHexPrefix(value string) string {
	if len(value) >= 2 && value[:2] == "0x" {
		return value[2:]
	}
	return value
}

func (c *Client) FromAddress() string {
	return c.from
}

// SendBaseToken sends MATIC/POL from the manager to a target address.
func (c *Client) SendBaseToken(ctx context.Context, to string, amountWei *big.Int) (*types.Transaction, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	opts, err := c.prepareTransactOptsWithKeyLocked(ctx, c.privateKey)
	if err != nil {
		return nil, err
	}

	tx := types.NewTransaction(opts.Nonce.Uint64(), common.HexToAddress(to), amountWei, opts.GasLimit, opts.GasPrice, nil)
	signedTx, err := opts.Signer(opts.From, tx)
	if err != nil {
		return nil, fmt.Errorf("sign base token tx: %w", err)
	}

	err = c.EthClient.SendTransaction(ctx, signedTx)
	if err != nil {
		c.nonceInit = false // Desync nonce on failure
		return nil, fmt.Errorf("send base token tx: %w", err)
	}

	return signedTx, nil
}
