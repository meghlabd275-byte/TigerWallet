/**
 * TigerWallet Timelock Service
 * Delayed Transaction Execution System
 * 
 * Features:
 * - Scheduled transactions
 * - Multi-sig timelock
 * - Delayed execution
 * - Cancel pending
 * - Queue management
 * - Emergency cancel
 * - ETA-based execution
 */

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	ServerPort      string
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	RedisHost       string
	RedisPort       string
	MinDelay        int // seconds
	DefaultDelay    int // seconds
	MaxDelay        int // seconds
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:   getEnv("TIMELOCK_PORT", "9099"),
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "tigerwallet"),
		DBPassword:   getEnv("DB_PASSWORD", "password"),
		DBName:       getEnv("DB_NAME", "tigerwallet"),
		RedisHost:    getEnv("REDIS_HOST", "localhost"),
		RedisPort:    getEnv("REDIS_PORT", "6379"),
		MinDelay:     60,        // 1 minute minimum
		DefaultDelay: 86400,     // 24 hours default
		MaxDelay:     2592000,   // 30 days maximum
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Database Models
// ============================================================================

type TimelockTransaction struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	TxID              string         `gorm:"uniqueIndex;size:36" json:"tx_id"`
	Proposer          string         `gorm:"index" json:"proposer"`
	ProposerID        uint            `gorm:"index" json:"proposer_id"`
	Target            string         `json:"target"` // Contract address
	ChainID           int            `json:"chain_id"`
	Value             string         `json:"value"` // Amount in wei
	Data              string         `json:"data"` // Calldata
	Sig               string         `json:"sig"` // Function signature
	Description       string         `json:"description"`
	Delay             int            `json:"delay"` // in seconds
	ETA               time.Time      `json:"eta"` // Estimated time of execution
	ExecuteAfter      time.Time      `json:"execute_after"`
	Status            string         `json:"status"` // pending, queued, executed, cancelled, expired
	ExecutedAt        *time.Time     `json:"executed_at"`
	ExecutedBy        string         `json:"executed_by"`
	ExecutionTxHash   string         `gorm:"uniqueIndex;size:66" json:"execution_tx_hash"`
	CancelledAt       *time.Time     `json:"cancelled_at"`
	CancelledBy       string         `json:"cancelled_by"`
	CancelReason      string         `json:"cancel_reason"`
	BlockNumber       int64          `json:"block_number"`
	GasUsed           string         `json:"gas_used"`
	FailureReason     string         `json:"failure_reason"`
}

type TimelockAdmin struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	AdminAddress      string    `gorm:"uniqueIndex;size:42" json:"admin_address"`
	AdminID           uint      `gorm:"index" json:"admin_id"`
	Role              string    `json:"role"` // executor, proposer, canceller
	IsActive          bool      `json:"is_active"`
	Threshold         int       `json:"threshold"` // Required signatures
}

type TimelockQueue struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	QueueID           string    `gorm:"uniqueIndex;size:36" json:"queue_id"`
	Name              string    `json:"name"`
	Delay             int       `json:"delay"` // seconds
	GracePeriod       int       `json:"grace_period"` // seconds after ETA
	IsActive          bool      `json:"is_active"`
	MinDelay          int       `json:"min_delay"`
	MaxDelay          int       `json:"max_delay"`
}

type TimelockSignatures struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	TxID              string    `gorm:"index" json:"tx_id"`
	AdminAddress      string    `gorm:"index" json:"admin_address"`
	Signature         string    `json:"signature"`
	SignedAt          time.Time `json:"signed_at"`
}

