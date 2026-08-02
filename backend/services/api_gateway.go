/**
 * TigerWallet API Gateway
 * 
 * Central API gateway that connects all backend services
 * Provides unified REST/gRPC interface for all features
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	
	"github.com/gin-gonic/gin"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	
	"tigerwallet/backend/services/privacy"
	"tigerwallet/backend/services/passkey"
	"tigerwallet/backend/services/enterprise_mpc"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Server struct {
		Port         string `json:"port"`
		ReadTimeout  int    `json:"read_timeout"`
		WriteTimeout int    `json:"write_timeout"`
		IdleTimeout  int    `json:"idle_timeout"`
	} `json:"server"`
	
	Privacy    *privacy.PrivacyConfig    `json:"privacy"`
	Passkey    *passkey.PasskeyConfig   `json:"passkey"`
	Enterprise *enterprise_mpc.MPCConfig `json:"enterprise"`
}

func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.Server.Port = "8080"
	cfg.Server.ReadTimeout = 30
	cfg.Server.WriteTimeout = 30
	cfg.Server.IdleTimeout = 120
	cfg.Privacy = privacy.DefaultPrivacyConfig()
	cfg.Passkey = passkey.DefaultPasskeyConfig()
	cfg.Enterprise = enterprise_mpc.DefaultMPCConfig()
	return cfg
}

// ============================================================================
// API Response Types
// ============================================================================

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Code    int        `json:"code"`
}

type HealthResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
	Uptime   int64           `json:"uptime"`
	Version  string           `json:"version"`
}

// ============================================================================
// API Server
// ============================================================================

type APIServer struct {
	config          *Config
	router          *gin.Engine
	privacyService  *privacy.PrivacyService
	passkeyService *passkey.PasskeyService
	enterpriseMPC   *enterprise_mpc.EnterpriseMPCService
	startTime       int64
}

// NewAPIServer creates a new API server
func NewAPIServer(config *Config) *APIServer {
	server := &APIServer{
		config:          config,
		privacyService:  privacy.NewPrivacyService(config.Privacy),
		passkeyService: passkey.NewPasskeyService(config.Passkey),
		enterpriseMPC:  enterprise_mpc.NewEnterpriseMPCService(config.Enterprise),
		startTime:      time.Now().UnixMilli(),
	}
	
	server.setupRouter()
	return server
}

func (s *APIServer) setupRouter() {
	gin.SetMode(gin.ReleaseMode)
	s.router = gin.New()
	s.router.Use(gin.Recovery())
	s.router.Use(gin.Logger())
	s.router.Use(handlers.CORS())
	
	// Health check
	s.router.GET("/health", s.HealthCheck)
	s.router.GET("/api/v1/health", s.HealthCheck)
	
	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Privacy routes
		privacy := v1.Group("/privacy")
		{
			privacy.POST("/address", s.CreateShieldedAddress)
			privacy.POST("/address/rotate", s.RotateAddress)
			privacy.POST("/transfer", s.CreateShieldedTransfer)
			privacy.POST("/transfer/verify", s.VerifyTransfer)
			privacy.GET("/address/:address", s.GetShieldedAddress)
			privacy.GET("/address/:address/balance", s.GetShieldedBalance)
			privacy.POST("/coinjoin/start", s.StartCoinJoin)
			privacy.POST("/coinjoin/join", s.JoinCoinJoin)
			privacy.POST("/coinjoin/finalize", s.FinalizeCoinJoin)
			privacy.GET("/coinjoin/:round_id", s.GetCoinJoinStatus)
		}
		
		// Passkey routes
		passkey := v1.Group("/passkey")
		{
			passkey.POST("/user", s.CreateUser)
			passkey.GET("/user/:username", s.GetUser)
			passkey.DELETE("/user/:username", s.DeleteUser)
			passkey.POST("/registration/start", s.BeginRegistration)
			passkey.POST("/registration/complete", s.CompleteRegistration)
			passkey.POST("/authentication/start", s.BeginAuthentication)
			passkey.POST("/authentication/complete", s.CompleteAuthentication)
			passkey.GET("/credentials/:username", s.GetCredentials)
			passkey.DELETE("/credentials/:username/:credential_id", s.DeleteCredential)
		}
		
		// Enterprise MPC routes
		mpc := v1.Group("/mpc")
		{
			mpc.POST("/keygen/start", s.StartKeyGeneration)
			mpc.POST("/keygen/share", s.AddKeyShare)
			mpc.GET("/keygen/:session_id", s.GetKeyGenerationSession)
			mpc.GET("/keygen/share/:share_id", s.GetKeyShare)
			mpc.POST("/signing/start", s.StartSigning)
			mpc.POST("/signing/partial", s.AddPartialSignature)
			mpc.GET("/signing/:session_id", s.GetSigningSession)
			mpc.GET("/audit/:session_id", s.GetAuditLogs)
			mpc.GET("/audit", s.GetAllAuditLogs)
			mpc.POST("/hsm/key", s.GenerateHSMKey)
			mpc.POST("/hsm/sign", s.SignWithHSM)
		}
		
		// Wallet routes
		wallet := v1.Group("/wallet")
		{
			wallet.POST("/create", s.CreateWallet)
			wallet.POST("/import", s.ImportWallet)
			wallet.GET("/:address", s.GetWalletInfo)
			wallet.POST("/:address/balance", s.GetBalance)
			wallet.POST("/send", s.SendTransaction)
			wallet.POST("/swap", s.SwapTokens)
			wallet.POST("/stake", s.StakeTokens)
			wallet.POST("/unstake", s.UnstakeTokens)
		}
		
		// Admin routes
		admin := v1.Group("/admin")
		admin.Use(s.AdminAuthMiddleware())
		{
			admin.GET("/users", s.ListUsers)
			admin.GET("/stats", s.GetStats)
			admin.POST("/config", s.UpdateConfig)
			admin.GET("/logs", s.GetLogs)
		}
		
		// Blockchain routes
		blockchain := v1.Group("/blockchain")
		{
			blockchain.GET("/chains", s.ListChains)
			blockchain.GET("/chains/:chain_id", s.GetChainInfo)
			blockchain.GET("/tx/:tx_hash", s.GetTransaction)
			blockchain.GET("/block/:block_num", s.GetBlock)
			blockchain.POST("/broadcast", s.BroadcastTransaction)
		}
	}
	
	// WebSocket
	s.router.GET("/ws", s.HandleWebSocket)
	
	// Metrics
	s.router.GET("/metrics", s.GetMetrics)
	
	// Version
	s.router.GET("/version", s.GetVersion)
}

// ============================================================================
// Health Check
// ============================================================================

func (s *APIServer) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status: "healthy",
		Services: map[string]string{
			"privacy":    "ok",
			"passkey":   "ok",
			"enterprise": "ok",
			"wallet":    "ok",
			"blockchain": "ok",
		},
		Uptime:  time.Now().UnixMilli() - s.startTime,
		Version: "1.0.0",
	})
}

// ============================================================================
// Privacy Handlers
// ============================================================================

func (s *APIServer) CreateShieldedAddress(c *gin.Context) {
	var req privacy.CreateShieldedAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error(), Code: 400})
		return
	}
	
	addr, err := s.privacyService.CreateShieldedAddress(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: addr, Code: 200})
}

func (s *APIServer) RotateAddress(c *gin.Context) {
	var req struct {
		Address  string `json:"address"`
		NewIndex uint32 `json:"new_index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error(), Code: 400})
		return
	}
	
	addr, err := s.privacyService.RotateAddress(context.Background(), req.Address, req.NewIndex)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: addr, Code: 200})
}

func (s *APIServer) CreateShieldedTransfer(c *gin.Context) {
	var req privacy.CreateShieldedTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error(), Code: 400})
		return
	}
	
	transfer, err := s.privacyService.CreateShieldedTransfer(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: transfer, Code: 200})
}

func (s *APIServer) VerifyTransfer(c *gin.Context) {
	var req struct {
		TransferID string `json:"transfer_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error(), Code: 400})
		return
	}
	
	valid, err := s.privacyService.VerifyTransfer(context.Background(), req.TransferID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]bool{"valid": valid}, Code: 200})
}

func (s *APIServer) GetShieldedAddress(c *gin.Context) {
	address := c.Param("address")
	
	addr, err := s.privacyService.GetShieldedAddress(context.Background(), address)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: err.Error(), Code: 404})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: addr, Code: 200})
}

func (s *APIServer) GetShieldedBalance(c *gin.Context) {
	address := c.Param("address")
	tokenID := c.Query("token_id")
	
	balance, err := s.privacyService.GetShieldedBalance(context.Background(), address, tokenID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]uint64{"balance": balance}, Code: 200})
}

func (s *APIServer) StartCoinJoin(c *gin.Context) {
	var req privacy.CoinJoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error(), Code: 400})
		return
	}
	
	round, err := s.privacyService.StartCoinJoin(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: round, Code: 200})
}

func (s *APIServer) JoinCoinJoin(c *gin.Context) {
	var req struct {
		RoundID     string                `json:"round_id"`
		Participant privacy.Participant `json:"participant"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error(), Code: 400})
		return
	}
	
	err := s.privacyService.JoinCoinJoin(context.Background(), req.RoundID, req.Participant)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Code: 200})
}

func (s *APIServer) FinalizeCoinJoin(c *gin.Context) {
	var req struct {
		RoundID string `json:"round_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error(), Code: 400})
		return
	}
	
	transfers, err := s.privacyService.FinalizeCoinJoin(context.Background(), req.RoundID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: transfers, Code: 200})
}

func (s *APIServer) GetCoinJoinStatus(c *gin.Context) {
	roundID := c.Param("round_id")
	
	round, err := s.privacyService.GetCoinJoinStatus(context.Background(), roundID)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: err.Error(), Code: 404})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: round, Code: 200})
}

// ============================================================================
// Passkey Handlers
// ============================================================================

func (s *APIServer) CreateUser(c *gin.Context) {
	var req struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error(), Code: 400})
		return
	}
	
	user, err := s.passkeyService.CreateUser(context.Background(), req.Username, req.DisplayName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: user, Code: 200})
}

func (s *APIServer) GetUser(c *gin.Context) {
	username := c.Param("username")
	
	user, err := s.passkeyService.GetUser(context.Background(), username)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: err.Error(), Code: 404})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: user, Code: 200})
}

func (s *APIServer) DeleteUser(c *gin.Context) {
	username := c.Param("username")
	
	err := s.passkeyService.DeleteUser(context.Background(), username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Code: 200})
}

func (s *APIServer) BeginRegistration(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error(), Code: 400})
		return
	}
	
	session, request, err := s.passkeyService.BeginRegistration(context.Background(), req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"session": session,
		"request": request,
	}, Code: 200})
}

func (s *APIServer) CompleteRegistration(c *gin.Context) {
	var req struct {
		SessionID string                  `json:"session_id"`
		Response  passkey.WebAuthnResponse `json:"response"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error(), Code: 400})
		return
	}
	
	credential, err := s.passkeyService.CompleteRegistration(context.Background(), req.SessionID, &req.Response)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: credential, Code: 200})
}

func (s *APIServer) BeginAuthentication(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error(), Code: 400})
		return
	}
	
	session, request, err := s.passkeyService.BeginAuthentication(context.Background(), req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{
		"session": session,
		"request": request,
	}, Code: 200})
}

func (s *APIServer) CompleteAuthentication(c *gin.Context) {
	var req struct {
		SessionID string                  `json:"session_id"`
		Response  passkey.WebAuthnResponse `json:"response"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error(), Code: 400})
		return
	}
	
	user, err := s.passkeyService.CompleteAuthentication(context.Background(), req.SessionID, &req.Response)
	if err != nil {
		c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: err.Error(), Code: 401})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: user, Code: 200})
}

func (s *APIServer) GetCredentials(c *gin.Context) {
	username := c.Param("username")
	
	credentials, err := s.passkeyService.GetCredentials(context.Background(), username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: credentials, Code: 200})
}

func (s *APIServer) DeleteCredential(c *gin.Context) {
	vars := mux.Vars(c.Request)
	username := vars["username"]
	credentialID := vars["credential_id"]
	
	err := s.passkeyService.DeleteCredential(context.Background(), username, credentialID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Code: 200})
}

// ============================================================================
// Enterprise MPC Handlers
// ============================================================================

func (s *APIServer) StartKeyGeneration(c *gin.Context) {
	var req enterprise_mpc.KeyGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error(), Code: 400})
		return
	}
	
	session, err := s.enterpriseMPC.CreateKeyGenerationSession(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: session, Code: 200})
}

func (s *APIServer) AddKeyShare(c *gin.Context) {
	var req struct {
		SessionID   string `json:"session_id"`
		SignerID    string `json:"signer_id"`
		ShareData   string `json:"share_data"`
		VerifierData string `json:"verifier_data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error(), Code: 400})
		return
	}
	
	share, err := s.enterpriseMPC.AddKeyShare(context.Background(), req.SessionID, req.SignerID, req.ShareData, req.VerifierData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: share, Code: 200})
}

func (s *APIServer) GetKeyGenerationSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	
	session, err := s.enterpriseMPC.GetKeyGenerationSession(context.Background(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: err.Error(), Code: 404})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: session, Code: 200})
}

func (s *APIServer) GetKeyShare(c *gin.Context) {
	shareID := c.Param("share_id")
	
	share, err := s.enterpriseMPC.GetKeyShare(context.Background(), shareID)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: err.Error(), Code: 404})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: share, Code: 200})
}

func (s *APIServer) StartSigning(c *gin.Context) {
	var req enterprise_mpc.SigningRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error(), Code: 400})
		return
	}
	
	session, err := s.enterpriseMPC.CreateSigningSession(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: session, Code: 200})
}

func (s *APIServer) AddPartialSignature(c *gin.Context) {
	var req struct {
		SessionID   string `json:"session_id"`
		SignerID    string `json:"signer_id"`
		PartialSig  string `json:"partial_sig"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error(), Code: 400})
		return
	}
	
	err := s.enterpriseMPC.AddPartialSignature(context.Background(), req.SessionID, req.SignerID, req.PartialSig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Code: 200})
}

func (s *APIServer) GetSigningSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	
	session, err := s.enterpriseMPC.GetSigningSession(context.Background(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: err.Error(), Code: 404})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: session, Code: 200})
}

func (s *APIServer) GetAuditLogs(c *gin.Context) {
	sessionID := c.Param("session_id")
	
	logs, err := s.enterpriseMPC.GetAuditLogs(context.Background(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: logs, Code: 200})
}

func (s *APIServer) GetAllAuditLogs(c *gin.Context) {
	logs, err := s.enterpriseMPC.GetAllAuditLogs(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: logs, Code: 200})
}

func (s *APIServer) GenerateHSMKey(c *gin.Context) {
	var req struct {
		KeyType string `json:"key_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error(), Code: 400})
		return
	}
	
	key, err := s.enterpriseMPC.GenerateHSMKey(context.Background(), req.KeyType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: key, Code: 200})
}

func (s *APIServer) SignWithHSM(c *gin.Context) {
	var req struct {
		KeyID string `json:"key_id"`
		Data  string `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: err.Error(), Code: 400})
		return
	}
	
	sig, err := s.enterpriseMPC.SignWithHSM(context.Background(), req.KeyID, req.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error(), Code: 500})
		return
	}
	
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]string{"signature": sig}, Code: 200})
}

// ============================================================================
// Placeholder Handlers (Wallet, Admin, Blockchain)
// ============================================================================

func (s *APIServer) CreateWallet(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]string{"address": "0x...", "message": "Wallet created"}, Code: 200})
}

func (s *APIServer) ImportWallet(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]string{"address": "0x...", "message": "Wallet imported"}, Code: 200})
}

func (s *APIServer) GetWalletInfo(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{"address": c.Param("address"), "balance": 0}, Code: 200})
}

func (s *APIServer) GetBalance(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]uint64{"balance": 0}, Code: 200})
}

func (s *APIServer) SendTransaction(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]string{"tx_hash": "0x..."}, Code: 200})
}

func (s *APIServer) SwapTokens(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]string{"tx_hash": "0x..."}, Code: 200})
}

func (s *APIServer) StakeTokens(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]string{"tx_hash": "0x..."}, Code: 200})
}

func (s *APIServer) UnstakeTokens(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]string{"tx_hash": "0x..."}, Code: 200})
}

func (s *APIServer) AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Simplified - in production use proper auth
		c.Next()
	}
}

func (s *APIServer) ListUsers(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: []string{}, Code: 200})
}

func (s *APIServer) GetStats(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{}, Code: 200})
}

func (s *APIServer) UpdateConfig(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Code: 200})
}

func (s *APIServer) GetLogs(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: []string{}, Code: 200})
}

func (s *APIServer) ListChains(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: []string{"Ethereum", "Polygon", "Solana"}}, Code: 200})
}

func (s *APIServer) GetChainInfo(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{}, Code: 200})
}

func (s *APIServer) GetTransaction(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{}, Code: 200})
}

func (s *APIServer) GetBlock(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{}, Code: 200})
}

func (s *APIServer) BroadcastTransaction(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]string{"tx_hash": "0x..."}, Code: 200})
}

func (s *APIServer) HandleWebSocket(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]string{"message": "WebSocket endpoint"}, Code: 200})
}

func (s *APIServer) GetMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]interface{}{}, Code: 200})
}

func (s *APIServer) GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: map[string]string{"version": "1.0.0", "build": "2026.08.02"}, Code: 200})
}

// ============================================================================
// Server
// ============================================================================

func (s *APIServer) Start() error {
	srv := &http.Server{
		Addr:         ":" + s.config.Server.Port,
		Handler:      s.router,
		ReadTimeout:  time.Duration(s.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.config.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(s.config.Server.IdleTimeout) * time.Second,
	}
	
	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()
	
	log.Printf("TigerWallet API Gateway started on port %s", s.config.Server.Port)
	
	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("Shutting down server...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		return err
	}
	
	log.Println("Server stopped")
	return nil
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := DefaultConfig()
	
	// Override with environment variables if set
	if port := os.Getenv("PORT"); port != "" {
		config.Server.Port = port
	}
	
	server := NewAPIServer(config)
	
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
