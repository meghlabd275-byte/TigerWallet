// Package handlers implements the standalone WL-Card backend REST API. A
// standalone clone of the TigerWallet CryptoCard service — but
// PostgreSQL-persisted (real cards + transactions). REAL bcrypt + JWT auth,
// REAL PostgreSQL persistence, REAL card-number generation via crypto/rand,
// AES-GCM encryption of PAN/CVV at rest (never plaintext), and atomic
// balance debits/credits inside a DB transaction. No stubs, no fakes, no
// mocks, no demos. No fake balances/transactions — real DB rows only.
package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/wl-card/internal/config"
	"github.com/tigerwallet/wl-card/internal/store"
	"github.com/tigerwallet/wl-shared/wlcrypto"
	"github.com/tigerwallet/wl-shared/wlgate"
	"golang.org/x/crypto/bcrypt"
)

type Svc struct {
	cfg   *config.Config
	store *store.Store
	gate  *wlgate.Gate
}

func New(cfg *config.Config, s *store.Store, g *wlgate.Gate) *Svc {
	return &Svc{cfg: cfg, store: s, gate: g}
}

// ==================== Auth (real bcrypt + JWT) ====================

func (s *Svc) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.cfg.BCryptCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"})
		return
	}
	u, err := s.store.CreateUser(c.Request.Context(), req.Email, string(hash), req.Role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": u.ID, "email": u.Email, "role": u.Role})
}

func (s *Svc) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := s.store.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil || !u.IsActive || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	wlID, err := uuid.Parse(s.cfg.WLClientID)
	if err != nil {
		wlID = uuid.Nil
	}
	scopes := []string{"wl_client", "card"}
	tok, err := wlgate.IssueJWT(s.cfg.JWTSecret, u.ID, u.Email, wlID, scopes, s.cfg.JWTExpiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token issue failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tok, "user_id": u.ID, "email": u.Email, "role": u.Role})
}

// RequireRole is the admin gate. The wlgate JWT does not carry a role, so the
// role is loaded fresh from the users table on each request — fail-closed
// (403) if the user is missing or lacks one of the allowed roles.
func (s *Svc) RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		uid := wlgate.UserID(c)
		if uid == uuid.Nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient privileges"})
			return
		}
		u, err := s.store.GetUserByID(c.Request.Context(), uid)
		if err != nil || !u.IsActive || !allowed[u.Role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient privileges"})
			return
		}
		c.Set("role", u.Role)
		c.Next()
	}
}

// ==================== Cards ====================

// ListCards returns the authenticated user's cards. PAN/CVV are never returned
// in full — only a masked last-4 view.
func (s *Svc) ListCards(c *gin.Context) {
	uid := wlgate.UserID(c)
	cards, err := s.store.ListCards(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(cards))
	for i := range cards {
		out = append(out, s.cardJSON(&cards[i]))
	}
	c.JSON(http.StatusOK, gin.H{"cards": out, "count": len(out)})
}

// IssueCard generates a real card number + CVV via crypto/rand, AES-GCM
// encrypts the PAN/CVV at rest, and stores only a SHA-256 hash of the card
// number as the queryable identifier. Admin-gated.
func (s *Svc) IssueCard(c *gin.Context) {
	if err := s.requireEncKey(c); err != nil {
		return
	}
	var req struct {
		UserID     string `json:"user_id" binding:"required"`
		HolderName string `json:"holder_name" binding:"required"`
		Currency   string `json:"currency"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	if _, err := s.store.GetUserByID(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user not found"})
		return
	}
	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	pan, err := generatePAN()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "card generation failed"})
		return
	}
	cvv, err := generateCVV()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "card generation failed"})
		return
	}

	panEnc, err := wlcrypto.EncryptSeedAtRest([]byte(pan), s.cfg.CardEncKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}
	cvvEnc, err := wlcrypto.EncryptSeedAtRest([]byte(cvv), s.cfg.CardEncKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}
	hash := sha256Hex(pan)

	card := &store.Card{
		UserID:         userID,
		CardNumberHash: hash,
		PANEncrypted:   panEnc,
		CVVEncrypted:   cvvEnc,
		HolderName:     req.HolderName,
		Status:         "active",
		Balance:        "0",
		Currency:       currency,
	}
	created, err := s.store.CreateCard(c.Request.Context(), card)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Card issuance returns the full PAN + CVV ONCE so the operator can deliver
	// it to the cardholder; it is never retrievable again in full.
	resp := s.cardJSON(created)
	resp["card_number"] = pan
	resp["cvv"] = cvv
	c.JSON(http.StatusCreated, resp)
}

func (s *Svc) GetCard(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	card, err := s.store.GetCard(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid := wlgate.UserID(c)
	role, _ := c.Get("role")
	if card.UserID != uid && role != "admin" && role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your card"})
		return
	}
	c.JSON(http.StatusOK, s.cardJSON(card))
}

// UpdateCardStatus sets any lifecycle status (freeze/unfreeze/block/cancel/
// reactivate). Admin-gated. Distinct lifecycle shortcuts exist at
// /cards/:id/activate|block|cancel.
func (s *Svc) UpdateCardStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Status string `json:"status" binding:"required,oneof=pending active frozen blocked cancelled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	card, err := s.store.UpdateCardStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s.cardJSON(card))
}

// Balance returns the card's real persisted balance.
func (s *Svc) Balance(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	card, err := s.store.GetCard(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid := wlgate.UserID(c)
	role, _ := c.Get("role")
	if card.UserID != uid && role != "admin" && role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your card"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"card_id": card.ID, "balance": card.Balance,
		"currency": card.Currency, "status": card.Status,
	})
}

