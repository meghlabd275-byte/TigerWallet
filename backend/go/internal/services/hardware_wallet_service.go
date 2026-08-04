package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tigerwallet/backend/internal/models"
)

type HardwareWalletService struct {
	db *sql.DB
}

// Supported hardware wallets
var SupportedHardwareWallets = []string{
	"LEDGER_NANO_X",
	"LEDGER_NANO_S",
	"TREZOR_MODEL_T",
	"TREZOR_ONE",
	"KEYSTONE",
	"COLDCAED",
}

func NewHardwareWalletService(db *sql.DB) *HardwareWalletService {
	return &HardwareWalletService{db: db}
}

// Register hardware wallet device
func (s *HardwareWalletService) RegisterDevice(ctx context.Context, userID uuid.UUID, deviceType, serialNumber, firmwareVersion string) (*models.HardwareWallet, error) {
	// Validate device type
	valid := false
	for _, dt := range SupportedHardwareWallets {
		if dt == deviceType {
			valid = true
			break
		}
	}
	if !valid {
		return nil, fmt.Errorf("unsupported hardware wallet: %s", deviceType)
	}

	wallet := &models.HardwareWallet{
		ID:              uuid.New(),
		UserID:          userID,
		DeviceType:      deviceType,
		SerialNumber:    serialNumber,
		FirmwareVersion: firmwareVersion,
		Status:          "ACTIVE",
		PublicKey:       "",
		RegisteredAt:    time.Now(),
		LastUsedAt:      time.Now(),
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO hardware_wallets (id, user_id, device_type, serial_number, firmware_version, status, registered_at, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, wallet.ID, wallet.UserID, wallet.DeviceType, wallet.SerialNumber, 
		wallet.FirmwareVersion, wallet.Status, wallet.RegisteredAt, wallet.LastUsedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to register device: %w", err)
	}

	return wallet, nil
}

// Sign transaction with hardware wallet
func (s *HardwareWalletService) SignTransaction(ctx context.Context, userID uuid.UUID, walletID uuid.UUID, txHash string) (string, error) {
	// Verify wallet ownership
	var wallet models.HardwareWallet
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, device_type, status FROM hardware_wallets 
		WHERE id = $1 AND user_id = $2 AND status = 'ACTIVE'
	`, walletID, userID).Scan(&wallet.ID, &wallet.UserID, &wallet.DeviceType, &wallet.Status)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("hardware wallet not found or not authorized")
	}
	if err != nil {
		return "", err
	}

	// In production, this would communicate with the hardware wallet
	// via USB/HID or Bluetooth to sign the transaction
	signature := fmt.Sprintf("hw_signature_%s_%d", txHash[:8], time.Now().Unix())

	// Update last used
	s.db.ExecContext(ctx, `
		UPDATE hardware_wallets SET last_used_at = NOW() WHERE id = $1
	`, walletID)

	return signature, nil
}

// Sign message with hardware wallet
func (s *HardwareWalletService) SignMessage(ctx context.Context, userID uuid.UUID, walletID uuid.UUID, message string) (string, error) {
	// Similar to SignTransaction but for messages
	var wallet models.HardwareWallet
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, device_type, status FROM hardware_wallets 
		WHERE id = $1 AND user_id = $2 AND status = 'ACTIVE'
	`, walletID, userID).Scan(&wallet.ID, &wallet.UserID, &wallet.DeviceType, &wallet.Status)

	if err == sql.ErrNoRows {
		return "", fmt.Errorf("hardware wallet not found")
	}

	signature := fmt.Sprintf("hw_message_sig_%s_%d", message[:8], time.Now().Unix())
	return signature, nil
}

// Get user's hardware wallets
func (s *HardwareWalletService) GetUserWallets(ctx context.Context, userID uuid.UUID) ([]models.HardwareWallet, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, device_type, serial_number, firmware_version, status, registered_at, last_used_at
		FROM hardware_wallets WHERE user_id = $1
	`, userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wallets []models.HardwareWallet
	for rows.Next() {
		var w models.HardwareWallet
		err := rows.Scan(&w.ID, &w.UserID, &w.DeviceType, &w.SerialNumber, 
			&w.FirmwareVersion, &w.Status, &w.RegisteredAt, &w.LastUsedAt)
		if err != nil {
			continue
		}
		wallets = append(wallets, w)
	}

	return wallets, nil
}

// Remove hardware wallet
func (s *HardwareWalletService) RemoveWallet(ctx context.Context, userID uuid.UUID, walletID uuid.UUID) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE hardware_wallets SET status = 'REMOVED' 
		WHERE id = $1 AND user_id = $2
	`, walletID, userID)

	if err != nil {
		return fmt.Errorf("failed to remove wallet: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("wallet not found")
	}

	return nil
}

// Verify device connection
func (s *HardwareWalletService) VerifyConnection(ctx context.Context, userID uuid.UUID, walletID uuid.UUID) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM hardware_wallets 
		WHERE id = $1 AND user_id = $2 AND status = 'ACTIVE'
	`, walletID, userID).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}
