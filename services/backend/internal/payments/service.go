package payments

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

const resendFromAddress = "Acme <onboarding@resend.dev>"

type Service struct {
	repo                 *Repository
	httpClient           *http.Client
	razorpayKeyID        string
	razorpayKeySecret    string
	appBaseURL           string
	resendAPIKey         string
	defaultReminderEmail string
}

type SessionDetails struct {
	AmountDue   float64   `json:"amount"`
	FundID      string    `json:"fund_id"`
	CycleNumber int       `json:"cycle_no"`
	DueDate     time.Time `json:"due_date"`
}

type CreateOrderResult struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
	KeyID   string `json:"key_id"`
}

type VerifyInput struct {
	SessionID         string
	RazorpayOrderID   string
	RazorpayPaymentID string
	RazorpaySignature string
}

func NewService(repo *Repository) *Service {
	baseURL := strings.TrimSpace(os.Getenv("APP_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://yourapp.com"
	}
	return &Service{
		repo:                 repo,
		httpClient:           &http.Client{Timeout: 15 * time.Second},
		razorpayKeyID:        strings.TrimSpace(os.Getenv("RAZORPAY_KEY_ID")),
		razorpayKeySecret:    strings.TrimSpace(os.Getenv("RAZORPAY_KEY_SECRET")),
		appBaseURL:           strings.TrimRight(baseURL, "/"),
		resendAPIKey:         strings.TrimSpace(os.Getenv("RESEND_API_KEY")),
		defaultReminderEmail: strings.TrimSpace(os.Getenv("PAYMENT_REMINDER_FALLBACK_EMAIL")),
	}
}

func (s *Service) StartDailyReminderCron() *cron.Cron {
	c := cron.New(cron.WithLocation(time.Local))
	_, err := c.AddFunc("0 9 * * *", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := s.RunDailyReminderJob(ctx); err != nil {
			log.Printf("payment reminder cron error: %v", err)
		}
	})
	if err != nil {
		log.Printf("failed to register payment reminder cron: %v", err)
		return c
	}

	c.Start()
	return c
}

func (s *Service) RunDailyReminderJob(ctx context.Context) error {
	if err := s.repo.EnsureContributionsForActiveFunds(ctx); err != nil {
		return err
	}
	if err := s.repo.ExpireSessions(ctx); err != nil {
		return err
	}

	items, err := s.repo.ListUpcomingPendingContributions(ctx)
	if err != nil {
		return err
	}

	for _, item := range items {
		session, err := s.repo.GetActiveSessionForContribution(ctx, item.ContributionID)
		if err != nil {
			log.Printf("get active session failed for contribution %s: %v", item.ContributionID, err)
			continue
		}
		if session == nil {
			session, err = s.repo.CreatePaymentSession(ctx, item.ContributionID, item.UserID, item.AmountDue, time.Now().Add(48*time.Hour))
			if err != nil {
				log.Printf("create session failed for contribution %s: %v", item.ContributionID, err)
				continue
			}
		}

		link := fmt.Sprintf("%s/pay?session_id=%s", s.appBaseURL, session.ID)
		if err := s.sendReminderEmail(ctx, item.Email, link); err != nil {
			log.Printf("send reminder email failed for user %s: %v", item.UserID, err)
		}
	}

	return nil
}

func (s *Service) GeneratePaymentSession(ctx context.Context, userID, fundID string, cycleNumber int) (string, error) {
	contribID, amountDue, _, status, err := s.repo.GetPendingContributionInfo(ctx, fundID, userID, cycleNumber)
	if err != nil {
		return "", err
	}
	if status == "paid" {
		return "", fmt.Errorf("contribution already paid")
	}

	activeSession, err := s.repo.GetActiveSessionForContribution(ctx, contribID)
	if err != nil {
		return "", err
	}
	if activeSession != nil {
		return activeSession.ID, nil
	}

	// Create new session valid for 2 hours
	newSession, err := s.repo.CreatePaymentSession(ctx, contribID, userID, amountDue, time.Now().Add(2*time.Hour))
	if err != nil {
		return "", fmt.Errorf("failed to create payment session: %w", err)
	}

	return newSession.ID, nil
}

