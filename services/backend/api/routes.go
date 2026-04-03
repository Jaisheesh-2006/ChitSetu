package api

import (
	"context"
	"net/http"
	"time"

	"github.com/Jaisheesh-2006/ChitSetu/handlers"
	"github.com/Jaisheesh-2006/ChitSetu/internal/auth"
	"github.com/Jaisheesh-2006/ChitSetu/internal/users"
	"github.com/Jaisheesh-2006/ChitSetu/middleware"
	"github.com/Jaisheesh-2006/ChitSetu/models"
	"github.com/Jaisheesh-2006/ChitSetu/pkg/database"
	pkgmiddleware "github.com/Jaisheesh-2006/ChitSetu/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// DBPinger captures the minimum database behavior needed for health checks.

func SetupRouter(store *database.Store, authService *auth.Service) *gin.Engine {
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
	return router
}
