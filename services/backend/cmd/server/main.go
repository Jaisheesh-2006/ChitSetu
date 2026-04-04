package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Jaisheesh-2006/ChitSetu/api"
	"github.com/Jaisheesh-2006/ChitSetu/internal/auction"
	"github.com/Jaisheesh-2006/ChitSetu/internal/auth"
	"github.com/Jaisheesh-2006/ChitSetu/internal/chat"
	"github.com/Jaisheesh-2006/ChitSetu/internal/chitfund"
	"github.com/Jaisheesh-2006/ChitSetu/internal/payments"
	"github.com/Jaisheesh-2006/ChitSetu/internal/ws"
	"github.com/Jaisheesh-2006/ChitSetu/pkg/database"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	loadEnv()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	store, err := database.Connect(ctx, database.LoadConfigFromEnv())
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer func() {
		disconnectCtx, cancelDisconnect := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelDisconnect()
		if err := store.Close(disconnectCtx); err != nil {
			log.Printf("database disconnect failed: %v", err)
		}
	}()

	log.Println("backend startup: mongodb connected")
	indexCtx, cancelIndexes := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelIndexes()
	if err := store.EnsureIndexes(indexCtx); err != nil {
		log.Fatalf("database index bootstrap failed: %v", err)
	}
	authService := auth.NewService(store.Database)
	// 3. Application Services
	Repo := payments.NewRepository(store.Database)
	paymentService := payments.NewService(paymentRepo)
	paymentHandler := payments.NewHandler(paymentService)
	paymentCron := paymentService.StartDailyReminderCron()
	defer paymentCron.Stop()

	wsManager := ws.NewManager()
	auctionRepo := auction.NewRepository(store.Database)
	chitfundRepo := chitfupaymentnd.NewRepository(store.Database)

	// Broadcast participant count only for explicit auction-room joins/leaves.
	wsManager.OnAuctionParticipantChange = func(fundID string, count int) {
		_ = wsManager.Broadcast(fundID, map[string]any{
			"type":  "participants",
			"count": count,
		})
	}

	auctionService := auction.NewService(auctionRepo, wsManager)
	auctionHandler := auction.NewHandler(auctionService, wsManager)

	auctionSchedulerCtx, stopAuctionScheduler := context.WithCancel(context.Background())
	defer stopAuctionScheduler()
	auctionService.StartScheduler(auctionSchedulerCtx)

	chitfundService := chitfund.NewService(chitfundRepo, wsManager)
	chitfundHandler := chitfund.NewHandler(chitfundService)

	chatRepo := chat.NewRepository(store.Database)
	chatHandler := chat.NewHandler(chatRepo, wsManager)
	// Setup router.
	router := api.SetupRouter(store, auctionHandler, authService, chitfundHandler, paymentHandler, chatHandler)
	port := getenvOrDefault("PORT", "8080")
	addr := ":" + port
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("backend startup: listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed to start: %v", err)
		}
	}()

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-sigCtx.Done()
	log.Println("shutdown signal received")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		if closeErr := server.Close(); closeErr != nil {
			log.Printf("force close failed: %v", closeErr)
		}
	}

	log.Println("backend shutdown complete")

}

func getenvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func loadEnv() {
	paths := []string{".env", "../../.env", "../../../.env", "../../../../.env"}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			if err := godotenv.Load(p); err == nil {
				log.Printf("environment loaded from %s", p)
				return
			}
		}
	}

	log.Println("environment file not found, relying on existing process environment")
}