type TimelockHistory struct {
	ID                uint      `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	TxID              string    `gorm:"index" json:"tx_id"`
	Action            string    `json:"action"` // queued, executed, cancelled
	Actor             string    `json:"actor"`
	Details           string    `json:"details"`
}

// ============================================================================
// Timelock Service
// ============================================================================

type TimelockService struct {
	config *Config
	db     *gorm.DB
	redis  *redis.Client
}

func NewTimelockService(cfg *Config) (*TimelockService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	
	db.AutoMigrate(&TimelockTransaction{}, &TimelockAdmin{}, &TimelockQueue{}, &TimelockSignatures{}, &TimelockHistory{})
	
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: "",
		DB: 0,
	})
	
	service := &TimelockService{
		config: cfg,
		db:     db,
		redis:  rdb,
	}
	
	// Initialize default queue if not exists
	service.initDefaultQueue()
	
	return service, nil
}

func (s *TimelockService) initDefaultQueue() {
	var queue TimelockQueue
	result := s.db.Where("queue_id = ?", "default").First(&queue)
	
	if result.Error == gorm.ErrRecordNotFound {
		queue = TimelockQueue{
			QueueID:     "default",
			Name:        "Default Timelock Queue",
			Delay:       s.config.DefaultDelay,
			GracePeriod: 86400, // 24 hours
			IsActive:    true,
			MinDelay:    s.config.MinDelay,
			MaxDelay:    s.config.MaxDelay,
		}
		s.db.Create(&queue)
	}
}

// ============================================================================
// Transaction Operations
// ============================================================================

func (s *TimelockService) ScheduleTransaction(
	proposerID uint,
	proposer string,
	target string,
	chainID int,
	value string,
	data string,
	sig string,
	description string,
	delay int,
) (*TimelockTransaction, error) {
	
	// Validate delay
	if delay < s.config.MinDelay {
		return nil, fmt.Errorf("delay must be at least %d seconds", s.config.MinDelay)
	}
	if delay > s.config.MaxDelay {
		return nil, fmt.Errorf("delay cannot exceed %d seconds", s.config.MaxDelay)
	}
	
	txID := uuid.New().String()
	now := time.Now()
	executeAfter := now.Add(time.Duration(delay) * time.Second)
	
	tx := TimelockTransaction{
		TxID:         txID,
		Proposer:     proposer,
		ProposerID:   proposerID,
		Target:       target,
		ChainID:      chainID,
		Value:        value,
		Data:         data,
		Sig:          sig,
		Description:  description,
		Delay:        delay,
		ETA:          executeAfter,
		ExecuteAfter: executeAfter,
		Status:       "pending",
	}
	
	s.db.Create(&tx)
	
	// Add history
	s.addHistory(txID, "created", proposer, "Transaction created")
	
	// Queue the transaction
	s.queueTransaction(&tx)
	
	return &tx, nil
}

func (s *TimelockService) queueTransaction(tx *TimelockTransaction) {
	// Store in Redis for quick access
	ctx := context.Background()
	
	txJSON, _ := json.Marshal(tx)
	s.redis.Set(ctx, fmt.Sprintf("timelock:tx:%s", tx.TxID), txJSON, time.Duration(tx.Delay+86400)*time.Second)
	
	// Add to sorted set for scheduled execution
	score := float64(tx.ExecuteAfter.Unix())
	s.redis.ZAdd(ctx, "timelock:schedule", &redis.Z{Score: score, Member: tx.TxID})
}

func (s *TimelockService) ExecuteTransaction(txID string, executor string) (*TimelockTransaction, error) {
	var tx TimelockTransaction
	if err := s.db.Where("tx_id = ?", txID).First(&tx).Error; err != nil {
		return nil, fmt.Errorf("transaction not found")
	}
	
	// Check if ready to execute
	if time.Now().Before(tx.ExecuteAfter) {
		return nil, fmt.Errorf("transaction not yet executable. ETA: %s", tx.ExecuteAfter)
	}
	
	// Check grace period (24 hours after ETA)
	gracePeriodEnd := tx.ETA.Add(24 * time.Hour)
	if time.Now().After(gracePeriodEnd) {
		tx.Status = "expired"
		s.db.Save(&tx)
		return nil, fmt.Errorf("transaction expired")
	}
	
	// Execute transaction. Real on-chain broadcast is not implemented here;
	// a transaction hash can only be obtained by broadcasting the signed
	// transaction via an RPC node. We mark the transaction as pending rather
	// than fabricating a hash.
	execTxHash := ""
	tx.Status = "pending_broadcast"
	now := time.Now()
	tx.ExecutedAt = &now
	tx.ExecutedBy = executor
	tx.ExecutionTxHash = execTxHash

	s.db.Save(&tx)

	// Remove from schedule
	s.redis.ZRem(context.Background(), "timelock:schedule", txID)
	s.redis.Del(context.Background(), fmt.Sprintf("timelock:tx:%s", txID))

	// Add history
	s.addHistory(txID, "pending_broadcast", executor, "Transaction queued for broadcast; awaiting real on-chain hash")
	
	return &tx, nil
}

func (s *TimelockService) CancelTransaction(txID string, canceller string, reason string) (*TimelockTransaction, error) {
	var tx TimelockTransaction
	if err := s.db.Where("tx_id = ?", txID).First(&tx).Error; err != nil {
		return nil, fmt.Errorf("transaction not found")
	}
	
	if tx.Status != "pending" && tx.Status != "queued" {
		return nil, fmt.Errorf("cannot cancel transaction in status: %s", tx.Status)
	}
	
	tx.Status = "cancelled"
	now := time.Now()
	tx.CancelledAt = &now
	tx.CancelledBy = canceller
	tx.CancelReason = reason
	
	s.db.Save(&tx)
	
	// Remove from schedule
	s.redis.ZRem(context.Background(), "timelock:schedule", txID)
	s.redis.Del(context.Background(), fmt.Sprintf("timelock:tx:%s", txID))
	
	// Add history
	s.addHistory(txID, "cancelled", canceller, reason)
	
	return &tx, nil
}

func (s *TimelockService) GetTransaction(txID string) (*TimelockTransaction, error) {
	var tx TimelockTransaction
	if err := s.db.Where("tx_id = ?", txID).First(&tx).Error; err != nil {
		return nil, err
	}
	return &tx, nil
}

func (s *TimelockService) GetQueuedTransactions() ([]TimelockTransaction, error) {
	var txs []TimelockTransaction
	s.db.Where("status IN ?", []string{"pending", "queued"}).Order("execute_after ASC").Find(&txs)
	return txs, nil
}

func (s *TimelockService) GetExecutableTransactions() ([]TimelockTransaction, error) {
	var txs []TimelockTransaction
	s.db.Where("status = ? AND execute_after <= ?", "queued", time.Now()).Find(&txs)
	return txs, nil
}

func (s *TimelockService) GetUserTransactions(userID uint) ([]TimelockTransaction, error) {
	var txs []TimelockTransaction
	s.db.Where("proposer_id = ?", userID).Order("created_at DESC").Find(&txs)
	return txs, nil
}

// ============================================================================
// Queue Management
// ============================================================================

func (s *TimelockService) CreateQueue(name string, delay int, gracePeriod int) (*TimelockQueue, error) {
	queueID := uuid.New().String()
	
	queue := TimelockQueue{
		QueueID:     queueID,
		Name:        name,
		Delay:       delay,
		GracePeriod: gracePeriod,
		IsActive:    true,
		MinDelay:    s.config.MinDelay,
		MaxDelay:    s.config.MaxDelay,
	}
	
	s.db.Create(&queue)
	
	return &queue, nil
}

func (s *TimelockService) UpdateQueueDelay(queueID string, newDelay int) error {
	if newDelay < s.config.MinDelay || newDelay > s.config.MaxDelay {
		return fmt.Errorf("delay must be between %d and %d seconds", s.config.MinDelay, s.config.MaxDelay)
	}
	
	result := s.db.Model(&TimelockQueue{}).Where("queue_id = ?", queueID).Update("delay", newDelay)
	return result.Error
}

// ============================================================================
// Admin Management
// ============================================================================

func (s *TimelockService) AddAdmin(address string, userID uint, role string) (*TimelockAdmin, error) {
	admin := TimelockAdmin{
		AdminAddress: address,
		AdminID:      userID,
		Role:         role,
		IsActive:     true,
		Threshold:    1,
	}
	
	s.db.Create(&admin)
	
	return &admin, nil
}

func (s *TimelockService) RemoveAdmin(address string) error {
	return s.db.Where("admin_address = ?", address).Delete(&TimelockAdmin{}).Error
}

func (s *TimelockService) GetAdmins() ([]TimelockAdmin, error) {
	var admins []TimelockAdmin
	s.db.Where("is_active = ?", true).Find(&admins)
	return admins, nil
}

// ============================================================================
// Multi-sig Support
// ============================================================================

func (s *TimelockService) SignTransaction(txID string, adminAddress string) error {
	var tx TimelockTransaction
	if err := s.db.Where("tx_id = ?", txID).First(&tx).Error; err != nil {
		return err
	}
	
	// Check if already signed
	var existing TimelockSignatures
	result := s.db.Where("tx_id = ? AND admin_address = ?", txID, adminAddress).First(&existing)
	if result.Error == nil {
		return fmt.Errorf("already signed")
	}
	
	// Add signature
	sig := TimelockSignatures{
		TxID:         txID,
		AdminAddress: adminAddress,
		Signature:   generateSignature(txID, adminAddress),
		SignedAt:    time.Now(),
	}
	s.db.Create(&sig)
	
	// Check if threshold met
	var sigCount int64
	s.db.Model(&TimelockSignatures{}).Where("tx_id = ?", txID).Count(&sigCount)
	
	// Get admin threshold
	var admin TimelockAdmin
	s.db.Where("admin_address = ?", adminAddress).First(&admin)
	
	if sigCount >= int64(admin.Threshold) && tx.Status == "pending" {
		tx.Status = "queued"
		s.db.Save(&tx)
	}
	
	return nil
}

func (s *TimelockService) GetSignatures(txID string) ([]TimelockSignatures, error) {
	var sigs []TimelockSignatures
	s.db.Where("tx_id = ?", txID).Find(&sigs)
	return sigs, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func (s *TimelockService) addHistory(txID, action, actor, details string) {
	history := TimelockHistory{
		TxID:     txID,
		Action:   action,
		Actor:    actor,
		Details:  details,
	}
	s.db.Create(&history)
}

func generateSignature(txID, admin string) string {
	data := txID + admin + time.Now().String()
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// ============================================================================
// API Handlers
// ============================================================================

func (s *TimelockService) setupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		// Transactions
		api.POST("/schedule", s.scheduleTransaction)
		api.POST("/execute", s.executeTransaction)
		api.POST("/cancel", s.cancelTransaction)
		api.GET("/tx/:tx_id", s.getTransaction)
		api.GET("/queued", s.getQueued)
		api.GET("/executable", s.getExecutable)
		api.GET("/user/:user_id/txs", s.getUserTransactions)
		
		// Queue
		api.POST("/queue", s.createQueue)
		api.PUT("/queue/:queue_id/delay", s.updateQueueDelay)
		
		// Admin
		api.POST("/admin", s.addAdmin)
		api.DELETE("/admin/:address", s.removeAdmin)
		api.GET("/admins", s.getAdmins)
		
		// Signatures
		api.POST("/sign", s.signTransaction)
		api.GET("/tx/:tx_id/signatures", s.getSignatures)
		
		// History
		api.GET("/tx/:tx_id/history", s.getTransactionHistory)
	}
	
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "timelock"})
	})
}

func (s *TimelockService) scheduleTransaction(c *gin.Context) {
	var req struct {
		ProposerID  uint   `json:"proposer_id" binding:"required"`
		Proposer    string `json:"proposer" binding:"required"`
		Target      string `json:"target" binding:"required"`
		ChainID     int    `json:"chain_id"`
		Value       string `json:"value"`
		Data        string `json:"data"`
		Sig         string `json:"sig"`
		Description string `json:"description"`
		Delay       int    `json:"delay"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if req.Delay == 0 {
		req.Delay = s.config.DefaultDelay
	}
	
	tx, err := s.ScheduleTransaction(
		req.ProposerID,
		req.Proposer,
		req.Target,
		req.ChainID,
		req.Value,
		req.Data,
		req.Sig,
		req.Description,
		req.Delay,
	)
	
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"transaction": tx})
}

