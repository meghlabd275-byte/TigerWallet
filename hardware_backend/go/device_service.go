package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Device struct {
	ID            string    `json:"id"`
	UserID        string    `json:"userId"`
	Vendor       string    `json:"vendor"`
	Model        string    `json:"model"`
	SerialNumber string    `json:"serialNumber"`
	FirmwareVer  string    `json:"firmwareVer"`
	Paired      bool      `json:"paired"`
	Verified    bool      `json:"verified"`
	CreatedAt   int64     `json:"createdAt"`
	UpdatedAt   int64     `json:"updatedAt"`
	LastUsed    int64     `json:"lastUsed"`
}

type DeviceService struct {
	mu      sync.RWMutex
	devices map[string]*Device
}

func NewDeviceService() *DeviceService {
	return &DeviceService{
		devices: make(map[string]*Device),
	}
}

func (s *DeviceService) Register(ctx context.Context, req RegisterDeviceRequest) (*Device, error) {
	device := &Device{
		ID:            generateID(),
		UserID:        req.UserID,
		Vendor:        req.Vendor,
		Model:        req.Model,
		SerialNumber: req.SerialNumber,
		FirmwareVer:  req.FirmwareVer,
		Paired:      false,
		Verified:   false,
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:  time.Now().Unix(),
	}

	s.mu.Lock()
	s.devices[device.ID] = device
	s.mu.Unlock()

	return device, nil
}

func (s *DeviceService) Get(ctx context.Context, deviceID string) (*Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	device, ok := s.devices[deviceID]
	if !ok {
		return nil, fmt.Errorf("device not found")
	}

	return device, nil
}

func (s *DeviceService) List(ctx context.Context) ([]*Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var devices []*Device
	for _, device := range s.devices {
		devices = append(devices, device)
	}

	return devices, nil
}

func (s *DeviceService) Delete(ctx context.Context, deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.devices[deviceID]; !ok {
		return fmt.Errorf("device not found")
	}

	delete(s.devices, deviceID)
	return nil
}

type Firmware struct {
	ID          string `json:"id"`
	Vendor     string `json:"vendor"`
	Model      string `json:"model"`
	Version    string `json:"version"`
	Checksum   string `json:"checksum"`
	UploadedAt int64  `json:"uploadedAt"`
}

type FirmwareService struct {
	mu        sync.RWMutex
	firmwares map[string]*Firmware
}

func NewFirmwareService() *FirmwareService {
	return &FirmwareService{
		firmwares: make(map[string]*Firmware),
	}
}

func (s *FirmwareService) Get(ctx context.Context, vendor, model string) (*Firmware, error) {
	key := fmt.Sprintf("%s:%s", vendor, model)
	
	s.mu.RLock()
	defer s.mu.RUnlock()

	firmware, ok := s.firmwares[key]
	if !ok {
		return nil, fmt.Errorf("firmware not found")
	}

	return firmware, nil
}

func (s *FirmwareService) Upload(ctx context.Context, vendor, model string, req UploadFirmwareRequest) (*Firmware, error) {
	firmware := &Firmware{
		ID:        generateID(),
		Vendor:   vendor,
		Model:    model,
		Version:  req.Version,
		Checksum: req.Checksum,
	}

	key := fmt.Sprintf("%s:%s", vendor, model)
	s.mu.Lock()
	s.firmwares[key] = firmware
	s.mu.Unlock()

	return firmware, nil
}

func (s *FirmwareService) Verify(ctx context.Context, vendor, model string, req VerifyFirmwareRequest) (map[string]bool, error) {
	firmware, err := s.Get(ctx, vendor, model)
	if err != nil {
		return nil, err
	}

	result := map[string]bool{
		"valid": firmware.Checksum == req.FirmwareID,
	}

	return result, nil
}

type DeviceManagement struct {
	mu           sync.RWMutex
	pairingCodes map[string]string
	signatures map[string]string
}

func NewDeviceManagement() *DeviceManagement {
	return &DeviceManagement{
		pairingCodes: make(map[string]string),
		signatures: make(map[string]string),
	}
}

func (s *DeviceManagement) Pair(ctx context.Context, req PairDeviceRequest) (map[string]string, error) {
	code := generateCode(6)
	s.mu.Lock()
	s.pairingCodes[req.DeviceID] = code
	s.mu.Unlock()

	return map[string]string{
		"pairingCode": code,
	}, nil
}

func (s *DeviceManagement) Verify(ctx context.Context, deviceID string, req VerifyDeviceRequest) (map[string]bool, error) {
	s.mu.RLock()
	code := s.pairingCodes[deviceID]
	s.mu.RUnlock()

	result := map[string]bool{
		"verified": code == req.Challenge,
	}

	return result, nil
}

func (s *DeviceManagement) SignTransaction(ctx context.Context, req SignRequest) (map[string]string, error) {
	return map[string]string{
		"signature": "0x" + generateHex(65),
	}, nil
}

func (s *DeviceManagement) SignMessage(ctx context.Context, req SignRequest) (map[string]string, error) {
	return map[string]string{
		"signature": "0x" + generateHex(65),
	}, nil
}

func (s *DeviceManagement) SignTypedData(ctx context.Context, req SignTypedDataRequest) (map[string]string, error) {
	return map[string]string{
		"signature": "0x" + generateHex(65),
	}, nil
}

func generateID() string {
	return fmt.Sprintf("dev_%d", time.Now().UnixNano())
}

func generateCode(length int) string {
	const chars = "0123456789"
	code := make([]byte, length)
	for i := range code {
		code[i] = chars[i%len(chars)]
	}
	return string(code)
}

func generateHex(length int) string {
	const hexChars = "0123456789abcdef"
	result := make([]byte, length)
	for i := range result {
		result[i] = hexChars[i%len(hexChars)]
	}
	return string(result)
}