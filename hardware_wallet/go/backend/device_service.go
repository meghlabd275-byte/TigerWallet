package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

// Device is a hardware wallet registered by a user.
//
// The address is derived from the public key the device reports, so a device
// cannot claim an address it does not control.
type Device struct {
	ID           string `json:"id"`
	UserID       string `json:"userId"`
	Vendor       string `json:"vendor"`
	Model        string `json:"model"`
	SerialNumber string `json:"serialNumber"`
	FirmwareVer  string `json:"firmwareVer"`
	PublicKey    string `json:"publicKey"`
	Address      string `json:"address"`
	// DerivationPath is the BIP-32 path the public key was read from.
	DerivationPath string `json:"derivationPath"`
	Paired         bool   `json:"paired"`
	Verified       bool   `json:"verified"`
	CreatedAt      int64  `json:"createdAt"`
	UpdatedAt      int64  `json:"updatedAt"`
	LastUsed       int64  `json:"lastUsed"`
}

// pairing is an outstanding challenge for a device.
type pairing struct {
	challenge   string
	address     string
	issuedAt    time.Time
	expiresAt   time.Time
	attempts    int
	maxAttempts int
}

// pairingTTL bounds how long a challenge stays valid. A challenge that never
// expires is a replay opportunity.
const pairingTTL = 5 * time.Minute

// maxPairingAttempts bounds how many signatures may be tried against one
// challenge, so the endpoint cannot be used as a signature oracle.
const maxPairingAttempts = 5

// DeviceService stores devices and their outstanding pairing challenges.
type DeviceService struct {
	mu       sync.RWMutex
	devices  map[string]*Device
	pairings map[string]*pairing
}

// NewDeviceService creates an empty device service.
func NewDeviceService() *DeviceService {
	return &DeviceService{
		devices:  make(map[string]*Device),
		pairings: make(map[string]*pairing),
	}
}

// Register stores a device, deriving its address from the public key it
// presents.
//
// The public key is required: without it there is nothing to verify a signature
// against, and the device could not be trusted to sign anything.
func (s *DeviceService) Register(ctx context.Context, req RegisterDeviceRequest) (*Device, error) {
	if strings.TrimSpace(req.UserID) == "" {
		return nil, errors.New("a user id is required")
	}
	if strings.TrimSpace(req.Vendor) == "" {
		return nil, errors.New("a vendor is required")
	}
	if strings.TrimSpace(req.PublicKey) == "" {
		return nil, errors.New("a public key is required")
	}

	address, err := DeriveAddressFromPublicKey(req.PublicKey)
	if err != nil {
		return nil, err
	}

	// A device that claims an address must present the key for that address.
	if req.Address != "" {
		claimed, err := checksumAddress(req.Address)
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(claimed, address) {
			return nil, fmt.Errorf(
				"the public key derives the address %s, but %s was claimed",
				address,
				claimed,
			)
		}
	}

	id, err := newDeviceID()
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	device := &Device{
		ID:             id,
		UserID:         req.UserID,
		Vendor:         req.Vendor,
		Model:          req.Model,
		SerialNumber:   req.SerialNumber,
		FirmwareVer:    req.FirmwareVer,
		PublicKey:      req.PublicKey,
		Address:        address,
		DerivationPath: req.DerivationPath,
		Paired:         false,
		Verified:       false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	s.mu.Lock()
	s.devices[device.ID] = device
	s.mu.Unlock()

	return device, nil
}

// Get returns a device by id.
func (s *DeviceService) Get(ctx context.Context, deviceID string) (*Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	device, ok := s.devices[deviceID]
	if !ok {
		return nil, errors.New("the device was not found")
	}

	return device, nil
}

// List returns every registered device.
func (s *DeviceService) List(ctx context.Context) ([]*Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	devices := make([]*Device, 0, len(s.devices))
	for _, device := range s.devices {
		devices = append(devices, device)
	}

	return devices, nil
}

// Delete removes a device and any outstanding pairing.
func (s *DeviceService) Delete(ctx context.Context, deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.devices[deviceID]; !ok {
		return errors.New("the device was not found")
	}

	delete(s.devices, deviceID)
	delete(s.pairings, deviceID)

	return nil
}

// Pair issues a challenge for a device.
//
// The device must sign the challenge to prove it holds the key. A pairing code
// alone proves nothing, because anyone who learned the code could present it.
func (s *DeviceService) Pair(ctx context.Context, req PairDeviceRequest) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, ok := s.devices[req.DeviceID]
	if !ok {
		return nil, errors.New("the device was not found")
	}

	if req.UserID != "" && req.UserID != device.UserID {
		return nil, errors.New("the device belongs to a different user")
	}

	challenge, err := newChallenge()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	s.pairings[device.ID] = &pairing{
		challenge:   challenge,
		address:     device.Address,
		issuedAt:    now,
		expiresAt:   now.Add(pairingTTL),
		maxAttempts: maxPairingAttempts,
	}

	return map[string]string{
		"deviceId":  device.ID,
		"challenge": challenge,
		"address":   device.Address,
		"expiresAt": fmt.Sprintf("%d", now.Add(pairingTTL).Unix()),
	}, nil
}

