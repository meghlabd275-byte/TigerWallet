/**
 * TigerWallet Admin - Crypto Cards Handler
 * Backend implementation for crypto card management.
 *
 * Card numbers (PANs) are generated with crypto/rand and a Luhn check digit,
 * stored AES-256-GCM encrypted, and never returned in full: every read path
 * emits a masked last-4 view. Full PANs are only decrypted for the issuing
 * processor, not for the admin UI.
 */

package handlers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/tigerwallet/admin/internal/models"
)

// cryptoCardIssuerBIN is the 6-digit issuer identification number prefix for
// TigerWallet-issued virtual cards.
const cryptoCardIssuerBIN = "453219"

// cryptoCardPANLength is the total number of digits in a generated PAN,
// including the trailing Luhn check digit.
const cryptoCardPANLength = 16

type CryptoCardHandler struct {
	db        *gorm.DB
	encrypter *cardPANEncrypter
}

// NewCryptoCardHandler builds the handler. When CARD_PAN_ENCRYPTION_KEY is set
// the handler stores PANs encrypted; when it is absent the handler refuses card
// creation rather than persisting a PAN in the clear.
func NewCryptoCardHandler(db *gorm.DB) *CryptoCardHandler {
	return &CryptoCardHandler{db: db, encrypter: newCardPANEncrypter(os.Getenv("CARD_PAN_ENCRYPTION_KEY"))}
}

// CryptoCard is the persisted card record (models.CryptoCard). CardNumber holds
// the encrypted PAN and is never serialised to JSON.
type CryptoCard = models.CryptoCard

// cardView is the API representation of a card: the PAN is always masked.
type cardView struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"user_id"`
	UserName  string    `json:"user_name"`
	MaskedPAN string    `json:"card_number_masked"`
	Currency  string    `json:"currency"`
	Balance   float64   `json:"balance"`
	Limit     float64   `json:"limit"`
	Status    string    `json:"status"`
	CardType  string    `json:"card_type"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// toView renders a card for API responses, masking the PAN. If the PAN cannot
// be decrypted (rotated key, corrupt row) the mask degrades to a fixed
// placeholder instead of leaking ciphertext.
func (h *CryptoCardHandler) toView(card CryptoCard) cardView {
	masked := "**** **** **** ****"
	if pan, err := h.encrypter.decrypt(card.CardNumber); err == nil {
		masked = maskPAN(pan)
	}
	return cardView{
		ID:        card.ID,
		UserID:    card.UserID,
		UserName:  card.UserName,
		MaskedPAN: masked,
		Currency:  card.Currency,
		Balance:   card.Balance,
		Limit:     card.Limit,
		Status:    card.Status,
		CardType:  card.CardType,
		CreatedAt: card.CreatedAt,
		UpdatedAt: card.UpdatedAt,
	}
}

// GetAll handles GET /crypto-cards
func (h *CryptoCardHandler) GetAll(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 20
	}
	offset := (page - 1) * limit

	query := h.db.Model(&CryptoCard{})
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count cards"})
		return
	}

	var cards []CryptoCard
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&cards).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cards"})
		return
	}

	views := make([]cardView, 0, len(cards))
	for _, card := range cards {
		views = append(views, h.toView(card))
	}

	c.JSON(http.StatusOK, gin.H{
		"data": views,
		"meta": gin.H{"total": total, "page": page, "limit": limit},
	})
}

// GetByID handles GET /crypto-cards/:id
func (h *CryptoCardHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid card id"})
		return
	}

	var card CryptoCard
	if err := h.db.First(&card, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch card"})
		return
	}

	c.JSON(http.StatusOK, h.toView(card))
}

