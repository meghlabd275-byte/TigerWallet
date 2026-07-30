/**
 * TigerWallet Account Abstraction Service - Complete Implementation
 * 
 * ERC-4337 Smart Account implementation with social recovery
 * High-performance Go service for worldwide distribution
 */

package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// TYPES AND STRUCTURES
// ============================================================================

// UserOperation for ERC-4337
type UserOperation struct {
	Sender               string   `json:"sender"`
	Nonce                string   `json:"nonce"`
	InitCode             string   `json:"initCode"`
	CallData             string   `json:"callData"`
	CallGasLimit         string   `json:"callGasLimit"`
	VerificationGasLimit string   `json:"verificationGasLimit"`
	PreVerificationGas   string   `json:"preVerificationGas"`
	MaxFeePerGas        string   `json:"maxFeePerGas"`
	MaxPriorityFeePerGas string   `json:"maxPriorityFeePerGas"`
	PaymasterAndData    string   `json:"paymasterAndData"`
	Signature           string   `json:"signature"`
}

// SmartAccount configuration
type SmartAccount struct {
	ID                string    `json:"id"`
	Owner             string    `json:"owner"`
	Address           string    `json:"address"`
	FactoryAddress    string    `json:"factory_address"`
	EntryPointAddress string    `json:"entry_point_address"`
	ChainID           uint64    `json:"chain_id"`
	IsDeployed        bool      `json:"is_deployed"`
	Nonce             uint64    `json:"nonce"`
	Guardians         []Guardian `json:"guardians"`
	Threshold         uint8     `json:"threshold"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Guardian for social recovery
type Guardian struct {
	Address   string    `json:"address"`
	Name      string    `json:"name"`
	Weight    uint8     `json:"weight"`
	IsActive  bool      `json:"is_active"`
	AddedAt   time.Time `json:"added_at"`
}

// Paymaster configuration
type PaymasterConfig struct {
	ID             string   `json:"id"`
	Address       string   `json:"address"`
	Owner         string   `json:"owner"`
	ChainIDs      []uint64 `json:"chain_ids"`
	FeePercentage float64  `json:"fee_percentage"`
	IsActive      bool     `json:"is_active"`
	Whitelist     []string `json:"whitelist"`
	Blacklist     []string `json:"blacklist"`
}

// Session key for gasless transactions
type SessionKey struct {
	ID             string    `json:"id"`
	AccountID      string    `json:"account_id"`
	Address        string    `json:"address"`
	KeyHash        string    `json:"key_hash"`
	Permissions    string    `json:"permissions"`
	SpendingLimit  string    `json:"spending_limit"`
	Expiration     time.Time `json:"expiration"`
	IsActive       bool      `json:"is_active"`
	RemainingUses  uint64    `json:"remaining_uses"`
	MaxUses        uint64    `json:"max_uses"`
}

// Operation status
type OperationStatus string

const (
	StatusPending    OperationStatus = "pending"
	StatusQueued    OperationStatus = "queued"
	StatusSponsored  OperationStatus = "sponsored"
	StatusVerifying OperationStatus = "verifying"
	StatusConfirmed OperationStatus = "confirmed"
	StatusFailed    OperationStatus = "failed"
)

// Bundler transaction
type BundlerTransaction struct {
	ID            string          `json:"id"`
	UserOpHash   string          `json:"user_op_hash"`
	UserOp       *UserOperation  `json:"user_op"`
	Status       OperationStatus `json:"status"`
	GasFees      GasFees         `json:"gas_fees"`
	BlockNumber  uint64          `json:"block_number"`
	BlockHash    string          `json:"block_hash"`
	TransactionHash string       `json:"transaction_hash"`
	Confirmations uint64         `json:"confirmations"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// Gas fees structure
type GasFees struct {
	PreVerificationGas string `json:"pre_verification_gas"`
	VerificationGas    string `json:"verification_gas"`
	CallGasLimit      string `json:"call_gas_limit"`
	MaxFeePerGas     string `json:"max_fee_per_gas"`
	MaxPriorityFee    string `json:"max_priority_fee"`
	TotalGasCost     string `json:"total_gas_cost"`
	UserOpHash       string `json:"user_op_hash"`
}

// Signature verification result
type SignatureVerification struct {
	IsValid bool   `json:"is_valid"`
	Signer  string `json:"signer"`
	Error   string `json:"error,omitempty"`
}

// ============================================================================
// SERVICE IMPLEMENTATION
// ============================================================================

// AccountAbstractionService main service
type AccountAbstractionService struct {
	mu               sync.RWMutex
	accounts         map[string]*SmartAccount
	operations       map[string]*BundlerTransaction
	sessionKeys      map[string]*SessionKey
	paymasters       map[string]*PaymasterConfig
	factoryAddress   string
	entryPointAddress string
}

// NewAccountAbstractionService creates new service
func NewAccountAbstractionService() *AccountAbstractionService {
	return &AccountAbstractionService{
		accounts:          make(map[string]*SmartAccount),
		operations:        make(map[string]*BundlerTransaction),
		sessionKeys:       make(map[string]*SessionKey),
		paymasters:        make(map[string]*PaymasterConfig),
		factoryAddress:    "0x...", // Deploy factory contract
		entryPointAddress: "0x5FF137D4b0FD9D6A97D4cE4d8F8f4f3f2E8d9cA", // ERC-4337 EntryPoint
	}
}

// ============================================================================
// SMART ACCOUNT FUNCTIONS
// ============================================================================

// CreateSmartAccount creates a new smart account
func (s *AccountAbstractionService) CreateSmartAccount(ctx context.Context, owner string, chainID uint64) (*SmartAccount, error) {
	// Validate owner address
	if owner == "" {
		return nil, fmt.Errorf("owner address required")
	}

	// Generate account address (in production, this would call factory contract)
	accountID := generateID("account")
	accountAddress := s.generateAccountAddress(owner, chainID)

	account := &SmartAccount{
		ID:                accountID,
		Owner:             owner,
		Address:           accountAddress,
		FactoryAddress:    s.factoryAddress,
		EntryPointAddress: s.entryPointAddress,
		ChainID:           chainID,
		IsDeployed:        false,
		Nonce:             0,
		Guardians:         []Guardian{},
		Threshold:         1,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	s.mu.Lock()
	s.accounts[accountID] = account
	s.mu.Unlock()

	return account, nil
}

// GetSmartAccount returns account by ID
func (s *AccountAbstractionService) GetSmartAccount(accountID string) (*SmartAccount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	account, ok := s.accounts[accountID]
	if !ok {
		return nil, fmt.Errorf("account not found: %s", accountID)
	}

	return account, nil
}

// GetSmartAccountByAddress returns account by address
func (s *AccountAbstractionService) GetSmartAccountByAddress(address string) (*SmartAccount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, account := range s.accounts {
		if strings.EqualFold(account.Address, address) {
			return account, nil
		}
	}

	return nil, fmt.Errorf("account not found: %s", address)
}

// GetUserAccounts returns all accounts for a user
func (s *AccountAbstractionService) GetUserAccounts(owner string) []*SmartAccount {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var accounts []*SmartAccount
	for _, account := range s.accounts {
		if strings.EqualFold(account.Owner, owner) {
			accounts = append(accounts, account)
		}
	}

	return accounts
}

// AddGuardian adds a guardian to the account
func (s *AccountAbstractionService) AddGuardian(accountID, address, name string, weight uint8) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	account, ok := s.accounts[accountID]
	if !ok {
		return fmt.Errorf("account not found")
	}

	guardian := Guardian{
		Address:  address,
		Name:     name,
		Weight:   weight,
		IsActive: true,
		AddedAt:  time.Now(),
	}

	account.Guardians = append(account.Guardians, guardian)
	account.UpdatedAt = time.Now()

	return nil
}

// RemoveGuardian removes a guardian from account
func (s *AccountAbstractionService) RemoveGuardian(accountID, address string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	account, ok := s.accounts[accountID]
	if !ok {
		return fmt.Errorf("account not found")
	}

	newGuardians := []Guardian{}
	for _, g := range account.Guardians {
		if !strings.EqualFold(g.Address, address) {
			newGuardians = append(newGuardians, g)
		}
	}

	account.Guardians = newGuardians
	account.UpdatedAt = time.Now()

	return nil
}

// UpdateThreshold updates the signature threshold
func (s *AccountAbstractionService) UpdateThreshold(accountID string, threshold uint8) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	account, ok := s.accounts[accountID]
	if !ok {
		return fmt.Errorf("account not found")
	}

	if threshold == 0 || threshold > uint8(len(account.Guardians)) {
		return fmt.Errorf("invalid threshold")
	}

	account.Threshold = threshold
	account.UpdatedAt = time.Now()

	return nil
}