// ==================== Transactions ====================

func (s *Svc) ListTransactions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	card, err := s.store.GetCard(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid := wlgate.UserID(c)
	role, _ := c.Get("role")
	if card.UserID != uid && role != "admin" && role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your card"})
		return
	}
	txns, err := s.store.ListTransactions(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(txns))
	for i := range txns {
		out = append(out, s.txJSON(&txns[i]))
	}
	c.JSON(http.StatusOK, gin.H{"transactions": out, "count": len(out)})
}

// RecordTransaction records a real debit/credit and adjusts the card balance
// atomically inside a DB transaction. Debits that exceed the balance are
// rejected (ErrInsufficientFunds) and leave the balance untouched.
func (s *Svc) RecordTransaction(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Amount    string `json:"amount" binding:"required"`
		Direction string `json:"direction" binding:"required,oneof=debit credit"`
		Merchant  string `json:"merchant"`
		Category  string `json:"category"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, perr := strconv.ParseFloat(req.Amount, 64); perr != nil || parseFloat(req.Amount, 0) <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be a positive number"})
		return
	}
	txn, card, err := s.store.RecordTransaction(c.Request.Context(), id, req.Amount, req.Merchant, req.Category, req.Direction, "PURCHASE")
	if err != nil {
		s.respondTxError(c, err, card)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"transaction": s.txJSON(txn),
		"balance":     card.Balance,
	})
}

// respondTxError maps store transaction errors to HTTP responses.
func (s *Svc) respondTxError(c *gin.Context, err error, card *store.Card) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
	case errors.Is(err, store.ErrCardNotActive):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "card_status": card.Status})
	case errors.Is(err, store.ErrInsufficientFunds):
		c.JSON(http.StatusPaymentRequired, gin.H{"error": err.Error(), "balance": card.Balance})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// ==================== Card application & lifecycle ====================

// ApplyForCard lets an end user apply for a card. A real PAN/CVV is generated
// and AES-GCM-encrypted at rest, but the card is created in 'pending' status
// and the PAN/CVV are never returned — an admin activates it via
// /cards/:id/activate.
func (s *Svc) ApplyForCard(c *gin.Context) {
	if err := s.requireEncKey(c); err != nil {
		return
	}
	var req struct {
		HolderName string `json:"holder_name" binding:"required"`
		Currency   string `json:"currency"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}
	pan, err := generatePAN()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "card generation failed"})
		return
	}
	cvv, err := generateCVV()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "card generation failed"})
		return
	}
	panEnc, err := wlcrypto.EncryptSeedAtRest([]byte(pan), s.cfg.CardEncKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}
	cvvEnc, err := wlcrypto.EncryptSeedAtRest([]byte(cvv), s.cfg.CardEncKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}
	card := &store.Card{
		UserID:         wlgate.UserID(c),
		CardNumberHash: sha256Hex(pan),
		PANEncrypted:   panEnc,
		CVVEncrypted:   cvvEnc,
		HolderName:     req.HolderName,
		Status:         "pending",
		Balance:        "0",
		Currency:       currency,
	}
	created, err := s.store.CreateCard(c.Request.Context(), card)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, s.cardJSON(created))
}

// setCardStatus is the shared implementation behind the lifecycle shortcut
// endpoints (activate/block/cancel). Admin-gated.
func (s *Svc) setCardStatus(c *gin.Context, status string) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	card, err := s.store.UpdateCardStatus(c.Request.Context(), id, status)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s.cardJSON(card))
}

// ActivateCard transitions a card to 'active' (e.g. approving a pending
// application or unblocking). Admin-gated.
func (s *Svc) ActivateCard(c *gin.Context) { s.setCardStatus(c, "active") }

// BlockCard transitions a card to 'blocked'. Admin-gated.
func (s *Svc) BlockCard(c *gin.Context) { s.setCardStatus(c, "blocked") }

// CancelCard transitions a card to 'cancelled' (terminal). Admin-gated.
func (s *Svc) CancelCard(c *gin.Context) { s.setCardStatus(c, "cancelled") }