// Create handles POST /crypto-cards
func (h *CryptoCardHandler) Create(c *gin.Context) {
	var req struct {
		UserID   uint    `json:"user_id"`
		UserName string  `json:"user_name"`
		Currency string  `json:"currency" binding:"required"`
		Limit    float64 `json:"limit"`
		CardType string  `json:"card_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid card payload"})
		return
	}
	if req.CardType == "" {
		req.CardType = "virtual"
	}
	if req.CardType != "virtual" && req.CardType != "physical" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "card_type must be virtual or physical"})
		return
	}
	if req.Limit < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must not be negative"})
		return
	}
	if !h.encrypter.configured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "card issuance is disabled: CARD_PAN_ENCRYPTION_KEY is not configured",
		})
		return
	}

	pan, err := generatePAN()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate card number"})
		return
	}

	encryptedPAN, err := h.encrypter.encrypt(pan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to secure card number"})
		return
	}

	card := CryptoCard{
		UserID:     req.UserID,
		UserName:   req.UserName,
		CardNumber: encryptedPAN,
		Currency:   strings.ToUpper(strings.TrimSpace(req.Currency)),
		Limit:      req.Limit,
		Status:     "pending",
		CardType:   req.CardType,
	}

	if err := h.db.Create(&card).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create card"})
		return
	}

	c.JSON(http.StatusCreated, h.toView(card))
}

// Block handles POST /crypto-cards/:id/block
func (h *CryptoCardHandler) Block(c *gin.Context) {
	h.setStatus(c, "blocked", "card blocked successfully")
}

// Activate handles POST /crypto-cards/:id/activate
func (h *CryptoCardHandler) Activate(c *gin.Context) {
	h.setStatus(c, "active", "card activated successfully")
}

func (h *CryptoCardHandler) setStatus(c *gin.Context, status, message string) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid card id"})
		return
	}

	result := h.db.Model(&CryptoCard{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update card status"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": message, "status": status})
}

// SetLimit handles PUT /crypto-cards/:id/limit
func (h *CryptoCardHandler) SetLimit(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid card id"})
		return
	}

	var input struct {
		Limit float64 `json:"limit" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit payload"})
		return
	}
	if input.Limit < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must not be negative"})
		return
	}

	result := h.db.Model(&CryptoCard{}).Where("id = ?", id).Update("limit", input.Limit)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update limit"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "limit updated successfully", "limit": input.Limit})
}

// generatePAN builds a Luhn-valid card number using crypto/rand.
func generatePAN() (string, error) {
	digits := make([]byte, 0, cryptoCardPANLength)
	digits = append(digits, cryptoCardIssuerBIN...)

	randomLen := cryptoCardPANLength - len(cryptoCardIssuerBIN) - 1
	buf := make([]byte, randomLen)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("entropy unavailable: %w", err)
	}
	for _, b := range buf {
		digits = append(digits, byte('0'+b%10))
	}

	return string(digits) + strconv.Itoa(luhnCheckDigit(string(digits))), nil
}

// luhnCheckDigit computes the check digit that makes partial Luhn-valid.
func luhnCheckDigit(partial string) int {
	sum := 0
	double := true // the check digit position means the rightmost payload digit is doubled

	for i := len(partial) - 1; i >= 0; i-- {
		d := int(partial[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}

	return (10 - (sum % 10)) % 10
}

// luhnValid reports whether a full card number satisfies the Luhn checksum.
func luhnValid(number string) bool {
	if number == "" {
		return false
	}
	sum := 0
	double := false

	for i := len(number) - 1; i >= 0; i-- {
		if number[i] < '0' || number[i] > '9' {
			return false
		}
		d := int(number[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}

	return sum%10 == 0
}

// maskPAN renders a card number as "**** **** **** 1234".
func maskPAN(pan string) string {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, pan)

	if len(digits) < 4 {
		return "**** **** **** ****"
	}
	return "**** **** **** " + digits[len(digits)-4:]
}

// cardPANEncrypter seals PANs with AES-256-GCM under a key derived from the
// configured secret. A missing secret means card storage is unavailable.
type cardPANEncrypter struct {
	key []byte
}

func newCardPANEncrypter(secret string) *cardPANEncrypter {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return &cardPANEncrypter{}
	}
	sum := sha256.Sum256([]byte("tigerwallet:card-pan:v1|" + secret))
	return &cardPANEncrypter{key: sum[:]}
}

func (e *cardPANEncrypter) configured() bool {
	return e != nil && len(e.key) == 32
}

func (e *cardPANEncrypter) encrypt(plaintext string) (string, error) {
	if !e.configured() {
		return "", errors.New("card PAN encryption key is not configured")
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func (e *cardPANEncrypter) decrypt(ciphertext string) (string, error) {
	if !e.configured() {
		return "", errors.New("card PAN encryption key is not configured")
	}

	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}

	nonce, sealed := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
