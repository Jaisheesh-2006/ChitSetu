package chitfund

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

type createFundRequest struct {
	Name                string  `json:"name" binding:"required"`
	Description         string  `json:"description" binding:"required"`
	TotalAmount         float64 `json:"total_amount" binding:"required"`
	MonthlyContribution float64 `json:"monthly_contribution" binding:"required"`
	MaxMembers          int     `json:"max_members" binding:"required"`
	StartDate           string  `json:"start_date" binding:"required"`
}

type approveRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateFund(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	var req createFundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	startDate, err := time.Parse(time.RFC3339, strings.TrimSpace(req.StartDate))
	if err != nil {
		startDate, err = time.Parse("2006-01-02", strings.TrimSpace(req.StartDate))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "start_date must be RFC3339 or YYYY-MM-DD"})
			return
		}
	}

	fund, err := h.service.CreateFund(c.Request.Context(), userID, CreateFundInput{
		Name:                req.Name,
		Description:         req.Description,
		TotalAmount:         req.TotalAmount,
		MonthlyContribution: req.MonthlyContribution,
		MaxMembers:          req.MaxMembers,
		StartDate:           startDate,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, fund)
}

func (h *Handler) ListFunds(c *gin.Context) {
	funds, err := h.service.ListFunds(c.Request.Context())
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, funds)
}

func (h *Handler) GetFund(c *gin.Context) {
	fundID := strings.TrimSpace(c.Param("id"))
	fund, err := h.service.GetFundDetails(c.Request.Context(), fundID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, fund)
}

func (h *Handler) Apply(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	fundID := strings.TrimSpace(c.Param("id"))
	result, err := h.service.ApplyToFund(c.Request.Context(), userID, fundID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) Approve(c *gin.Context) {
	organizerID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	fundID := strings.TrimSpace(c.Param("id"))
	var req approveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	result, err := h.service.ApproveMember(c.Request.Context(), organizerID, fundID, strings.TrimSpace(req.UserID))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) Reject(c *gin.Context) {
	organizerID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	fundID := strings.TrimSpace(c.Param("id"))
	var req approveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	result, err := h.service.RejectMember(c.Request.Context(), organizerID, fundID, strings.TrimSpace(req.UserID))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ApplicationStatus(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	fundID := strings.TrimSpace(c.Param("id"))
	status, err := h.service.GetApplicationStatus(c.Request.Context(), userID, fundID)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, status)
}

func (h *Handler) Members(c *gin.Context) {
	organizerID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	fundID := strings.TrimSpace(c.Param("id"))
	members, err := h.service.ListMembers(c.Request.Context(), organizerID, fundID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, members)
}

func (h *Handler) CurrentCycleContributions(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	fundID := strings.TrimSpace(c.Param("id"))
	result, err := h.service.GetCurrentCycleContributions(c.Request.Context(), userID, fundID)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) respondError(c *gin.Context, err error) {
	if appErr, ok := err.(*AppError); ok {
		c.JSON(appErr.StatusCode, gin.H{"error": appErr.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}

func authenticatedUserID(c *gin.Context) (string, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
		return "", false
	}

	userID, ok := value.(string)
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authenticated user"})
		return "", false
	}

	return strings.TrimSpace(userID), true
}
