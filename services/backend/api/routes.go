package api

import (
	"context"
	"net/http"
	"time"

	"github.com/Jaisheesh-2006/ChitSetu/pkg/database"
	"github.com/gin-gonic/gin"
)

// DBPinger captures the minimum database behavior needed for health checks.


func SetupRouter(store *database.Store) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

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

	return router
}