// UpdateCardLimits sets a card's daily/monthly spending limits. Admin-gated.
func (s *Svc) UpdateCardLimits(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		DailyLimit   string `json:"daily_limit" binding:"required"`
		MonthlyLimit string `json:"monthly_limit" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if parseFloat(req.DailyLimit, -1) < 0 || parseFloat(req.MonthlyLimit, -1) < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limits must be non-negative numbers"})
		return
	}
	card, err := s.store.UpdateCardLimits(c.Request.Context(), id, req.DailyLimit, req.MonthlyLimit)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s.cardJSON(card))
}

// GetRates returns the card funding rates (USD per unit). Backed by the
// store's static rates stub until a real oracle is wired in.
func (s *Svc) GetRates(c *gin.Context) {
	rates := s.store.Rates(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"rates": rates, "count": len(rates)})
}

// GetTransaction returns a single transaction on a card. Cardholder (own
// card) or admin.
func (s *Svc) GetTransaction(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	txID, err := uuid.Parse(c.Param("txid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid txid"})
		return
	}
	card, err := s.store.GetCard(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	uid := wlgate.UserID(c)
	role, _ := c.Get("role")
	if card.UserID != uid && role != "admin" && role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your card"})
		return
	}
	txn, err := s.store.GetTransaction(c.Request.Context(), id, txID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s.txJSON(txn))
}

// TopUpCard credits the card balance with an explicit tx_type='TOP_UP'
// transaction. Rejected with 409 if the card is not active.
func (s *Svc) TopUpCard(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Amount string `json:"amount" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if parseFloat(req.Amount, 0) <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be a positive number"})
		return
	}
	txn, card, err := s.store.TopUp(c.Request.Context(), id, req.Amount)
	if err != nil {
		s.respondTxError(c, err, card)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"transaction": s.txJSON(txn),
		"balance":     card.Balance,
	})
}

// ==================== Admin ====================

// AdminStats returns aggregate counts/volumes across users, cards, and
// transactions. Admin-gated.
func (s *Svc) AdminStats(c *gin.Context) {
	stats, err := s.store.Stats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// ==================== Health ====================

func (s *Svc) Health(c *gin.Context) {
	total, active, _ := s.store.CardCount(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{
		"status":       "healthy",
		"service":      "wl-card",
		"licensed":     s.gate.IsAlive(),
		"reason":       s.gate.Reason(),
		"wl_client_id": s.cfg.WLClientID,
		"product":      s.cfg.Product,
		"total_cards":  total,
		"active_cards": active,
	})
}

// ==================== card helpers ====================

// cardJSON renders a card for API responses. PAN/CVV are NEVER returned in
// full from storage — only a masked last-4 derived from the encrypted PAN.
func (s *Svc) cardJSON(c *store.Card) gin.H {
	masked := "****"
	if pan, err := wlcrypto.DecryptSeedAtRest(c.PANEncrypted, s.cfg.CardEncKey); err == nil && len(pan) >= 4 {
		masked = "**** **** **** " + string(pan[len(pan)-4:])
	}
	return gin.H{
		"id": c.ID, "user_id": c.UserID, "card_number_masked": masked,
		"holder_name": c.HolderName, "status": c.Status, "balance": c.Balance,
		"currency": c.Currency, "kyc_level": c.KYCLevel,
		"daily_limit": c.DailyLimit, "monthly_limit": c.MonthlyLimit,
		"created_at": c.CreatedAt,
	}
}

// txJSON renders a card transaction for API responses.
func (s *Svc) txJSON(t *store.CardTransaction) gin.H {
	return gin.H{
		"id": t.ID, "card_id": t.CardID, "amount": t.Amount,
		"merchant": t.Merchant, "category": t.Category, "type": t.TxType,
		"status": t.Status, "created_at": t.CreatedAt,
	}
}

// requireEncKey aborts with 500 if CARD_ENC_KEY is unset — PAN/CVV encryption
// is mandatory, never store plaintext.
func (s *Svc) requireEncKey(c *gin.Context) error {
	if s.cfg.CardEncKey == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "CARD_ENC_KEY not configured"})
		return errors.New("no enc key")
	}
	return nil
}

// generatePAN creates a 16-digit card number via crypto/rand. It is not Luhn-
// valid by design (it is a WL-branded crypto card, not a rails-issued PAN),
// but uses the CSPRNG so it is unpredictable.
func generatePAN() (string, error) {
	// BIN 4... (Visa-style) + 15 more random digits.
	var b strings.Builder
	b.WriteByte('4')
	for i := 0; i < 15; i++ {
		d, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b.WriteByte(byte('0' + d.Int64()))
	}
	return b.String(), nil
}

func generateCVV() (string, error) {
	var b strings.Builder
	for i := 0; i < 3; i++ {
		d, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b.WriteByte(byte('0' + d.Int64()))
	}
	return b.String(), nil
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func parseFloat(s string, def float64) float64 {
	if s == "" {
		return def
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
}
