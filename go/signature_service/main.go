package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Server   ServerConfig
	Redis    RedisConfig
	Security SecurityConfig
}

type ServerConfig struct {
	Port string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type SecurityConfig struct {
	RequireApproval      bool
	MaxSignaturesPerHour int
	AllowedChains        []uint64
}

func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8444"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     6379,
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		},
		Security: SecurityConfig{
			RequireApproval:      getEnv("REQUIRE_APPROVAL", "false") == "true",
			MaxSignaturesPerHour: 1000,
			AllowedChains:        []uint64{1, 5, 137, 42161, 10, 43114, 56, 8453},
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Models
// ============================================================================

type SignatureRequest struct {
	ID            string     `json:"id"`
	UserID        string     `json:"userId"`
	WalletAddress string     `json:"walletAddress"`
	ChainID       uint64     `json:"chainId"`
	Message       string     `json:"message"`
	MessageHash   string     `json:"messageHash"`
	Signature     string     `json:"signature,omitempty"`
	Status        string     `json:"status"`        // pending, signed, failed, cancelled
	SignatureType string     `json:"signatureType"` // personal_sign, eth_sign, typed_data
	IPAddress     string     `json:"ipAddress"`
	UserAgent     string     `json:"userAgent"`
	ApprovedBy    string     `json:"approvedBy,omitempty"`
	ApprovedAt    *time.Time `json:"approvedAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
}

type SignatureApproval struct {
	ID            string    `json:"id"`
	RequestID     string    `json:"requestId"`
	ApproverID    string    `json:"approverId"`
	ApproverEmail string    `json:"approverEmail"`
	Status        string    `json:"status"` // approved, rejected
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"createdAt"`
}

type KeyRotation struct {
	ID           string     `json:"id"`
	UserID       string     `json:"userId"`
	OldPublicKey string     `json:"oldPublicKey"`
	NewPublicKey string     `json:"newPublicKey"`
	Status       string     `json:"status"`       // pending, completed, failed
	RotationType string     `json:"rotationType"` // scheduled, emergency, compromised
	CreatedAt    time.Time  `json:"createdAt"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
}

type AuditLog struct {
	ID         string    `json:"id"`
	UserID     string    `json:"userId"`
	Action     string    `json:"action"`
	Resource   string    `json:"resource"`
	ResourceID string    `json:"resourceId"`
	Details    string    `json:"details"`
	IPAddress  string    `json:"ipAddress"`
	UserAgent  string    `json:"userAgent"`
	CreatedAt  time.Time `json:"createdAt"`
}

// ============================================================================
// Services
// ============================================================================

type SignatureService struct {
	config      *Config
	redis       *redis.Client
	pg          *pgxpool.Pool
	auditLogs   []AuditLog
	mu          sync.RWMutex
	rateLimiter *RateLimiter
}

func NewSignatureService(config *Config, redisClient *redis.Client) *SignatureService {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	svc := &SignatureService{
		config:      config,
		redis:       redisClient,
		pg:          pool,
		auditLogs:   make([]AuditLog, 0),
		rateLimiter: NewRateLimiter(config.Security.MaxSignaturesPerHour),
	}
	if err := svc.Migrate(context.Background()); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	return svc
}

const signatureSchema = `
CREATE TABLE IF NOT EXISTS signature_requests (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL,
    wallet_address  TEXT NOT NULL,
    chain_id        BIGINT NOT NULL,
    message         TEXT NOT NULL,
    message_hash    TEXT NOT NULL,
    signature       TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL,
    signature_type  TEXT NOT NULL,
    ip_address      TEXT NOT NULL DEFAULT '',
    user_agent      TEXT NOT NULL DEFAULT '',
    approved_by      TEXT NOT NULL DEFAULT '',
    approved_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL,
    completed_at    TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_signature_requests_user ON signature_requests(user_id);

CREATE TABLE IF NOT EXISTS signature_approvals (
    id              TEXT PRIMARY KEY,
    request_id      TEXT NOT NULL,
    approver_id      TEXT NOT NULL,
    approver_email   TEXT NOT NULL,
    status          TEXT NOT NULL,
    notes           TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_signature_approvals_request ON signature_approvals(request_id);

CREATE TABLE IF NOT EXISTS key_rotations (
    id               TEXT PRIMARY KEY,
    user_id          TEXT NOT NULL,
    old_public_key   TEXT NOT NULL,
    new_public_key   TEXT NOT NULL,
    status           TEXT NOT NULL,
    rotation_type    TEXT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL,
    completed_at     TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_key_rotations_user ON key_rotations(user_id);
`

// Migrate creates the signature/approval/rotation tables if they do not exist.
func (s *SignatureService) Migrate(ctx context.Context) error {
	if s.pg == nil {
		return fmt.Errorf("database not configured")
	}
	_, err := s.pg.Exec(ctx, signatureSchema)
	return err
}

// ============================================================================
// Rate Limiter
// ============================================================================

type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   time.Hour,
	}
}

func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	oneHourAgo := now.Add(-r.window)

	// Clean old requests
	var recent []time.Time
	for _, t := range r.requests[key] {
		if t.After(oneHourAgo) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= r.limit {
		r.requests[key] = recent
		return false
	}

	r.requests[key] = append(recent, now)
	return true
}

// ============================================================================
// Signature Operations
// ============================================================================

func (s *SignatureService) CreateSignatureRequest(
	userID,
	walletAddress string,
	chainID uint64,
	message string,
	signatureType string,
	ipAddress,
	userAgent string,
) (*SignatureRequest, error) {
	// Check rate limit
	if !s.rateLimiter.Allow(userID) {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	// Validate chain
	if !s.isChainAllowed(chainID) {
		return nil, fmt.Errorf("chain %d not allowed", chainID)
	}

	// Validate wallet address
	if !common.IsHexAddress(walletAddress) {
		return nil, fmt.Errorf("invalid wallet address")
	}

	// Calculate message hash
	messageHash := calculateMessageHash(message)

	request := &SignatureRequest{
		ID:            uuid.New().String(),
		UserID:        userID,
		WalletAddress: walletAddress,
		ChainID:       chainID,
		Message:       message,
		MessageHash:   messageHash,
		Status:        "pending",
		SignatureType: signatureType,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
		CreatedAt:     time.Now(),
	}

	if s.pg == nil {
		return nil, fmt.Errorf("database not configured")
	}
	_, err := s.pg.Exec(context.Background(), `INSERT INTO signature_requests
		(id,user_id,wallet_address,chain_id,message,message_hash,signature,status,signature_type,ip_address,user_agent,approved_by,approved_at,created_at,completed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		request.ID, request.UserID, request.WalletAddress, request.ChainID, request.Message,
		request.MessageHash, request.Signature, request.Status, request.SignatureType,
		request.IPAddress, request.UserAgent, request.ApprovedBy, request.ApprovedAt,
		request.CreatedAt, request.CompletedAt)
	if err != nil {
		return nil, err
	}

	// Log audit
	s.logAudit(userID, "CREATE_SIGNATURE_REQUEST", "signature_request", request.ID,
		fmt.Sprintf("Created request for wallet %s on chain %d", walletAddress, chainID),
		ipAddress, userAgent)

	// If approval required, don't auto-sign
	if s.config.Security.RequireApproval {
		return request, nil
	}

	// When approval is not required, the request stays in "pending" status
	// and is signed on demand via SignMessage (with a real ECDSA key). We do
	// NOT auto-sign here — signing always requires an explicit key holder call.
	return request, nil
}

func (s *SignatureService) SignMessage(
	requestID string,
	privateKey *ecdsa.PrivateKey,
) (*SignatureRequest, error) {
	if s.pg == nil {
		return nil, fmt.Errorf("database not configured")
	}
	ctx := context.Background()
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var request SignatureRequest
	if err := tx.QueryRow(ctx, `SELECT id,user_id,wallet_address,chain_id,message,message_hash,
		signature,status,signature_type,ip_address,user_agent,approved_by,approved_at,created_at,completed_at
		FROM signature_requests WHERE id=$1 FOR UPDATE`, requestID).
		Scan(&request.ID, &request.UserID, &request.WalletAddress, &request.ChainID, &request.Message,
			&request.MessageHash, &request.Signature, &request.Status, &request.SignatureType,
			&request.IPAddress, &request.UserAgent, &request.ApprovedBy, &request.ApprovedAt,
			&request.CreatedAt, &request.CompletedAt); err != nil {
		return nil, fmt.Errorf("request not found")
	}

	if request.Status != "pending" {
		return nil, fmt.Errorf("request already processed")
	}

	// Sign the message
	var signature []byte
	var signErr error

	switch request.SignatureType {
	case "personal_sign":
		signature, signErr = crypto.Sign(accounts.TextHash([]byte(request.Message)), privateKey)
	case "eth_sign":
		// eth_sign signs the raw keccak256 hash of the message (no Ethereum prefix).
		signature, signErr = crypto.Sign(crypto.Keccak256([]byte(request.Message)), privateKey)
	default:
		signature, signErr = crypto.Sign(accounts.TextHash([]byte(request.Message)), privateKey)
	}

	if signErr != nil {
		if _, e := tx.Exec(ctx, `UPDATE signature_requests SET status='failed' WHERE id=$1`, requestID); e != nil {
			return nil, fmt.Errorf("signing failed: %w", signErr)
		}
		_ = tx.Commit(ctx)
		return nil, fmt.Errorf("signing failed: %w", signErr)
	}

	request.Signature = hexutil.Encode(signature)
	request.Status = "signed"
	now := time.Now()
	request.CompletedAt = &now

	if _, err := tx.Exec(ctx, `UPDATE signature_requests
		SET signature=$1, status=$2, completed_at=$3 WHERE id=$4`,
		request.Signature, request.Status, request.CompletedAt, request.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Log audit
	s.logAudit(request.UserID, "SIGN_MESSAGE", "signature_request", request.ID,
		fmt.Sprintf("Signed message with hash %s", request.MessageHash),
		request.IPAddress, request.UserAgent)

	return &request, nil
}

func (s *SignatureService) GetSignatureRequest(id string) (*SignatureRequest, error) {
	if s.pg == nil {
		return nil, fmt.Errorf("database not configured")
	}
	var request SignatureRequest
	err := s.pg.QueryRow(context.Background(), `SELECT id,user_id,wallet_address,chain_id,message,message_hash,
		signature,status,signature_type,ip_address,user_agent,approved_by,approved_at,created_at,completed_at
		FROM signature_requests WHERE id=$1`, id).
		Scan(&request.ID, &request.UserID, &request.WalletAddress, &request.ChainID, &request.Message,
			&request.MessageHash, &request.Signature, &request.Status, &request.SignatureType,
			&request.IPAddress, &request.UserAgent, &request.ApprovedBy, &request.ApprovedAt,
			&request.CreatedAt, &request.CompletedAt)
	if err != nil {
		return nil, fmt.Errorf("request not found")
	}
	return &request, nil
}

func (s *SignatureService) GetUserSignatureRequests(userID string, limit int) ([]SignatureRequest, error) {
	if s.pg == nil {
		return nil, fmt.Errorf("database not configured")
	}
	rows, err := s.pg.Query(context.Background(), `SELECT id,user_id,wallet_address,chain_id,message,message_hash,
		signature,status,signature_type,ip_address,user_agent,approved_by,approved_at,created_at,completed_at
		FROM signature_requests WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []SignatureRequest
	for rows.Next() {
		var req SignatureRequest
		if err := rows.Scan(&req.ID, &req.UserID, &req.WalletAddress, &req.ChainID, &req.Message,
			&req.MessageHash, &req.Signature, &req.Status, &req.SignatureType,
			&req.IPAddress, &req.UserAgent, &req.ApprovedBy, &req.ApprovedAt,
			&req.CreatedAt, &req.CompletedAt); err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return requests, rows.Err()
}

func (s *SignatureService) CancelSignatureRequest(id, userID string) error {
	if s.pg == nil {
		return fmt.Errorf("database not configured")
	}
	ctx := context.Background()
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var request SignatureRequest
	if err := tx.QueryRow(ctx, `SELECT id,user_id,wallet_address,chain_id,message,message_hash,
		signature,status,signature_type,ip_address,user_agent,approved_by,approved_at,created_at,completed_at
		FROM signature_requests WHERE id=$1 FOR UPDATE`, id).
		Scan(&request.ID, &request.UserID, &request.WalletAddress, &request.ChainID, &request.Message,
			&request.MessageHash, &request.Signature, &request.Status, &request.SignatureType,
			&request.IPAddress, &request.UserAgent, &request.ApprovedBy, &request.ApprovedAt,
			&request.CreatedAt, &request.CompletedAt); err != nil {
		return fmt.Errorf("request not found")
	}

	if request.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	if request.Status != "pending" {
		return fmt.Errorf("request cannot be cancelled")
	}

	if _, err := tx.Exec(ctx, `UPDATE signature_requests SET status='cancelled' WHERE id=$1`, id); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// Log audit
	s.logAudit(userID, "CANCEL_SIGNATURE_REQUEST", "signature_request", id,
		"Cancelled signature request", request.IPAddress, request.UserAgent)

	return nil
}

// ============================================================================
// Approval Management
// ============================================================================

func (s *SignatureService) ApproveSignatureRequest(
	requestID,
	approverID,
	approverEmail,
	notes string,
) (*SignatureApproval, error) {
	if s.pg == nil {
		return nil, fmt.Errorf("database not configured")
	}
	ctx := context.Background()
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var request SignatureRequest
	if err := tx.QueryRow(ctx, `SELECT id,user_id,wallet_address,chain_id,message,message_hash,
		signature,status,signature_type,ip_address,user_agent,approved_by,approved_at,created_at,completed_at
		FROM signature_requests WHERE id=$1 FOR UPDATE`, requestID).
		Scan(&request.ID, &request.UserID, &request.WalletAddress, &request.ChainID, &request.Message,
			&request.MessageHash, &request.Signature, &request.Status, &request.SignatureType,
			&request.IPAddress, &request.UserAgent, &request.ApprovedBy, &request.ApprovedAt,
			&request.CreatedAt, &request.CompletedAt); err != nil {
		return nil, fmt.Errorf("request not found")
	}

	if request.Status != "pending" {
		return nil, fmt.Errorf("request already processed")
	}

	approval := &SignatureApproval{
		ID:            uuid.New().String(),
		RequestID:     requestID,
		ApproverID:    approverID,
		ApproverEmail: approverEmail,
		Status:        "approved",
		Notes:         notes,
		CreatedAt:     time.Now(),
	}

	now := time.Now()
	if _, err := tx.Exec(ctx, `UPDATE signature_requests SET approved_by=$1, approved_at=$2 WHERE id=$3`,
		approverID, &now, requestID); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `INSERT INTO signature_approvals
		(id,request_id,approver_id,approver_email,status,notes,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		approval.ID, approval.RequestID, approval.ApproverID, approval.ApproverEmail,
		approval.Status, approval.Notes, approval.CreatedAt); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Log audit
	s.logAudit(approverID, "APPROVE_SIGNATURE", "signature_request", requestID,
		fmt.Sprintf("Approved request from user %s", request.UserID),
		"", "")

	return approval, nil
}

func (s *SignatureService) RejectSignatureRequest(
	requestID,
	approverID,
	approverEmail,
	notes string,
) error {
	if s.pg == nil {
		return fmt.Errorf("database not configured")
	}
	ctx := context.Background()
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var request SignatureRequest
	if err := tx.QueryRow(ctx, `SELECT id,user_id,wallet_address,chain_id,message,message_hash,
		signature,status,signature_type,ip_address,user_agent,approved_by,approved_at,created_at,completed_at
		FROM signature_requests WHERE id=$1 FOR UPDATE`, requestID).
		Scan(&request.ID, &request.UserID, &request.WalletAddress, &request.ChainID, &request.Message,
			&request.MessageHash, &request.Signature, &request.Status, &request.SignatureType,
			&request.IPAddress, &request.UserAgent, &request.ApprovedBy, &request.ApprovedAt,
			&request.CreatedAt, &request.CompletedAt); err != nil {
		return fmt.Errorf("request not found")
	}

	approval := &SignatureApproval{
		ID:            uuid.New().String(),
		RequestID:     requestID,
		ApproverID:    approverID,
		ApproverEmail: approverEmail,
		Status:        "rejected",
		Notes:         notes,
		CreatedAt:     time.Now(),
	}

	if _, err := tx.Exec(ctx, `UPDATE signature_requests SET status='rejected' WHERE id=$1`, requestID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO signature_approvals
		(id,request_id,approver_id,approver_email,status,notes,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		approval.ID, approval.RequestID, approval.ApproverID, approval.ApproverEmail,
		approval.Status, approval.Notes, approval.CreatedAt); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// Log audit
	s.logAudit(approverID, "REJECT_SIGNATURE", "signature_request", requestID,
		fmt.Sprintf("Rejected request from user %s", request.UserID),
		"", "")

	return nil
}

// ============================================================================
// Key Rotation
// ============================================================================

func (s *SignatureService) InitiateKeyRotation(
	userID,
	oldPublicKey,
	newPublicKey,
	rotationType string,
) (*KeyRotation, error) {
	if s.pg == nil {
		return nil, fmt.Errorf("database not configured")
	}
	rotation := &KeyRotation{
		ID:           uuid.New().String(),
		UserID:       userID,
		OldPublicKey: oldPublicKey,
		NewPublicKey: newPublicKey,
		Status:       "pending",
		RotationType: rotationType,
		CreatedAt:    time.Now(),
	}

	if _, err := s.pg.Exec(context.Background(), `INSERT INTO key_rotations
		(id,user_id,old_public_key,new_public_key,status,rotation_type,created_at,completed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		rotation.ID, rotation.UserID, rotation.OldPublicKey, rotation.NewPublicKey,
		rotation.Status, rotation.RotationType, rotation.CreatedAt, rotation.CompletedAt); err != nil {
		return nil, err
	}

	// Log audit
	s.logAudit(userID, "INITIATE_KEY_ROTATION", "key_rotation", rotation.ID,
		fmt.Sprintf("Initiated %s key rotation", rotationType),
		"", "")

	return rotation, nil
}

func (s *SignatureService) CompleteKeyRotation(rotationID string) error {
	if s.pg == nil {
		return fmt.Errorf("database not configured")
	}
	ctx := context.Background()
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var rotation KeyRotation
	if err := tx.QueryRow(ctx, `SELECT id,user_id,old_public_key,new_public_key,status,rotation_type,created_at,completed_at
		FROM key_rotations WHERE id=$1 FOR UPDATE`, rotationID).
		Scan(&rotation.ID, &rotation.UserID, &rotation.OldPublicKey, &rotation.NewPublicKey,
			&rotation.Status, &rotation.RotationType, &rotation.CreatedAt, &rotation.CompletedAt); err != nil {
		return fmt.Errorf("rotation not found")
	}

	if rotation.Status != "pending" {
		return fmt.Errorf("rotation already processed")
	}

	now := time.Now()
	if _, err := tx.Exec(ctx, `UPDATE key_rotations SET status='completed', completed_at=$1 WHERE id=$2`,
		&now, rotationID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// Log audit
	s.logAudit(rotation.UserID, "COMPLETE_KEY_ROTATION", "key_rotation", rotationID,
		"Completed key rotation",
		"", "")

	return nil
}

// ============================================================================
// Verification
// ============================================================================

func (s *SignatureService) VerifySignature(
	walletAddress,
	message,
	signature string,
) (bool, error) {
	// Parse signature
	sigBytes, err := hexutil.Decode(signature)
	if err != nil {
		return false, fmt.Errorf("invalid signature format")
	}

	// Recover public key
	if len(sigBytes) != 65 {
		return false, fmt.Errorf("invalid signature length")
	}

	sigBytes[64] = 0 // V value

	pubKey, err := crypto.SigToPub(accounts.TextHash([]byte(message)), sigBytes)
	if err != nil {
		return false, fmt.Errorf("signature verification failed")
	}

	recoveredAddress := crypto.PubkeyToAddress(*pubKey).Hex()

	// Compare addresses (case insensitive)
	return strings.EqualFold(recoveredAddress, walletAddress), nil
}

// ============================================================================
// Utilities
// ============================================================================

func (s *SignatureService) isChainAllowed(chainID uint64) bool {
	for _, allowed := range s.config.Security.AllowedChains {
		if chainID == allowed {
			return true
		}
	}
	return false
}

func (s *SignatureService) logAudit(userID, action, resource, resourceID, details, ipAddress, userAgent string) {
	log := AuditLog{
		ID:         uuid.New().String(),
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		CreatedAt:  time.Now(),
	}

	s.mu.Lock()
	s.auditLogs = append(s.auditLogs, log)
	s.mu.Unlock()
}

func (s *SignatureService) GetAuditLogs(userID string, limit int) []AuditLog {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var logs []AuditLog
	count := 0

	for i := len(s.auditLogs) - 1; i >= 0 && count < limit; i-- {
		if s.auditLogs[i].UserID == userID {
			logs = append(logs, s.auditLogs[i])
			count++
		}
	}

	return logs
}

func calculateMessageHash(message string) string {
	// Ethereum personal_sign hash: keccak256("\x19Ethereum Signed Message:\n" + len + msg).
	// Must use keccak256 (NOT sha256) so the stored hash matches the signature
	// produced by crypto.Sign(accounts.TextHash(...)).
	prefix := "\x19Ethereum Signed Message:\n"
	fullMessage := prefix + fmt.Sprintf("%d", len(message)) + message
	hash := crypto.Keccak256([]byte(fullMessage))
	return hex.EncodeToString(hash)
}

// ============================================================================
// Generate Key Pair
// ============================================================================

func GenerateKeyPair() (string, string, error) {
	// Ethereum uses secp256k1 (NOT NIST P-256). crypto.GenerateKey() produces a
	// cryptographically-secure secp256k1 key compatible with Ethereum addresses,
	// EIP-191/712 signing, and eth_sendRawTransaction.
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate key: %w", err)
	}

	privateKeyHex := hex.EncodeToString(privateKey.D.Bytes())
	publicKeyHex := hex.EncodeToString(crypto.FromECDSAPub(&privateKey.PublicKey))

	return privateKeyHex, publicKeyHex, nil
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *SignatureService) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "signature-service"})
	})

	api := r.Group("/api/v1")
	{
		// Signature requests
		api.POST("/signature/request", s.handleCreateRequest)
		api.GET("/signature/requests", s.handleGetUserRequests)
		api.GET("/signature/request/:id", s.handleGetRequest)
		api.POST("/signature/request/:id/cancel", s.handleCancelRequest)
		api.POST("/signature/request/:id/sign", s.handleSignRequest)
		api.POST("/signature/verify", s.handleVerifySignature)

		// Approvals
		api.POST("/signature/request/:id/approve", s.handleApproveRequest)
		api.POST("/signature/request/:id/reject", s.handleRejectRequest)

		// Key rotation
		api.POST("/key-rotation", s.handleInitiateRotation)
		api.POST("/key-rotation/:id/complete", s.handleCompleteRotation)

		// Audit
		api.GET("/audit-logs", s.handleGetAuditLogs)

		// Utility
		api.POST("/key/generate", s.handleGenerateKey)
	}
}

func (s *SignatureService) handleCreateRequest(c *gin.Context) {
	var req struct {
		UserID        string `json:"userId" binding:"required"`
		WalletAddress string `json:"walletAddress" binding:"required"`
		ChainID       uint64 `json:"chainId" binding:"required"`
		Message       string `json:"message" binding:"required"`
		SignatureType string `json:"signatureType"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sigType := req.SignatureType
	if sigType == "" {
		sigType = "personal_sign"
	}

	request, err := s.CreateSignatureRequest(
		req.UserID,
		req.WalletAddress,
		req.ChainID,
		req.Message,
		sigType,
		c.ClientIP(),
		c.Request.UserAgent(),
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, request)
}

func (s *SignatureService) handleGetUserRequests(c *gin.Context) {
	userID := c.Query("userId")
	limit := 50

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId required"})
		return
	}

	requests, err := s.GetUserSignatureRequests(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

func (s *SignatureService) handleGetRequest(c *gin.Context) {
	id := c.Param("id")

	request, err := s.GetSignatureRequest(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, request)
}

func (s *SignatureService) handleCancelRequest(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		UserID string `json:"userId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.CancelSignatureRequest(id, req.UserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Request cancelled"})
}

func (s *SignatureService) handleSignRequest(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		PrivateKey string `json:"privateKey" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse private key
	privateKeyBytes, err := hex.DecodeString(req.PrivateKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid private key"})
		return
	}

	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid private key"})
		return
	}

	request, err := s.SignMessage(id, privateKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, request)
}

func (s *SignatureService) handleVerifySignature(c *gin.Context) {
	var req struct {
		WalletAddress string `json:"walletAddress" binding:"required"`
		Message       string `json:"message" binding:"required"`
		Signature     string `json:"signature" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	valid, err := s.VerifySignature(req.WalletAddress, req.Message, req.Signature)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": valid})
}

func (s *SignatureService) handleApproveRequest(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		ApproverID    string `json:"approverId" binding:"required"`
		ApproverEmail string `json:"approverEmail" binding:"required"`
		Notes         string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	approval, err := s.ApproveSignatureRequest(id, req.ApproverID, req.ApproverEmail, req.Notes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, approval)
}

func (s *SignatureService) handleRejectRequest(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		ApproverID    string `json:"approverId" binding:"required"`
		ApproverEmail string `json:"approverEmail" binding:"required"`
		Notes         string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.RejectSignatureRequest(id, req.ApproverID, req.ApproverEmail, req.Notes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Request rejected"})
}

func (s *SignatureService) handleInitiateRotation(c *gin.Context) {
	var req struct {
		UserID       string `json:"userId" binding:"required"`
		OldPublicKey string `json:"oldPublicKey" binding:"required"`
		NewPublicKey string `json:"newPublicKey" binding:"required"`
		RotationType string `json:"rotationType"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rotType := req.RotationType
	if rotType == "" {
		rotType = "scheduled"
	}

	rotation, err := s.InitiateKeyRotation(req.UserID, req.OldPublicKey, req.NewPublicKey, rotType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, rotation)
}

func (s *SignatureService) handleCompleteRotation(c *gin.Context) {
	id := c.Param("id")

	if err := s.CompleteKeyRotation(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Key rotation completed"})
}

func (s *SignatureService) handleGetAuditLogs(c *gin.Context) {
	userID := c.Query("userId")
	limit := 100

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId required"})
		return
	}

	logs := s.GetAuditLogs(userID, limit)
	c.JSON(http.StatusOK, gin.H{"auditLogs": logs})
}

func (s *SignatureService) handleGenerateKey(c *gin.Context) {
	privateKey, publicKey, err := GenerateKeyPair()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"privateKey": privateKey,
		"publicKey":  publicKey,
	})
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Redis.Host, config.Redis.Port),
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
	})

	// Initialize service
	service := NewSignatureService(config, redisClient)

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// Register routes
	service.RegisterRoutes(r)

	// Create server
	srv := &http.Server{
		Addr:    ":" + config.Server.Port,
		Handler: r,
	}

	// Start server
	go func() {
		log.Printf("Signature service starting on port %s", config.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx := context.Background()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

func c() {
	// Dummy function to fix syntax
}