// Verify checks a signature over the device's outstanding challenge.
//
// Only a signature that recovers to the device's own address verifies, and each
// challenge may be tried a bounded number of times before it is discarded.
func (s *DeviceService) Verify(ctx context.Context, deviceID string, req VerifyDeviceRequest) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, ok := s.devices[deviceID]
	if !ok {
		return nil, errors.New("the device was not found")
	}

	pending, ok := s.pairings[deviceID]
	if !ok {
		return nil, errors.New("no pairing challenge is outstanding for this device")
	}

	if time.Now().After(pending.expiresAt) {
		delete(s.pairings, deviceID)
		return nil, errors.New("the pairing challenge has expired")
	}

	// A caller may echo the challenge; if it does, it must match.
	if req.Challenge != "" && req.Challenge != pending.challenge {
		return nil, errors.New("the challenge does not match the one issued")
	}

	pending.attempts++
	if pending.attempts > pending.maxAttempts {
		delete(s.pairings, deviceID)
		return nil, errors.New("too many pairing attempts; request a new challenge")
	}

	recovered, err := verifyChallenge(pending.challenge, req.Signature, device.Address)
	if err != nil {
		return nil, err
	}

	device.Paired = true
	device.Verified = true
	device.UpdatedAt = time.Now().Unix()
	device.LastUsed = device.UpdatedAt

	// A challenge is single use.
	delete(s.pairings, deviceID)

	return map[string]any{
		"verified": true,
		"address":  recovered.Hex(),
		"deviceId": device.ID,
	}, nil
}

// RecordSignatureUse marks a device as used and returns it.
//
// A signature is only accepted from a device that has completed pairing, so an
// unpaired device cannot sign.
func (s *DeviceService) RecordSignatureUse(deviceID string) (*Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, ok := s.devices[deviceID]
	if !ok {
		return nil, errors.New("the device was not found")
	}

	if !device.Verified {
		return nil, errors.New("the device has not completed pairing; verify it before signing")
	}

	device.LastUsed = time.Now().Unix()
	device.UpdatedAt = device.LastUsed

	return device, nil
}

// Firmware is an uploaded firmware image record.
type Firmware struct {
	ID           string `json:"id"`
	Vendor       string `json:"vendor"`
	Model        string `json:"model"`
	Version      string `json:"version"`
	SHA256       string `json:"sha256"`
	Size         int    `json:"size"`
	ReleaseNotes string `json:"releaseNotes"`
	UploadedAt   int64  `json:"uploadedAt"`
}

// FirmwareService stores firmware images and their digests.
type FirmwareService struct {
	mu        sync.RWMutex
	firmwares map[string]*Firmware
}

// NewFirmwareService creates an empty firmware service.
func NewFirmwareService() *FirmwareService {
	return &FirmwareService{firmwares: make(map[string]*Firmware)}
}

// key identifies a firmware line by vendor and model.
func firmwareKey(vendor, model string) string {
	return strings.ToLower(vendor) + ":" + strings.ToLower(model)
}

// Get returns the firmware for a vendor and model.
func (s *FirmwareService) Get(ctx context.Context, vendor, model string) (*Firmware, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	firmware, ok := s.firmwares[firmwareKey(vendor, model)]
	if !ok {
		return nil, errors.New("no firmware is registered for this vendor and model")
	}

	return firmware, nil
}

// Upload records a firmware image.
//
// The digest is computed from the image itself. A caller-supplied checksum is
// not trusted, because the point of the digest is to detect a tampered image.
func (s *FirmwareService) Upload(ctx context.Context, vendor, model string, req UploadFirmwareRequest) (*Firmware, error) {
	if strings.TrimSpace(req.Version) == "" {
		return nil, errors.New("a firmware version is required")
	}
	if strings.TrimSpace(req.FirmwareBin) == "" {
		return nil, errors.New("the firmware image is required")
	}

	image, err := decodeHexBytes(req.FirmwareBin)
	if err != nil {
		return nil, fmt.Errorf("the firmware image is not valid hex: %w", err)
	}

	id, err := newDeviceID()
	if err != nil {
		return nil, err
	}

	firmware := &Firmware{
		ID:           strings.Replace(id, "dev_", "fw_", 1),
		Vendor:       vendor,
		Model:        model,
		Version:      req.Version,
		SHA256:       fmt.Sprintf("%x", crypto.Keccak256(image)),
		Size:         len(image),
		ReleaseNotes: req.ReleaseNotes,
		UploadedAt:   time.Now().Unix(),
	}

	s.mu.Lock()
	s.firmwares[firmwareKey(vendor, model)] = firmware
	s.mu.Unlock()

	return firmware, nil
}

// Verify reports whether an image matches the stored firmware.
//
// The comparison is over the digest of the supplied image, so a caller cannot
// pass by echoing back a value it was given.
func (s *FirmwareService) Verify(ctx context.Context, vendor, model string, req VerifyFirmwareRequest) (map[string]any, error) {
	firmware, err := s.Get(ctx, vendor, model)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(req.FirmwareBin) == "" {
		return nil, errors.New("the firmware image is required")
	}

	image, err := decodeHexBytes(req.FirmwareBin)
	if err != nil {
		return nil, fmt.Errorf("the firmware image is not valid hex: %w", err)
	}

	digest := fmt.Sprintf("%x", crypto.Keccak256(image))

	return map[string]any{
		"valid":           digest == firmware.SHA256,
		"version":         firmware.Version,
		"expectedSha256":  firmware.SHA256,
		"presentedSha256": digest,
	}, nil
}
