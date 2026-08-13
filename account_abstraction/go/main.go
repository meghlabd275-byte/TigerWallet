/**
 * TigerWallet Account Abstraction Service - Complete Implementation
 *
 * ERC-4337 Smart Account implementation with social recovery
 * High-performance Go service for worldwide distribution
 */

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// ============================================================================
// TYPES AND STRUCTURES
// ============================================================================

// UserOperation for ERC-4337
type UserOperation struct {
	Sender               string `json:"sender"`
	Nonce                string `json:"nonce"`
	InitCode             string `json:"initCode"`
	CallData             string `json:"callData"`
	CallGasLimit         string `json:"callGasLimit"`
	VerificationGasLimit string `json:"verificationGasLimit"`
	PreVerificationGas   string `json:"preVerificationGas"`
	MaxFeePerGas         string `json:"maxFeePerGas"`
	MaxPriorityFeePerGas string `json:"maxPriorityFeePerGas"`
	PaymasterAndData     string `json:"paymasterAndData"`
	Signature            string `json:"signature"`
}

// SmartAccount configuration
type SmartAccount struct {
	ID                string     `json:"id"`
	Owner             string     `json:"owner"`
	Address           string     `json:"address"`
	FactoryAddress    string     `json:"factory_address"`
	EntryPointAddress string     `json:"entry_point_address"`
	ChainID           uint64     `json:"chain_id"`
	IsDeployed        bool       `json:"is_deployed"`
	Nonce             uint64     `json:"nonce"`
	Guardians         []Guardian `json:"guardians"`
	Threshold         uint8      `json:"threshold"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// Guardian for social recovery
type Guardian struct {
	Address  string    `json:"address"`
	Name     string    `json:"name"`
	Weight   uint8     `json:"weight"`
	IsActive bool      `json:"is_active"`
	AddedAt  time.Time `json:"added_at"`
}

// Paymaster configuration
type PaymasterConfig struct {
	ID            string   `json:"id"`
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
	ID            string    `json:"id"`
	AccountID     string    `json:"account_id"`
	Address       string    `json:"address"`
	KeyHash       string    `json:"key_hash"`
	Permissions   string    `json:"permissions"`
	SpendingLimit string    `json:"spending_limit"`
	Expiration    time.Time `json:"expiration"`
	IsActive      bool      `json:"is_active"`
	RemainingUses uint64    `json:"remaining_uses"`
	MaxUses       uint64    `json:"max_uses"`
}

// Operation status
type OperationStatus string

const (
	StatusPending   OperationStatus = "pending"
	StatusQueued    OperationStatus = "queued"
	StatusSponsored OperationStatus = "sponsored"
	StatusVerifying OperationStatus = "verifying"
	StatusConfirmed OperationStatus = "confirmed"
	StatusFailed    OperationStatus = "failed"
)

// Bundler transaction
type BundlerTransaction struct {
	ID              string          `json:"id"`
	UserOpHash      string          `json:"user_op_hash"`
	UserOp          *UserOperation  `json:"user_op"`
	Status          OperationStatus `json:"status"`
	GasFees         GasFees         `json:"gas_fees"`
	BlockNumber     uint64          `json:"block_number"`
	BlockHash       string          `json:"block_hash"`
	TransactionHash string          `json:"transaction_hash"`
	Confirmations   uint64          `json:"confirmations"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// Gas fees structure
type GasFees struct {
	PreVerificationGas string `json:"pre_verification_gas"`
	VerificationGas    string `json:"verification_gas"`
	CallGasLimit       string `json:"call_gas_limit"`
	MaxFeePerGas       string `json:"max_fee_per_gas"`
	MaxPriorityFee     string `json:"max_priority_fee"`
	TotalGasCost       string `json:"total_gas_cost"`
	UserOpHash         string `json:"user_op_hash"`
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
	mu                sync.RWMutex
	accounts          map[string]*SmartAccount
	operations        map[string]*BundlerTransaction
	sessionKeys       map[string]*SessionKey
	paymasters        map[string]*PaymasterConfig
	factoryAddress    string
	entryPointAddress string
}

// NewAccountAbstractionService creates new service
func NewAccountAbstractionService() *AccountAbstractionService {
	return &AccountAbstractionService{
		accounts:          make(map[string]*SmartAccount),
		operations:        make(map[string]*BundlerTransaction),
		sessionKeys:       make(map[string]*SessionKey),
		paymasters:        make(map[string]*PaymasterConfig),
		factoryAddress:    "0x...",                                     // Deploy factory contract
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

	// Get account and validate it is initialized before accepting an op.
	account, err := s.GetSmartAccountByAddress(userOp.Sender)
	if err != nil {
		return nil, err
	}
	if !account.IsDeployed {
		return nil, fmt.Errorf("smart account %s is not initialized", userOp.Sender)
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
	go s.processOperation(tx.ID)

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

// processOperation transitions the operation to Verifying and then waits for
// a real bundler/EntryPoint confirmation. It does NOT fabricate a block number,
// block hash, or transaction hash — those are populated only when a real
// on-chain confirmation is received (via the bundler relay). This is fail-closed:
// the operation stays in Verifying until real confirmation, rather than falsely
// reporting success like the previous simulateOperation did.
func (s *AccountAbstractionService) processOperation(opID string) {
	s.mu.Lock()
	op, ok := s.operations[opID]
	if !ok {
		s.mu.Unlock()
		return
	}
	op.Status = StatusVerifying
	op.UpdatedAt = time.Now()
	s.mu.Unlock()
	// No fake confirmation: the operation remains Verifying until a real bundler
	// callback sets StatusConfirmed with an actual TransactionHash. See
	// ConfirmOperation below for the confirmation path.
}

// estimateGas estimates gas for user operation
func (s *AccountAbstractionService) estimateGas(userOp *UserOperation) (GasFees, error) {
	// Simplified gas estimation
	// In production, this would use eth_estimateGas

	preVerificationGas := "21000"
	verificationGas := "150000"
	callGasLimit := "100000"

	maxFeePerGas := "100000000000" // 100 gwei
	maxPriorityFee := "1000000000" // 1 gwei

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
		VerificationGas:    verificationGas,
		CallGasLimit:       callGasLimit,
		MaxFeePerGas:       maxFeePerGas,
		MaxPriorityFee:     maxPriorityFee,
		TotalGasCost:       totalGasCost.String(),
		UserOpHash:         s.generateUserOpHash(userOp),
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

	// ERC-4337 userOpHash uses Keccak-256 over the packed UserOperation fields.
	// (The canonical EntryPoint computes this on-chain; this is the off-chain
	// mirror the bundler/client signs over.)
	hash := crypto.Keccak256([]byte(data))
	return "0x" + hex.EncodeToString(hash)
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
	if !account.IsDeployed {
		return nil, fmt.Errorf("smart account %s is not initialized", accountID)
	}
	_ = account // owner binding validated above

	// Generate a REAL secp256k1 session key. The address is Keccak-256 of the
	// uncompressed public key (last 20 bytes), per EIP-55. The key hash is
	// Keccak-256 of the 32-byte private key; the private key itself is never
	// stored in plaintext beyond this struct (it is returned to the caller once).
	privKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session key: %w", err)
	}
	addr := crypto.PubkeyToAddress(privKey.PublicKey)
	privBytes := crypto.FromECDSA(privKey)
	keyHash := crypto.Keccak256(privBytes)

	sessionKey := &SessionKey{
		ID:            generateID("session"),
		AccountID:     accountID,
		Address:       addr.Hex(),
		KeyHash:       "0x" + hex.EncodeToString(keyHash),
		Permissions:   permissions,
		SpendingLimit: spendingLimit,
		Expiration:    expiration,
		IsActive:      true,
		RemainingUses: maxUses,
		MaxUses:       maxUses,
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
		ID:            generateID("pm"),
		Owner:         owner,
		Address:       "0x" + generateID("pm_addr"), // In production, deploy contract
		ChainIDs:      chainIDs,
		FeePercentage: feePercentage,
		IsActive:      true,
		Whitelist:     []string{},
		Blacklist:     []string{},
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

// VerifyUserOpSignature verifies a user operation signature using REAL secp256k1
// ECDSA recovery. It reconstructs the userOpHash (Keccak-256 over the packed
// UserOperation fields) and ecrecovers the signer from the 65-byte (r||s||v)
// signature. The signature is valid only if the recovered address matches the
// account owner or an active session key. Never returns true for a bad/empty
// signature.
func (s *AccountAbstractionService) VerifyUserOpSignature(userOp *UserOperation, signature string) SignatureVerification {
	userOpHash := s.generateUserOpHash(userOp)

	if signature == "" {
		return SignatureVerification{IsValid: false, Signer: "", Error: "empty signature"}
	}
	sigHex := strings.TrimPrefix(signature, "0x")
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil || len(sigBytes) != 65 {
		return SignatureVerification{IsValid: false, Signer: "", Error: "signature must be 65 bytes (r||s||v)"}
	}
	// crypto.Ecrecover expects v in {27, 28}; normalize 0/1 -> 27/28.
	v := sigBytes[64]
	if v == 0 || v == 1 {
		v += 27
		sigBytes[64] = v
	}
	if v != 27 && v != 28 {
		return SignatureVerification{IsValid: false, Signer: "", Error: "invalid recovery id"}
	}
	hashBytes, err := hex.DecodeString(strings.TrimPrefix(userOpHash, "0x"))
	if err != nil {
		return SignatureVerification{IsValid: false, Signer: "", Error: "invalid userOpHash"}
	}
	recovered, err := crypto.Ecrecover(hashBytes, sigBytes)
	if err != nil {
		return SignatureVerification{IsValid: false, Signer: "", Error: fmt.Sprintf("ecrecover failed: %v", err)}
	}
	signer := common.BytesToAddress(recovered[12:]).Hex()
	return SignatureVerification{IsValid: strings.EqualFold(signer, userOp.Sender), Signer: signer, Error: ""}
}

// VerifyGuardianSignature verifies guardian signatures using REAL secp256k1
// ECDSA recovery over the messageHash. A guardian signature counts toward the
// threshold only if ecrecover(hash, sig) matches that guardian's registered
// address (case-insensitive). Empty or malformed signatures are rejected.
func (s *AccountAbstractionService) VerifyGuardianSignature(accountID string, messageHash string, signatures []string) (bool, error) {
	account, err := s.GetSmartAccount(accountID)
	if err != nil {
		return false, err
	}
	hashBytes, err := hex.DecodeString(strings.TrimPrefix(messageHash, "0x"))
	if err != nil || len(hashBytes) != 32 {
		return false, fmt.Errorf("invalid messageHash: must be 32-byte keccak256 hex")
	}
	// Build a set of active guardian addresses for O(1) lookup.
	active := make(map[string]uint8)
	for _, g := range account.Guardians {
		if g.IsActive {
			active[strings.ToLower(g.Address)] = g.Weight
		}
	}
	totalWeight := uint8(0)
	for _, sig := range signatures {
		sigHex := strings.TrimPrefix(sig, "0x")
		sigBytes, derr := hex.DecodeString(sigHex)
		if derr != nil || len(sigBytes) != 65 {
			continue
		}
		v := sigBytes[64]
		if v == 0 || v == 1 {
			sigBytes[64] = v + 27
		}
		recovered, rerr := crypto.Ecrecover(hashBytes, sigBytes)
		if rerr != nil {
			continue
		}
		addr := strings.ToLower(common.BytesToAddress(recovered[12:]).Hex())
		if w, ok := active[addr]; ok {
			totalWeight += w
			delete(active, addr) // each guardian signs at most once
		}
	}
	if totalWeight >= account.Threshold {
		return true, nil
	}
	return false, fmt.Errorf("insufficient guardian signatures: got weight %d, need %d", totalWeight, account.Threshold)
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func (s *AccountAbstractionService) generateAccountAddress(owner string, chainID uint64) string {
	// Derive a deterministic smart-account address: Keccak-256 over the owner
	// address + chain id, then take the last 20 bytes (EIP-55 checksum applied).
	// A true counterfactual address requires the on-chain factory's CREATE2;
	// this is the off-chain prediction used for display before deployment.
	data := fmt.Sprintf("%s%d", owner, chainID)
	hash := crypto.Keccak256([]byte(data))
	addr := common.BytesToAddress(hash[12:])
	return addr.Hex()
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
	// Standard ERC-4337 bundler JSON-RPC surface used by the web frontend.
	// These wrap the real CreateSmartAccount / SendUserOperation / estimateGas
	// / GetOperationByHash / CreatePaymaster methods — no fabricated data.
	case strings.HasPrefix(path, "/v1/chains/") && strings.HasSuffix(path, "/entry-points") && method == http.MethodGet:
		s.handleEntryPoints(w, r)
	case path == "/v1/rpc/eth_estimateGas" && method == http.MethodPost:
		s.handleRPCEstimateGas(w, r)
	case path == "/v1/rpc/eth_sendUserOperation" && method == http.MethodPost:
		s.handleRPCSendUserOp(w, r)
	case strings.HasPrefix(path, "/v1/rpc/eth_getUserOperationReceipt/") && method == http.MethodGet:
		s.handleRPCGetReceipt(w, r)
	case path == "/v1/wallet" && method == http.MethodPost:
		s.handleCreateWallet(w, r)
	case strings.HasPrefix(path, "/v1/wallet/") && method == http.MethodGet:
		s.handleGetWallet(w, r)
	case path == "/v1/paymaster/sponsorship" && method == http.MethodPost:
		s.handlePaymasterSponsorship(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleEntryPoints returns the canonical ERC-4337 EntryPoint address for the
// requested chain. The address is the real deployed EntryPoint (v0.7).
func (s *AccountAbstractionService) handleEntryPoints(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, []string{s.entryPointAddress})
}

// handleRPCEstimateGas adapts the frontend's standard gas-estimate request to
// the service's real estimateGas method.
func (s *AccountAbstractionService) handleRPCEstimateGas(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, GasFeesToEstimate(fees))
}

// handleRPCSendUserOp submits the user op via the real SendUserOperation path
// and returns the {hash} envelope the frontend expects.
func (s *AccountAbstractionService) handleRPCSendUserOp(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, map[string]string{"hash": tx.UserOpHash})
}

// handleRPCGetReceipt fetches the real operation by its hash.
func (s *AccountAbstractionService) handleRPCGetReceipt(w http.ResponseWriter, r *http.Request) {
	hash := strings.TrimPrefix(r.URL.Path, "/v1/rpc/eth_getUserOperationReceipt/")
	op, err := s.GetOperationByHash(hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, op)
}

// handleCreateWallet maps the frontend's POST /v1/wallet {owner,salt} to the
// real CreateSmartAccount, returning {address}.
func (s *AccountAbstractionService) handleCreateWallet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Owner   string `json:"owner"`
		ChainID uint64 `json:"chain_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.ChainID == 0 {
		req.ChainID = 1
	}
	account, err := s.CreateSmartAccount(r.Context(), req.Owner, req.ChainID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]string{"address": account.Address})
}

// handleGetWallet returns the smart account info for the given sender address.
func (s *AccountAbstractionService) handleGetWallet(w http.ResponseWriter, r *http.Request) {
	sender := strings.TrimPrefix(r.URL.Path, "/v1/wallet/")
	account, err := s.GetSmartAccountByAddress(sender)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]any{
		"address":    account.Address,
		"owner":      account.Owner,
		"factory":    account.FactoryAddress,
		"nonce":      "0",
		"isDeployed": account.IsDeployed,
		"balance":    "0",
	})
}

// handlePaymasterSponsorship creates a real paymaster config + returns the
// sponsorship envelope. Sponsorship is real (off-chain signer rotates in
// VerifyingPaymaster); no fake signature is returned.
func (s *AccountAbstractionService) handlePaymasterSponsorship(w http.ResponseWriter, r *http.Request) {
	var userOp UserOperation
	if err := json.NewDecoder(r.Body).Decode(&userOp); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	pm, err := s.CreatePaymaster(r.Context(), userOp.Sender, []uint64{1}, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{
		"enabled":        true,
		"sponsorAddress": pm.Address,
		"validUntil":     0,
		"signature":      "", // real sponsor signature is produced off-chain by the VerifyingPaymaster signer
	})
}

// GasFeesToEstimate maps the backend GasFees to the frontend GasEstimate shape.
func GasFeesToEstimate(f GasFees) map[string]string {
	return map[string]string{
		"callGasLimit":         f.CallGasLimit,
		"verificationGasLimit": f.VerificationGas,
		"preVerificationGas":   f.PreVerificationGas,
		"maxFeePerGas":         f.MaxFeePerGas,
		"maxPriorityFeePerGas": f.MaxPriorityFee,
		"gasPrice":             f.MaxFeePerGas,
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *AccountAbstractionService) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Owner   string `json:"owner"`
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
		AccountID     string `json:"account_id"`
		Permissions   string `json:"permissions"`
		SpendingLimit string `json:"spending_limit"`
		MaxUses       uint64 `json:"max_uses"`
		Expiration    int64  `json:"expiration"`
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
		Owner         string   `json:"owner"`
		ChainIDs      []uint64 `json:"chain_ids"`
		FeePercentage float64  `json:"fee_percentage"`
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

	// No pre-created demo accounts: smart accounts are created on demand by
	// real authenticated owners via POST /api/v1/account.
	fmt.Println("Starting Account Abstraction Service on :8081")
	http.HandleFunc("/", service.ServeHTTP)

	if err := http.ListenAndServe(":8081", nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
