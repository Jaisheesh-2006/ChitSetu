package payments

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

type createOrderRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

type verifyPaymentRequest struct {
	SessionID         string `json:"session_id" binding:"required"`
	RazorpayOrderID   string `json:"razorpay_order_id" binding:"required"`
	RazorpayPaymentID string `json:"razorpay_payment_id" binding:"required"`
	RazorpaySignature string `json:"razorpay_signature" binding:"required"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// resolveSessionID converts a composite "fundID-userID-cycleNo" string into a
// real payment session UUID, creating it in MongoDB if needed.
// A composite key has the form:
//
//	<uuid-fund-id>-<uuid-user-id>-<cycleNumber>
//
// which when split on "-" yields 11 segments (5+5+1).
// If the input is already a plain UUID it is returned unchanged.
func (h *Handler) resolveSessionID(c *gin.Context, userID, raw string) string {
	parts := strings.Split(raw, "-")
	if len(parts) < 11 {
		return raw
	}

	fundID := strings.Join(parts[0:5], "-")
	qUserID := strings.Join(parts[5:10], "-")
	cycleStr := strings.Join(parts[10:], "-")

	cycleNo, err := strconv.Atoi(cycleStr)
	if err != nil || cycleNo <= 0 {
		return raw
	}
	// Security: the embedded user_id must match the authenticated caller.
	if qUserID != userID {
		return raw
	}

	realID, err := h.service.GeneratePaymentSession(c.Request.Context(), userID, fundID, cycleNo)
	if err != nil {
		return raw
	}
	return realID
}

func (h *Handler) GetSession(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	sessionID := strings.TrimSpace(c.Param("id"))
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session id is required"})
		return
	}

	sessionID = h.resolveSessionID(c, userID, sessionID)

	result, err := h.service.GetSessionDetails(c.Request.Context(), userID, sessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) CreateOrder(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	sessionID := h.resolveSessionID(c, userID, strings.TrimSpace(req.SessionID))

	result, err := h.service.CreateOrder(c.Request.Context(), userID, sessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) Verify(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	var req verifyPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	sessionID := h.resolveSessionID(c, userID, strings.TrimSpace(req.SessionID))

	alreadyPaid, err := h.service.VerifyPayment(c.Request.Context(), userID, VerifyInput{
		SessionID:         sessionID,
		RazorpayOrderID:   strings.TrimSpace(req.RazorpayOrderID),
		RazorpayPaymentID: strings.TrimSpace(req.RazorpayPaymentID),
		RazorpaySignature: strings.TrimSpace(req.RazorpaySignature),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if alreadyPaid {
		c.JSON(http.StatusOK, gin.H{"status": "paid", "idempotent": true})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "paid"})
}

func authenticatedUserID(c *gin.Context) (string, bool) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
		return "", false
	}

	userID, ok := userIDValue.(string)
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authenticated user"})
		return "", false
	}

	return strings.TrimSpace(userID), true
}
