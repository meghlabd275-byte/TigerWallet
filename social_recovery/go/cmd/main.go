package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ============ Configuration ============

type Config struct {
	Port            string
	RedisURL        string
	AdminPrivateKey string
	RecoveryDelay   time.Duration
	MaxGuardians    int
	MinGuardians    int
	EmailEnabled    bool
	SMSEnabled      bool
	// Guardian notification delivery. Invitations and recovery notices are only
	// sent when the matching channel is configured; see guardian.go.
	SMTPHost        string
	SMTPPort        int
	SMTPUsername    string
	SMTPPassword    string
	SMTPFrom        string
	SMSWebhookURL   string
	SMSWebhookToken string
	// On-chain recovery. Without all three the service can approve a
	// recovery but will not claim one has happened; see ExecuteRecovery.
	RPCURL             string
	RecoveryContract   string
	ExecutorPrivateKey string
	ChainID            int64
}

// parseChainID reads the configured chain id, falling back to 1 so a typo
// cannot silently target a different network.
func parseChainID(raw string) int64 {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value <= 0 {
		log.Printf("SOCIAL_RECOVERY_CHAIN_ID %q is not a valid chain id; using 1", raw)
		return 1
	}
	return value
}

// parseInt parses an integer, falling back to def on any error.
func parseInt(raw string, def int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
		return n
	}
	return def
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============ Models ============