// ============================================================================
// USER OPERATION FUNCTIONS
// ============================================================================

// SendUserOperation sends a user operation
func (s *AccountAbstractionService) SendUserOperation(ctx context.Context, userOp *UserOperation) (*BundlerTransaction, error) {
	// Validate user operation
	if userOp.Sender == "" {
		return nil, fmt.Errorf("sender required")
	}

	// Get account
	account, err := s.GetSmartAccountByAddress(userOp.Sender)
	if err != nil {
		return nil, err
	}

	// Calculate gas fees
	gasFees := s.calculateGasFees(userOp)

	// Generate user operation hash
	userOpHash := s.generateUserOpHash(userOp)

	tx := &BundlerTransaction{
		ID:              generateID("op"),
		UserOpHash:      userOpHash,
		UserOp:          userOp,
		Status:          StatusPending,
		GasFees:         gasFees,
		BlockNumber:     0,
		BlockHash:       "",
		TransactionHash: "",
		Confirmations:   0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	s.mu.Lock()
	s.operations[tx.ID] = tx
	s.mu.Unlock()

	// Simulate operation
	go s.simulateOperation(tx.ID)

	return tx, nil
}

// GetOperation returns operation by ID
func (s *AccountAbstractionService) GetOperation(opID string) (*BundlerTransaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	op, ok := s.operations[opID]
	if !ok {
		return nil, fmt.Errorf("operation not found")
	}

	return op, nil
}

// GetOperationByHash returns operation by user operation hash
func (s *AccountAbstractionService) GetOperationByHash(opHash string) (*BundlerTransaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, op := range s.operations {
		if op.UserOpHash == opHash {
			return op, nil
		}
	}

	return nil, fmt.Errorf("operation not found")
}

// GetAccountOperations returns all operations for an account
func (s *AccountAbstractionService) GetAccountOperations(accountAddress string) []*BundlerTransaction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var ops []*BundlerTransaction
	for _, op := range s.operations {
		if op.UserOp != nil && strings.EqualFold(op.UserOp.Sender, accountAddress) {
			ops = append(ops, op)
		}
	}

	return ops
}

