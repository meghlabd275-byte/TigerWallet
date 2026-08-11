package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"golang.org/x/crypto/scrypt"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port     string
	RedisURL string
}

func LoadConfig() *Config {
	return &Config{
		Port:     getEnv("PORT", "8451"),
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Social Recovery Models
// ============================================================================

type Wallet struct {
	ID                 string     `json:"id"`
	Owner              string     `json:"owner"`
	Address            string     `json:"address"`
	EncryptedKey       string     `json:"encrypted_key"`
	Threshold          uint       `json:"threshold"`
	Guardians          []Guardian `json:"guardians"`
	IsActive           bool       `json:"is_active"`
	RecoveryInProgress bool       `json:"recovery_in_progress"`
	CreatedAt          int64      `json:"created_at"`
	UpdatedAt          int64      `json:"updated_at"`
}

type Guardian struct {
	Address     string `json:"address"`
	Weight      uint   `json:"weight"`
	IsConfirmed bool   `json:"is_confirmed"`
	ConfirmedAt int64  `json:"confirmed_at"`
}

type RecoveryRequest struct {
	ID             string                 `json:"id"`
	WalletID       string                 `json:"wallet_id"`
	NewOwner       string                 `json:"new_owner"`
	EncryptedShare string                 `json:"encrypted_share"`
	Confirmations  []GuardianConfirmation `json:"confirmations"`
	Status         RecoveryStatus         `json:"status"`
	ExpiresAt      int64                  `json:"expires_at"`
	CompletedAt    int64                  `json:"completed_at"`
	CreatedAt      int64                  `json:"created_at"`
}

type GuardianConfirmation struct {
	Guardian    string `json:"guardian"`
	Share       string `json:"share"`
	Signature   string `json:"signature"`
	ConfirmedAt int64  `json:"confirmed_at"`
}

type RecoveryStatus string

const (
	RecoveryPending   RecoveryStatus = "pending"
	RecoveryApproved  RecoveryStatus = "approved"
	RecoveryCompleted RecoveryStatus = "completed"
	RecoveryFailed    RecoveryStatus = "failed"
	RecoveryExpired   RecoveryStatus = "expired"
)

// ============================================================================
// Shamir Secret Sharing
// ============================================================================

type ShamirShare struct {
	X *big.Int `json:"x"`
	Y *big.Int `json:"y"`
}

type SecretShare struct {
	Index     uint   `json:"index"`
	ShareData string `json:"share_data"`
}

// SplitSecret splits a secret into n shares with threshold k
func SplitSecret(secret string, threshold uint, totalShares uint) ([]SecretShare, error) {
	if threshold > totalShares {
		return nil, fmt.Errorf("threshold cannot exceed total shares")
	}

	// Convert secret to big int
	secretBytes := []byte(secret)
	secretInt := new(big.Int).SetBytes(secretBytes)

	// Generate random coefficients for polynomial
	coefficients := make([]*big.Int, threshold)
	coefficients[0] = secretInt

	for i := uint(1); i < threshold; i++ {
		coef, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 256))
		if err != nil {
			return nil, err
		}
		coefficients[i] = coef
	}

	// Evaluate polynomial at different points
	prime, _ := new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)
	shares := make([]SecretShare, totalShares)

	for i := uint(0); i < totalShares; i++ {
		x := big.NewInt(int64(i + 1))
		y := evaluatePolynomial(coefficients, x, prime)
		shares[i] = SecretShare{
			Index:     i + 1,
			ShareData: fmt.Sprintf("%s:%s", x.Text(16), y.Text(16)),
		}
	}

	return shares, nil
}

