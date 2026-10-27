package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gorilla/mux"
)

// Config holds the server configuration.
type Config struct {
	Port string
	// EnableTLS serves over TLS when set. The certificate and key must be
	// present, because serving plaintext on a port that is expected to be
	// encrypted would expose every signature that passes through it.
	EnableTLS bool
	CertFile  string
	KeyFile   string
}

// Server is the hardware wallet backend.
//
// The service verifies signatures produced by a device. It never produces one:
// a hardware wallet's key never leaves the device, so a "signature" generated
// here would be meaningless. The signing endpoints therefore take a signature
// and a payload, and check that the signature recovers to the registered device
// address over that exact payload.
type Server struct {
	config      Config
	router      *mux.Router
	httpServer  *http.Server
	deviceSvc   *DeviceService
	firmwareSvc *FirmwareService
}

// NewServer builds the server and its routes.
func NewServer(cfg Config) *Server {
	s := &Server{
		config:      cfg,
		router:      mux.NewRouter(),
		deviceSvc:   NewDeviceService(),
		firmwareSvc: NewFirmwareService(),
	}

	s.setupRoutes()

	s.httpServer = &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           s.router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	return s
}

func (s *Server) setupRoutes() {
	api := s.router.PathPrefix("/api/v1").Subrouter()

	api.HandleFunc("/devices/register", s.registerDevice).Methods(http.MethodPost)
	api.HandleFunc("/devices", s.listDevices).Methods(http.MethodGet)
	api.HandleFunc("/devices/pair", s.pairDevice).Methods(http.MethodPost)
	api.HandleFunc("/devices/{deviceID}", s.getDevice).Methods(http.MethodGet)
	api.HandleFunc("/devices/{deviceID}", s.deleteDevice).Methods(http.MethodDelete)
	api.HandleFunc("/devices/{deviceID}/verify", s.verifyDevice).Methods(http.MethodPost)

	api.HandleFunc("/firmware/{vendor}/{model}", s.getFirmware).Methods(http.MethodGet)
	api.HandleFunc("/firmware/{vendor}/{model}", s.uploadFirmware).Methods(http.MethodPost)
	api.HandleFunc("/firmware/{vendor}/{model}/verify", s.verifyFirmware).Methods(http.MethodPost)

	api.HandleFunc("/sign/transaction", s.verifyTransactionSignature).Methods(http.MethodPost)
	api.HandleFunc("/sign/message", s.verifyMessageSignature).Methods(http.MethodPost)
	api.HandleFunc("/sign/typed-data", s.verifyTypedDataSignature).Methods(http.MethodPost)
	api.HandleFunc("/sign/typed-data/digest", s.typedDataDigest).Methods(http.MethodPost)

	s.router.HandleFunc("/health", s.healthCheck).Methods(http.MethodGet)
	s.router.HandleFunc("/readyz", s.readyCheck).Methods(http.MethodGet)
}

// Start serves until the listener stops.
func (s *Server) Start() error {
	log.Printf("Starting the hardware wallet backend on port %s", s.config.Port)

	if s.config.EnableTLS {
		if s.config.CertFile == "" || s.config.KeyFile == "" {
			return errors.New("TLS is enabled but no certificate or key was configured")
		}

		return s.httpServer.ListenAndServeTLS(s.config.CertFile, s.config.KeyFile)
	}

	return s.httpServer.ListenAndServe()
}

// Stop shuts the server down.
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}

// --- handlers ---------------------------------------------------------------

func (s *Server) registerDevice(w http.ResponseWriter, r *http.Request) {
	var req RegisterDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	device, err := s.deviceSvc.Register(r.Context(), req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusCreated, device)
}

func (s *Server) getDevice(w http.ResponseWriter, r *http.Request) {
	device, err := s.deviceSvc.Get(r.Context(), mux.Vars(r)["deviceID"])
	if err != nil {
		WriteError(w, r, http.StatusNotFound, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, device)
}

func (s *Server) listDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := s.deviceSvc.List(r.Context())
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, devices)
}

