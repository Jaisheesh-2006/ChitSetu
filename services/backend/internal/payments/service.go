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
		resendFromEmail:      strings.TrimSpace(os.Getenv("RESEND_FROM_EMAIL")),
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

func (s *Service) GetSessionDetails(ctx context.Context, userID, sessionID string) (map[string]interface{}, error) {

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
	if s.resendAPIKey == "" || s.resendFromEmail == "" {
		log.Printf("resend not configured, reminder link for %s: %s", recipient, paymentLink)
		return nil
	}

	payload := map[string]any{
		"from":    s.resendFromEmail,
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
