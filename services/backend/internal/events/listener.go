package events

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Listener watches for smart contract events
type Listener struct {
	client   *ethclient.Client
	contract common.Address
	abi      abi.ABI
}

// Initialize listener
func NewListener(client *ethclient.Client, contractAddressHex string) (*Listener, error) {

	abiStr := `[{
		"anonymous":false,
		"inputs":[
			{"indexed":true,"name":"round","type":"uint256"},
			{"indexed":true,"name":"winner","type":"address"},
			{"indexed":false,"name":"payout","type":"uint256"},
			{"indexed":false,"name":"discount","type":"uint256"}
		],
		"name":"AuctionWinner",
		"type":"event"
	}]`

	parsedABI, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		return nil, err
	}

	return &Listener{
		client:   client,
		contract: common.HexToAddress(contractAddressHex),
		abi:      parsedABI,
	}, nil
}

// Start listening
func (l *Listener) Start(ctx context.Context) {
	go func() {
		log.Println("Starting Web3 Event Listener:", l.contract.Hex())

		logs := make(chan types.Log, 32)

		query := ethereum.FilterQuery{
			Addresses: []common.Address{l.contract},
			Topics: [][]common.Hash{
				{l.abi.Events["AuctionWinner"].ID},
			},
		}

		for {
			// Check context before subscribing
			select {
			case <-ctx.Done():
				log.Println("Listener stopped")
				return
			default:
			}

			// Subscribe once — only retry if this fails or drops
			sub, err := l.client.SubscribeFilterLogs(ctx, query, logs)
			if err != nil {
				log.Println("Subscription failed, retrying in 5s:", err)
				time.Sleep(5 * time.Second)
				continue
			}

			log.Println("Web3 event subscription active")

			// Process events until subscription dies or context cancels
		eventLoop:
			for {
				select {
				case <-ctx.Done():
					sub.Unsubscribe()
					log.Println("Listener stopped")
					return

				case err := <-sub.Err():
					log.Println("Subscription dropped, reconnecting in 5s:", err)
					sub.Unsubscribe()
					time.Sleep(5 * time.Second)
					break eventLoop // restart outer loop to re-subscribe once

				case vLog := <-logs:
					l.handleAuctionWinner(vLog)
				}
			}
		}
	}()
}

// Handle AuctionWinner event
func (l *Listener) handleAuctionWinner(vLog types.Log) {

	// Indexed params
	round := new(big.Int).SetBytes(vLog.Topics[1].Bytes())
	winnerAddr := common.BytesToAddress(vLog.Topics[2].Bytes())

	// Non-indexed data
	type AuctionWinnerData struct {
		Payout   *big.Int
		Discount *big.Int
	}

	var data AuctionWinnerData

	err := l.abi.UnpackIntoInterface(&data, "AuctionWinner", vLog.Data)
	if err != nil {
		log.Println("Failed to decode event:", err)
		return
	}

	// Convert tokens → INR
	weiMultiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	fiatAmount := new(big.Int).Div(data.Payout, weiMultiplier)

	log.Printf("AUCTION EVENT DETECTED\n")
	log.Printf("Round: %s\n", round.String())
	log.Printf("Winner: %s\n", winnerAddr.Hex())
	log.Printf("Payout (tokens): %s\n", data.Payout.String())
	log.Printf("Discount: %s\n", data.Discount.String())
	log.Printf("INR Equivalent: %s\n", fiatAmount.String())

	// Trigger payout
	triggerFiatPayout(winnerAddr.Hex(), fiatAmount)
}

// Mock payout function
func triggerFiatPayout(wallet string, amount *big.Int) {

	fmt.Printf("MOCK UPI TRANSFER: ₹%s → Wallet %s\n", amount.String(), wallet)
}
