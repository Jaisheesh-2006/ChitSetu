package web3

import (
	"context"
	"fmt"
	"strings"

	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const defaultContractABI = `[
  {
    "inputs": [
      {"internalType":"address","name":"member","type":"address"}
    ],
    "name": "joinFund",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {"internalType":"address","name":"member","type":"address"}
    ],
    "name": "depositContribution",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {"internalType":"address","name":"winner","type":"address"},
      {"internalType":"uint256","name":"discount","type":"uint256"}
    ],
    "name": "finalizeAuction",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {"internalType":"address","name":"","type":"address"}
    ],
    "name": "hasPaid",
    "outputs": [
      {"internalType":"bool","name":"","type":"bool"}
    ],
    "stateMutability": "view",
    "type": "function"
  }
]`

type Contract struct {
	address common.Address
	abiDef  abi.ABI
	bound   *bind.BoundContract
	client  *Client
}

func NewContract(client *Client, contractAddress string, abiJSON string) (*Contract, error) {
	if client == nil || client.EthClient == nil {
		return nil, fmt.Errorf("web3 client is not initialized")
	}
	if !common.IsHexAddress(strings.TrimSpace(contractAddress)) {
		return nil, fmt.Errorf("invalid contract address")
	}
	if strings.TrimSpace(abiJSON) == "" {
		abiJSON = defaultContractABI
	}

	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, fmt.Errorf("parse contract abi: %w", err)
	}

	addr := common.HexToAddress(strings.TrimSpace(contractAddress))
	bound := bind.NewBoundContract(addr, parsedABI, client.EthClient, client.EthClient, client.EthClient)

	return &Contract{
		address: addr,
		abiDef:  parsedABI,
		bound:   bound,
		client:  client,
	}, nil
}

func (c *Contract) JoinFundWithOpts(opts *bind.TransactOpts, member string) (*types.Transaction, error) {
	return c.bound.Transact(opts, "joinFund", common.HexToAddress(member))
}

func (c *Contract) DepositContribution(ctx context.Context, member string) (*types.Transaction, error) {
	auth, err := c.client.PrepareTransactOpts(ctx)
	if err != nil {
		return nil, err
	}
	return c.DepositContributionWithOpts(auth, member)
}

func (c *Contract) DepositContributionWithOpts(opts *bind.TransactOpts, member string) (*types.Transaction, error) {
	return c.bound.Transact(opts, "depositContribution", common.HexToAddress(member))
}

func (c *Contract) FinalizeAuction(ctx context.Context, winner string, discount *big.Int) (*types.Transaction, error) {
	auth, err := c.client.PrepareTransactOpts(ctx)
	if err != nil {
		return nil, err
	}
	return c.FinalizeAuctionWithOpts(auth, winner, discount)
}

func (c *Contract) FinalizeAuctionWithOpts(opts *bind.TransactOpts, winner string, discount *big.Int) (*types.Transaction, error) {
	return c.bound.Transact(opts, "finalizeAuction", common.HexToAddress(winner), discount)
}

func (c *Contract) IsMemberPaid(ctx context.Context, member string) (bool, error) {
	var out []interface{}
	err := c.bound.Call(&bind.CallOpts{Context: ctx}, &out, "hasPaid", common.HexToAddress(member))
	if err != nil {
		return false, err
	}
	if len(out) == 0 {
		return false, fmt.Errorf("no output from hasPaid")
	}
	return out[0].(bool), nil
}
