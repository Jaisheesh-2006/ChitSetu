package auction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Jaisheesh-2006/ChitSetu/internal/ws"

	"github.com/Jaisheesh-2006/ChitSetu/internal/wallet"
	"github.com/Jaisheesh-2006/ChitSetu/internal/web3"
)

const (
	schedulerInterval = time.Second
	idleWindow        = 20 * time.Second
)

type Service struct {
	repo            *Repository
	wsManager       *ws.Manager
	httpClient      *http.Client
	payoutMode      string
	keyID           string
	keySecret       string
	accountNo       string
	fundAcctID      string
	contractService *web3.ContractService
	walletService   *wallet.Service
}

type PlaceBidInput struct {
	FundID    string
	UserID    string
	Increment int
}

func NewService(repo *Repository, wsManager *ws.Manager, contractService *web3.ContractService, walletService *wallet.Service) *Service {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("PAYOUT_MODE")))
	if mode == "" {
		mode = "simulate"
	}
	return &Service{
		repo:            repo,
		wsManager:       wsManager,
		httpClient:      &http.Client{Timeout: 20 * time.Second},
		payoutMode:      mode,
		keyID:           strings.TrimSpace(os.Getenv("RAZORPAY_KEY_ID")),
		keySecret:       strings.TrimSpace(os.Getenv("RAZORPAY_KEY_SECRET")),
		accountNo:       strings.TrimSpace(os.Getenv("RAZORPAY_X_ACCOUNT_NUMBER")),
		fundAcctID:      strings.TrimSpace(os.Getenv("RAZORPAY_PAYOUT_FUND_ACCOUNT_ID")),
		contractService: contractService,
		walletService:   walletService,
	}
}

