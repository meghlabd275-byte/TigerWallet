package main

import (
"context"
"crypto/sha256"
"encoding/hex"
"encoding/json"
"fmt"
"log"
"math/big"
"net/http"
"os"
"os/signal"
"strings"
"syscall"
"time"

"github.com/gin-gonic/gin"
"github.com/google/uuid"
)

// Configuration
type Config struct {
Port              string
DatabaseURL       string
RecoveryTimeout   int // hours
MinGuardians     int
MaxGuardians     int
RecoveryThreshold int
}

// Guardian
type Guardian struct {
ID          string    `json:"id"`
WalletID    string    `json:"wallet_id"`
Address     string    `json:"address"`
Name        string    `json:"name"`
PublicKey   string    `json:"public_key"`
AddedAt     time.Time `json:"added_at"`
IsConfirmed bool      `json:"is_confirmed"`
}

// Recovery Request
type RecoveryRequest struct {
ID              string     `json:"id"`
WalletID        string     `json:"wallet_id"`
NewOwnerAddress string     `json:"new_owner_address"`
Status          string     `json:"status"` // pending, confirmed, executed, cancelled
Threshold       int        `json:"threshold"`
ConfirmedBy     []string   `json:"confirmed_by"`
ExecutedAt      *time.Time `json:"executed_at,omitempty"`
CreatedAt       time.Time  `json:"created_at"`
ExpiresAt       time.Time  `json:"expires_at"`
}

// Guardian Signature
type GuardianSignature struct {
ID            string    `json:"id"`
RequestID    string    `json:"request_id"`
GuardianID   string    `json:"guardian_id"`
Signature    string    `json:"signature"`
SignedAt     time.Time `json:"signed_at"`
}

// Recovery Service
type RecoveryService struct {
config     *Config
httpServer *http.Server
}

// New creates a new recovery service
func New(config *Config) *RecoveryService {
return &RecoveryService{config: config}
}

func (s *RecoveryService) Start() error {
gin.SetMode(gin.ReleaseMode)
router := gin.New()
router.Use(gin.Recovery())
router.Use(gin.Logger())

router.GET("/health", func(c *gin.Context) {
(http.StatusOK, gin.H{"status": "healthy", "timestamp": time.Now().Unix()})
})

api := router.Group("/api/v1")
api.POST("/guardians", s.addGuardian)
api.GET("/guardians/:wallet_id", s.listGuardians)
api.DELETE("/guardians/:id", s.removeGuardian)
api.POST("/recovery/request", s.createRecoveryRequest)
api.GET("/recovery/requests/:wallet_id", s.listRecoveryRequests)
api.POST("/recovery/sign", s.signRecoveryRequest)
api.POST("/recovery/execute", s.executeRecovery)
api.POST("/recovery/cancel", s.cancelRecovery)
api.POST("/recovery/verify", s.verifyGuardian)

s.httpServer = &http.Server{
        ":" + s.config.Port,
dler:      router,
 15 * time.Second,
15 * time.Second,
}

go func() {
tf("Starting social recovery service on port %s", s.config.Port)
AndServe()
}()

return nil
}

func (s *RecoveryService) Stop() error {
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
return s.httpServer.Shutdown(ctx)
}

// Add Guardian
func (s *RecoveryService) addGuardian(c *gin.Context) {
var req struct {
 string `json:"wallet_id" binding:"required"`
  string `json:"address" binding:"required"`
ame      string `json:"name" binding:"required"`
 string `json:"public_key"`
}

if err := c.ShouldBindJSON(&req); err != nil {
(http.StatusBadRequest, gin.H{"error": err.Error()})

}

guardian := Guardian{
         uuid.New().String(),
   req.WalletID,
    req.Address,
ame:        req.Name,
:   req.PublicKey,
    time.Now(),
firmed: false,
}

// In production, would send confirmation email/message to guardian
log.Printf("Added guardian %s for wallet %s", guardian.ID, guardian.WalletID)

c.JSON(http.StatusCreated, guardian)
}

