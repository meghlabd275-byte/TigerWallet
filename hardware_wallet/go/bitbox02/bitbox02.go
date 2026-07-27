package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// BitBox02 Wallet Implementation
// ============================================================================

// BitBox02Wallet implements HardwareWallet for BitBox02 devices
type BitBox02Wallet struct {
	mu          sync.RWMutex
	device      *BitBox02Device
	isConnected bool
	session     *BitBox02Session
}

type BitBox02Device struct {
	Path        string
	ProductID   uint16
	VendorID    uint16
	Serial      string
	Firmware    string
	Edition     string // Standard or Advanced
}

type BitBox02Session struct {
	ID        string
	SessionID string
}

// NewBitBox02Wallet creates a new BitBox02 wallet instance
func NewBitBox02Wallet() *BitBox02Wallet {
	return &BitBox02Wallet{
		device:      nil,
		isConnected: false,
		session:     nil,
	}
}

// Connect connects to a BitBox02 device
func (w *BitBox02Wallet) Connect(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	devices, err := discoverBitBox02Devices()
	if err != nil {
		return fmt.Errorf("failed to discover devices: %w", err)
	}
	
	if len(devices) == 0 {
		return fmt.Errorf("no BitBox02 device found")
	}
	
	device := devices[0]
	
	session, err := startBitBox02Session(device)
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	
	w.device = device
	w.session = session
	w.isConnected = true
	
	fmt.Printf("Connected to BitBox02: %s Edition (%s)\n", device.Edition, device.Firmware)
	
	return nil
}

// Disconnect disconnects from the BitBox02 device
func (w *BitBox02Wallet) Disconnect() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if w.session != nil {
		endBitBox02Session(w.session)
	}
	
	w.device = nil
	w.session = nil
	w.isConnected = false
	
	return nil
}

// IsConnected returns connection status
func (w *BitBox02Wallet) IsConnected() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.isConnected
}

// GetPublicKey gets the public key for a derivation path
func (w *BitBox02Wallet) GetPublicKey(path string) (string, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	
	if !w.isConnected {
		return "", fmt.Errorf("wallet not connected")
	}
	
	derivationPath, err := parseBitBox02Path(path)
	if err != nil {
		return "", err
	}
	
	pubKey, err := getBitBox02PublicKey(w.session, derivationPath)
	if err != nil {
		return "", fmt.Errorf("failed to get public key: %w", err)
	}
	
	return hex.EncodeToString(pubKey), nil
}

// GetAddress gets the address for a derivation path
func (w *BitBox02Wallet) GetAddress(path string) (string, error) {
	// Support both ETH and BTC
	pubKey, err := w.GetPublicKey(path)
	if err != nil {
		return "", err
	}
	
	// Check if path suggests BTC or ETH
	if contains(path, "0'") { // Bitcoin
		return deriveBTCAddress(pubKey), nil
	}
	return deriveEVMAddress(pubKey), nil // Ethereum
}

// SignTransaction signs a transaction
func (w *BitBox02Wallet) SignTransaction(path string, tx []byte) ([]byte, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	
	if !w.isConnected {
		return nil, fmt.Errorf("wallet not connected")
	}
	
	derivationPath, err := parseBitBox02Path(path)
	if err != nil {
		return nil, err
	}
	
	signature, err := signBitBox02Transaction(w.session, derivationPath, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}
	
	return signature, nil
}

// SignMessage signs a message
func (w *BitBox02Wallet) SignMessage(path string, message []byte) ([]byte, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	
	if !w.isConnected {
		return nil, fmt.Errorf("wallet not connected")
	}
	
	derivationPath, err := parseBitBox02Path(path)
	if err != nil {
		return nil, err
	}
	
	signature, err := signBitBox02Message(w.session, derivationPath, message)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}
	
	return signature, nil
}

// GetFeatures returns device features
func (w *BitBox02Wallet) GetFeatures() (WalletFeatures, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	
	if !w.isConnected {
		return WalletFeatures{}, fmt.Errorf("wallet not connected")
	}
	
	return WalletFeatures{
		DeviceID:        w.device.Serial,
		DeviceModel:     "BitBox02 " + w.device.Edition,
		Manufacturer:   "Shift Crypto Security AG",
		FirmwareVersion: w.device.Firmware,
		BootloaderMode: false,
		Capabilities:   []string{"signing", "messages", "derivation", "mnemonic", "backup"},
		SupportsETH:    true,
		SupportsBTC:    true,
		SupportsSolana: false,
		SupportsNEAR:   false,
		SupportsAptos:  false,
	}, nil
}

// ============================================================================
// BitBox02 Session Management
// ============================================================================

func discoverBitBox02Devices() ([]*BitBox02Device, error) {
	// Simplified - return mock device
	return []*BitBox02Device{
		{
			Path:        "/dev/hidraw3",
			ProductID:  0x2403,
			VendorID:   0x03EB,
			Serial:     "BITBOX_001",
			Firmware:   "9.15.0",
			Edition:    "Advanced",
		},
	}, nil
}

