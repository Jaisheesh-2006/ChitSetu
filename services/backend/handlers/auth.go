package handlers

import (
	"net/http"
	"strings"

	"github.com/Jaisheesh-2006/ChitSetu/internal/auth"
	"github.com/Jaisheesh-2006/ChitSetu/middleware"
	"github.com/Jaisheesh-2006/ChitSetu/models"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	verifier    *middleware.Verifier
	userStore   *models.UserStore
	authService *auth.Service
}

type verifyRequest struct {
	AccessToken string `json:"access_token" binding:"required"`
}

func NewAuthHandler(verifier *middleware.Verifier, userStore *models.UserStore, authService *auth.Service) *AuthHandler {
	return &AuthHandler{verifier: verifier, userStore: userStore, authService: authService}
}

// Verify validates Supabase access token and upserts user profile locally.
func (h *AuthHandler) Verify(c *gin.Context) {
	var req verifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	claims, err := h.verifier.ParseAndValidateAccessToken(strings.TrimSpace(req.AccessToken))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if _, err := h.userStore.UpsertFromSupabase(c.Request.Context(), claims); err != nil {
		// Token was valid but DB sync failed — this is a server error, not auth failure
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to sync user"})
		return
	}

	tokens, err := h.authService.IssueTokenPair(c.Request.Context(), claims.Subject)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to issue app tokens"})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// Me returns current authenticated user profile from local store.
func (h *AuthHandler) Me(c *gin.Context) {
	userIDRaw := c.Request.Context().Value(middleware.UserIDKey)
	userID, _ := userIDRaw.(string)
	if strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
		return
	}

	user, err := h.userStore.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load user profile"})
		return
	}
	if user == nil {
		emailRaw := c.Request.Context().Value(middleware.EmailKey)
		email, _ := emailRaw.(string)
		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"email":   email,
		})
		return
	}

	c.JSON(http.StatusOK, user)
}
