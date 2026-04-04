package chitfund

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repository *Repository
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
) *Service {

	return &Service{
		repository: repository,
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
	if input.DurationMonths <= 0 {
		return nil, &AppError{StatusCode: http.StatusBadRequest, Message: "duration_months must be greater than 0"}
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

	now := time.Now()
	id := uuid.NewString()

	created, err := s.repository.CreateFund(ctx, Fund{
		ID:                  id,
		Name:                strings.TrimSpace(input.Name),
		Description:         strings.TrimSpace(input.Description),
		TotalAmount:         input.TotalAmount,
		MonthlyContribution: input.MonthlyContribution,
		DurationMonths:      input.DurationMonths,
		MaxMembers:          input.MaxMembers,
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

	return map[string]string{"message": "member approved", "user_id": targetUserID}, nil
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
