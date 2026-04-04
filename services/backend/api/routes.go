package api

import (
	"context"
	"net/http"
	"time"

	"github.com/Jaisheesh-2006/ChitSetu/handlers"
	"github.com/Jaisheesh-2006/ChitSetu/internal/auction"
	"github.com/Jaisheesh-2006/ChitSetu/internal/auth"
	"github.com/Jaisheesh-2006/ChitSetu/internal/chat"
	"github.com/Jaisheesh-2006/ChitSetu/internal/chitfund"
	"github.com/Jaisheesh-2006/ChitSetu/internal/payments"
	"github.com/Jaisheesh-2006/ChitSetu/internal/users"
	"github.com/Jaisheesh-2006/ChitSetu/middleware"
	"github.com/Jaisheesh-2006/ChitSetu/models"
	"github.com/Jaisheesh-2006/ChitSetu/pkg/database"
	pkgmiddleware "github.com/Jaisheesh-2006/ChitSetu/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// DBPinger captures the minimum database behavior needed for health checks.

func SetupRouter(store *database.Store, auctionHandler *auction.Handler, authService *auth.Service, chitfundHandler *chitfund.Handler, paymentHandler *payments.Handler, chatHandler *chat.Handler) *gin.Engine {
	router := gin.New()
	router.Use(middleware.CORS(), gin.Logger(), gin.Recovery())
	authHandler := auth.NewHandler(authService)
	authMiddleware := pkgmiddleware.JWTAuth(authService.JWTSecret())
	supabaseVerifier, err := middleware.NewVerifierFromEnv()
	if err != nil {
		panic(err)
	}
	userStore := models.NewUserStore(store.Database)
	supabaseAuthHandler := handlers.NewAuthHandler(supabaseVerifier, userStore, authService)
	supabaseRequireAuth := middleware.RequireAuth(supabaseVerifier)
	profileRepo := users.NewRepository(store.Database)
	profileHandler := users.NewHandler(profileRepo)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "backend"})
	})

	router.GET("/health/db", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := store.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "error": "database unavailable"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok", "db": "reachable"})
	})
	authGroup := router.Group("/auth")
	authGroup.POST("/register", authHandler.Register)
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/refresh", authHandler.Refresh)
	authGroup.POST("/forgot-password", authHandler.ForgotPassword)
	authGroup.POST("/reset-password", authHandler.ResetPassword)
	authGroup.POST("/verify", supabaseAuthHandler.Verify)
	authGroup.GET("/me", supabaseRequireAuth, supabaseAuthHandler.Me)

	userGroup := router.Group("/user")
	userGroup.Use(authMiddleware)
	userGroup.POST("/profile", profileHandler.UpsertProfile)
	userGroup.POST("/kyc/verify-pan", profileHandler.VerifyPAN)
	userGroup.POST("/kyc/fetch-history", profileHandler.FetchHistory)
	userGroup.POST("/kyc/run-ml", profileHandler.RunML)

	usersGroup := router.Group("/users")
	usersGroup.Use(authMiddleware)
	usersGroup.GET("/profile", profileHandler.GetProfile)
	usersGroup.GET("/risk-score", profileHandler.GetRiskScore)
	usersGroup.GET("/kyc/status", profileHandler.GetKYCStatus)
	usersGroup.GET("/me/funds", profileHandler.GetMyFunds)
	usersGroup.GET("/me/contributions", profileHandler.GetMyContributions)

	fundGroup := router.Group("/funds")
	fundGroup.Use(authMiddleware)
	fundGroup.POST("", chitfundHandler.CreateFund)
	fundGroup.GET("", chitfundHandler.ListFunds)
	fundGroup.GET("/:id", chitfundHandler.GetFund)
	fundGroup.POST("/:id/apply", chitfundHandler.Apply)
	fundGroup.GET("/:id/application-status", chitfundHandler.ApplicationStatus)
	fundGroup.POST("/:id/approve", chitfundHandler.Approve)
	fundGroup.POST("/:id/reject", chitfundHandler.Reject)
	fundGroup.GET("/:id/members", chitfundHandler.Members)

	paymentsGroup := router.Group("/payments")
	paymentsGroup.Use(authMiddleware)
	paymentsGroup.GET("/session/:id", paymentHandler.GetSession)
	paymentsGroup.POST("/create-order", paymentHandler.CreateOrder)
	paymentsGroup.POST("/verify", paymentHandler.Verify)

	fundGroup.GET("/:id/contributions/current", chitfundHandler.CurrentCycleContributions)
	fundGroup.POST("/:id/auction/start", auctionHandler.StartAuction)
	fundGroup.POST("/:id/auction/activate", auctionHandler.ActivateAuction)
	fundGroup.POST("/:id/auction/bid", auctionHandler.PlaceBid)
	fundGroup.GET("/:id/auction", auctionHandler.GetAuction)
	fundGroup.POST("/:id/chat", chatHandler.SendMessage)
	fundGroup.GET("/:id/chat", chatHandler.GetMessages)

	router.GET("/ws/funds/:id", authMiddleware, auctionHandler.FundWebSocket)
	return router
}
