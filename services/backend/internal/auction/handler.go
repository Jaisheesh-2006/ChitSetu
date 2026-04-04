package auction

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Jaisheesh-2006/ChitSetu/internal/ws"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service   *Service
	wsManager *ws.Manager
}

type placeBidRequest struct {
	Increment int `json:"increment" binding:"required"`
}

func NewHandler(service *Service, wsManager *ws.Manager) *Handler {
	return &Handler{service: service, wsManager: wsManager}
}

func (h *Handler) StartAuction(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	fundID := strings.TrimSpace(c.Param("id"))
	if fundID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fund id is required"})
		return
	}

	session, err := h.service.StartAuction(c.Request.Context(), fundID, userID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, session)
}

func (h *Handler) ActivateAuction(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	fundID := strings.TrimSpace(c.Param("id"))
	if fundID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fund id is required"})
		return
	}

	err := h.service.ActivateAuction(c.Request.Context(), fundID, userID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "bidding started"})
}

func (h *Handler) PlaceBid(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	fundID := strings.TrimSpace(c.Param("id"))
	if fundID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fund id is required"})
		return
	}

	var req placeBidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	bid, session, err := h.service.PlaceBid(c.Request.Context(), PlaceBidInput{
		FundID:    fundID,
		UserID:    userID,
		Increment: req.Increment,
	})
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bid":     bid,
		"auction": session,
	})
}

func (h *Handler) GetAuction(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	fundID := strings.TrimSpace(c.Param("id"))
	if fundID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fund id is required"})
		return
	}

	snapshot, err := h.service.GetAuction(c.Request.Context(), fundID, userID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, snapshot)
}

func (h *Handler) FundWebSocket(c *gin.Context) {
	fundID := strings.TrimSpace(c.Param("id"))
	if fundID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fund id is required"})
		return
	}

	// Get user ID from auth context (set by JWT middleware)
	userID := ""
	if val, exists := c.Get("user_id"); exists {
		if uid, ok := val.(string); ok {
			userID = uid
		}
	}

	if err := h.wsManager.ServeFundConnection(c.Writer, c.Request, fundID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "websocket upgrade failed"})
		return
	}
}

func (h *Handler) respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrAuctionNotLive):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrAuctionAlreadyLive):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, ErrAuctionStartDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, ErrAuctionParticipantsNotReady):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrNotFundMember):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, ErrContributionUnpaid):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrUserAlreadyWon):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrInvalidIncrement):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrDiscountCapExceeded):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrConsecutiveBid):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
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
