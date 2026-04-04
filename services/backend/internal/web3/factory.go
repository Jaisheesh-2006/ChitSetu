package web3

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const factoryABI = `[
  {
    "inputs": [
      {"internalType":"address","name":"token","type":"address"},
      {"internalType":"uint256","name":"members","type":"uint256"},
      {"internalType":"uint256","name":"contribution","type":"uint256"},
      {"internalType":"string","name":"name","type":"string"}
    ],
    "name": "createFund",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "anonymous": false,
    "inputs": [
      {"indexed":true,"internalType":"address","name":"fund","type":"address"},
      {"indexed":true,"internalType":"address","name":"creator","type":"address"},
      {"indexed":false,"internalType":"string","name":"name","type":"string"}
    ],
    "name": "FundCreated",
    "type": "event"
  }
]`

type FactoryContract struct {
	address common.Address
	abiDef  abi.ABI
	bound   *bind.BoundContract
	client  *Client
}

func NewFactoryContract(client *Client, factoryAddress string) (*FactoryContract, error) {
	if !common.IsHexAddress(strings.TrimSpace(factoryAddress)) {
		return nil, fmt.Errorf("invalid factory address")
	}

	parsedABI, err := abi.JSON(strings.NewReader(factoryABI))
	if err != nil {
		return nil, fmt.Errorf("parse factory abi: %w", err)
	}

	addr := common.HexToAddress(strings.TrimSpace(factoryAddress))
	bound := bind.NewBoundContract(addr, parsedABI, client.EthClient, client.EthClient, client.EthClient)

	return &FactoryContract{
		address: addr,
		abiDef:  parsedABI,
		bound:   bound,
		client:  client,
	}, nil
}

// CreateFundWithOpts deploys a new ChitFund child contract via the Factory with provided options.
func (f *FactoryContract) CreateFundWithOpts(
	opts *bind.TransactOpts,
	tokenAddress string,
	members uint64,
	contribution *big.Int,
	name string,
) (*types.Transaction, error) {
	return f.bound.Transact(opts, "createFund",
		common.HexToAddress(tokenAddress),
		new(big.Int).SetUint64(members),
		contribution,
		name,
	)
}

// ExtractFundAddressFromReceipt parses a FundCreated event from a transaction receipt.
func (f *FactoryContract) ExtractFundAddressFromReceipt(receipt *types.Receipt) (string, error) {
	eventID := f.abiDef.Events["FundCreated"].ID

	for _, logLine := range receipt.Logs {
		if len(logLine.Topics) > 0 && logLine.Topics[0] == eventID {
			// Topics[1] is the indexed "fund" address
			if len(logLine.Topics) >= 2 {
				fundAddr := common.BytesToAddress(logLine.Topics[1].Bytes())
				return fundAddr.Hex(), nil
			}
		}
	}

	// Debugging for Sepolia: List all topics to find where it went wrong
	fmt.Printf("DEBUG Factory.Extract: FundCreated event not found in %d logs.\n", len(receipt.Logs))
	for i, l := range receipt.Logs {
		fmt.Printf(" [Log #%d] Address: %s\n", i, l.Address.Hex())
		for ti, topic := range l.Topics {
			fmt.Printf("   Topic[%d]: %s\n", ti, topic.Hex())
		}
	}
	if len(receipt.Logs) > 0 {
		fmt.Printf(" Target EventID: %s\n", eventID.Hex())
	}

	return "", fmt.Errorf("FundCreated event not found in receipt (checked %d logs)", len(receipt.Logs))
}