// simulateOperation simulates the operation execution
func (s *AccountAbstractionService) simulateOperation(opID string) {
	s.mu.Lock()
	op, ok := s.operations[opID]
	if !ok {
		s.mu.Unlock()
		return
	}

	op.Status = StatusVerifying
	s.mu.Unlock()

	// Simulate verification delay
	time.Sleep(500 * time.Millisecond)

	s.mu.Lock()
	op.Status = StatusSponsored
	op.UpdatedAt = time.Now()
	s.mu.Unlock()

	// Simulate confirmation delay
	time.Sleep(1 * time.Second)

	s.mu.Lock()
	op.Status = StatusConfirmed
	op.BlockNumber = 19000000
	op.BlockHash = "0x..."
	op.TransactionHash = "0x" + generateID("tx")
	op.Confirmations = 1
	op.UpdatedAt = time.Now()
	s.mu.Unlock()
}

// estimateGas estimates gas for user operation
func (s *AccountAbstractionService) estimateGas(userOp *UserOperation) (GasFees, error) {
	// Simplified gas estimation
	// In production, this would use eth_estimateGas

	preVerificationGas := "21000"
	verificationGas := "150000"
	callGasLimit := "100000"

	maxFeePerGas := "100000000000"  // 100 gwei
	maxPriorityFee := "1000000000"   // 1 gwei

	// Calculate total
	preVer := new(big.Int)
	preVer.SetString(preVerificationGas, 10)
	ver := new(big.Int)
	ver.SetString(verificationGas, 10)
	call := new(big.Int)
	call.SetString(callGasLimit, 10)
	fee := new(big.Int)
	fee.SetString(maxFeePerGas, 10)

	total := new(big.Int).Add(preVer, ver)
	total = new(big.Int).Add(total, call)
	totalGasCost := new(big.Int).Mul(total, fee)

	return GasFees{
		PreVerificationGas: preVerificationGas,
		VerificationGas:     verificationGas,
		CallGasLimit:        callGasLimit,
		MaxFeePerGas:        maxFeePerGas,
		MaxPriorityFee:      maxPriorityFee,
		TotalGasCost:        totalGasCost.String(),
		UserOpHash:          s.generateUserOpHash(userOp),
	}, nil
}