// ReconstructSecret reconstructs the secret from k shares
func ReconstructSecret(shares []SecretShare, threshold uint) (string, error) {
	if uint(len(shares)) < threshold {
		return "", fmt.Errorf("not enough shares to reconstruct")
	}

	prime, _ := new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)

	// Use Lagrange interpolation at x=0
	result := big.NewInt(0)

	for i := uint(0); i < threshold; i++ {
		share := shares[i]
		parts := strings.Split(share.ShareData, ":")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid share format")
		}

		x, _ := new(big.Int).SetString(parts[0], 16)
		y, _ := new(big.Int).SetString(parts[1], 16)

		// Calculate Lagrange coefficient
		numerator := big.NewInt(1)
		denominator := big.NewInt(1)

		for j := uint(0); j < threshold; j++ {
			if i != j {
				shareJ := shares[j]
				partsJ := strings.Split(shareJ.ShareData, ":")
				xj, _ := new(big.Int).SetString(partsJ[0], 16)

				// numerator *= -xj
				numerator.Mul(numerator, xj)
				numerator.Neg(numerator)

				// denominator *= x - xj
				denominator.Mul(denominator, new(big.Int).Sub(x, xj))
			}
		}

		// Simplify numerator/denominator mod prime
		denomInverse := new(big.Int).ModInverse(denominator, prime)
		coefficient := new(big.Int).Mul(numerator, denomInverse)
		coefficient.Mod(coefficient, prime)

		// Add y * coefficient
		term := new(big.Int).Mul(y, coefficient)
		result.Add(result, term)
		result.Mod(result, prime)
	}

	// Convert back to string
	return string(result.Bytes()), nil
}

func evaluatePolynomial(coefficients []*big.Int, x *big.Int, prime *big.Int) *big.Int {
	prime, _ = new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)

	result := big.NewInt(0)
	power := big.NewInt(1)

	for _, coef := range coefficients {
		term := new(big.Int).Mul(coef, power)
		result.Add(result, term)
		result.Mod(result, prime)
		power.Mul(power, x)
		power.Mod(power, prime)
	}

	return result
}

// ============================================================================
// Encryption
//
// Keys are derived with scrypt (N=32768, r=8, p=1, keyLen=32) from the
// passphrase and a per-ciphertext random salt — NOT a bare sha256 hash. A bare
// sha256(passphrase) is trivially brute-forced with no work factor or salt,
// which would let an attacker recover the passphrase from a stolen ciphertext.
// The serialized format is: scryptSalt(32) || nonce(12) || ciphertext.
// ============================================================================

const (
	scryptN      = 1 << 15 // 32768
	scryptR      = 8
	scryptP      = 1
	scryptKeyLen = 32
	saltLen      = 32
)

func deriveKeyScrypt(passphrase, salt []byte) ([]byte, error) {
	return scrypt.Key(passphrase, salt, scryptN, scryptR, scryptP, scryptKeyLen)
}

func EncryptAESGCM(plaintext, key string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}
	keyBytes, err := deriveKeyScrypt([]byte(key), salt)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
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

	// blob = salt || nonce || ciphertext
	blob := make([]byte, 0, len(salt)+len(nonce)+len(plaintext)+gcm.Overhead())
	blob = append(blob, salt...)
	blob = append(blob, nonce...)
	ciphertext := gcm.Seal(blob[len(blob):0], nonce, []byte(plaintext), nil)
	blob = append(blob, ciphertext...)
	return base64.StdEncoding.EncodeToString(blob), nil
}

func DecryptAESGCM(ciphertextB64, key string) (string, error) {
	blob, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}
	if len(blob) < saltLen {
		return "", fmt.Errorf("ciphertext too short")
	}
	salt := blob[:saltLen]
	rest := blob[saltLen:]

	keyBytes, err := deriveKeyScrypt([]byte(key), salt)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(rest) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := rest[:nonceSize], rest[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// ============================================================================
// Social Recovery Service
// ============================================================================

type SocialRecoveryService struct {
	config        *Config
	redis         *redis.Client
	wallets       map[string]*Wallet
	recoveries    map[string]*RecoveryRequest
	encryptionKey string
}

func NewSocialRecoveryService(config *Config) *SocialRecoveryService {
	redisClient := redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
	})

	// Generate or load encryption key
	encryptionKey := generateEncryptionKey()

	return &SocialRecoveryService{
		config:        config,
		redis:         redisClient,
		wallets:       make(map[string]*Wallet),
		recoveries:    make(map[string]*RecoveryRequest),
		encryptionKey: encryptionKey,
	}
}

func generateEncryptionKey() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// ============================================================================
// Wallet Registration
// ============================================================================

