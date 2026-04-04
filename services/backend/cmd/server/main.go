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
	"github.com/Jaisheesh-2006/ChitSetu/internal/wallet"
	"github.com/Jaisheesh-2006/ChitSetu/internal/web3"
	"github.com/Jaisheesh-2006/ChitSetu/internal/ws"
	"github.com/Jaisheesh-2006/ChitSetu/pkg/database"
	"github.com/joho/godotenv"
)
unc main() {
	loadEnv()
	// validateRequiredAuthEnv()

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

	if err := store.EnsureIndexes(indexCtx); err != nil {
		cancelIndexes()
		log.Fatalf("database index bootstrap failed: %v", err)
	}
	cancelIndexes()
	
	// 1. Core Services
	walletRepo := wallet.NewRepository(store.Database)
	walletService := wallet.NewService(walletRepo)
	authService := auth.NewService(store.Database, walletService)

	// 2. Web3 Stack
	var contractService *web3.ContractService

	web3Client, err := web3.NewClient(os.Getenv("WEB3_RPC_URL"), os.Getenv("WEB3_PRIVATE_KEY"))
	if err != nil {
		log.Printf("Web3 disabled: %v", err)
	} else {
		contractService, err = web3.NewContractService(web3Client)
		if err != nil {
			log.Printf("Web3 smart contracts disabled: %v", err)
		} else {
			// One-time max allowance on the specific factory/child if needed.
			// The backend now handles this per-fund anyway, but keeping for compatibility.
			chitFundAddr := os.Getenv("CHIT_CONTRACT_ADDRESS")
			if chitFundAddr != "" {
				contractService.ApproveInfinite(chitFundAddr)
			}
		}
	}

	// 3. Application Services
	Repo := payments.NewRepository(store.Database)
	paymentService := payments.NewService(paymentRepo, contractService, walletService)
	paymentHandler := payments.NewHandler(paymentService)
	paymentCron := paymentService.StartDailyReminderCron()
	defer paymentCron.Stop()

	wsManager := ws.NewManager()
	auctionRepo := auction.NewRepository(store.Database)
	chitfundRepo := chitfupaymentnd.NewRepository(store.Database)
	chatRepo := chat.NewRepository(store.Database)

	// Broadcast participant count only for explicit auction-room joins/leaves.
	wsManager.OnAuctionParticipantChange = func(fundID string, count int) {
		_ = wsManager.Broadcast(fundID, map[string]any{
			"type":  "participants",
			"count": count,
		})
	}

	web3Handlers := api.NewWeb3Handlers(walletService, contractService, chitfundRepo)

	auctionService := auction.NewService(auctionRepo, wsManager, contractService, walletService)
	auctionHandler := auction.NewHandler(auctionService, wsManager)

	auctionSchedulerCtx, stopAuctionScheduler := context.WithCancel(context.Background())
	defer stopAuctionScheduler()
	auctionService.StartScheduler(auctionSchedulerCtx)

	chitfundService := chitfund.NewService(chitfundRepo, contractService, walletService, wsManager)
	chitfundHandler := chitfund.NewHandler(chitfundService)

	chatHandler := chat.NewHandler(chatRepo, wsManager)

	// 4. Setup Router
	router := api.SetupRouter(
		store,
		paymentHandler,
		auctionHandler,
		chitfundHandler,
		chatHandler,
		authService,
		walletService,
		web3Handlers,
	)

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
				return
			}
		}
	}
}

func validateRequiredAuthEnv() {
	required := []string{
		"SUPABASE_URL",
		"SUPABASE_ANON_KEY",
		"SUPABASE_JWT_SECRET",
		"GOOGLE_CLIENT_ID",
		"GOOGLE_CLIENT_SECRET",
		"AUTH_CALLBACK_URL",
	}

	missing := make([]string, 0)
	for _, key := range required {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		panic("missing required environment variables: " + strings.Join(missing, ", "))
	}
}
