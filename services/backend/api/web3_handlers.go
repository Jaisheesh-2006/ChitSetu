package api

import (
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/Jaisheesh-2006/ChitSetu/internal/chitfund"

	"github.com/Jaisheesh-2006/ChitSetu/internal/wallet"
	"github.com/Jaisheesh-2006/ChitSetu/web3"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/gin-gonic/gin"
)

type Web3Handlers struct {
	walletService   *wallet.Service
	contractService *web3.ContractService
	fundRepo        *chitfund.Repository
}

func NewWeb3Handlers(ws *wallet.Service, cs *web3.ContractService, fr *chitfund.Repository) *Web3Handlers {
	return &Web3Handlers{
		walletService:   ws,
		contractService: cs,
		fundRepo:        fr,
	}
}

func (h *Web3Handlers) CreateWallet(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	address, err := h.walletService.CreateWallet(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"address": address})
}

func (h *Web3Handlers) GetWalletInfo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	address, _, err := h.walletService.GetWalletByUserID(c.Request.Context(), userID.(string))
	if err != nil {
		// Auto-provision wallet if it doesn't exist
		newAddr, createErr := h.walletService.CreateWallet(c.Request.Context(), userID.(string))
		if createErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to auto-provision wallet: " + createErr.Error()})
			return
		}
		address = newAddr
	}

	balance := big.NewInt(0)
	if h.contractService != nil {
		bal, err := h.contractService.GetTokenBalance(c.Request.Context(), address)
		if err == nil {
			balance = bal
		}
	}

	// Format balance: Wei (10^18) to human readable
	fbal := new(big.Float).SetInt(balance)
	divider := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	humanBalance, _ := new(big.Float).Quo(fbal, divider).Float64()

	c.JSON(http.StatusOK, gin.H{
		"address": address,
		"balance": humanBalance,
	})
}

func (h *Web3Handlers) MintTokens(c *gin.Context) {
	if h.contractService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "web3 service not initialized"})
		return
	}

	var req struct {
		Amount uint64 `json:"amount"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if req.Amount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be greater than 0"})
		return
	}

	walletAddr, _, err := h.walletService.GetWalletByUserID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "wallet not found"})
		return
	}

	// API accepts whole CHIT units; convert to 18-decimal base units for ERC20 mint.
	amountWei := web3.INRToWei(float64(req.Amount))
	txHash, err := h.contractService.MintTokens(c.Request.Context(), walletAddr, amountWei)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	status := "pending"
	if receipt, waitErr := h.contractService.WaitForReceiptWithTimeout(txHash, 10*time.Second); waitErr == nil && receipt != nil && receipt.Status == types.ReceiptStatusSuccessful {
		status = "confirmed"
	}

	c.JSON(http.StatusOK, gin.H{"tx_hash": txHash, "status": status})
}

func (h *Web3Handlers) JoinFund(c *gin.Context) {
	var req struct {
		FundAddress string `json:"fund_address"`
	}
	_ = c.ShouldBindJSON(&req)

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	walletAddr, _, err := h.walletService.GetWalletByUserID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "wallet not found"})
		return
	}

	txHash, err := h.contractService.JoinFund(c.Request.Context(), req.FundAddress, walletAddr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tx_hash": txHash, "status": "pending"})
}

func (h *Web3Handlers) DepositContribution(c *gin.Context) {
	var req struct {
		FundAddress string `json:"fund_address"`
	}
	_ = c.ShouldBindJSON(&req)

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	walletAddr, _, err := h.walletService.GetWalletByUserID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "wallet not found"})
		return
	}

	txHash, err := h.contractService.DepositContribution(c.Request.Context(), req.FundAddress, walletAddr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tx_hash": txHash, "status": "pending"})
}

func (h *Web3Handlers) FinalizeAuction(c *gin.Context) {
	var req struct {
		FundAddress   string `json:"fund_address"`
		WinnerAddress string `json:"winner_address"`
		Discount      uint64 `json:"discount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	discountWei := web3.INRToWei(float64(req.Discount))
	txHash, err := h.contractService.FinalizeAuction(c.Request.Context(), req.FundAddress, req.WinnerAddress, discountWei)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tx_hash": txHash, "status": "pending"})
}

func (h *Web3Handlers) ApproveToken(c *gin.Context) {
	var req struct {
		FundAddress string `json:"fund_address"`
	}
	_ = c.ShouldBindJSON(&req)

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	_, privKey, err := h.walletService.GetWalletByUserID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "wallet not found"})
		return
	}

	txHash, err := h.contractService.ApproveFundForUser(c.Request.Context(), privKey, req.FundAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "approved", "tx_hash": txHash, "fund": req.FundAddress})
}

func (h *Web3Handlers) CheckTxStatus(c *gin.Context) {
	txHash := c.Query("hash")
	if txHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hash query parameter required"})
		return
	}

	status, blockNumber, err := h.contractService.GetTransactionStatus(txHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tx_hash":      txHash,
		"status":       status,
		"block_number": blockNumber,
	})
}
func (h *Web3Handlers) GetWalletHistory(c *gin.Context) {
	address := c.Param("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address is required"})
		return
	}

	// Retrieve user ID for cycle lookup if available
	tokenUserID, _ := c.Get("user_id")

	if h.contractService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "web3 service not initialized"})
		return
	}

	transfers, err := h.contractService.GetTokenTransfers(c.Request.Context(), address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Enrich transfers for premium 'user experience'
	type EnrichedTransfer struct {
		web3.TokenTransfer
		CounterpartyName string `json:"counterparty_name,omitempty"`
		Description      string `json:"description,omitempty"`
	}

	enriched := make([]EnrichedTransfer, len(transfers))
	for i, t := range transfers {
		enriched[i] = EnrichedTransfer{TokenTransfer: t}

		otherAddr := t.To
		if t.Type == "credit" {
			otherAddr = t.From
		}

		// Try to find the fund by contract address
		if h.fundRepo != nil {
			fund, _ := h.fundRepo.GetFundByContractAddress(c.Request.Context(), otherAddr)
			if fund != nil {
				enriched[i].CounterpartyName = fund.Name
				if t.Type == "debit" {
					enriched[i].Description = "Contribution to " + fund.Name

					// Deep lookup: Match cycle number by amount and user
					if uid, ok := tokenUserID.(string); ok && uid != "" {
						contrib, _ := h.fundRepo.GetContributionByAmount(c.Request.Context(), fund.ID, uid, t.Value)
						if contrib != nil {
							enriched[i].Description = fmt.Sprintf("Cycle %d Contribution: %s", contrib.CycleNumber, fund.Name)
						}
					}
				} else {
					enriched[i].Description = "Auction Win from " + fund.Name
				}
			} else if otherAddr == h.contractService.Client().FromAddress() {
				enriched[i].CounterpartyName = "ChitSetu Manager"
				enriched[i].Description = "System Token Mint"
			} else {
				enriched[i].CounterpartyName = otherAddr[:6] + "..." + otherAddr[len(otherAddr)-4:]
				enriched[i].Description = "External Transfer"
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"address": address,
		"history": enriched,
	})
}
