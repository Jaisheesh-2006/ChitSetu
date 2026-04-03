package users

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ── Regex patterns ───────────────────────────────────────────────────────────

var (
	// Indian bank account: 9–18 digits
	bankAccountRegex = regexp.MustCompile(`^[0-9]{9,18}$`)
	// IFSC code: 4 letters, '0', then 6 alphanumerics — e.g. HDFC0001234
	ifscRegex = regexp.MustCompile(`^[A-Z]{4}0[A-Z0-9]{6}$`)
)

// ── ML service HTTP helper ───────────────────────────────────────────────────

func mlServiceBaseURL() string {
	if url := os.Getenv("ML_SERVICE_URL"); url != "" {
		return strings.TrimRight(url, "/")
	}
	return "http://localhost:8000"
}

func callMLService(ctx context.Context, path string, payload interface{}) (map[string]interface{}, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal ml payload: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		mlServiceBaseURL()+path,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("build ml request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 40 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ml service %s unreachable: %w", path, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read ml response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ml service %s returned %d: %s", path, resp.StatusCode, string(respBytes))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("decode ml response: %w", err)
	}
	return result, nil
}

// bsonToNative recursively converts BSON types (primitive.D, primitive.A)
// into plain Go maps and slices so that json.Marshal produces standard JSON
// objects instead of key/value pair arrays.
func bsonToNative(v interface{}) interface{} {
	switch val := v.(type) {
	case bson.D:
		m := make(map[string]interface{}, len(val))
		for _, e := range val {
			m[e.Key] = bsonToNative(e.Value)
		}
		return m
	case bson.A:
		s := make([]interface{}, len(val))
		for i, e := range val {
			s[i] = bsonToNative(e)
		}
		return s
	case bson.M:
		m := make(map[string]interface{}, len(val))
		for k, e := range val {
			m[k] = bsonToNative(e)
		}
		return m
	case []interface{}:
		s := make([]interface{}, len(val))
		for i, e := range val {
			s[i] = bsonToNative(e)
		}
		return s
	case map[string]interface{}:
		m := make(map[string]interface{}, len(val))
		for k, e := range val {
			m[k] = bsonToNative(e)
		}
		return m
	default:
		return v
	}
}

// ── VerifyPAN ────────────────────────────────────────────────────────────────

// VerifyPAN simulates PAN verification and CIBIL check via the ML service.
// Transitions kyc.status: pending → pan_verified.
//
// POST /user/kyc/verify-pan
// (No body required — PAN and profile are read from DB)
func (h *Handler) VerifyPAN(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	profile, err := h.repository.GetKYCData(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch profile"})
		return
	}
	if profile == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "please complete your profile first"})
		return
	}

	// Idempotency: if already past pending, return current state without re-doing.
	if profile.KYCStatus != "" && profile.KYCStatus != "pending" {
		c.JSON(http.StatusOK, gin.H{
			"pan_verified": true,
			"has_cibil":    profile.CibilScore != nil,
			"cibil_score":  profile.CibilScore,
			"kyc_status":   profile.KYCStatus,
			"skipped":      true,
		})
		return
	}

	// Call ML service: /generate-credit
	mlPayload := map[string]interface{}{
		"pan":              profile.PAN,
		"age":              profile.Age,
		"income":           profile.Income,
		"employment_years": profile.EmploymentYears,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()

	mlResult, err := callMLService(ctx, "/generate-credit", mlPayload)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":  "credit check service unavailable",
			"detail": err.Error(),
		})
		return
	}

	panVerified, _ := mlResult["pan_verified"].(bool)
	if !panVerified {
		c.JSON(http.StatusOK, gin.H{
			"pan_verified": false,
			"kyc_status":   "pending",
			"error":        "PAN could not be verified with the credit bureau",
		})
		return
	}

	hasCibil, _ := mlResult["has_cibil"].(bool)
	var cibilScore *int
	if hasCibil {
		if scoreFloat, ok := mlResult["cibil_score"].(float64); ok {
			score := int(scoreFloat)
			cibilScore = &score
		}
	}

	if err := h.repository.StorePANVerified(c.Request.Context(), userID, cibilScore); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save verification result"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pan_verified": true,
		"has_cibil":    hasCibil,
		"cibil_score":  cibilScore,
		"kyc_status":   "pan_verified",
	})
}

// ── FetchHistory ─────────────────────────────────────────────────────────────

type fetchHistoryRequest struct {
	BankAccount string `json:"bank_account" binding:"required"`
	IFSCCode    string `json:"ifsc_code" binding:"required"`
}