type Wallet struct {
	ID            string     `json:"id"`
	Address       string     `json:"address"`
	Chain         string     `json:"chain"`
	Guardians     []Guardian `json:"guardians"`
	Threshold     int        `json:"threshold"`
	Status        string     `json:"status"`
	RecoveryCount int        `json:"recoveryCount"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type Guardian struct {
	ID          string       `json:"id"`
	Address     string       `json:"address"` // Ethereum address
	Email       string       `json:"email,omitempty"`
	Phone       string       `json:"phone,omitempty"`
	Type        GuardianType `json:"type"`
	Confirmed   bool         `json:"confirmed"`
	InvitedAt   time.Time    `json:"invitedAt"`
	ConfirmedAt *time.Time   `json:"confirmedAt,omitempty"`
}

type GuardianType string

const (
	GuardianTypeAddress GuardianType = "address"
	GuardianTypeEmail   GuardianType = "email"
	GuardianTypePhone   GuardianType = "phone"
	GuardianTypeSocial  GuardianType = "social"
)

type RecoveryRequest struct {
	ID              string              `json:"id"`
	WalletID        string              `json:"walletId"`
	NewOwnerAddress string              `json:"newOwnerAddress"`
	Status          RecoveryStatus      `json:"status"`
	ThresholdMet    bool                `json:"thresholdMet"`
	Signatures      []GuardianSignature `json:"signatures"`
	DelayEnd        time.Time           `json:"delayEnd"`
	ExecutedAt      *time.Time          `json:"executedAt,omitempty"`
	ExecutedTxHash  string              `json:"executedTxHash,omitempty"`
	Challenge       string              `json:"challenge,omitempty"`
	CancelledAt     *time.Time          `json:"cancelledAt,omitempty"`
	CreatedAt       time.Time           `json:"createdAt"`
	ExpiresAt       time.Time           `json:"expiresAt"`
}

type RecoveryStatus string

const (
	RecoveryPending   RecoveryStatus = "pending"
	RecoveryApproved  RecoveryStatus = "approved"
	RecoveryExecuting RecoveryStatus = "executing"
	RecoveryCompleted RecoveryStatus = "completed"
	RecoveryFailed    RecoveryStatus = "failed"
	RecoveryCancelled RecoveryStatus = "cancelled"
)

type GuardianSignature struct {
	GuardianID string    `json:"guardianId"`
	Signature  string    `json:"signature"`
	SignedAt   time.Time `json:"signedAt"`
	Message    string    `json:"message"`
}

type RecoverySession struct {
	WalletID    string    `json:"walletId"`
	Challenge   string    `json:"challenge"`
	ExpiresAt   time.Time `json:"expiresAt"`
	MaxAttempts int       `json:"maxAttempts"`
	Attempts    int       `json:"attempts"`
}

// ============ Service ============

type SocialRecoveryService struct {
	config *Config
	redis  *redis.Client
	// now is the clock the service reads. Tests use it to reach the end of the
	// recovery delay without waiting a day; production leaves it as time.Now.
	now func() time.Time
}

func NewSocialRecoveryService(config *Config) (*SocialRecoveryService, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisURL,
		Password: "",
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis not available: %v", err)
	}

	return &SocialRecoveryService{
		config: config,
		redis:  redisClient,
	}, nil
}

// Create Wallet
func (s *SocialRecoveryService) CreateWallet(ctx context.Context, address, chain string, threshold int, guardians []Guardian) (*Wallet, error) {
	wallet := &Wallet{
		ID:            uuid.New().String(),
		Address:       address,
		Chain:         chain,
		Guardians:     guardians,
		Threshold:     threshold,
		Status:        "active",
		RecoveryCount: 0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if s.redis != nil {
		data, _ := json.Marshal(wallet)
		s.redis.Set(ctx, fmt.Sprintf("wallet:%s", wallet.ID), data, 0)
		s.redis.Set(ctx, fmt.Sprintf("wallet:address:%s:%s", chain, address), wallet.ID, 0)
	}

	return wallet, nil
}

// Add Guardian
func (s *SocialRecoveryService) AddGuardian(ctx context.Context, walletID string, guardian Guardian) (*Wallet, error) {
	wallet, err := s.GetWallet(ctx, walletID)
	if err != nil {
		return nil, err
	}

	if len(wallet.Guardians) >= s.config.MaxGuardians {
		return nil, fmt.Errorf("maximum guardians reached")
	}

	guardian.ID = uuid.New().String()
	guardian.InvitedAt = time.Now()

	wallet.Guardians = append(wallet.Guardians, guardian)
	wallet.UpdatedAt = time.Now()

	if s.redis != nil {
		data, _ := json.Marshal(wallet)
		s.redis.Set(ctx, fmt.Sprintf("wallet:%s", wallet.ID), data, 0)
	}

	// Send invitation (in production, would send email/SMS)
	go s.sendGuardianInvitation(ctx, wallet.ID, guardian)

	return wallet, nil
}

// Remove Guardian
func (s *SocialRecoveryService) RemoveGuardian(ctx context.Context, walletID, guardianID string) (*Wallet, error) {
	wallet, err := s.GetWallet(ctx, walletID)
	if err != nil {
		return nil, err
	}

	for i, g := range wallet.Guardians {
		if g.ID == guardianID {
			wallet.Guardians = append(wallet.Guardians[:i], wallet.Guardians[i+1:]...)
			break
		}
	}

	wallet.UpdatedAt = time.Now()

	if s.redis != nil {
		data, _ := json.Marshal(wallet)
		s.redis.Set(ctx, fmt.Sprintf("wallet:%s", wallet.ID), data, 0)
	}

	return wallet, nil
}

// Initiate Recovery
func (s *SocialRecoveryService) InitiateRecovery(ctx context.Context, walletID, newOwnerAddress string) (*RecoveryRequest, error) {
	wallet, err := s.GetWallet(ctx, walletID)
	if err != nil {
		return nil, err
	}

	if len(wallet.Guardians) < wallet.Threshold {
		return nil, fmt.Errorf("not enough guardians")
	}

	// Check all guardians are confirmed
	confirmedCount := 0
	for _, g := range wallet.Guardians {
		if g.Confirmed {
			confirmedCount++
		}
	}

	if confirmedCount < wallet.Threshold {
		return nil, fmt.Errorf("not enough confirmed guardians")
	}

	request := &RecoveryRequest{
		ID:              uuid.New().String(),
		WalletID:        walletID,
		NewOwnerAddress: newOwnerAddress,
		Status:          RecoveryPending,
		ThresholdMet:    false,
		Signatures:      []GuardianSignature{},
		DelayEnd:        time.Now().Add(s.config.RecoveryDelay),
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(7 * 24 * time.Hour),
	}

	// The challenge is a nonce the guardians sign over. It used to be stored as
	// an entry in Signatures, which meant it counted towards the threshold: a
	// 2-of-3 wallet was satisfied by a single guardian's approval, because the
	// placeholder made up the second count.
	request.Challenge = generateChallenge()

	if s.redis != nil {
		data, _ := json.Marshal(request)
		s.redis.Set(ctx, fmt.Sprintf("recovery:%s", request.ID), data, 7*24*time.Hour)
		s.redis.Set(ctx, fmt.Sprintf("recovery:wallet:%s", walletID), request.ID, 7*24*time.Hour)
	}

	// Notify guardians
	go s.notifyGuardiansOfRecovery(*wallet, *request)

	return request, nil
}

// Sign Recovery
func (s *SocialRecoveryService) SignRecovery(ctx context.Context, requestID, guardianID, signature string) (*RecoveryRequest, error) {
	request, err := s.GetRecoveryRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}

	if request.Status != RecoveryPending {
		return nil, fmt.Errorf("recovery not pending")
	}

	wallet, err := s.GetWallet(ctx, request.WalletID)
	if err != nil {
		return nil, err
	}

	// Find guardian
	var guardian Guardian
	for _, g := range wallet.Guardians {
		if g.ID == guardianID {
			guardian = g
			break
		}
	}

	if guardian.ID == "" {
		return nil, fmt.Errorf("guardian not found")
	}

	// A guardian may approve a given request once. Without this, one guardian
	// could submit the same signature repeatedly and reach the threshold alone,
	// which defeats the whole point of distributing trust.
	for _, existing := range request.Signatures {
		if existing.GuardianID == guardianID {
			return nil, fmt.Errorf("guardian %s has already approved this request", guardianID)
		}
	}

	// The signature must recover to the guardian's own address. Accepting an
	// unverified signature here let anyone approve a recovery for any guardian
	// and hand the wallet to an attacker-chosen owner.
	approvalMessage := recoveryApprovalMessage(request)
	signer, err := recoverPersonalSigner(approvalMessage, signature)
	if err != nil {
		return nil, fmt.Errorf("the guardian signature is not valid: %w", err)
	}

	if !strings.EqualFold(signer, guardian.Address) {
		return nil, fmt.Errorf("the signature was produced by %s, not by guardian %s", signer, guardian.Address)
	}

	request.Signatures = append(request.Signatures, GuardianSignature{
		GuardianID: guardianID,
		Signature:  signature,
		SignedAt:   time.Now(),
		Message:    approvalMessage,
	})

	// Check threshold
	if len(request.Signatures) >= wallet.Threshold {
		request.ThresholdMet = true
		request.Status = RecoveryApproved
	}

	if s.redis != nil {
		data, _ := json.Marshal(request)
		s.redis.Set(ctx, fmt.Sprintf("recovery:%s", request.ID), data, 7*24*time.Hour)
	}

	return request, nil
}

// Execute Recovery (after delay)
func (s *SocialRecoveryService) ExecuteRecovery(ctx context.Context, requestID string) (*RecoveryRequest, error) {
	request, err := s.GetRecoveryRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}

	if !request.ThresholdMet {
		return nil, fmt.Errorf("threshold not met")
	}

	if s.clock().Before(request.DelayEnd) {
		return nil, fmt.Errorf("delay period not complete")
	}

	request.Status = RecoveryExecuting

	// Approvals and the elapsed delay are not the recovery itself: the wallet is
	// only recovered once the recovery contract has actually run. This service
	// holds no key and has no contract, so it hands the caller a ready-to-send
	// transaction instead of claiming the transfer already happened. Marking the
	// request completed here would tell every client that a wallet had moved to
	// the new owner while the chain still had the old one.
	if s.config == nil || strings.TrimSpace(s.config.RecoveryContract) == "" ||
		strings.TrimSpace(s.config.RPCURL) == "" || strings.TrimSpace(s.config.ExecutorPrivateKey) == "" {
		request.Status = RecoveryApproved
		if s.redis != nil {
			data, _ := json.Marshal(request)
			s.redis.Set(ctx, fmt.Sprintf("recovery:%s", request.ID), data, 7*24*time.Hour)
		}
		return request, fmt.Errorf(
			"the recovery is approved and ready to execute, but this service is not configured to broadcast: " +
				"set SOCIAL_RECOVERY_CONTRACT, SOCIAL_RECOVERY_RPC_URL and SOCIAL_RECOVERY_EXECUTOR_PRIVATE_KEY")
	}

	txHash, err := s.broadcastRecovery(ctx, request)
	if err != nil {
		request.Status = RecoveryApproved
		if s.redis != nil {
			data, _ := json.Marshal(request)
			s.redis.Set(ctx, fmt.Sprintf("recovery:%s", request.ID), data, 7*24*time.Hour)
		}
		return nil, fmt.Errorf("the recovery transaction was not broadcast: %w", err)
	}

	now := time.Now()
	request.ExecutedAt = &now
	request.ExecutedTxHash = txHash
	request.Status = RecoveryCompleted

	// Update wallet
	wallet, err := s.GetWallet(ctx, request.WalletID)
	if err == nil {
		wallet.RecoveryCount++
		wallet.UpdatedAt = time.Now()
		if s.redis != nil {
			data, _ := json.Marshal(wallet)
			s.redis.Set(ctx, fmt.Sprintf("wallet:%s", wallet.ID), data, 0)
		}
	}

	if s.redis != nil {
		data, _ := json.Marshal(request)
		s.redis.Set(ctx, fmt.Sprintf("recovery:%s", request.ID), data, 7*24*time.Hour)
	}

	return request, nil
}

// Cancel Recovery
func (s *SocialRecoveryService) CancelRecovery(ctx context.Context, requestID string) (*RecoveryRequest, error) {
	request, err := s.GetRecoveryRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}

	request.Status = RecoveryCancelled
	now := time.Now()
	request.CancelledAt = &now

	if s.redis != nil {
		data, _ := json.Marshal(request)
		s.redis.Set(ctx, fmt.Sprintf("recovery:%s", request.ID), data, 7*24*time.Hour)
	}

	return request, nil
}

// Get Wallet
func (s *SocialRecoveryService) GetWallet(ctx context.Context, walletID string) (*Wallet, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("wallet not found")
	}

	data, err := s.redis.Get(ctx, fmt.Sprintf("wallet:%s", walletID)).Bytes()
	if err != nil {
		return nil, fmt.Errorf("wallet not found")
	}

	var wallet Wallet
	json.Unmarshal(data, &wallet)
	return &wallet, nil
}

// Get Wallet by Address
func (s *SocialRecoveryService) GetWalletByAddress(ctx context.Context, chain, address string) (*Wallet, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("wallet not found")
	}

	walletID, err := s.redis.Get(ctx, fmt.Sprintf("wallet:address:%s:%s", chain, address)).Result()
	if err != nil {
		return nil, fmt.Errorf("wallet not found")
	}

	return s.GetWallet(ctx, walletID)
}

// Get Recovery Request
func (s *SocialRecoveryService) GetRecoveryRequest(ctx context.Context, requestID string) (*RecoveryRequest, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("request not found")
	}

	data, err := s.redis.Get(ctx, fmt.Sprintf("recovery:%s", requestID)).Bytes()
	if err != nil {
		return nil, fmt.Errorf("request not found")
	}

	var request RecoveryRequest
	json.Unmarshal(data, &request)
	return &request, nil
}

// Get Active Recovery Request for Wallet
func (s *SocialRecoveryService) GetActiveRecoveryRequest(ctx context.Context, walletID string) (*RecoveryRequest, error) {
	if s.redis == nil {
		return nil, nil
	}

	requestID, err := s.redis.Get(ctx, fmt.Sprintf("recovery:wallet:%s", walletID)).Result()
	if err != nil {
		return nil, nil
	}

	return s.GetRecoveryRequest(ctx, requestID)
}

// Helper functions
func generateChallenge() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ============ HTTP Handlers ============

type Handler struct {
	service *SocialRecoveryService
}

func NewHandler(service *SocialRecoveryService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateWallet(c *gin.Context) {
	var req struct {
		Address   string     `json:"address" binding:"required"`
		Chain     string     `json:"chain" binding:"required"`
		Threshold int        `json:"threshold" binding:"required"`
		Guardians []Guardian `json:"guardians"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wallet, err := h.service.CreateWallet(c.Request.Context(), req.Address, req.Chain, req.Threshold, req.Guardians)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (h *Handler) AddGuardian(c *gin.Context) {
	walletID := c.Param("walletId")

	var req struct {
		Guardian Guardian `json:"guardian" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wallet, err := h.service.AddGuardian(c.Request.Context(), walletID, req.Guardian)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (h *Handler) InitiateRecovery(c *gin.Context) {
	walletID := c.Param("walletId")

	var req struct {
		NewOwnerAddress string `json:"newOwnerAddress" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	request, err := h.service.InitiateRecovery(c.Request.Context(), walletID, req.NewOwnerAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, request)
}

func (h *Handler) SignRecovery(c *gin.Context) {
	requestID := c.Param("requestId")

	var req struct {
		GuardianID string `json:"guardianId" binding:"required"`
		Signature  string `json:"signature" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	request, err := h.service.SignRecovery(c.Request.Context(), requestID, req.GuardianID, req.Signature)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, request)
}

func (h *Handler) ExecuteRecovery(c *gin.Context) {
	requestID := c.Param("requestId")

	request, err := h.service.ExecuteRecovery(c.Request.Context(), requestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, request)
}

func (h *Handler) GetWallet(c *gin.Context) {
	walletID := c.Param("walletId")

	wallet, err := h.service.GetWallet(c.Request.Context(), walletID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (h *Handler) GetRecoveryRequest(c *gin.Context) {
	requestID := c.Param("requestId")

	request, err := h.service.GetRecoveryRequest(c.Request.Context(), requestID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, request)
}

func (h *Handler) GetWalletByAddress(c *gin.Context) {
	chain := c.Param("chain")
	address := c.Param("address")

	wallet, err := h.service.GetWalletByAddress(c.Request.Context(), chain, address)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (h *Handler) RemoveGuardian(c *gin.Context) {
	walletID := c.Param("walletId")
	guardianID := c.Param("guardianId")

	wallet, err := h.service.RemoveGuardian(c.Request.Context(), walletID, guardianID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (h *Handler) VerifyGuardian(c *gin.Context) {
	walletID := c.Param("walletId")
	guardianID := c.Param("guardianId")

	var req struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.VerifyGuardian(c.Request.Context(), walletID, guardianID, req.Code); err != nil {
		// The code was wrong or has expired; that is a client error, not a
		// server fault, and the caller may retry with a fresh code.
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wallet, err := h.service.GetWallet(c.Request.Context(), walletID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

func (h *Handler) GetActiveRecovery(c *gin.Context) {
	walletID := c.Param("walletId")

	request, err := h.service.GetActiveRecoveryRequest(c.Request.Context(), walletID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, request)
}

func (h *Handler) CancelRecovery(c *gin.Context) {
	requestID := c.Param("requestId")

	request, err := h.service.CancelRecovery(c.Request.Context(), requestID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, request)
}

// clock returns the service clock, defaulting to the wall clock.
func (s *SocialRecoveryService) clock() time.Time {
	if s.now == nil {
		return time.Now()
	}
	return s.now()
}

// recoveryApprovalMessage is the exact text a guardian signs to approve a
// recovery. It binds the approval to the request and to the new owner, so a
// signature harvested for one recovery cannot be replayed onto another.
func recoveryApprovalMessage(request *RecoveryRequest) string {
	return fmt.Sprintf(
		"TigerWallet social recovery approval\nrequest: %s\nwallet: %s\nnew owner: %s\nchallenge: %s",
		request.ID, request.WalletID, request.NewOwnerAddress, request.Challenge)
}

// recoverPersonalSigner recovers the address that produced an EIP-191
// personal-sign signature over message.
func recoverPersonalSigner(message, signature string) (string, error) {
	sig, err := hexutil.Decode(signature)
	if err != nil {
		return "", fmt.Errorf("the signature is not valid hex: %w", err)
	}
	if len(sig) != 65 {
		return "", fmt.Errorf("the signature must be 65 bytes, got %d", len(sig))
	}

	// Some signers report the recovery id as 27/28; go-ethereum expects 0/1.
	if sig[64] >= 27 {
		sig[64] -= 27
	}

	hash := accounts.TextHash([]byte(message))
	pub, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return "", fmt.Errorf("the signature could not be recovered: %w", err)
	}
	return crypto.PubkeyToAddress(*pub).Hex(), nil
}

// broadcastRecovery sends the on-chain call that moves the wallet to the new
// owner and waits for it to be mined. It returns the transaction hash only
// after a receipt confirms success, so a caller never records a recovery that
// reverted.
func (s *SocialRecoveryService) broadcastRecovery(ctx context.Context, request *RecoveryRequest) (string, error) {
	key, err := crypto.HexToECDSA(strings.TrimPrefix(s.config.ExecutorPrivateKey, "0x"))
	if err != nil {
		return "", fmt.Errorf("EXECUTOR_PRIVATE_KEY is not a valid private key: %w", err)
	}
	if !common.IsHexAddress(request.NewOwnerAddress) {
		return "", fmt.Errorf("the new owner %q is not a valid address", request.NewOwnerAddress)
	}

	client, err := ethclient.DialContext(ctx, s.config.RPCURL)
	if err != nil {
		return "", fmt.Errorf("cannot reach the RPC endpoint: %w", err)
	}
	defer client.Close()

	contract := common.HexToAddress(s.config.RecoveryContract)
	chainID := big.NewInt(s.config.ChainID)
	if s.config.ChainID == 0 {
		if chainID, err = client.ChainID(ctx); err != nil {
			return "", fmt.Errorf("cannot read the chain id: %w", err)
		}
	}

	from := crypto.PubkeyToAddress(key.PublicKey)

	// executeRecovery(address wallet, address newOwner, bytes[] signatures) is
	// the entry point the recovery contract exposes.
	selector := crypto.Keccak256([]byte("executeRecovery(address,address,bytes[])"))[:4]
	type args struct {
		Wallet     common.Address
		NewOwner   common.Address
		Signatures [][]byte
	}
	input, err := abi.Arguments{
		{Name: "wallet", Type: mustNewType("address")},
		{Name: "newOwner", Type: mustNewType("address")},
		{Name: "signatures", Type: mustNewType("bytes[]")},
	}.Pack(args{
		Wallet:     common.HexToAddress(request.WalletID),
		NewOwner:   common.HexToAddress(request.NewOwnerAddress),
		Signatures: collectedSignatures(request),
	})
	if err != nil {
		return "", fmt.Errorf("cannot encode the recovery call: %w", err)
	}

	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return "", fmt.Errorf("cannot read the account nonce: %w", err)
	}
	tip, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		return "", fmt.Errorf("cannot read the gas tip: %w", err)
	}
	head, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("cannot read the latest header: %w", err)
	}
	feeCap := new(big.Int).Add(new(big.Int).Mul(head.BaseFee, big.NewInt(2)), tip)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: tip,
		GasFeeCap: feeCap,
		Gas:       300000,
		To:        &contract,
		Value:     big.NewInt(0),
		Data:      append(selector, input...),
	})
	signed, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), key)
	if err != nil {
		return "", fmt.Errorf("cannot sign the recovery transaction: %w", err)
	}
	if err := client.SendTransaction(ctx, signed); err != nil {
		return "", fmt.Errorf("the node rejected the recovery transaction: %w", err)
	}

	receipt, err := bind.WaitMined(ctx, client, signed)
	if err != nil {
		return signed.Hash().Hex(), fmt.Errorf("the recovery transaction was sent but not confirmed: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return signed.Hash().Hex(), fmt.Errorf("the recovery transaction reverted in block %d", receipt.BlockNumber)
	}

	return signed.Hash().Hex(), nil
}

// collectedSignatures decodes the guardian signatures the request carries.
func collectedSignatures(request *RecoveryRequest) [][]byte {
	sigs := make([][]byte, 0, len(request.Signatures))
	for _, s := range request.Signatures {
		if decoded, err := hexutil.Decode(s.Signature); err == nil {
			sigs = append(sigs, decoded)
		}
	}
	return sigs
}

func mustNewType(kind string) abi.Type {
	t, err := abi.NewType(kind, "", nil)
	if err != nil {
		panic(err)
	}
	return t
}

// ============ Main ============

func main() {
	config := &Config{
		Port:               getEnv("PORT", "8080"),
		RedisURL:           getEnv("REDIS_URL", "localhost:6379"),
		RecoveryDelay:      24 * time.Hour,
		MaxGuardians:       10,
		MinGuardians:       2,
		EmailEnabled:       getEnv("EMAIL_ENABLED", "true") == "true",
		SMSEnabled:         getEnv("SMS_ENABLED", "true") == "true",
		SMTPHost:           os.Getenv("SMTP_HOST"),
		SMTPPort:           parseInt(getEnv("SMTP_PORT", "587"), 587),
		SMTPUsername:       os.Getenv("SMTP_USERNAME"),
		SMTPPassword:       os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:           os.Getenv("SMTP_FROM"),
		SMSWebhookURL:      os.Getenv("SMS_WEBHOOK_URL"),
		SMSWebhookToken:    os.Getenv("SMS_WEBHOOK_TOKEN"),
		RPCURL:             os.Getenv("SOCIAL_RECOVERY_RPC_URL"),
		RecoveryContract:   os.Getenv("SOCIAL_RECOVERY_CONTRACT"),
		ExecutorPrivateKey: os.Getenv("SOCIAL_RECOVERY_EXECUTOR_PRIVATE_KEY"),
		ChainID:            parseChainID(getEnv("SOCIAL_RECOVERY_CHAIN_ID", "1")),
	}

	service, err := NewSocialRecoveryService(config)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	handler := NewHandler(service)

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		api.POST("/wallets", handler.CreateWallet)
		api.GET("/wallets/:walletId", handler.GetWallet)
		// Resolves a wallet by chain and address, which is how a client knows
		// the chain and address but not the internal wallet id.
		api.GET("/wallets/by-address/:chain/:address", handler.GetWalletByAddress)

		api.POST("/wallets/:walletId/guardians", handler.AddGuardian)
		api.DELETE("/wallets/:walletId/guardians/:guardianId", handler.RemoveGuardian)
		// Verifies the code sent to an email or phone guardian.
		api.POST("/wallets/:walletId/guardians/:guardianId/verify", handler.VerifyGuardian)

		api.POST("/wallets/:walletId/recovery", handler.InitiateRecovery)
		// The active request for a wallet, so a client can resume a recovery it
		// did not start in this session.
		api.GET("/wallets/:walletId/recovery", handler.GetActiveRecovery)
		api.POST("/recovery/:requestId/sign", handler.SignRecovery)
		api.POST("/recovery/:requestId/execute", handler.ExecuteRecovery)
		api.POST("/recovery/:requestId/cancel", handler.CancelRecovery)
		api.GET("/recovery/:requestId", handler.GetRecoveryRequest)
	}

	// Start server
	addr := fmt.Sprintf(":%s", config.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("Starting Social Recovery service on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