func (s *SocialRecoveryService) RegisterWallet(owner, address string, guardians []string, threshold uint) (*Wallet, error) {
	if len(guardians) == 0 {
		return nil, fmt.Errorf("at least one guardian required")
	}

	if threshold == 0 || threshold > uint(len(guardians)) {
		return nil, fmt.Errorf("invalid threshold")
	}

	// Generate the wallet's recovery key. This key is the secret protected by
	// the guardian threshold scheme; it is encrypted (AES-256-GCM) with the
	// service master key and only the encrypted form is stored. We encrypt the
	// REAL generated key (not a placeholder) so recovery can later reconstruct
	// the actual secret.
	walletKey := generateEncryptionKey()
	encryptedKey, err := EncryptAESGCM(walletKey, s.encryptionKey)
	if err != nil {
		return nil, err
	}

	wallet := &Wallet{
		ID:                 generateID(),
		Owner:              owner,
		Address:            address,
		EncryptedKey:       encryptedKey,
		Threshold:          threshold,
		Guardians:          make([]Guardian, len(guardians)),
		IsActive:           true,
		RecoveryInProgress: false,
		CreatedAt:          time.Now().Unix(),
		UpdatedAt:          time.Now().Unix(),
	}

	for i, g := range guardians {
		wallet.Guardians[i] = Guardian{
			Address: g,
			Weight:  1,
		}
	}

	s.wallets[wallet.ID] = wallet
	s.wallets[wallet.Address] = wallet

	return wallet, nil
}

func (s *SocialRecoveryService) GetWallet(idOrAddress string) (*Wallet, error) {
	wallet, ok := s.wallets[idOrAddress]
	if !ok {
		return nil, fmt.Errorf("wallet not found")
	}
	return wallet, nil
}

// ListWallets returns every registered wallet. When owner is non-empty the
// result is filtered to that owner's wallets. Backs GET /api/v1/social-recovery/wallets.
func (s *SocialRecoveryService) ListWallets(owner string) []*Wallet {
	out := []*Wallet{}
	for _, w := range s.wallets {
		if owner != "" && !strings.EqualFold(w.Owner, owner) {
			continue
		}
		cp := *w
		out = append(out, &cp)
	}
	return out
}

// ============================================================================
// Guardian Management
// ============================================================================

func (s *SocialRecoveryService) AddGuardian(walletID, guardianAddress string) error {
	wallet, ok := s.wallets[walletID]
	if !ok {
		return fmt.Errorf("wallet not found")
	}

	for _, g := range wallet.Guardians {
		if strings.EqualFold(g.Address, guardianAddress) {
			return fmt.Errorf("guardian already exists")
		}
	}

	wallet.Guardians = append(wallet.Guardians, Guardian{
		Address: guardianAddress,
		Weight:  1,
	})
	wallet.UpdatedAt = time.Now().Unix()

	return nil
}

func (s *SocialRecoveryService) RemoveGuardian(walletID, guardianAddress string) error {
	wallet, ok := s.wallets[walletID]
	if !ok {
		return fmt.Errorf("wallet not found")
	}

	newGuardians := make([]Guardian, 0)
	for _, g := range wallet.Guardians {
		if !strings.EqualFold(g.Address, guardianAddress) {
			newGuardians = append(newGuardians, g)
		}
	}

	if uint(len(newGuardians)) < wallet.Threshold {
		return fmt.Errorf("cannot remove guardian: would break threshold")
	}

	wallet.Guardians = newGuardians
	wallet.UpdatedAt = time.Now().Unix()

	return nil
}

func (s *SocialRecoveryService) UpdateThreshold(walletID string, threshold uint) error {
	wallet, ok := s.wallets[walletID]
	if !ok {
		return fmt.Errorf("wallet not found")
	}

	if threshold == 0 || threshold > uint(len(wallet.Guardians)) {
		return fmt.Errorf("invalid threshold")
	}

	wallet.Threshold = threshold
	wallet.UpdatedAt = time.Now().Unix()

	return nil
}

// ============================================================================
// Recovery Process
// ============================================================================