func (s *Service) GetSessionDetails(ctx context.Context, userID, sessionID string) (*SessionDetails, error) {
	session, err := s.repo.GetSessionForUser(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}
	if session.ContributionSt == "paid" {
		return nil, fmt.Errorf("contribution already paid")
	}
	if session.ExpiresAt.Before(time.Now()) {
		_ = s.repo.ExpireSessions(ctx)
		return nil, fmt.Errorf("session expired")
	}

	return &SessionDetails{
		AmountDue:   session.AmountDue,
		FundID:      session.FundID,
		CycleNumber: session.CycleNumber,
		DueDate:     session.DueDate,
	}, nil
}

func (s *Service) CreateOrder(ctx context.Context, userID, sessionID string) (*CreateOrderResult, error) {
	if s.razorpayKeyID == "" || s.razorpayKeySecret == "" {
		return nil, fmt.Errorf("razorpay keys are not configured")
	}

	session, err := s.repo.GetSessionForUser(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}
	if session.ContributionSt == "paid" {
		return nil, fmt.Errorf("contribution already paid")
	}
	if session.ExpiresAt.Before(time.Now()) {
		_ = s.repo.ExpireSessions(ctx)
		return nil, fmt.Errorf("session expired")
	}
	if session.Status != "created" {
		return nil, fmt.Errorf("session is not payable")
	}

	amountPaise := int64(math.Round(session.AmountDue * 100))
	payload := map[string]any{
		"amount":   amountPaise,
		"currency": "INR",
		"receipt":  session.ID,
	}

	orderID, err := s.createRazorpayOrder(ctx, payload)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpsertOrderForSession(ctx, session.ID, orderID); err != nil {
		return nil, err
	}

	return &CreateOrderResult{OrderID: orderID, Amount: amountPaise, KeyID: s.razorpayKeyID}, nil
}

func (s *Service) VerifyPayment(ctx context.Context, userID string, input VerifyInput) (bool, error) {

	if strings.TrimSpace(input.SessionID) == "" ||
		strings.TrimSpace(input.RazorpayOrderID) == "" ||
		strings.TrimSpace(input.RazorpayPaymentID) == "" ||
		strings.TrimSpace(input.RazorpaySignature) == "" {
		return false, fmt.Errorf("missing verify fields")
	}

	expectedOrderID, err := s.repo.GetOrderForSession(ctx, input.SessionID)
	if err != nil {
		return false, err
	}
	if expectedOrderID != input.RazorpayOrderID {
		return false, fmt.Errorf("order mismatch")
	}

	if !s.verifyRazorpaySignature(input.RazorpayOrderID, input.RazorpayPaymentID, input.RazorpaySignature) {
		return false, fmt.Errorf("invalid signature")
	}

	// DB UPDATE
	alreadyPaid, contribution, err := s.repo.MarkPaymentVerified(ctx, userID, input.SessionID, input.RazorpayPaymentID)
	if err != nil {
		return false, err
	}

	if alreadyPaid {
		log.Println("already processed")
		return true, nil
	}

	if contribution == nil {
		return false, nil
	}

	return true, nil
}

func (s *Service) createRazorpayOrder(ctx context.Context, payload map[string]any) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal razorpay order payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.razorpay.com/v1/orders", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create razorpay request: %w", err)
	}
	req.SetBasicAuth(s.razorpayKeyID, s.razorpayKeySecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call razorpay orders api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("razorpay orders api returned status %d", resp.StatusCode)
	}

	var parsed struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode razorpay order response: %w", err)
	}
	if strings.TrimSpace(parsed.ID) == "" {
		return "", fmt.Errorf("razorpay order id missing")
	}

	return parsed.ID, nil
}

func (s *Service) verifyRazorpaySignature(orderID, paymentID, signature string) bool {
	mac := hmac.New(sha256.New, []byte(s.razorpayKeySecret))
	mac.Write([]byte(orderID + "|" + paymentID))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (s *Service) sendReminderEmail(ctx context.Context, toEmail, paymentLink string) error {
	recipient := strings.TrimSpace(toEmail)
	if recipient == "" {
		recipient = s.defaultReminderEmail
	}
	if recipient == "" {
		return nil
	}
	if s.resendAPIKey == "" {
		log.Printf("resend not configured, reminder link for %s: %s", recipient, paymentLink)
		return nil
	}

	payload := map[string]any{
		"from":    resendFromAddress,
		"to":      []string{recipient},
		"subject": "Contribution Payment Reminder",
		"html":    fmt.Sprintf("<p>Your contribution payment is due soon.</p><p><a href=\"%s\">Pay Now</a></p>", paymentLink),
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

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call resend api: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("resend api returned status %d", resp.StatusCode)
	}
	return nil
}