// FetchHistory validates bank account + IFSC, generates synthetic transaction
// history via the ML service (Gemini-backed), and stores it in MongoDB.
// Transitions kyc.status: pan_verified → credit_fetched.
//
// POST /user/kyc/fetch-history
// Body: { "bank_account": "...", "ifsc_code": "..." }
func (h *Handler) FetchHistory(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	var req fetchHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bank_account and ifsc_code are required"})
		return
	}

	account := strings.TrimSpace(req.BankAccount)
	ifsc := strings.ToUpper(strings.TrimSpace(req.IFSCCode))

	if !bankAccountRegex.MatchString(account) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid bank account number: must be 9 to 18 digits",
			"field": "bank_account",
		})
		return
	}
	if !ifscRegex.MatchString(ifsc) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid IFSC code format (example: HDFC0001234)",
			"field": "ifsc_code",
		})
		return
	}

	profile, err := h.repository.GetKYCData(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch profile"})
		return
	}
	if profile == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "profile not found"})
		return
	}

	// Idempotency: if already past pan_verified, skip re-generation.
	if profile.KYCStatus != "" && profile.KYCStatus != "pan_verified" {
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"kyc_status": profile.KYCStatus,
			"skipped":    true,
		})
		return
	}

	if profile.KYCStatus != "pan_verified" {
		c.JSON(http.StatusConflict, gin.H{
			"error":      "PAN must be verified first",
			"kyc_status": profile.KYCStatus,
		})
		return
	}

	// Call ML service: /generate-history
	hasCibil := profile.CibilScore != nil
	mlPayload := map[string]interface{}{
		"has_cibil":        hasCibil,
		"cibil_score":      profile.CibilScore,
		"age":              profile.Age,
		"income":           profile.Income,
		"employment_years": profile.EmploymentYears,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()

	history, err := callMLService(ctx, "/generate-history", mlPayload)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":  "history generation service unavailable",
			"detail": err.Error(),
		})
		return
	}

	if err := h.repository.StoreSyntheticHistory(c.Request.Context(), userID, history); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store transaction history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"kyc_status": "credit_fetched",
		"history":    history,
	})
}

// ── RunML ────────────────────────────────────────────────────────────────────

// RunML reads the stored synthetic history, calls the ML /predict endpoint,
// stores the trust score, and transitions kyc.status: credit_fetched → verified.
//
// POST /user/kyc/run-ml
// (No body required)
func (h *Handler) RunML(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	profile, err := h.repository.GetKYCData(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch profile"})
		return
	}
	if profile == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "profile not found"})
		return
	}

	// Idempotency: already scored → return stored result.
	// Accept both "verified" (new) and "ml_ready" (legacy) as the completed state.
	if profile.KYCStatus == "verified" || profile.KYCStatus == "ml_ready" {
		c.JSON(http.StatusOK, gin.H{
			"score":               profile.TrustScore,
			"risk_band":           profile.RiskBand,
			"default_probability": profile.DefaultProbability,
			"kyc_status":          "verified",
			"skipped":             true,
		})
		return
	}

	if profile.KYCStatus != "credit_fetched" {
		c.JSON(http.StatusConflict, gin.H{
			"error":      "transaction history not yet generated",
			"kyc_status": profile.KYCStatus,
		})
		return
	}
	if profile.SyntheticHistory == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "synthetic history missing — please retry fetch-history"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()

	// Convert BSON types (primitive.D/A) to standard Go maps/slices
	// so json.Marshal produces a proper JSON object, not an array of key-value pairs.
	nativeHistory := bsonToNative(profile.SyntheticHistory)
	mlResult, err := callMLService(ctx, "/predict", nativeHistory)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":  "ML prediction service unavailable",
			"detail": err.Error(),
		})
		return
	}

	scoreFloat, _ := mlResult["score"].(float64)
	riskBand, _ := mlResult["risk_band"].(string)
	defaultProb, _ := mlResult["default_probability"].(float64)

	if err := h.repository.StoreTrustScore(
		c.Request.Context(),
		userID,
		int(scoreFloat),
		riskBand,
		defaultProb,
	); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store trust score"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"score":               int(scoreFloat),
		"risk_band":           riskBand,
		"default_probability": defaultProb,
		"kyc_status":          "verified",
	})
}

// ── GetKYCStatus ─────────────────────────────────────────────────────────────

// GetKYCStatus returns the current KYC state so the frontend can resume
// an interrupted wizard at the correct step.
//
// GET /users/kyc/status
func (h *Handler) GetKYCStatus(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	data, err := h.repository.GetKYCData(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch KYC status"})
		return
	}
	if data == nil {
		c.JSON(http.StatusOK, gin.H{"kyc_status": "none"})
		return
	}

	// Normalise legacy "ml_ready" to "verified" so the frontend only needs
	// to handle the current canonical status string.
	kycStatus := data.KYCStatus
	if kycStatus == "ml_ready" {
		kycStatus = "verified"
	}

	resp := gin.H{
		"kyc_status":          kycStatus,
		"has_cibil":           data.CibilScore != nil,
		"cibil_score":         data.CibilScore,
		"trust_score":         data.TrustScore,
		"risk_band":           data.RiskBand,
		"default_probability": data.DefaultProbability,
	}
	if !data.CheckedAt.IsZero() {
		resp["checked_at"] = data.CheckedAt
	}

	c.JSON(http.StatusOK, resp)
}