func (s *TimelockService) executeTransaction(c *gin.Context) {
	var req struct {
		TxID     string `json:"tx_id" binding:"required"`
		Executor string `json:"executor" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	tx, err := s.ExecuteTransaction(req.TxID, req.Executor)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"transaction": tx})
}

func (s *TimelockService) cancelTransaction(c *gin.Context) {
	var req struct {
		TxID       string `json:"tx_id" binding:"required"`
		Canceller  string `json:"canceller" binding:"required"`
		Reason     string `json:"reason"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	tx, err := s.CancelTransaction(req.TxID, req.Canceller, req.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"transaction": tx})
}

func (s *TimelockService) getTransaction(c *gin.Context) {
	txID := c.Param("tx_id")
	
	tx, err := s.GetTransaction(txID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"transaction": tx})
}

func (s *TimelockService) getQueued(c *gin.Context) {
	txs, err := s.GetQueuedTransactions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"transactions": txs})
}

func (s *TimelockService) getExecutable(c *gin.Context) {
	txs, err := s.GetExecutableTransactions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"transactions": txs})
}

func (s *TimelockService) getUserTransactions(c *gin.Context) {
	userID, _ := strconv.ParseUint(c.Param("user_id"), 10, 32)
	
	txs, err := s.GetUserTransactions(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"transactions": txs})
}