// List Guardians
func (s *RecoveryService) listGuardians(c *gin.Context) {
walletID := c.Param("wallet_id")

// Mock data
guardians := []Guardian{
"1", WalletID: walletID, Address: "0xABC123", Name: "Alice", 
time.Now().Add(-30 * 24 * time.Hour), IsConfirmed: true,
"2", WalletID: walletID, Address: "0xDEF456", Name: "Bob", 
time.Now().Add(-20 * 24 * time.Hour), IsConfirmed: true,
"3", WalletID: walletID, Address: "0xGHI789", Name: "Charlie", 
time.Now().Add(-10 * 24 * time.Hour), IsConfirmed: true,
(http.StatusOK, gin.H{"guardians": guardians, "total": len(guardians)})
}

// Remove Guardian
func (s *RecoveryService) removeGuardian(c *gin.Context) {
id := c.Param("id")
log.Printf("Removed guardian %s", id)
c.JSON(http.StatusOK, gin.H{"message": "Guardian removed"})
}

// Create Recovery Request
func (s *RecoveryService) createRecoveryRequest(c *gin.Context) {
var req struct {
       string `json:"wallet_id" binding:"required"`
ewOwnerAddress string `json:"new_owner_address" binding:"required"`
          string `json:"reason"`
}

if err := c.ShouldBindJSON(&req); err != nil {
(http.StatusBadRequest, gin.H{"error": err.Error()})

}

now := time.Now()
expiry := now.Add(time.Duration(s.config.RecoveryTimeout) * time.Hour)

request := RecoveryRequest{
             uuid.New().String(),
       req.WalletID,
ewOwnerAddress: req.NewOwnerAddress,
         "pending",
      s.config.RecoveryThreshold,
firmedBy:     []string{},
      now,
      expiry,
}

// In production, would notify all guardians
log.Printf("Created recovery request %s for wallet %s", request.ID, request.WalletID)

c.JSON(http.StatusCreated, request)
}

// List Recovery Requests
func (s *RecoveryService) listRecoveryRequests(c *gin.Context) {
walletID := c.Param("wallet_id")

requests := []RecoveryRequest{
"1", WalletID: walletID, NewOwnerAddress: "0xNewOwner123",
"pending", Threshold: 2, ConfirmedBy: []string{"guardian1"},
time.Now().Add(-1 * time.Hour),
time.Now().Add(23 * time.Hour),
(http.StatusOK, gin.H{"requests": requests})
}

// Sign Recovery Request
func (s *RecoveryService) signRecoveryRequest(c *gin.Context) {
var req struct {
 string `json:"request_id" binding:"required"`
ID string `json:"guardian_id" binding:"required"`
ature  string `json:"signature" binding:"required"`
}

if err := c.ShouldBindJSON(&req); err != nil {
(http.StatusBadRequest, gin.H{"error": err.Error()})

}

sig := GuardianSignature{
         uuid.New().String(),
  req.RequestID,
ID:  req.GuardianID,
ature:   req.Signature,
edAt:    time.Now(),
}

log.Printf("Guardian %s signed recovery request %s", req.GuardianID, req.RequestID)

c.JSON(http.StatusCreated, gin.H{"signature": sig, "confirmations_needed": 1})
}

// Execute Recovery
func (s *RecoveryService) executeRecovery(c *gin.Context) {
var req struct {
string `json:"request_id" binding:"required"`
}

if err := c.ShouldBindJSON(&req); err != nil {
(http.StatusBadRequest, gin.H{"error": err.Error()})

}

// In production, would call smart contract to change ownership
now := time.Now()
response := map[string]interface{}{
         req.RequestID,
     "executed",
now.Unix(),
ew_owner":   "0xNewOwner123", // Would be from request
}

log.Printf("Executed recovery request %s", req.RequestID)

c.JSON(http.StatusOK, response)
}

// Cancel Recovery
func (s *RecoveryService) cancelRecovery(c *gin.Context) {
var req struct {
string `json:"request_id" binding:"required"`
 string `json:"wallet_id" binding:"required"`
}

if err := c.ShouldBindJSON(&req); err != nil {
(http.StatusBadRequest, gin.H{"error": err.Error()})

}

log.Printf("Cancelled recovery request %s", req.RequestID)

c.JSON(http.StatusOK, gin.H{"message": "Recovery cancelled", "request_id": req.RequestID})
}

// Verify Guardian
func (s *RecoveryService) verifyGuardian(c *gin.Context) {
var req struct {
ID string `json:"guardian_id" binding:"required"`
      string `json:"code" binding:"required"`
}

if err := c.ShouldBindJSON(&req); err != nil {
(http.StatusBadRequest, gin.H{"error": err.Error()})

}

// Verify the confirmation code
verified := len(req.Code) == 6 // Simple validation

if verified {
(http.StatusOK, gin.H{"verified": true, "guardian_id": req.GuardianID})
} else {
(http.StatusBadRequest, gin.H{"verified": false, "error": "Invalid verification code"})
}
}

// Helper: Hash message
func hashMessage(msg string) string {
h := sha256.Sum256([]byte(msg))
return hex.EncodeToString(h[:])
}

// Helper: Verify signature
func verifySignature(pubKey, msg, sig string) bool {
// In production, would use proper cryptographic verification
return len(sig) > 0
}

func main() {
config := &Config{
             "8082",
Timeout:   24, // 24 hours
Guardians:     3,
s:     7,
Threshold: 2,
}

if p := os.Getenv("PORT"); p != "" {
fig.Port = p
}

service := New(config)
service.Start()

quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

log.Println("Shutting down social recovery service...")
service.Stop()
}
