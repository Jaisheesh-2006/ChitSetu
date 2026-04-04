package chat

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Jaisheesh-2006/ChitSetu/internal/ws"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo      *Repository
	wsManager *ws.Manager
}

func NewHandler(repo *Repository, wsManager *ws.Manager) *Handler {
	return &Handler{repo: repo, wsManager: wsManager}
}

type sendMessageRequest struct {
	Message     string `json:"message" binding:"required"`
	ChatType    string `json:"chat_type" binding:"required"`
	CycleNumber int    `json:"cycle_number,omitempty"`
}

func (h *Handler) SendMessage(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	fundID := strings.TrimSpace(c.Param("id"))
	if fundID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fund id is required"})
		return
	}

	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.ChatType != "fund" && req.ChatType != "auction" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chat_type must be 'fund' or 'auction'"})
		return
	}

	// Verify user is an active member
	isMember, err := h.repo.IsActiveMember(c.Request.Context(), fundID, userID)
	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not an active member of this fund"})
		return
	}

	fullName := h.repo.GetUserFullName(c.Request.Context(), userID)

	msg := &Message{
		FundID:      fundID,
		UserID:      userID,
		FullName:    fullName,
		MessageText: strings.TrimSpace(req.Message),
		ChatType:    req.ChatType,
		CycleNumber: req.CycleNumber,
	}

	if err := h.repo.SaveMessage(c.Request.Context(), msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save message"})
		return
	}

	// Broadcast via WebSocket
	_ = h.wsManager.Broadcast(fundID, map[string]any{
		"type":         "chat_message",
		"fund_id":      msg.FundID,
		"user_id":      msg.UserID,
		"full_name":    msg.FullName,
		"message":      msg.MessageText,
		"chat_type":    msg.ChatType,
		"cycle_number": msg.CycleNumber,
		"created_at":   msg.CreatedAt,
	})

	c.JSON(http.StatusOK, msg)
}

func (h *Handler) GetMessages(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	fundID := strings.TrimSpace(c.Param("id"))
	if fundID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fund id is required"})
		return
	}

	// Verify user is an active member
	isMember, err := h.repo.IsActiveMember(c.Request.Context(), fundID, userID)
	if err != nil || !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not an active member of this fund"})
		return
	}

	chatType := c.DefaultQuery("type", "fund")
	limitStr := c.DefaultQuery("limit", "50")
	cycleStr := c.DefaultQuery("cycle", "0")

	limit, _ := strconv.Atoi(limitStr)
	cycle, _ := strconv.Atoi(cycleStr)

	var before *time.Time
	if beforeStr := c.Query("before"); beforeStr != "" {
		if t, err := time.Parse(time.RFC3339, beforeStr); err == nil {
			before = &t
		}
	}

	messages, err := h.repo.GetMessages(c.Request.Context(), fundID, chatType, cycle, limit, before)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch messages"})
		return
	}

	if messages == nil {
		messages = []Message{}
	}

	c.JSON(http.StatusOK, messages)
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
