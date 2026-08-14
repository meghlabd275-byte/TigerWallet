/**
 * TigerWallet Admin - Export Handler
 * CSV/JSON exports of real PostgreSQL data for users, tokens, withdrawals and
 * transactions. Ports the orphan admin_service's export endpoints onto the
 * canonical GORM-backed admin backend.
 */

package handlers

import (
	"bytes"
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tigerwallet/admin/internal/models"
	"github.com/tigerwallet/admin/pkg/database"
)

// ExportHandler handles data export endpoints
type ExportHandler struct {
	db *database.PostgresDB
}

// NewExportHandler creates a new export handler
func NewExportHandler(db *database.PostgresDB) *ExportHandler {
	return &ExportHandler{db: db}
}

// writeCSV serializes the given rows and sends them as a CSV download.
func writeCSV(c *gin.Context, filename string, header []string, rows [][]string) {
	buf := &bytes.Buffer{}
	w := csv.NewWriter(buf)
	_ = w.Write(header)
	for _, row := range rows {
		_ = w.Write(row)
	}
	w.Flush()

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "text/csv", buf.Bytes())
}

// ExportUsers exports all users as CSV.
func (h *ExportHandler) ExportUsers(c *gin.Context) {
	var users []models.User
	if err := h.db.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	header := []string{"id", "email", "username", "phone", "wallet_address", "status", "kyc_status", "kyc_level", "two_factor_enabled", "referral_code", "referred_by", "white_label_id", "last_login_at", "registration_ip", "risk_score", "created_at"}
	rows := make([][]string, 0, len(users))
	for _, u := range users {
		phone := ""
		if u.Phone.Valid {
			phone = u.Phone.String
		}
		lastLogin := ""
		if u.LastLoginAt != nil {
			lastLogin = u.LastLoginAt.Format(time.RFC3339)
		}
		rows = append(rows, []string{
			strconv.FormatUint(uint64(u.ID), 10),
			u.Email,
			u.Username,
			phone,
			u.WalletAddress,
			u.Status,
			u.KYCStatus,
			strconv.Itoa(u.KYCLevel),
			strconv.FormatBool(u.TwoFactorEnabled),
			u.ReferralCode,
			u.ReferredBy,
			uintPtrToStr(u.WhiteLabelID),
			lastLogin,
			u.RegistrationIP,
			strconv.Itoa(u.RiskScore),
			u.CreatedAt.Format(time.RFC3339),
		})
	}

	writeCSV(c, "users.csv", header, rows)
}

// ExportTokens exports all tokens as CSV.
func (h *ExportHandler) ExportTokens(c *gin.Context) {
	var tokens []models.Token
	if err := h.db.Find(&tokens).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tokens"})
		return
	}

	header := []string{"id", "name", "symbol", "contract_address", "chain", "decimals", "total_supply", "price", "market_cap", "volume_24h", "price_change_24h", "is_active", "is_verified", "listed_by", "listed_at", "created_at"}
	rows := make([][]string, 0, len(tokens))
	for _, t := range tokens {
		listedAt := ""
		if t.ListedAt != nil {
			listedAt = t.ListedAt.Format(time.RFC3339)
		}
		rows = append(rows, []string{
			strconv.FormatUint(uint64(t.ID), 10),
			t.Name,
			t.Symbol,
			t.ContractAddress,
			t.Chain,
			strconv.Itoa(t.Decimals),
			t.TotalSupply,
			t.Price,
			t.MarketCap,
			t.Volume24h,
			t.PriceChange24h,
			strconv.FormatBool(t.IsActive),
			strconv.FormatBool(t.IsVerified),
			strconv.FormatUint(uint64(t.ListedBy), 10),
			listedAt,
			t.CreatedAt.Format(time.RFC3339),
		})
	}

	writeCSV(c, "tokens.csv", header, rows)
}

// ExportWithdrawals exports all withdrawal requests as CSV.
func (h *ExportHandler) ExportWithdrawals(c *gin.Context) {
	var withdrawals []models.Withdrawal
	if err := h.db.Find(&withdrawals).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch withdrawals"})
		return
	}

	header := []string{"id", "user_id", "amount", "token", "chain", "to_address", "status", "approved_at", "approved_by", "rejected_at", "rejected_by", "rejection_reason", "processed_at", "tx_hash", "fee", "ip_address", "created_at"}
	rows := make([][]string, 0, len(withdrawals))
	for _, w := range withdrawals {
		rows = append(rows, []string{
			strconv.FormatUint(uint64(w.ID), 10),
			strconv.FormatUint(uint64(w.UserID), 10),
			w.Amount,
			w.Token,
			w.Chain,
			w.ToAddress,
			w.Status,
			timePtrToStr(w.ApprovedAt),
			uintPtrToStr(w.ApprovedBy),
			timePtrToStr(w.RejectedAt),
			uintPtrToStr(w.RejectedBy),
			w.RejectionReason,
			timePtrToStr(w.ProcessedAt),
			w.TxHash,
			w.Fee,
			w.IPAddress,
			w.CreatedAt.Format(time.RFC3339),
		})
	}

	writeCSV(c, "withdrawals.csv", header, rows)
}

// ExportTransactions exports all transactions as CSV.
func (h *ExportHandler) ExportTransactions(c *gin.Context) {
	var txs []models.Transaction
	if err := h.db.Find(&txs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
		return
	}

	header := []string{"id", "user_id", "hash", "type", "chain", "from_address", "to_address", "amount", "token", "token_amount", "fee", "status", "block_number", "block_hash", "gas_used", "gas_price", "nonce", "timestamp", "flagged", "flag_reason", "created_at"}
	rows := make([][]string, 0, len(txs))
	for _, t := range txs {
		rows = append(rows, []string{
			strconv.FormatUint(uint64(t.ID), 10),
			strconv.FormatUint(uint64(t.UserID), 10),
			t.Hash,
			t.Type,
			t.Chain,
			t.FromAddress,
			t.ToAddress,
			t.Amount,
			t.Token,
			t.TokenAmount,
			t.Fee,
			t.Status,
			strconv.FormatInt(t.BlockNumber, 10),
			t.BlockHash,
			t.GasUsed,
			t.GasPrice,
			strconv.FormatInt(t.Nonce, 10),
			t.Timestamp.Format(time.RFC3339),
			strconv.FormatBool(t.Flagged),
			t.FlagReason,
			t.CreatedAt.Format(time.RFC3339),
		})
	}

	writeCSV(c, "transactions.csv", header, rows)
}

func uintPtrToStr(p *uint) string {
	if p == nil {
		return ""
	}
	return strconv.FormatUint(uint64(*p), 10)
}

func timePtrToStr(p *time.Time) string {
	if p == nil {
		return ""
	}
	return p.Format(time.RFC3339)
}