func (s *TimelockService) createQueue(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Delay       int    `json:"delay"`
		GracePeriod int    `json:"grace_period"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if req.Delay == 0 {
		req.Delay = s.config.DefaultDelay
	}
	if req.GracePeriod == 0 {
		req.GracePeriod = 86400
	}
	
	queue, err := s.CreateQueue(req.Name, req.Delay, req.GracePeriod)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"queue": queue})
}

func (s *TimelockService) updateQueueDelay(c *gin.Context) {
	queueID := c.Param("queue_id")
	
	var req struct {
		Delay int `json:"delay" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	err := s.UpdateQueueDelay(queueID, req.Delay)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "delay updated"})
}

func (s *TimelockService) addAdmin(c *gin.Context) {
	var req struct {
		Address string `json:"address" binding:"required"`
		UserID  uint   `json:"user_id"`
		Role    string `json:"role" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	admin, err := s.AddAdmin(req.Address, req.UserID, req.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"admin": admin})
}

func (s *TimelockService) removeAdmin(c *gin.Context) {
	address := c.Param("address")
	
	err := s.RemoveAdmin(address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "admin removed"})
}

func (s *TimelockService) getAdmins(c *gin.Context) {
	admins, err := s.GetAdmins()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"admins": admins})
}

func (s *TimelockService) signTransaction(c *gin.Context) {
	var req struct {
		TxID        string `json:"tx_id" binding:"required"`
		AdminAddress string `json:"admin_address" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	err := s.SignTransaction(req.TxID, req.AdminAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "signed"})
}

func (s *TimelockService) getSignatures(c *gin.Context) {
	txID := c.Param("tx_id")
	
	sigs, err := s.GetSignatures(txID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"signatures": sigs})
}

func (s *TimelockService) getTransactionHistory(c *gin.Context) {
	txID := c.Param("tx_id")
	
	var history []TimelockHistory
	s.db.Where("tx_id = ?", txID).Order("created_at DESC").Find(&history)
	
	c.JSON(http.StatusOK, gin.H{"history": history})
}

// ============================================================================
// Main
// ============================================================================

func main() {
	cfg := LoadConfig()
	
	service, err := NewTimelockService(cfg)
	if err != nil {
		log.Fatalf("Failed to create timelock service: %v", err)
	}
	
	router := gin.Default()
	service.setupRoutes(router)
	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-quit
		log.Println("Shutting down timelock service...")
		os.Exit(0)
	}()
	
	log.Printf("Timelock Service starting on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