func startBitBox02Session(device *BitBox02Device) (*BitBox02Session, error) {
	// Simplified - in production use bitbox02-api-go
	return &BitBox02Session{
		ID:        device.Serial,
		SessionID: fmt.Sprintf("session_%d", time.Now().Unix()),
	}, nil
}

func endBitBox02Session(session *BitBox02Session) {
	// Simplified
}

func getBitBox02PublicKey(session *BitBox02Session, path []uint32) ([]byte, error) {
	// Simplified
	return []byte{}, nil
}

func signBitBox02Transaction(session *BitBox02Session, path []uint32, tx []byte) ([]byte, error) {
	// Simplified
	return []byte{}, nil
}

func signBitBox02Message(session *BitBox02Session, path []uint32, message []byte) ([]byte, error) {
	// Simplified
	return []byte{}, nil
}

func parseBitBox02Path(path string) ([]uint32, error) {
	// Parse BIP44 path - support both BTC and ETH
	// For ETH: m/44'/60'/0'/0/0
	// For BTC: m/44'/0'/0'/0/0
	return []uint32{44 + 0x80000000, 60 + 0x80000000, 0 + 0x80000000, 0, 0}, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================================================
// BitBox02 Manager
// ============================================================================

type BitBox02Manager struct {
	mu       sync.RWMutex
	wallets  map[string]*BitBox02Wallet
	selected string
}

func NewBitBox02Manager() *BitBox02Manager {
	return &BitBox02Manager{
		wallets: make(map[string]*BitBox02Wallet),
	}
}

func (m *BitBox02Manager) Connect(ctx context.Context) (string, error) {
	wallet := NewBitBox02Wallet()
	if err := wallet.Connect(ctx); err != nil {
		return "", err
	}
	
	deviceID := fmt.Sprintf("bitbox02_%d", time.Now().Unix())
	m.mu.Lock()
	m.wallets[deviceID] = wallet
	m.selected = deviceID
	m.mu.Unlock()
	
	return deviceID, nil
}

func (m *BitBox02Manager) Disconnect(deviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	wallet, ok := m.wallets[deviceID]
	if !ok {
		return fmt.Errorf("device not found")
	}
	
	wallet.Disconnect()
	delete(m.wallets, deviceID)
	
	if m.selected == deviceID {
		m.selected = ""
	}
	
	return nil
}

func (m *BitBox02Manager) GetWallet(deviceID string) (*BitBox02Wallet, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	wallet, ok := m.wallets[deviceID]
	if !ok {
		return nil, fmt.Errorf("device not found")
	}
	
	return wallet, nil
}

// ============================================================================
// Unified Hardware Wallet Manager
// ============================================================================

type UnifiedWalletManager struct {
	ledger    *LedgerManager
	trezor    *TrezorManager
	coldcard  *ColdcardManager
	bitbox02  *BitBox02Manager
}

func NewUnifiedWalletManager() *UnifiedWalletManager {
	return &UnifiedWalletManager{
		ledger:   NewLedgerManager(),
		trezor:    NewTrezorManager(),
		coldcard:  NewColdcardManager(),
		bitbox02:  NewBitBox02Manager(),
	}
}

func (m *UnifiedWalletManager) ConnectAny(ctx context.Context) (string, string, error) {
	// Try each wallet type
	deviceID, err := m.ledger.Connect(ctx)
	if err == nil {
		return deviceID, "ledger", nil
	}
	
	deviceID, err = m.trezor.Connect(ctx)
	if err == nil {
		return deviceID, "trezor", nil
	}
	
	deviceID, err = m.coldcard.Connect(ctx)
	if err == nil {
		return deviceID, "coldcard", nil
	}
	
	deviceID, err = m.bitbox02.Connect(ctx)
	if err == nil {
		return deviceID, "bitbox02", nil
	}
	
	return "", "", fmt.Errorf("no hardware wallet found")
}

func main() {
	fmt.Println("TigerWallet - BitBox02 Integration")
	fmt.Println("==================================")
	
	manager := NewBitBox02Manager()
	ctx := context.Background()
	
	deviceID, err := manager.Connect(ctx)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}
	
	fmt.Printf("Connected to device: %s\n", deviceID)
	
	wallet, _ := manager.GetWallet(deviceID)
	
	// Get ETH address
	ethAddress, _ := wallet.GetAddress("m/44'/60'/0'/0/0")
	fmt.Printf("ETH Address: %s\n", ethAddress)
	
	// Get BTC address
	btcAddress, _ := wallet.GetAddress("m/44'/0'/0'/0/0")
	fmt.Printf("BTC Address: %s\n", btcAddress)
	
	features, _ := wallet.GetFeatures()
	fmt.Printf("Device: %s (%s)\n", features.DeviceModel, features.FirmwareVersion)
	
	manager.Disconnect(deviceID)
	fmt.Println("Disconnected")
}
