package web3

import (
	"context"
	"math/big"
)

type ContractService struct{}

func (s *ContractService) CreateFund(ctx context.Context, tokenAddr string, maxMembers uint64, contributionWei *big.Int, name string) (string, string, error) {
	return "", "", nil
}

func (s *ContractService) ApproveInfinite(contractAddr string) error {
	return nil
}

func (s *ContractService) RegisterMember(ctx context.Context, contractAddr, memberAddr string) (string, error) {
	return "", nil
}
