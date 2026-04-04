package chitfund

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Jaisheesh-2006/ChitSetu/internal/wallet"
	"github.com/Jaisheesh-2006/ChitSetu/internal/web3"
	"github.com/Jaisheesh-2006/ChitSetu/internal/ws"
	"github.com/google/uuid"

	"github.com/robfig/cron/v3"
)

const resendFromAddress = "Acme <onboarding@resend.dev>"

type Service struct {
	repository      *Repository
	contractService *web3.ContractService
	walletService   *wallet.Service
	wsManager       *ws.Manager
	resendAPIKey    string
}

type CreateFundInput struct {
	Name                string
	Description         string
	TotalAmount         float64
	MonthlyContribution float64
	DurationMonths      int
	MaxMembers          int
	StartDate           time.Time
}

type AppError struct {
	StatusCode int
	Message    string
}

type ApplicationStatusResponse struct {
	Status string `json:"status"`
}

type CurrentCycleContributions struct {
	FundID             string             `json:"fund_id"`
	CycleNumber        int                `json:"cycle_number"`
	Contributions      []FundContribution `json:"contributions"`
	TotalDueAmount     float64            `json:"total_due_amount"`
	CurrentMemberCount int64              `json:"current_member_count"`
}

func (e *AppError) Error() string {
	return e.Message
}

func NewService(
	repository *Repository,
	contractService *web3.ContractService,
	walletService *wallet.Service,
	wsManager *ws.Manager,
) *Service {

	return &Service{
		repository:      repository,
		contractService: contractService,
		walletService:   walletService,
		wsManager:       wsManager,
		resendAPIKey:    strings.TrimSpace(os.Getenv("RESEND_API_KEY")),
	}
}

func (s *Service) CreateFund(ctx context.Context, creatorID string, input CreateFundInput) (*FundWithCount, error) {
	if strings.TrimSpace(creatorID) == "" {
		return nil, &AppError{StatusCode: http.StatusUnauthorized, Message: "missing authenticated user"}
	}
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Description) == "" {
		return nil, &AppError{StatusCode: http.StatusBadRequest, Message: "name and description are required"}
	}
	if input.TotalAmount <= 0 {
		return nil, &AppError{StatusCode: http.StatusBadRequest, Message: "total_amount must be greater than 0"}
	}
	if input.MonthlyContribution <= 0 {
		return nil, &AppError{StatusCode: http.StatusBadRequest, Message: "monthly_contribution must be greater than 0"}
	}
	if input.MaxMembers < 2 {
		return nil, &AppError{StatusCode: http.StatusBadRequest, Message: "max_members must be at least 2"}
	}
	if !input.StartDate.After(time.Now()) {
		return nil, &AppError{StatusCode: http.StatusBadRequest, Message: "start_date must be a future date"}
	}

	// Verify creator has KYC before allowing them to create and auto-join
	user, err := s.repository.GetFundUser(ctx, creatorID)
	if err != nil {
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to fetch creator user"}
	}
	if user == nil {
		return nil, &AppError{StatusCode: http.StatusBadRequest, Message: "user not found"}
	}
	if user.KYC == nil || (strings.TrimSpace(user.KYC.Status) != "verified" && strings.TrimSpace(user.KYC.Status) != "ml_ready") {
		return nil, &AppError{StatusCode: http.StatusBadRequest, Message: "kyc verification is required before creating a fund"}
	}
	if user.Credit == nil || user.Credit.Score <= 0 || user.Credit.CheckedAt.IsZero() {
		return nil, &AppError{StatusCode: http.StatusBadRequest, Message: "trust score is required before creating a fund"}
	}

	durationMonths := input.MaxMembers
	contractAddress := ""

	if s.contractService != nil {
		tokenAddress := strings.TrimSpace(os.Getenv("TOKEN_CONTRACT_ADDRESS"))
		if tokenAddress == "" {
			return nil, &AppError{StatusCode: http.StatusServiceUnavailable, Message: "token contract is not configured"}
		}

		_, deployedAddress, deployErr := s.contractService.CreateFund(
			ctx,
			tokenAddress,
			uint64(input.MaxMembers),
			web3.INRToWei(input.MonthlyContribution),
			strings.TrimSpace(input.Name),
		)
		if deployErr != nil {
			return nil, &AppError{StatusCode: http.StatusBadGateway, Message: fmt.Sprintf("failed to deploy fund contract: %v", deployErr)}
		}
		contractAddress = deployedAddress
	} else {
		contractAddress = "pending:web3_not_configured"
	}

	now := time.Now()
	id := uuid.NewString()

	created, err := s.repository.CreateFund(ctx, Fund{
		ID:                  id,
		Name:                strings.TrimSpace(input.Name),
		Description:         strings.TrimSpace(input.Description),
		TotalAmount:         input.TotalAmount,
		MonthlyContribution: input.MonthlyContribution,
		DurationMonths:      durationMonths,
		MaxMembers:          input.MaxMembers,
		ContractAddress:     contractAddress,
		Status:              "open",
		StartDate:           input.StartDate,
		CreatorID:           creatorID,
		CreatedAt:           now,
		UpdatedAt:           now,
	})
	if err != nil {
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to create fund"}
	}

	// Auto-enroll creator
	if err := s.repository.CreatePendingApplication(ctx, id, creatorID); err != nil {
		log.Printf("Failed to create pending application for auto-enroll: %v", err)
	} else {
		if _, err := s.repository.ApprovePendingMemberAndCreateContributions(ctx, *created, creatorID); err != nil {
			log.Printf("Failed to auto-approve creator: %v", err)
		}
	}

	return &FundWithCount{Fund: *created, CurrentMemberCount: 1}, nil
}