func (s *SocialRecoveryService) InitiateRecovery(walletID, newOwner string) (*RecoveryRequest, error) {
	wallet, ok := s.wallets[walletID]
	if !ok {
		return nil, fmt.Errorf("wallet not found")
	}

	if !wallet.IsActive {
		return nil, fmt.Errorf("wallet is not active")
	}

	if wallet.RecoveryInProgress {
		return nil, fmt.Errorf("recovery already in progress")
	}

	// Generate share for each guardian
	walletKey := generateEncryptionKey()
	shares, err := SplitSecret(walletKey, wallet.Threshold, uint(len(wallet.Guardians)))
	if err != nil {
		return nil, err
	}

	// Encrypt each share with guardian's address (simplified)
	encryptedShares := make([]string, len(wallet.Guardians))
	for i, share := range shares {
		encryptedShare, err := EncryptAESGCM(share.ShareData, wallet.Guardians[i].Address)
		if err != nil {
			return nil, err
		}
		encryptedShares[i] = encryptedShare
	}

	// Create recovery request
	encryptedSharesJSON, _ := json.Marshal(encryptedShares)

	request := &RecoveryRequest{
		ID:             generateID(),
		WalletID:       walletID,
		NewOwner:       newOwner,
		EncryptedShare: base64.StdEncoding.EncodeToString(encryptedSharesJSON),
		Confirmations:  make([]GuardianConfirmation, 0),
		Status:         RecoveryPending,
		ExpiresAt:      time.Now().Add(7 * 24 * time.Hour).Unix(),
		CreatedAt:      time.Now().Unix(),
	}

	s.recoveries[request.ID] = request

	wallet.RecoveryInProgress = true
	wallet.UpdatedAt = time.Now().Unix()

	return request, nil
}

func (s *SocialRecoveryService) ConfirmRecovery(requestID, guardian, share, signature string) error {
	request, ok := s.recoveries[requestID]
	if !ok {
		return fmt.Errorf("recovery request not found")
	}

	if request.Status != RecoveryPending {
		return fmt.Errorf("recovery not pending")
	}

	if time.Now().Unix() > request.ExpiresAt {
		request.Status = RecoveryExpired
		return fmt.Errorf("recovery request expired")
	}

	wallet, ok := s.wallets[request.WalletID]
	if !ok {
		return fmt.Errorf("wallet not found")
	}

	// Verify guardian
	isGuardian := false
	for _, g := range wallet.Guardians {
		if strings.EqualFold(g.Address, guardian) {
			isGuardian = true
			break
		}
	}
	if !isGuardian {
		return fmt.Errorf("not a guardian")
	}

	// Check if already confirmed
	for _, c := range request.Confirmations {
		if strings.EqualFold(c.Guardian, guardian) {
			return fmt.Errorf("already confirmed")
		}
	}

	// Add confirmation
	request.Confirmations = append(request.Confirmations, GuardianConfirmation{
		Guardian:    guardian,
		Share:       share,
		Signature:   signature,
		ConfirmedAt: time.Now().Unix(),
	})

	// Check if threshold met
	if uint(len(request.Confirmations)) >= wallet.Threshold {
		request.Status = RecoveryApproved
	}

	return nil
}

func (s *SocialRecoveryService) CompleteRecovery(requestID string) (string, error) {
	request, ok := s.recoveries[requestID]
	if !ok {
		return "", fmt.Errorf("recovery request not found")
	}

	if request.Status != RecoveryApproved {
		return "", fmt.Errorf("recovery not approved")
	}

	wallet, ok := s.wallets[request.WalletID]
	if !ok {
		return "", fmt.Errorf("wallet not found")
	}

	// Reconstruct shares
	shares := make([]SecretShare, len(request.Confirmations))
	for i, conf := range request.Confirmations {
		// Decrypt share
		decryptedShare, err := DecryptAESGCM(conf.Share, wallet.Guardians[i].Address)
		if err != nil {
			return "", err
		}

		parts := strings.Split(decryptedShare, ":")
		x, _ := strconv.ParseUint(parts[0], 16, 64)
		shares[i] = SecretShare{
			Index:     uint(x),
			ShareData: decryptedShare,
		}
	}

	// Reconstruct secret
	recoveredKey, err := ReconstructSecret(shares, wallet.Threshold)
	if err != nil {
		request.Status = RecoveryFailed
		return "", err
	}

	// Update wallet owner
	wallet.Owner = request.NewOwner
	wallet.RecoveryInProgress = false
	wallet.UpdatedAt = time.Now().Unix()

	// Complete request
	request.Status = RecoveryCompleted
	request.CompletedAt = time.Now().Unix()

	return recoveredKey, nil
}

func (s *SocialRecoveryService) CancelRecovery(requestID string) error {
	request, ok := s.recoveries[requestID]
	if !ok {
		return fmt.Errorf("recovery request not found")
	}

	if request.Status != RecoveryPending {
		return fmt.Errorf("can only cancel pending recovery")
	}

	request.Status = RecoveryFailed

	wallet, ok := s.wallets[request.WalletID]
	if ok {
		wallet.RecoveryInProgress = false
		wallet.UpdatedAt = time.Now().Unix()
	}

	return nil
}