// calculateGasFees calculates gas fees
func (s *AccountAbstractionService) calculateGasFees(userOp *UserOperation) GasFees {
	fees, _ := s.estimateGas(userOp)
	return fees
}

// generateUserOpHash generates user operation hash
func (s *AccountAbstractionService) generateUserOpHash(userOp *UserOperation) string {
	data := fmt.Sprintf("%s%s%s%s%s%s%s%s%s",
		userOp.Sender,
		userOp.Nonce,
		userOp.InitCode,
		userOp.CallData,
		userOp.CallGasLimit,
		userOp.VerificationGasLimit,
		userOp.PreVerificationGas,
		userOp.MaxFeePerGas,
		userOp.MaxPriorityFeePerGas,
	)

	hash := sha256.Sum256([]byte(data))
	return "0x" + hex.EncodeToString(hash[:])
}

// ============================================================================
// SESSION KEY FUNCTIONS
// ============================================================================

// CreateSessionKey creates a new session key
func (s *AccountAbstractionService) CreateSessionKey(ctx context.Context, accountID, permissions string, spendingLimit string, maxUses uint64, expiration time.Time) (*SessionKey, error) {
	account, err := s.GetSmartAccount(accountID)
	if err != nil {
		return nil, err
	}

	// Generate session key (in production, use proper key generation)
	keyAddress := generateID("key")
	keyHash := sha256.Sum256([]byte(keyAddress))

	sessionKey := &SessionKey{
		ID:             generateID("session"),
		AccountID:      accountID,
		Address:        "0x" + hex.EncodeToString(keyHash[:20]),
		KeyHash:        "0x" + hex.EncodeToString(keyHash[:]),
		Permissions:    permissions,
		SpendingLimit:  spendingLimit,
		Expiration:     expiration,
		IsActive:       true,
		RemainingUses:  maxUses,
		MaxUses:        maxUses,
	}

	s.mu.Lock()
	s.sessionKeys[sessionKey.ID] = sessionKey
	s.mu.Unlock()

	return sessionKey, nil
}

// GetSessionKey returns session key by ID
func (s *AccountAbstractionService) GetSessionKey(keyID string) (*SessionKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, ok := s.sessionKeys[keyID]
	if !ok {
		return nil, fmt.Errorf("session key not found")
	}

	return key, nil
}

// RevokeSessionKey revokes a session key
func (s *AccountAbstractionService) RevokeSessionKey(keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, ok := s.sessionKeys[keyID]
	if !ok {
		return fmt.Errorf("session key not found")
	}

	key.IsActive = false
	return nil
}

// UseSessionKey decrements remaining uses
func (s *AccountAbstractionService) UseSessionKey(keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, ok := s.sessionKeys[keyID]
	if !ok {
		return fmt.Errorf("session key not found")
	}

	if !key.IsActive {
		return fmt.Errorf("session key is not active")
	}

	if key.RemainingUses == 0 {
		return fmt.Errorf("session key has no remaining uses")
	}

	key.RemainingUses--
	if key.RemainingUses == 0 {
		key.IsActive = false
	}

	return nil
}

// ============================================================================
// PAYMASTER FUNCTIONS
// ============================================================================

// CreatePaymaster creates a new paymaster
func (s *AccountAbstractionService) CreatePaymaster(ctx context.Context, owner string, chainIDs []uint64, feePercentage float64) (*PaymasterConfig, error) {
	paymaster := &PaymasterConfig{
		ID:             generateID("pm"),
		Owner:          owner,
		Address:        "0x" + generateID("pm_addr"), // In production, deploy contract
		ChainIDs:       chainIDs,
		FeePercentage:  feePercentage,
		IsActive:       true,
		Whitelist:      []string{},
		Blacklist:      []string{},
	}

	s.mu.Lock()
	s.paymasters[paymaster.ID] = paymaster
	s.mu.Unlock()

	return paymaster, nil
}