func (s *Service) StartUnderfilledFundCleanupScheduler(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Hour
	}

	run := func() {
		runCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := s.RunStartDateUnderfilledFundCleanup(runCtx); err != nil {
			log.Printf("underfilled fund cleanup failed: %v", err)
		}
	}

	go func() {
		run() // run once at startup

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}

func (s *Service) StartDailyUnderfilledFundCleanupCron() *cron.Cron {
	c := cron.New(cron.WithLocation(time.Local))
	_, err := c.AddFunc("10 0 * * *", func() {
		runCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := s.RunStartDateUnderfilledFundCleanup(runCtx); err != nil {
			log.Printf("underfilled fund daily cleanup cron failed: %v", err)
		}
	})
	if err != nil {
		log.Printf("failed to register underfilled fund cleanup cron: %v", err)
		return c
	}

	c.Start()
	return c
}

func (s *Service) RunStartDateUnderfilledFundCleanup(ctx context.Context) error {
	funds, err := s.repository.ListOpenFunds(ctx)
	if err != nil {
		return fmt.Errorf("list open funds for cleanup: %w", err)
	}

	today := time.Now()
	for _, fund := range funds {
		if !sameUTCDate(today, fund.StartDate) {
			continue
		}

		activeCount, err := s.repository.CountActiveMembers(ctx, fund.ID)
		if err != nil {
			log.Printf("cleanup: failed counting members for fund %s: %v", fund.ID, err)
			continue
		}
		if activeCount >= int64(fund.MaxMembers) {
			continue
		}

		members, err := s.repository.ListMembersByFund(ctx, fund.ID)
		if err != nil {
			log.Printf("cleanup: failed listing members for fund %s: %v", fund.ID, err)
			continue
		}

		userIDSet := make(map[string]struct{}, len(members))
		for _, member := range members {
			if member.UserID != "" {
				userIDSet[member.UserID] = struct{}{}
			}
		}

		userIDs := make([]string, 0, len(userIDSet))
		for userID := range userIDSet {
			userIDs = append(userIDs, userID)
		}

		usersByID, err := s.repository.ListUsersByIDs(ctx, userIDs)
		if err != nil {
			log.Printf("cleanup: failed fetching user emails for fund %s: %v", fund.ID, err)
			continue
		}

		reason := fmt.Sprintf(
			"Your fund '%s' was deleted because it did not reach required members by the start date (%d/%d joined).",
			fund.Name,
			activeCount,
			fund.MaxMembers,
		)

		notified := make(map[string]struct{})
		for _, user := range usersByID {
			email := strings.TrimSpace(user.Email)
			if email == "" {
				continue
			}
			if _, seen := notified[email]; seen {
				continue
			}
			notified[email] = struct{}{}

			if err := s.sendFundDeletedEmail(ctx, email, fund.Name, reason); err != nil {
				log.Printf("cleanup: failed sending fund deletion email to %s for fund %s: %v", email, fund.ID, err)
			}
		}

		if err := s.repository.DeleteFundCascade(ctx, fund.ID); err != nil {
			log.Printf("cleanup: failed deleting underfilled fund %s: %v", fund.ID, err)
			continue
		}

		log.Printf("cleanup: deleted underfilled fund %s (%d/%d active members)", fund.ID, activeCount, fund.MaxMembers)
	}

	return nil
}