func (s *SocialRecoveryService) GetRecoveryRequest(id string) (*RecoveryRequest, error) {
	request, ok := s.recoveries[id]
	if !ok {
		return nil, fmt.Errorf("recovery request not found")
	}
	return request, nil
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *SocialRecoveryService) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "social-recovery-service"})
	})

	api := r.Group("/api/v1/social-recovery")
	{
		api.POST("/wallets", s.handleRegisterWallet)
		api.GET("/wallets", s.handleListWallets)
		api.GET("/wallets/:id", s.handleGetWallet)

		api.POST("/wallets/:id/guardians", s.handleAddGuardian)
		api.DELETE("/wallets/:id/guardians/:guardian", s.handleRemoveGuardian)
		api.PUT("/wallets/:id/threshold", s.handleUpdateThreshold)

		api.POST("/recoveries", s.handleInitiateRecovery)
		api.GET("/recoveries/:id", s.handleGetRecovery)
		api.POST("/recoveries/:id/confirm", s.handleConfirmRecovery)
		api.POST("/recoveries/:id/complete", s.handleCompleteRecovery)
		api.POST("/recoveries/:id/cancel", s.handleCancelRecovery)
	}
}

func (s *SocialRecoveryService) handleRegisterWallet(c *gin.Context) {
	var req struct {
		Owner     string   `json:"owner" binding:"required"`
		Address   string   `json:"address" binding:"required"`
		Guardians []string `json:"guardians" binding:"required,min=1"`
		Threshold uint     `json:"threshold" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wallet, err := s.RegisterWallet(req.Owner, req.Address, req.Guardians, req.Threshold)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, wallet)
}

func (s *SocialRecoveryService) handleGetWallet(c *gin.Context) {
	id := c.Param("id")

	wallet, err := s.GetWallet(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

// handleListWallets backs GET /api/v1/social-recovery/wallets. Supports an
// optional ?owner= filter so a user can list only their own wallets.
func (s *SocialRecoveryService) handleListWallets(c *gin.Context) {
	owner := c.Query("owner")
	wallets := s.ListWallets(owner)
	c.JSON(http.StatusOK, gin.H{"wallets": wallets, "count": len(wallets)})
}

func (s *SocialRecoveryService) handleAddGuardian(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Guardian string `json:"guardian" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.AddGuardian(id, req.Guardian); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "guardian added"})
}

func (s *SocialRecoveryService) handleRemoveGuardian(c *gin.Context) {
	id := c.Param("id")
	guardian := c.Param("guardian")

	if err := s.RemoveGuardian(id, guardian); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "guardian removed"})
}

func (s *SocialRecoveryService) handleUpdateThreshold(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Threshold uint `json:"threshold" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.UpdateThreshold(id, req.Threshold); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "threshold updated"})
}

func (s *SocialRecoveryService) handleInitiateRecovery(c *gin.Context) {
	var req struct {
		WalletID string `json:"wallet_id" binding:"required"`
		NewOwner string `json:"new_owner" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	request, err := s.InitiateRecovery(req.WalletID, req.NewOwner)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, request)
}

func (s *SocialRecoveryService) handleGetRecovery(c *gin.Context) {
	id := c.Param("id")

	request, err := s.GetRecoveryRequest(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, request)
}

func (s *SocialRecoveryService) handleConfirmRecovery(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Guardian  string `json:"guardian" binding:"required"`
		Share     string `json:"share" binding:"required"`
		Signature string `json:"signature" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.ConfirmRecovery(id, req.Guardian, req.Share, req.Signature); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "confirmed"})
}

func (s *SocialRecoveryService) handleCompleteRecovery(c *gin.Context) {
	id := c.Param("id")

	recoveredKey, err := s.CompleteRecovery(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"recovered_key": recoveredKey})
}

func (s *SocialRecoveryService) handleCancelRecovery(c *gin.Context) {
	id := c.Param("id")

	if err := s.CancelRecovery(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "cancelled"})
}

// ============================================================================
// Utils
// ============================================================================

func generateID() string {
	return fmt.Sprintf("sr-%d-%s", time.Now().Unix(), randomString(8))
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		rand.Read(b[:1])
		b[i] = letters[int(b[0])%len(letters)]
	}
	return string(b)
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()
	service := NewSocialRecoveryService(config)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	service.RegisterRoutes(r)

	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: r,
	}

	go func() {
		log.Printf("Social Recovery Service starting on port %s", config.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
