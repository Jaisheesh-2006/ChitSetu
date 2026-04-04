package payments

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

type VerifyInput struct {
	SessionID         string
	RazorpayOrderID   string
	RazorpayPaymentID string
	RazorpaySignature string
}

type ReminderCron struct {
	stop chan struct{}
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) StartDailyReminderCron() *ReminderCron {
	cron := &ReminderCron{stop: make(chan struct{})}
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-cron.stop:
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				_ = s.repo.EnsureContributionsForActiveFunds(ctx)
				_ = s.repo.ExpireSessions(ctx)
				cancel()
			}
		}
	}()
	return cron
}

func (c *ReminderCron) Stop() {
	close(c.stop)
}

func (s *Service) GeneratePaymentSession(ctx context.Context, userID, fundID string, cycleNumber int) (string, error) {
	contributionID, amountDue, _, status, err := s.repo.GetPendingContributionInfo(ctx, fundID, userID, cycleNumber)
	if err != nil {
		return "", err
	}
	if status == "paid" {
		return "", fmt.Errorf("contribution already paid")
	}

	existing, err := s.repo.GetActiveSessionForContribution(ctx, contributionID)
	if err != nil {
		return "", err
	}
	if existing != nil {
		return existing.ID, nil
	}

	expiresAt := time.Now().Add(15 * time.Minute)
	session, err := s.repo.CreatePaymentSession(ctx, contributionID, userID, amountDue, expiresAt)
	if err != nil {
		return "", err
	}
	return session.ID, nil
}

func (s *Service) GetSessionDetails(ctx context.Context, userID, sessionID string) (map[string]interface{}, error) {
	session, err := s.repo.GetSessionForUser(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"session_id":      session.ID,
		"fund_id":         session.FundID,
		"cycle_number":    session.CycleNumber,
		"amount_due":      session.AmountDue,
		"status":          session.Status,
		"expires_at":      session.ExpiresAt,
		"due_date":        session.DueDate,
		"contribution_id": session.ContributionID,
	}, nil
}

func (s *Service) CreateOrder(ctx context.Context, userID, sessionID string) (map[string]interface{}, error) {
	session, err := s.repo.GetSessionForUser(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}
	if session.Status != "created" {
		return nil, fmt.Errorf("session is not payable")
	}
	if session.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("session expired")
	}

	orderID := "order_" + uuid.NewString()
	if err := s.repo.UpsertOrderForSession(ctx, sessionID, orderID); err != nil {
		return nil, err
	}

	amountPaise := int64(session.AmountDue * 100)
	return map[string]interface{}{
		"order_id":     orderID,
		"session_id":   sessionID,
		"amount_paise": amountPaise,
		"currency":     "INR",
	}, nil
}

func (s *Service) VerifyPayment(ctx context.Context, userID string, input VerifyInput) (bool, error) {
	if strings.TrimSpace(input.SessionID) == "" {
		return false, fmt.Errorf("session_id is required")
	}
	if strings.TrimSpace(input.RazorpayOrderID) == "" || strings.TrimSpace(input.RazorpayPaymentID) == "" || strings.TrimSpace(input.RazorpaySignature) == "" {
		return false, fmt.Errorf("razorpay verification fields are required")
	}

	storedOrderID, err := s.repo.GetOrderForSession(ctx, input.SessionID)
	if err != nil {
		return false, err
	}
	if storedOrderID != input.RazorpayOrderID {
		return false, fmt.Errorf("order mismatch for session")
	}

	alreadyPaid, _, err := s.repo.MarkPaymentVerified(ctx, userID, input.SessionID, input.RazorpayPaymentID)
	if err != nil {
		return false, err
	}
	return alreadyPaid, nil
}