func sameUTCDate(a, b time.Time) bool {
	au := a.UTC()
	bu := b.UTC()
	ay, am, ad := au.Date()
	by, bm, bd := bu.Date()
	return ay == by && am == bm && ad == bd
}

func (s *Service) GetApplicationStatus(ctx context.Context, userID, fundID string) (*ApplicationStatusResponse, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, &AppError{StatusCode: http.StatusUnauthorized, Message: "missing authenticated user"}
	}
	if strings.TrimSpace(fundID) == "" {
		return nil, &AppError{StatusCode: http.StatusBadRequest, Message: "fund id is required"}
	}

	status, err := s.repository.GetMembershipStatus(ctx, fundID, userID)
	if err != nil {
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to fetch application status"}
	}

	return &ApplicationStatusResponse{Status: status}, nil
}

func (s *Service) ListFunds(ctx context.Context) ([]FundWithCount, error) {
	funds, err := s.repository.ListOpenFunds(ctx)
	if err != nil {
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to list funds"}
	}

	result := make([]FundWithCount, 0, len(funds))
	for _, fund := range funds {
		count, err := s.repository.CountActiveMembers(ctx, fund.ID)
		if err != nil {
			return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to load fund member counts"}
		}
		result = append(result, FundWithCount{Fund: fund, CurrentMemberCount: count})
	}
	return result, nil
}

func (s *Service) GetFundDetails(ctx context.Context, fundID string) (*FundWithCount, error) {
	if strings.TrimSpace(fundID) == "" {
		return nil, &AppError{StatusCode: http.StatusBadRequest, Message: "fund id is required"}
	}

	fund, err := s.repository.GetFundByID(ctx, fundID)
	if err != nil {
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to fetch fund"}
	}
	if fund == nil {
		return nil, &AppError{StatusCode: http.StatusNotFound, Message: "fund not found"}
	}

	count, err := s.repository.CountActiveMembers(ctx, fundID)
	if err != nil {
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to count fund members"}
	}

	return &FundWithCount{Fund: *fund, CurrentMemberCount: count}, nil
}

func (s *Service) ApplyToFund(ctx context.Context, userID, fundID string) (map[string]string, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, &AppError{StatusCode: http.StatusUnauthorized, Message: "missing authenticated user"}
	}
	if strings.TrimSpace(fundID) == "" {
		return nil, &AppError{StatusCode: http.StatusBadRequest, Message: "fund id is required"}
	}

	fund, err := s.repository.GetFundByID(ctx, fundID)
	if err != nil {
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to fetch fund"}
	}
	if fund == nil || fund.Status != "open" {
		return nil, &AppError{StatusCode: http.StatusBadRequest, Message: "fund is not open for applications"}
	}

	user, err := s.repository.GetFundUser(ctx, userID)
	if err != nil {
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to fetch user"}
	}
	if user == nil {
		return nil, &AppError{StatusCode: http.StatusBadRequest, Message: "user not found"}
	}
	if user.KYC == nil || (strings.TrimSpace(user.KYC.Status) != "verified" && strings.TrimSpace(user.KYC.Status) != "ml_ready") {
		return nil, &AppError{StatusCode: http.StatusBadRequest, Message: "kyc verification is required before applying"}
	}
	if user.Credit == nil || user.Credit.Score <= 0 || user.Credit.CheckedAt.IsZero() {
		return nil, &AppError{StatusCode: http.StatusBadRequest, Message: "credit score is required before applying"}
	}

	exists, err := s.repository.HasMembershipRecord(ctx, fundID, userID)
	if err != nil {
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to validate membership"}
	}
	if exists {
		return nil, &AppError{StatusCode: http.StatusConflict, Message: "user already applied or is already a member"}
	}

	activeCount, err := s.repository.CountActiveMembers(ctx, fundID)
	if err != nil {
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to count active members"}
	}
	if activeCount >= int64(fund.MaxMembers) {
		return nil, &AppError{StatusCode: http.StatusBadRequest, Message: "fund is full"}
	}

	if err := s.repository.CreatePendingApplication(ctx, fundID, userID); err != nil {
		if err == ErrMembershipAlreadyExists {
			return nil, &AppError{StatusCode: http.StatusConflict, Message: "user already applied or is already a member"}
		}
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to submit application"}
	}

	return map[string]string{"message": "application submitted", "status": "pending"}, nil
}

