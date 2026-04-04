package web3

import (
	"context"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var errContractBindingsUnavailable = errors.New("contract bindings unavailable: generate or add ABI bindings")

type Contract struct {
	address common.Address
	abiDef  abi.ABI
}

func NewContract(_ *Client, address, _ string) (*Contract, error) {
	parsedABI, err := abi.JSON(strings.NewReader("[]"))
	if err != nil {
		return nil, err
	}
	return &Contract{address: common.HexToAddress(address), abiDef: parsedABI}, nil
}

func (c *Contract) JoinFundWithOpts(_ *bind.TransactOpts, _ string) (*types.Transaction, error) {
	return nil, errContractBindingsUnavailable
}

func (c *Contract) DepositContributionWithOpts(_ *bind.TransactOpts, _ string) (*types.Transaction, error) {
	return nil, errContractBindingsUnavailable
}

func (c *Contract) FinalizeAuctionWithOpts(_ *bind.TransactOpts, _ string, _ *big.Int) (*types.Transaction, error) {
	return nil, errContractBindingsUnavailable
}

func (c *Contract) IsMemberPaid(_ context.Context, _ string) (bool, error) {
	return false, errContractBindingsUnavailable
}

type TokenContract struct {
	address common.Address
}

func NewTokenContract(_ *Client, address, _ string) (*TokenContract, error) {
	return &TokenContract{address: common.HexToAddress(address)}, nil
}

func (t *TokenContract) MintWithOpts(_ *bind.TransactOpts, _ string, _ *big.Int) (*types.Transaction, error) {
	return nil, errContractBindingsUnavailable
}

func (t *TokenContract) ApproveWithKey(_ context.Context, _ string, _ string, _ *big.Int) (*types.Transaction, error) {
	return nil, errContractBindingsUnavailable
}

func (t *TokenContract) Approve(_ context.Context, _ string, _ *big.Int) (*types.Transaction, error) {
	return nil, errContractBindingsUnavailable
}

func (t *TokenContract) BalanceOf(_ context.Context, _ string) (*big.Int, error) {
	return big.NewInt(0), nil
}

type FactoryContract struct{}

func NewFactoryContract(_ *Client, _ string) (*FactoryContract, error) {
	return &FactoryContract{}, nil
}

func (f *FactoryContract) CreateFundWithOpts(_ *bind.TransactOpts, _ string, _ uint64, _ *big.Int, _ string) (*types.Transaction, error) {
	return nil, errContractBindingsUnavailable
}

func (f *FactoryContract) ExtractFundAddressFromReceipt(_ *types.Receipt) (string, error) {
	return "", errContractBindingsUnavailable
}

func INRToWei(amount float64) *big.Int {
	if amount <= 0 {
		return big.NewInt(0)
	}
	multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	result := new(big.Float).Mul(big.NewFloat(amount), new(big.Float).SetInt(multiplier))
	wei := new(big.Int)
	result.Int(wei)
	return wei
}