func (s *Service) StartScheduler(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(schedulerInterval)
		defer ticker.Stop()

		for {
			if err := s.RunSchedulerTick(ctx); err != nil {
				log.Printf("auction scheduler tick failed: %v", err)
			}

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (s *Service) RunSchedulerTick(ctx context.Context) error {
	now := time.Now().UTC()

	liveAuctions, err := s.repo.ListLiveAuctions(ctx, 200)
	if err != nil {
		return err
	}

	for _, live := range liveAuctions {
		reference := live.CreatedAt
		if live.BiddingStartedAt != nil {
			reference = *live.BiddingStartedAt
		}
		if live.LastBidAt != nil {
			reference = *live.LastBidAt
		}
		if now.Before(reference.Add(idleWindow)) {
			continue
		}

		if _, _, err := s.FinalizeAuction(ctx, live.FundID, live.CycleNumber); err != nil {
			if err == ErrAuctionNotFinalized {
				continue
			}
			return err
		}
	}

	if err := s.RunPayoutRetryJob(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Service) StartAuction(ctx context.Context, fundID, creatorUserID string) (*AuctionSession, error) {
	if strings.TrimSpace(fundID) == "" {
		return nil, fmt.Errorf("fund id is required")
	}
	if strings.TrimSpace(creatorUserID) == "" {
		return nil, fmt.Errorf("creator user id is required")
	}

	session, err := s.repo.StartAuction(ctx, fundID, creatorUserID, 0)
	if err != nil {
		return nil, err
	}

	_ = s.wsManager.Broadcast(fundID, map[string]any{
		"type":          "auction_started",
		"fund_id":       session.FundID,
		"cycle_number":  session.CycleNumber,
		"current_price": session.CurrentPrice,
		"started_at":    session.CreatedAt,
	})

	return session, nil
}

func (s *Service) ActivateAuction(ctx context.Context, fundID, userID string) error {
	if strings.TrimSpace(fundID) == "" {
		return fmt.Errorf("fund id is required")
	}
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("user id is required")
	}

	// Verify requester is the fund creator
	fund, err := s.repo.GetFundByID(ctx, fundID)
	if err != nil {
		return fmt.Errorf("verify creator: %w", err)
	}
	if fund == nil {
		return fmt.Errorf("fund not found")
	}
	if fund.CreatorID != userID {
		return ErrAuctionStartDenied
	}

	activeMembers, err := s.repo.CountActiveMembers(ctx, fundID)
	if err != nil {
		return fmt.Errorf("count active members before activation: %w", err)
	}
	if activeMembers == 0 {
		return ErrAuctionParticipantsNotReady
	}
	if s.wsManager == nil {
		return fmt.Errorf("auction websocket manager is unavailable")
	}

	joinedParticipants := s.wsManager.AuctionParticipantCount(fundID)
	if joinedParticipants < activeMembers {
		return fmt.Errorf("%w (%d/%d joined)", ErrAuctionParticipantsNotReady, joinedParticipants, activeMembers)
	}

	err = s.repo.ActivateAuction(ctx, fundID)
	if err != nil {
		return err
	}

	_ = s.wsManager.Broadcast(fundID, map[string]any{
		"type":    "bidding_started",
		"fund_id": fundID,
	})

	return nil
}

func (s *Service) PlaceBid(ctx context.Context, input PlaceBidInput) (*Bid, *AuctionSession, error) {
	if strings.TrimSpace(input.FundID) == "" {
		return nil, nil, fmt.Errorf("fund id is required")
	}
	if strings.TrimSpace(input.UserID) == "" {
		return nil, nil, fmt.Errorf("user id is required")
	}
	if !isAllowedIncrement(input.Increment) {
		return nil, nil, ErrInvalidIncrement
	}

	incrementValue := float64(input.Increment)
	bid, updatedSession, err := s.repo.PlaceIncrementBid(ctx, input.FundID, input.UserID, incrementValue)
	if err != nil {
		return nil, nil, err
	}

	_ = s.wsManager.Broadcast(input.FundID, map[string]any{
		"type":             "new_bid",
		"fund_id":          input.FundID,
		"cycle_number":     updatedSession.CycleNumber,
		"user_id":          input.UserID,
		"best_bid_user_id": updatedSession.LastBidUserID,
		"increment":        input.Increment,
		"new_price":        updatedSession.CurrentPrice,
		"timestamp":        bid.CreatedAt,
	})

	return bid, updatedSession, nil
}

func (s *Service) GetAuction(ctx context.Context, fundID, userID string) (*AuctionSnapshot, error) {
	if strings.TrimSpace(fundID) == "" {
		return nil, fmt.Errorf("fund id is required")
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("user id is required")
	}

	allowed, err := s.repo.IsFundParticipant(ctx, fundID, userID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrNotFundMember
	}

	snapshot, err := s.repo.GetAuctionSnapshot(ctx, fundID)
	if err != nil {
		return nil, err
	}

	if snapshot != nil {
		// Fetch member profiles for transparency
		members, err := s.repo.GetMembersProfileInfo(ctx, fundID)
		if err == nil {
			// Calculate dividend for non-winners
			var dividend float64
			if snapshot.Result != nil && len(members) > 1 {
				dividend = snapshot.Result.WinningPrice / float64(len(members)-1)
			}

			// Enrich members info
			for i := range members {
				if snapshot.Result != nil && members[i].UserID == snapshot.Result.WinnerUserID {
					members[i].IsWinner = true
				} else {
					members[i].IsWinner = false
					members[i].Dividend = dividend
				}
			}
			snapshot.MembersInfo = members
		}

		if snapshot.Session != nil && snapshot.Session.Status == "live" {
			reference := snapshot.Session.CreatedAt
			if snapshot.Session.BiddingStartedAt != nil {
				reference = *snapshot.Session.BiddingStartedAt
			}
			if snapshot.Session.LastBidAt != nil {
				reference = *snapshot.Session.LastBidAt
			}

			remaining := int64(reference.Add(idleWindow).Sub(time.Now().UTC()).Seconds())
			if remaining < 0 {
				remaining = 0
			}
			snapshot.LiveCountdownSeconds = &remaining
		}
	}

	return snapshot, nil
}

func (s *Service) FinalizeAuction(ctx context.Context, fundID string, cycleNumber int) (*AuctionResult, bool, error) {
	result, created, err := s.repo.FinalizeAuction(ctx, fundID, cycleNumber)
	if err != nil {
		return nil, false, err
	}
	if !created || result == nil {
		return result, false, nil
	}

	_ = s.wsManager.Broadcast(fundID, map[string]any{
		"type":           "auction_ended",
		"fund_id":        fundID,
		"cycle_number":   cycleNumber,
		"winner_user_id": result.WinnerUserID,
		"winning_price":  result.WinningPrice,
		"payout":         result.PayoutAmount,
	})

	// Trigger wallet balance settlement
	go s.settleAuctionWalletBalances(fundID, cycleNumber, result)

	// Trigger On-Chain Settlement
	if s.contractService != nil && s.walletService != nil {
		fund, err := s.repo.GetFundByID(ctx, fundID)
		if err != nil || fund == nil || fund.ContractAddress == "" {
			return result, true, nil
		}
		winnerAddr, _, err := s.walletService.GetWalletByUserID(context.Background(), result.WinnerUserID)
		if err != nil {
			log.Printf("automated on-chain auction finalization failed: %v", err)
			return result, true, nil
		}
		// Calculate discount and scale to Wei using web3.INRToWei
		totalAmountRaw := float64(fund.MaxMembers) * fund.MonthlyContribution
		discountRaw := totalAmountRaw - result.PayoutAmount
		if discountRaw < 0 {
			discountRaw = 0
		}
		discountWei := web3.INRToWei(discountRaw)

		go func() {
			txHash, err := s.contractService.FinalizeAuction(context.Background(), fund.ContractAddress, winnerAddr, discountWei)
			if err != nil {
				log.Printf("automated on-chain auction finalization failed for fund %s: %v", fundID, err)
			} else {
				log.Printf("automated on-chain auction finalized: fund=%s tx=%s", fundID, txHash)
			}
		}()
	}

	if err := s.TriggerPayout(context.Background(), fundID, cycleNumber, result.WinnerUserID, result.PayoutAmount); err != nil {
		log.Printf("trigger payout failed for fund=%s cycle=%d: %v", fundID, cycleNumber, err)
	}

	return result, true, nil
}

func (s *Service) TriggerPayout(ctx context.Context, fundID string, cycleNumber int, winnerUserID string, amount float64) error {
	if strings.TrimSpace(fundID) == "" || strings.TrimSpace(winnerUserID) == "" {
		return fmt.Errorf("fund id and winner user id are required")
	}
	if cycleNumber <= 0 {
		return fmt.Errorf("cycle number must be greater than 0")
	}
	if amount <= 0 {
		return fmt.Errorf("payout amount must be greater than 0")
	}

	payout, err := s.repo.UpsertPayoutInitiated(ctx, fundID, cycleNumber, winnerUserID, amount, s.payoutMode)
	if err != nil {
		return err
	}
	if payout.Status == "completed" {
		return nil
	}

	go s.executePayoutAttempt(*payout)
	return nil
}

func (s *Service) RunPayoutRetryJob(ctx context.Context) error {
	failed, err := s.repo.ListFailedPayouts(ctx, 50)
	if err != nil {
		return err
	}
	for _, payout := range failed {
		s.executePayoutAttempt(payout)
	}
	return nil
}

// settleAuctionWalletBalances handles wallet token transfer after auction finalization.
// Winner gets payout, non-winners get dividend from the discount pool.
func (s *Service) settleAuctionWalletBalances(fundID string, cycleNumber int, result *AuctionResult) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if s.walletService == nil || s.repo == nil {
		log.Printf("settle wallet balances: wallet or repo service unavailable")
		return
	}

	// Get fund for member count
	fund, err := s.repo.GetFundByID(ctx, fundID)
	if err != nil {
		log.Printf("settle wallet balances fund=%s cycle=%d: failed to get fund: %v", fundID, cycleNumber, err)
		return
	}

	// Get fund members
	members, err := s.repo.GetFundMembers(ctx, fundID)
	if err != nil {
		log.Printf("settle wallet balances fund=%s cycle=%d: failed to get members: %v", fundID, cycleNumber, err)
		return
	}

	if len(members) == 0 {
		log.Printf("settle wallet balances fund=%s cycle=%d: no members found", fundID, cycleNumber)
		return
	}

	// Step 1: Credit winner payout
	log.Printf("settle wallet balances: crediting winner %s with %.2f tokens", result.WinnerUserID, result.PayoutAmount)
	if err := s.walletService.CreditToBalance(ctx, result.WinnerUserID, result.PayoutAmount, "auction_winner_payout"); err != nil {
		log.Printf("settle wallet balances: failed to credit winner: %v", err)
	}

	// Step 2: Calculate and credit dividends to non-winners
	if len(members) > 1 {
		discountAmount := (float64(fund.MaxMembers) * fund.MonthlyContribution) - result.PayoutAmount
		if discountAmount > 0 {
			numNonWinners := len(members) - 1
			dividendPerMember := discountAmount / float64(numNonWinners)

			log.Printf("settle wallet balances: distributing dividend %.2f to %d non-winners", dividendPerMember, numNonWinners)

			for _, member := range members {
				if member.UserID != result.WinnerUserID {
					if err := s.walletService.CreditToBalance(ctx, member.UserID, dividendPerMember, "auction_dividend"); err != nil {
						log.Printf("settle wallet balances: failed to credit dividend to %s: %v", member.UserID, err)
					} else {
						log.Printf("settle wallet balances: credited dividend %.2f to member %s", dividendPerMember, member.UserID)
					}
				}
			}
		}
	}

	log.Printf("settle wallet balances: completed for fund=%s cycle=%d", fundID, cycleNumber)
}

func (s *Service) executePayoutAttempt(payout PayoutRecord) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	referenceID, err := s.dispatchPayout(ctx, payout)
	if err != nil {
		if markErr := s.repo.MarkPayoutFailed(ctx, payout.ID, err); markErr != nil {
			log.Printf("mark payout failed error payout_id=%s err=%v", payout.ID, markErr)
		}
		return
	}

	if err := s.repo.MarkPayoutCompleted(ctx, payout.ID, referenceID); err != nil {
		log.Printf("mark payout completed error payout_id=%s err=%v", payout.ID, err)
	}
}