func (s *Service) ApproveMember(ctx context.Context, organizerUserID, fundID, targetUserID string) (map[string]string, error) {
	if strings.TrimSpace(organizerUserID) == "" {
		return nil, &AppError{StatusCode: http.StatusUnauthorized, Message: "missing authenticated user"}
	}
	if strings.TrimSpace(targetUserID) == "" {
		return nil, &AppError{StatusCode: http.StatusBadRequest, Message: "user_id is required"}
	}

	fund, err := s.repository.GetFundByID(ctx, fundID)
	if err != nil {
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to fetch fund"}
	}
	if fund == nil {
		return nil, &AppError{StatusCode: http.StatusNotFound, Message: "fund not found"}
	}
	if fund.CreatorID != organizerUserID {
		return nil, &AppError{StatusCode: http.StatusForbidden, Message: "only the fund creator can approve members"}
	}

	updated, err := s.repository.ApprovePendingMemberAndCreateContributions(ctx, *fund, targetUserID)
	if err != nil {
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to approve member"}
	}
	if !updated {
		return nil, &AppError{StatusCode: http.StatusNotFound, Message: "pending application not found"}
	}

	if applicant, err := s.repository.GetFundUser(ctx, targetUserID); err != nil {
		log.Printf("failed to load applicant user for approval email: %v", err)
	} else if applicant != nil {
		if err := s.sendMembershipDecisionEmail(ctx, applicant.Email, fund.Name, "approved"); err != nil {
			log.Printf("failed to send approval email: %v", err)
		}
	}

	return map[string]string{"message": "member approved", "user_id": targetUserID}, nil
}

func (s *Service) RejectMember(ctx context.Context, organizerUserID, fundID, targetUserID string) (map[string]string, error) {
	if strings.TrimSpace(organizerUserID) == "" {
		return nil, &AppError{StatusCode: http.StatusUnauthorized, Message: "missing authenticated user"}
	}
	if strings.TrimSpace(targetUserID) == "" {
		return nil, &AppError{StatusCode: http.StatusBadRequest, Message: "user_id is required"}
	}

	fund, err := s.repository.GetFundByID(ctx, fundID)
	if err != nil {
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to fetch fund"}
	}
	if fund == nil {
		return nil, &AppError{StatusCode: http.StatusNotFound, Message: "fund not found"}
	}
	if fund.CreatorID != organizerUserID {
		return nil, &AppError{StatusCode: http.StatusForbidden, Message: "only the fund creator can reject members"}
	}

	updated, err := s.repository.RejectPendingMember(ctx, fundID, targetUserID)
	if err != nil {
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to reject member"}
	}
	if !updated {
		return nil, &AppError{StatusCode: http.StatusNotFound, Message: "pending application not found"}
	}

	if applicant, err := s.repository.GetFundUser(ctx, targetUserID); err != nil {
		log.Printf("failed to load applicant user for rejection email: %v", err)
	} else if applicant != nil {
		if err := s.sendMembershipDecisionEmail(ctx, applicant.Email, fund.Name, "rejected"); err != nil {
			log.Printf("failed to send rejection email: %v", err)
		}
	}

	return map[string]string{"message": "member rejected", "user_id": targetUserID}, nil
}