func (s *Server) deleteDevice(w http.ResponseWriter, r *http.Request) {
	if err := s.deviceSvc.Delete(r.Context(), mux.Vars(r)["deviceID"]); err != nil {
		WriteError(w, r, http.StatusNotFound, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) pairDevice(w http.ResponseWriter, r *http.Request) {
	var req PairDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.deviceSvc.Pair(r.Context(), req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, result)
}

func (s *Server) verifyDevice(w http.ResponseWriter, r *http.Request) {
	var req VerifyDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.deviceSvc.Verify(r.Context(), mux.Vars(r)["deviceID"], req)
	if err != nil {
		WriteError(w, r, http.StatusUnauthorized, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, result)
}

func (s *Server) getFirmware(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	firmware, err := s.firmwareSvc.Get(r.Context(), vars["vendor"], vars["model"])
	if err != nil {
		WriteError(w, r, http.StatusNotFound, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, firmware)
}

func (s *Server) uploadFirmware(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	var req UploadFirmwareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	firmware, err := s.firmwareSvc.Upload(r.Context(), vars["vendor"], vars["model"], req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusCreated, firmware)
}

func (s *Server) verifyFirmware(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	var req VerifyFirmwareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.firmwareSvc.Verify(r.Context(), vars["vendor"], vars["model"], req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, result)
}

// verifyTransactionSignature checks a device's signature over a transaction.
//
// The transaction is decoded and hashed with the rules the chain uses, and the
// signature must recover to the registered device address. A signature that
// does not recover, or that recovers to a different address, is rejected.
func (s *Server) verifyTransactionSignature(w http.ResponseWriter, r *http.Request) {
	var req VerifySignatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	device, err := s.deviceSvc.RecordSignatureUse(req.DeviceID)
	if err != nil {
		WriteError(w, r, http.StatusUnauthorized, err.Error())
		return
	}

	recovered, err := RecoverTransactionSignature(req.Transaction, req.Signature)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if !strings.EqualFold(recovered.Hex(), device.Address) {
		WriteError(w, r, http.StatusUnauthorized, fmt.Sprintf(
			"the signature recovers to %s, not the registered device address %s",
			recovered.Hex(),
			device.Address,
		))
		return
	}

	WriteJSON(w, r, http.StatusOK, map[string]any{
		"valid":     true,
		"address":   recovered.Hex(),
		"deviceId":  device.ID,
		"vendor":    device.Vendor,
		"model":     device.Model,
		"verifiedAt": time.Now().Unix(),
	})
}

// verifyMessageSignature checks a device's EIP-191 signature over a message.
func (s *Server) verifyMessageSignature(w http.ResponseWriter, r *http.Request) {
	var req VerifyMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	device, err := s.deviceSvc.RecordSignatureUse(req.DeviceID)
	if err != nil {
		WriteError(w, r, http.StatusUnauthorized, err.Error())
		return
	}

	if strings.TrimSpace(req.Message) == "" {
		WriteError(w, r, http.StatusBadRequest, "a message is required")
		return
	}

	recovered, err := RecoverPersonalSignature(req.Message, req.Signature)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if !strings.EqualFold(recovered.Hex(), device.Address) {
		WriteError(w, r, http.StatusUnauthorized, fmt.Sprintf(
			"the signature recovers to %s, not the registered device address %s",
			recovered.Hex(),
			device.Address,
		))
		return
	}

	WriteJSON(w, r, http.StatusOK, map[string]any{
		"valid":      true,
		"address":    recovered.Hex(),
		"deviceId":   device.ID,
		"verifiedAt": time.Now().Unix(),
	})
}

// verifyTypedDataSignature checks a device's EIP-712 signature.
func (s *Server) verifyTypedDataSignature(w http.ResponseWriter, r *http.Request) {
	var req VerifyTypedDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	device, err := s.deviceSvc.RecordSignatureUse(req.DeviceID)
	if err != nil {
		WriteError(w, r, http.StatusUnauthorized, err.Error())
		return
	}

	recovered, err := RecoverTypedDataSignature(req.TypedData, req.Signature)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	if !strings.EqualFold(recovered.Hex(), device.Address) {
		WriteError(w, r, http.StatusUnauthorized, fmt.Sprintf(
			"the signature recovers to %s, not the registered device address %s",
			recovered.Hex(),
			device.Address,
		))
		return
	}

	digest, separator, err := TypedDataDigest(req.TypedData)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, map[string]any{
		"valid":           true,
		"address":         recovered.Hex(),
		"deviceId":        device.ID,
		"digest":          digest,
		"domainSeparator": separator,
		"verifiedAt":      time.Now().Unix(),
	})
}

// typedDataDigest returns the digest a device should sign for a payload, so a
// client can show the user exactly what is being signed.
func (s *Server) typedDataDigest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TypedData string `json:"typedData"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	digest, separator, err := TypedDataDigest(req.TypedData)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, map[string]string{
		"digest":          digest,
		"domainSeparator": separator,
	})
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, r, http.StatusOK, map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	})
}

// readyCheck reports readiness. The service is ready once it can serve
// requests; it holds no external dependency that can be unavailable.
func (s *Server) readyCheck(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, r, http.StatusOK, map[string]any{
		"status": "ready",
	})
}

// --- helpers ----------------------------------------------------------------

// WriteJSON writes a JSON response.
func WriteJSON(w http.ResponseWriter, r *http.Request, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("the response could not be encoded: %v", err)
	}
}

// WriteError writes an error response.
func WriteError(w http.ResponseWriter, r *http.Request, status int, message string) {
	WriteJSON(w, r, status, map[string]string{"error": message})
}

// --- request and response types ---------------------------------------------

// RegisterDeviceRequest registers a device. The public key is required so the
// address can be derived rather than asserted.
type RegisterDeviceRequest struct {
	UserID         string `json:"userId"`
	Vendor         string `json:"vendor"`
	Model          string `json:"model"`
	SerialNumber   string `json:"serialNumber"`
	FirmwareVer    string `json:"firmwareVer"`
	PublicKey      string `json:"publicKey"`
	Address        string `json:"address"`
	DerivationPath string `json:"derivationPath"`
}

// PairDeviceRequest asks for a pairing challenge.
type PairDeviceRequest struct {
	UserID   string `json:"userId"`
	DeviceID string `json:"deviceId"`
}

// VerifyDeviceRequest answers a pairing challenge.
type VerifyDeviceRequest struct {
	Challenge string `json:"challenge"`
	Signature string `json:"signature"`
}

// UploadFirmwareRequest carries a firmware image.
type UploadFirmwareRequest struct {
	Version      string `json:"version"`
	FirmwareBin  string `json:"firmwareBin"`
	ReleaseNotes string `json:"releaseNotes"`
}

// VerifyFirmwareRequest carries a firmware image to check against the stored
// digest.
type VerifyFirmwareRequest struct {
	FirmwareID  string `json:"firmwareId"`
	FirmwareBin string `json:"firmwareBin"`
}

// VerifySignatureRequest carries a signed transaction.
type VerifySignatureRequest struct {
	UserID      string `json:"userId"`
	DeviceID    string `json:"deviceId"`
	Transaction string `json:"transaction"`
	Signature   string `json:"signature"`
	Path        string `json:"path"`
}

// VerifyMessageRequest carries a signed message.
type VerifyMessageRequest struct {
	UserID    string `json:"userId"`
	DeviceID  string `json:"deviceId"`
	Message   string `json:"message"`
	Signature string `json:"signature"`
	Path      string `json:"path"`
}

// VerifyTypedDataRequest carries signed EIP-712 data.
type VerifyTypedDataRequest struct {
	UserID    string `json:"userId"`
	DeviceID  string `json:"deviceId"`
	TypedData string `json:"typedData"`
	Signature string `json:"signature"`
	Path      string `json:"path"`
}

// --- entry point ------------------------------------------------------------

func main() {
	cfg := Config{
		Port:      getEnv("PORT", "8090"),
		EnableTLS: getEnv("ENABLE_TLS", "") == "true",
		CertFile:  getEnv("TLS_CERT_FILE", ""),
		KeyFile:   getEnv("TLS_KEY_FILE", ""),
	}

	server := NewServer(cfg)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down the hardware wallet backend...")

	if err := server.Stop(); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}

// decodeHexBytes decodes a hex string, with or without the `0x` prefix.
func decodeHexBytes(value string) ([]byte, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, errors.New("the value is empty")
	}

	if !strings.HasPrefix(trimmed, "0x") && !strings.HasPrefix(trimmed, "0X") {
		trimmed = "0x" + trimmed
	}

	return hexutil.Decode(trimmed)
}

// privateKeyFromEnv loads a signing key from the environment, if one is set.
//
// This is used by tests and by operators who need the service to hold a key for
// its own accounts. A device key is never configured here.
func privateKeyFromEnv() (*ecdsa.PrivateKey, error) {
	value := os.Getenv("HARDWARE_WALLET_SIGNING_KEY")
	if value == "" {
		return nil, errors.New("no signing key is configured")
	}

	key, err := crypto.HexToECDSA(strings.TrimPrefix(value, "0x"))
	if err != nil {
		return nil, fmt.Errorf("the configured signing key is not valid: %w", err)
	}

	return key, nil
}