func (s *Service) dispatchPayout(ctx context.Context, payout PayoutRecord) (string, error) {
	if s.payoutMode != "razorpay" {
		return fmt.Sprintf("simulated-%s", payout.ID), nil
	}
	if s.keyID == "" || s.keySecret == "" || s.accountNo == "" || s.fundAcctID == "" {
		return "", fmt.Errorf("razorpay payout credentials are incomplete")
	}

	amountPaise := int64(math.Round(payout.Amount * 100))
	if amountPaise <= 0 {
		return "", fmt.Errorf("invalid payout amount")
	}

	payload := map[string]any{
		"account_number":  s.accountNo,
		"fund_account_id": s.fundAcctID,
		"amount":          amountPaise,
		"currency":        "INR",
		"mode":            "UPI",
		"purpose":         "payout",
		"reference_id":    fmt.Sprintf("%s-%d", payout.FundID, payout.CycleNumber),
		"notes": map[string]string{
			"winner_user_id": payout.WinnerUserID,
			"fund_id":        payout.FundID,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal razorpay payout payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.razorpay.com/v1/payouts", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create razorpay payout request: %w", err)
	}
	req.SetBasicAuth(s.keyID, s.keySecret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Account-Number", s.accountNo)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call razorpay payout api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("razorpay payout api returned status %d", resp.StatusCode)
	}

	var parsed struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode razorpay payout response: %w", err)
	}
	if strings.TrimSpace(parsed.ID) == "" {
		return "", fmt.Errorf("razorpay payout id missing")
	}

	return parsed.ID, nil
}

func isAllowedIncrement(increment int) bool {
	return increment == 10 || increment == 100 || increment == 200
}
