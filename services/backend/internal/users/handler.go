package users

import (
	"net/http"
	"strings"
	"time"

	"github.com/Jaisheesh-2006/ChitSetu/internal/validation"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repository *Repository
}

type profileRequest struct {
	FullName        string  `json:"full_name" binding:"required"`
	Age             int     `json:"age" binding:"required"`
	PhoneNumber     string  `json:"phone_number" binding:"required"`
	PANNumber       string  `json:"pan_number" binding:"required"`
	MonthlyIncome   float64 `json:"monthly_income" binding:"required"`
	EmploymentYears int     `json:"employment_years" binding:"required"`
}

type ProfileInput struct {
	UserID          string
	FullName        string
	Age             int
	PhoneNumber     string
	PANNumber       string
	MonthlyIncome   float64
	EmploymentYears int
}

type userFundResponse struct {
	ID                  string     `json:"_id"`
	Name                string     `json:"name"`
	TotalAmount         float64    `json:"total_amount"`
	MonthlyContribution float64    `json:"monthly_contribution"`
	DurationMonths      int        `json:"duration_months"`
	Status              string     `json:"status"`
	StartDate           time.Time  `json:"start_date"`
	CreatorID           string     `json:"creator_id"`
	CurrentMemberCount  int64      `json:"current_member_count"`
	JoinedAt            *time.Time `json:"joined_at,omitempty"`
}

type userContributionResponse struct {
	FundID      string    `json:"fund_id"`
	FundName    string    `json:"fund_name"`
	AmountDue   float64   `json:"amount_due"`
	DueDate     time.Time `json:"due_date"`
	CycleNumber int       `json:"cycle_number"`
	Status      string    `json:"status"`
}

func NewHandler(repository *Repository) *Handler {
	return &Handler{repository: repository}
}

func (h *Handler) UpsertProfile(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	var req profileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	pan := strings.ToUpper(strings.TrimSpace(req.PANNumber))
	if !validation.IsValidPAN(pan) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid PAN format"})
		return
	}

	profile := ProfileInput{
		UserID:          userID,
		FullName:        strings.TrimSpace(req.FullName),
		Age:             req.Age,
		PhoneNumber:     strings.TrimSpace(req.PhoneNumber),
		PANNumber:       pan,
		MonthlyIncome:   req.MonthlyIncome,
		EmploymentYears: req.EmploymentYears,
	}

	if profile.FullName == "" || profile.PhoneNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "full_name and phone_number are required"})
		return
	}
	if !validation.IsValidPhone10(profile.PhoneNumber) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone_number must be exactly 10 digits"})
		return
	}
	if profile.Age <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "age must be greater than 0"})
		return
	}
	if profile.MonthlyIncome <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "monthly_income must be greater than 0"})
		return
	}
	if profile.EmploymentYears < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "employment_years cannot be negative"})
		return
	}

	if err := h.repository.UpsertProfile(c.Request.Context(), profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"kyc_status": "pending",
	})
}

func (h *Handler) GetProfile(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	profile, err := h.repository.GetUserProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user profile"})
		return
	}
	if profile == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user profile not found"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *Handler) GetRiskScore(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	credit, err := h.repository.GetUserCredit(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch risk score"})
		return
	}
	if credit == nil || credit.Score <= 0 || strings.TrimSpace(credit.RiskCategory) == "" || credit.CheckedAt.IsZero() {
		c.JSON(http.StatusNotFound, gin.H{"error": "credit check not yet performed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"score":         credit.Score,
		"risk_category": credit.RiskCategory,
		"checked_at":    credit.CheckedAt,
	})
}

func authenticatedUserID(c *gin.Context) (string, bool) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authenticated user"})
		return "", false
	}

	userID, ok := userIDValue.(string)
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authenticated user"})
		return "", false
	}

	return strings.TrimSpace(userID), true
}
