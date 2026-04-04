package web3

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const defaultTokenABI = `[
  {
    "inputs": [
      {"internalType":"address","name":"to","type":"address"},
      {"internalType":"uint256","name":"amount","type":"uint256"}
    ],
    "name": "mint",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {"internalType":"address","name":"spender","type":"address"},
      {"internalType":"uint256","name":"amount","type":"uint256"}
    ],
    "name": "approve",
    "outputs": [
      {"internalType":"bool","name":"","type":"bool"}
    ],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {"internalType":"address","name":"account","type":"address"}
    ],
    "name": "balanceOf",
    "outputs": [
      {"internalType":"uint256","name":"","type":"uint256"}
    ],
    "stateMutability": "view",
    "type": "function"
  }
]`

type TokenContract struct {
	address common.Address
	abiDef  abi.ABI
	bound   *bind.BoundContract
	client  *Client
}

func NewTokenContract(client *Client, contractAddress string, abiJSON string) (*TokenContract, error) {
	if client == nil || client.EthClient == nil {
		return nil, fmt.Errorf("web3 client is not initialized")
	}
	if !common.IsHexAddress(strings.TrimSpace(contractAddress)) {
		return nil, fmt.Errorf("invalid token contract address")
	}
	if strings.TrimSpace(abiJSON) == "" {
		abiJSON = defaultTokenABI
	}

	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, fmt.Errorf("parse token abi: %w", err)
	}

	addr := common.HexToAddress(strings.TrimSpace(contractAddress))
	bound := bind.NewBoundContract(addr, parsedABI, client.EthClient, client.EthClient, client.EthClient)

	return &TokenContract{
		address: addr,
		abiDef:  parsedABI,
		bound:   bound,
		client:  client,
	}, nil
}

func (c *TokenContract) MintTokens(ctx context.Context, to string, amount *big.Int) (*types.Transaction, error) {
	auth, err := c.client.PrepareTransactOpts(ctx)
	if err != nil {
		return nil, err
	}
	return c.MintWithOpts(auth, to, amount)
}

func (c *TokenContract) MintWithOpts(opts *bind.TransactOpts, to string, amount *big.Int) (*types.Transaction, error) {
	return c.bound.Transact(opts, "mint", common.HexToAddress(to), amount)
}

func (c *TokenContract) Approve(ctx context.Context, spender string, amount *big.Int) (*types.Transaction, error) {
	auth, err := c.client.PrepareTransactOpts(ctx)
	if err != nil {
		return nil, err
	}
	return c.ApproveWithOpts(auth, spender, amount)
}

func (c *TokenContract) ApproveWithOpts(opts *bind.TransactOpts, spender string, amount *big.Int) (*types.Transaction, error) {
	return c.bound.Transact(opts, "approve", common.HexToAddress(spender), amount)
}

func (c *TokenContract) ApproveWithKey(ctx context.Context, privateKey string, spender string, amount *big.Int) (*types.Transaction, error) {
	auth, err := c.client.PrepareTransactOptsFrom(ctx, privateKey)
	if err != nil {
		return nil, err
	}
	return c.bound.Transact(auth, "approve", common.HexToAddress(spender), amount)
}

func (c *TokenContract) BalanceOf(ctx context.Context, account string) (*big.Int, error) {
	var out []interface{}
	err := c.bound.Call(&bind.CallOpts{Context: ctx}, &out, "balanceOf", common.HexToAddress(account))
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no output from balanceOf")
	}
	return out[0].(*big.Int), nil
}