func (s *Service) sendMembershipDecisionEmail(ctx context.Context, toEmail, fundName, decision string) error {
	recipient := strings.TrimSpace(toEmail)
	if recipient == "" {
		return nil
	}

	decisionLabel := "Approved"
	if decision == "rejected" {
		decisionLabel = "Rejected"
	}
	subject := fmt.Sprintf("Application %s for %s", decisionLabel, fundName)
	textBody := fmt.Sprintf("Your application for the fund '%s' has been %s.", fundName, decision)

	if s.resendAPIKey == "" {
		log.Printf("resend not configured, membership decision for %s (%s): %s", recipient, fundName, decision)
		return nil
	}

	payload := map[string]interface{}{
		"from":    resendFromAddress,
		"to":      []string{recipient},
		"subject": subject,
		"text":    textBody,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal resend payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.resendAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("call resend api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("resend api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (s *Service) sendFundDeletedEmail(ctx context.Context, toEmail, fundName, reason string) error {
	recipient := strings.TrimSpace(toEmail)
	if recipient == "" {
		return nil
	}

	subject := fmt.Sprintf("Fund Deleted: %s", fundName)
	textBody := reason

	if s.resendAPIKey == "" {
		log.Printf("resend not configured, fund deletion notice for %s (%s): %s", recipient, fundName, reason)
		return nil
	}

	payload := map[string]interface{}{
		"from":    resendFromAddress,
		"to":      []string{recipient},
		"subject": subject,
		"text":    textBody,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal resend payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.resendAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("call resend api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("resend api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (s *Service) GetCurrentCycleContributions(ctx context.Context, requesterUserID, fundID string) (*CurrentCycleContributions, error) {
	if strings.TrimSpace(requesterUserID) == "" {
		return nil, &AppError{StatusCode: http.StatusUnauthorized, Message: "missing authenticated user"}
	}
	if strings.TrimSpace(fundID) == "" {
		return nil, &AppError{StatusCode: http.StatusBadRequest, Message: "fund id is required"}
	}

	fund, err := s.repository.GetFundByID(ctx, fundID)
	if err != nil {
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to fetch fund"}
	}
	if fund == nil {
		return nil, &AppError{StatusCode: http.StatusNotFound, Message: "fund not found"}
	}

	if requesterUserID != fund.CreatorID {
		isMember, err := s.repository.IsActiveMember(ctx, fundID, requesterUserID)
		if err != nil {
			return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to verify membership"}
		}
		if !isMember {
			return nil, &AppError{StatusCode: http.StatusForbidden, Message: "access denied for this fund"}
		}
	}

	now := time.Now()
	// Removed strict now.Before(startDate) exit so first cycle can be paid before midnight UTC
	monthsElapsed := (now.Year()-fund.StartDate.Year())*12 + int(now.Month()-fund.StartDate.Month())
	cycleNumber := monthsElapsed + 1
	if cycleNumber < 1 {
		cycleNumber = 1
	}
	if cycleNumber > fund.DurationMonths {
		cycleNumber = fund.DurationMonths
	}

	contributions, err := s.repository.ListContributionsByFundAndCycle(ctx, fundID, cycleNumber)
	if err != nil {
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to fetch contributions"}
	}

	var totalDue float64
	for _, contribution := range contributions {
		if contribution.Status != "paid" {
			totalDue += contribution.AmountDue
		}
	}

	memberCount, err := s.repository.CountActiveMembers(ctx, fundID)
	if err != nil {
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to count active members"}
	}

	return &CurrentCycleContributions{
		FundID:             fund.ID,
		CycleNumber:        cycleNumber,
		Contributions:      contributions,
		TotalDueAmount:     totalDue,
		CurrentMemberCount: memberCount,
	}, nil
}

func (s *Service) ListMembers(ctx context.Context, requesterUserID, fundID string) ([]MemberView, error) {
	fund, err := s.repository.GetFundByID(ctx, fundID)
	if err != nil {
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to fetch fund"}
	}
	if fund == nil {
		return nil, &AppError{StatusCode: http.StatusNotFound, Message: "fund not found"}
	}

	// Both the creator and any active member can view the member list
	if fund.CreatorID != requesterUserID {
		isMember, err := s.repository.HasActiveMembership(ctx, fundID, requesterUserID)
		if err != nil {
			return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to verify membership"}
		}
		if !isMember {
			return nil, &AppError{StatusCode: http.StatusForbidden, Message: "only fund members can view the member list"}
		}
	}

	members, err := s.repository.ListMembersByFund(ctx, fundID)
	if err != nil {
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to list fund members"}
	}

	userIDs := make([]string, 0, len(members))
	for _, member := range members {
		userIDs = append(userIDs, member.UserID)
	}
	usersByID, err := s.repository.ListUsersByIDs(ctx, userIDs)
	if err != nil {
		return nil, &AppError{StatusCode: http.StatusInternalServerError, Message: "failed to fetch member users"}
	}

	result := make([]MemberView, 0, len(members))
	for _, member := range members {
		user := usersByID[member.UserID]
		mv := MemberView{
			UserID:   member.UserID,
			FullName: user.FullName,
			Email:    user.Email,
			Status:   member.Status,
			JoinedAt: member.JoinedAt,
		}
		if user.Credit != nil {
			mv.TrustScore = user.Credit.Score
			mv.RiskBand = user.Credit.RiskCategory
			mv.DefaultProbability = user.Credit.DefaultProbability
		}
		result = append(result, mv)
	}
	return result, nil
}