// GetPaymaster returns paymaster by ID
func (s *AccountAbstractionService) GetPaymaster(pmID string) (*PaymasterConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pm, ok := s.paymasters[pmID]
	if !ok {
		return nil, fmt.Errorf("paymaster not found")
	}

	return pm, nil
}

// SponsorOperation checks if paymaster will sponsor operation
func (s *AccountAbstractionService) SponsorOperation(pmID, sender string) (bool, error) {
	pm, err := s.GetPaymaster(pmID)
	if err != nil {
		return false, err
	}

	if !pm.IsActive {
		return false, fmt.Errorf("paymaster is not active")
	}

	// Check whitelist/blacklist
	for _, addr := range pm.Blacklist {
		if strings.EqualFold(addr, sender) {
			return false, fmt.Errorf("sender is blacklisted")
		}
	}

	if len(pm.Whitelist) > 0 {
		allowed := false
		for _, addr := range pm.Whitelist {
			if strings.EqualFold(addr, sender) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false, fmt.Errorf("sender not in whitelist")
		}
	}

	return true, nil
}

// AddToWhitelist adds address to paymaster whitelist
func (s *AccountAbstractionService) AddToWhitelist(pmID, address string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pm, ok := s.paymasters[pmID]
	if !ok {
		return fmt.Errorf("paymaster not found")
	}

	for _, addr := range pm.Whitelist {
		if strings.EqualFold(addr, address) {
			return nil // Already in whitelist
		}
	}

	pm.Whitelist = append(pm.Whitelist, address)
	return nil
}

// AddToBlacklist adds address to paymaster blacklist
func (s *AccountAbstractionService) AddToBlacklist(pmID, address string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pm, ok := s.paymasters[pmID]
	if !ok {
		return fmt.Errorf("paymaster not found")
	}

	for _, addr := range pm.Blacklist {
		if strings.EqualFold(addr, address) {
			return nil // Already in blacklist
		}
	}

	pm.Blacklist = append(pm.Blacklist, address)
	return nil
}

// ============================================================================
// SIGNATURE VERIFICATION
// ============================================================================

// VerifyUserOpSignature verifies user operation signature
func (s *AccountAbstractionService) VerifyUserOpSignature(userOp *UserOperation, signature string) SignatureVerification {
	// In production, this would:
	// 1. Reconstruct userOpHash
	// 2. Verify signature against account owner or session key

	userOpHash := s.generateUserOpHash(userOp)

	if signature == "" {
		return SignatureVerification{
			IsValid: false,
			Signer:  "",
			Error:   "empty signature",
		}
	}

	// Simplified verification
	// In production, use proper ECDSA signature verification
	return SignatureVerification{
		IsValid: true,
		Signer:  userOp.Sender, // Simplified
		Error:   "",
	}
}

