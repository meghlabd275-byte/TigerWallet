package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// Config holds server configuration
type Config struct {
	Port         string
	DatabaseURL string
	RedisURL    string
}

// Server is the main hardware wallet backend server
type Server struct {
	config        Config
	router       *mux.Router
	httpServer   *http.Server
	deviceSvc    *DeviceService
	firmwareSvc  *FirmwareService
	deviceMgmt  *DeviceManagement
}

// NewServer creates a new hardware wallet backend server
func NewServer(cfg Config) *Server {
	s := &Server{
		config:      cfg,
		router:      mux.NewRouter(),
		deviceSvc:   NewDeviceService(),
		firmwareSvc: NewFirmwareService(),
		deviceMgmt: NewDeviceManagement(),
	}

	s.setupRoutes()
	s.setupMiddlewares()

	s.httpServer = &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return s
}

func (s *Server) setupRoutes() {
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Device registration
	api.HandleFunc("/devices/register", s.registerDevice).Methods("POST")
	api.HandleFunc("/devices/{deviceID}", s.getDevice).Methods("GET")
	api.HandleFunc("/devices", s.listDevices).Methods("GET")
	api.HandleFunc("/devices/{deviceID}", s.deleteDevice).Methods("DELETE")

	// Device pairing
	api.HandleFunc("/devices/pair", s.pairDevice).Methods("POST")
	api.HandleFunc("/devices/{deviceID}/verify", s.verifyDevice).Methods("POST")

	// Firmware
	api.HandleFunc("/firmware/{vendor}/{model}", s.getFirmware).Methods("GET")
	api.HandleFunc("/firmware/{vendor}/{model}", s.uploadFirmware).Methods("POST")
	api.HandleFunc("/firmware/{vendor}/{model}/verify", s.verifyFirmware).Methods("POST")

	// Signing
	api.HandleFunc("/sign/signTransaction", s.signTransaction).Methods("POST")
	api.HandleFunc("/sign/signMessage", s.signMessage).Methods("POST")
	api.HandleFunc("/sign/signTypedData", s.signTypedData).Methods("POST")

	// Health check
	s.router.HandleFunc("/health", s.healthCheck).Methods("GET")
}

func (s *Server) setupMiddlewares() {
	s.router.Use(handlers.RecoveryHandler())
	s.router.Use(handlers.Logger())
}

func (s *Server) Start() error {
	log.Printf("Starting TigerWallet Hardware Backend on port %s", s.config.Port)

	if s.config.EnableTLS {
		return s.httpServer.ListenAndServeTLS(s.config.CertFile, s.config.KeyFile)
	}

	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}

// HTTP Handlers

func (s *Server) registerDevice(w http.ResponseWriter, r *http.Request) {
	var req RegisterDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	device, err := s.deviceSvc.Register(r.Context(), req)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusCreated, device)
}

func (s *Server) getDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars["deviceID"]

	device, err := s.deviceSvc.Get(r.Context(), deviceID)
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
	vars := mux.Vars(r)
	deviceID := vars["deviceID"]

	if err := s.deviceSvc.Delete(r.Context(), deviceID); err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
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

	result, err := s.deviceMgmt.Pair(r.Context(), req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, result)
}

func (s *Server) verifyDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars["deviceID"]

	var req VerifyDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.deviceMgmt.Verify(r.Context(), deviceID, req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, result)
}

func (s *Server) getFirmware(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vendor := vars["vendor"]
	model := vars["model"]

	firmware, err := s.firmwareSvc.Get(r.Context(), vendor, model)
	if err != nil {
		WriteError(w, r, http.StatusNotFound, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, firmware)
}

func (s *Server) uploadFirmware(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vendor := vars["vendor"]
	model := vars["model"]

	var req UploadFirmwareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	firmware, err := s.firmwareSvc.Upload(r.Context(), vendor, model, req)
	if err != nil {
		WriteError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusCreated, firmware)
}

func (s *Server) verifyFirmware(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vendor := vars["vendor"]
	model := vars["model"]

	var req VerifyFirmwareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.firmwareSvc.Verify(r.Context(), vendor, model, req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, result)
}

func (s *Server) signTransaction(w http.ResponseWriter, r *http.Request) {
	var req SignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	signature, err := s.deviceMgmt.SignTransaction(r.Context(), req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, signature)
}

func (s *Server) signMessage(w http.ResponseWriter, r *http.Request) {
	var req SignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	signature, err := s.deviceMgmt.SignMessage(r.Context(), req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, signature)
}

func (s *Server) signTypedData(w http.ResponseWriter, r *http.Request) {
	var req SignTypedDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	signature, err := s.deviceMgmt.SignTypedData(r.Context(), req)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, r, http.StatusOK, signature)
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, r, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	})
}

// Helper functions

func WriteJSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func WriteError(w http.ResponseWriter, r *http.Request, status int, message string) {
	WriteJSON(w, r, status, map[string]string{
		"error": message,
	})
}

// Request/Response types

type RegisterDeviceRequest struct {
	UserID       string `json:"userId"`
	Vendor       string `json:"vendor"`
	Model        string `json:"model"`
	SerialNumber string `json:"serialNumber"`
	FirmwareVer  string `json:"firmwareVer"`
}

type PairDeviceRequest struct {
	UserID       string `json:"userId"`
	DeviceID    string `json:"deviceId"`
	PairingCode string `json:"pairingCode"`
}

type VerifyDeviceRequest struct {
	Challenge string `json:"challenge"`
	Signature string `json:"signature"`
}

type UploadFirmwareRequest struct {
	Version     string `json:"version"`
	Checksum   string `json:"checksum"`
	FirmwareBin string `json:"firmwareBin"`
	ReleaseNotes string `json:"releaseNotes"`
}

type VerifyFirmwareRequest struct {
	FirmwareID string `json:"firmwareId"`
	Signature  string `json:"signature"`
}

type SignRequest struct {
	UserID    string `json:"userId"`
	DeviceID string `json:"deviceId"`
	TxData    string `json:"txData"`
	Path      string `json:"path"`
}

type SignTypedDataRequest struct {
	UserID    string `json:"userId"`
	DeviceID  string `json:"deviceId"`
	TypedData string `json:"typedData"`
	Path      string `json:"path"`
}

func main() {
	cfg := Config{
		Port:         getEnv("PORT", "8082"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://localhost:5432/tigerwallet"),
		RedisURL:     getEnv("REDIS_URL", "localhost:6379"),
	}

	server := NewServer(cfg)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		if err := server.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")

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