// VerifyGuardianSignature verifies signature from guardians
func (s *AccountAbstractionService) VerifyGuardianSignature(accountID string, messageHash string, signatures []string) (bool, error) {
	account, err := s.GetSmartAccount(accountID)
	if err != nil {
		return false, err
	}

	// Collect weight of signers
	totalWeight := uint8(0)
	for i, sig := range signatures {
		if i >= len(account.Guardians) {
			break
		}
		if account.Guardians[i].IsActive && sig != "" {
			totalWeight += account.Guardians[i].Weight
		}
	}

	// Check if threshold met
	if totalWeight >= account.Threshold {
		return true, nil
	}

	return false, fmt.Errorf("insufficient signatures: got %d, need %d", totalWeight, account.Threshold)
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func (s *AccountAbstractionService) generateAccountAddress(owner string, chainID uint64) string {
	data := fmt.Sprintf("%s%d", owner, chainID)
	hash := sha256.Sum256([]byte(data))
	return "0x" + hex.EncodeToString(hash[:20])
}

func generateID(prefix string) string {
	timestamp := time.Now().UnixNano()
	buf := make([]byte, 16)
	rand.Read(buf)
	return fmt.Sprintf("%s_%d_%x", prefix, timestamp, buf[:8])
}

// ============================================================================
// HTTP HANDLERS
// ============================================================================

func (s *AccountAbstractionService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := r.URL.Path
	method := r.Method

	switch {
	case path == "/api/v1/account" && method == http.MethodPost:
		s.handleCreateAccount(w, r)
	case strings.HasPrefix(path, "/api/v1/account/") && method == http.MethodGet:
		s.handleGetAccount(w, r)
	case path == "/api/v1/account/guardians" && method == http.MethodPost:
		s.handleAddGuardian(w, r)
	case path == "/api/v1/account/operations" && method == http.MethodPost:
		s.handleSendOperation(w, r)
	case strings.HasPrefix(path, "/api/v1/operation/") && method == http.MethodGet:
		s.handleGetOperation(w, r)
	case path == "/api/v1/session" && method == http.MethodPost:
		s.handleCreateSessionKey(w, r)
	case path == "/api/v1/paymaster" && method == http.MethodPost:
		s.handleCreatePaymaster(w, r)
	case path == "/api/v1/estimate-gas" && method == http.MethodPost:
		s.handleEstimateGas(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *AccountAbstractionService) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Owner  string `json:"owner"`
		ChainID uint64 `json:"chain_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	account, err := s.CreateSmartAccount(r.Context(), req.Owner, req.ChainID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(account)
}

func (s *AccountAbstractionService) handleGetAccount(w http.ResponseWriter, r *http.Request) {
	accountID := strings.TrimPrefix(r.URL.Path, "/api/v1/account/")

	account, err := s.GetSmartAccount(accountID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(account)
}

func (s *AccountAbstractionService) handleAddGuardian(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AccountID string `json:"account_id"`
		Address   string `json:"address"`
		Name      string `json:"name"`
		Weight    uint8  `json:"weight"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.AddGuardian(req.AccountID, req.Address, req.Name, req.Weight); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *AccountAbstractionService) handleSendOperation(w http.ResponseWriter, r *http.Request) {
	var userOp UserOperation
	if err := json.NewDecoder(r.Body).Decode(&userOp); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tx, err := s.SendUserOperation(r.Context(), &userOp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(tx)
}

func (s *AccountAbstractionService) handleGetOperation(w http.ResponseWriter, r *http.Request) {
	opID := strings.TrimPrefix(r.URL.Path, "/api/v1/operation/")

	op, err := s.GetOperation(opID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(op)
}

func (s *AccountAbstractionService) handleCreateSessionKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AccountID     string  `json:"account_id"`
		Permissions   string  `json:"permissions"`
		SpendingLimit string  `json:"spending_limit"`
		MaxUses       uint64  `json:"max_uses"`
		Expiration    int64   `json:"expiration"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	key, err := s.CreateSessionKey(r.Context(), req.AccountID, req.Permissions, req.SpendingLimit, req.MaxUses, time.Unix(req.Expiration, 0))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(key)
}

func (s *AccountAbstractionService) handleCreatePaymaster(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Owner         string  `json:"owner"`
		ChainIDs      []uint64 `json:"chain_ids"`
		FeePercentage float64 `json:"fee_percentage"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pm, err := s.CreatePaymaster(r.Context(), req.Owner, req.ChainIDs, req.FeePercentage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(pm)
}

func (s *AccountAbstractionService) handleEstimateGas(w http.ResponseWriter, r *http.Request) {
	var userOp UserOperation
	if err := json.NewDecoder(r.Body).Decode(&userOp); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fees, err := s.estimateGas(&userOp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(fees)
}

// ============================================================================
// MAIN FUNCTION
// ============================================================================

func main() {
	service := NewAccountAbstractionService()

	// Pre-create some test accounts
	_, _ = service.CreateSmartAccount("0x742d35Cc6634C0532925a3b844Bc9e7595f8aB1E", 1)
	_, _ = service.CreateSmartAccount("0x1234567890abcdef1234567890abcdef12345678", 1)

	fmt.Println("Starting Account Abstraction Service on :8081")
	http.HandleFunc("/", service.ServeHTTP)

	if err := http.ListenAndServe(":8081", nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

// Utility to prevent unused import error
var _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
var _ = bytes.Buffer{}
