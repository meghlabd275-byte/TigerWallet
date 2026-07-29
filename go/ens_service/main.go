package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port       string
	RedisURL   string
	EthRPCURL  string
	EnsAddress string
}

func LoadConfig() *Config {
	return &Config{
		Port:      getEnv("PORT", "8453"),
		RedisURL:  getEnv("REDIS_URL", "redis://localhost:6379"),
		EthRPCURL: getEnv("ETH_RPC_URL", "https://eth.llamarpc.com"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// ENS Models
// ============================================================================

type ENSRecord struct {
	Name         string `json:"name"`
	Labelhash    string `json:"labelhash"`
	Nodehash     string `json:"nodehash"`
	Owner        string `json:"owner"`
	Resolver     string `json:"resolver"`
	TTL          uint64 `json:"ttl"`
	Address      string `json:"address,omitempty"`
	ContentHash  string `json:"content_hash,omitempty"`
	TextRecords  map[string]string `json:"text_records,omitempty"`
	Coins        map[string]string `json:"coins,omitempty"`
	ResolvedAt   int64 `json:"resolved_at"`
}

type ReverseRecord struct {
	Address   string `json:"address"`
	Name     string `json:"name"`
	ResolvedAt int64 `json:"resolved_at"`
}

type ENSService struct {
	config  *Config
	redis   *redis.Client
	client  *http.Client
	records map[string]*ENSRecord
	reverse map[string]*ReverseRecord
}

// ============================================================================
// ENS Contract ABIs (Simplified)
// ============================================================================

var (
	ENSRegistryABI = `[{"constant":true,"inputs":[{"name":"node","type":"bytes32"}],"name":"owner","outputs":[{"name":"","type":"address"}],"type":"function"}]`
	ResolverABI    = `[{"constant":true,"inputs":[{"name":"node","type":"bytes32"}],"name":"addr","outputs":[{"name":"ret","type":"address"}],"type":"function"}]`
)

// ============================================================================
// ENS Service
// ============================================================================

func NewENSService(config *Config) *ENSService {
	redisClient := redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
	})

	return &ENSService{
		config:  config,
		redis:   redisClient,
		client:  &http.Client{Timeout: 30 * time.Second},
		records: make(map[string]*ENSRecord),
		reverse: make(map[string]*ReverseRecord),
	}
}

// ============================================================================
// Name Processing
// ============================================================================

func (s *ENSService) nameHash(name string) string {
	// Simplified namehash calculation
	// In production, would follow ENS namehash algorithm
	
	if name == "" {
		return "0000000000000000000000000000000000000000000000000000000000000000"
	}

	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		// Not a valid ENS name
		return ""
	}

	// Calculate labelhash for first label
	label := parts[0]
	labelHash := sha256.Sum256([]byte(label))
	
	// Return simplified hash
	return hex.EncodeToString(labelHash[:])
}

func (s *ENSService) labelHash(label string) string {
	hash := sha256.Sum256([]byte(label))
	return hex.EncodeToString(hash[:])
}

// ============================================================================
// Resolution
// ============================================================================

func (s *ENSService) Resolve(name string) (*ENSRecord, error) {
	// Check cache first
	cached, err := s.redis.Get(context.Background(), "ens:"+name).Result()
	if err == nil {
		var record ENSRecord
		if json.Unmarshal([]byte(cached), &record) == nil {
			return &record, nil
		}
	}

	// Validate ENS name
	if !strings.HasSuffix(name, ".eth") {
		return nil, fmt.Errorf("invalid ENS name: must end with .eth")
	}

	// Simplified resolution - would query Ethereum
	nodehash := s.nameHash(name)

	// Create record (would query contracts)
	record := &ENSRecord{
		Name:         name,
		Labelhash:    s.labelHash(strings.TrimSuffix(name, ".eth")),
		Nodehash:     nodehash,
		Owner:        "0x" + hex.EncodeToString([]byte(name[:8])) + "000000000000000000000000",
		Resolver:     "0x0000000000000000000000000000000000000001",
		TTL:         86400,
		Address:     "0x0000000000000000000000000000000000000000",
		TextRecords: make(map[string]string),
		Coins:       make(map[string]string),
		ResolvedAt:   time.Now().Unix(),
	}

	// Cache result
	if data, err := json.Marshal(record); err == nil {
		s.redis.Set(context.Background(), "ens:"+name, string(data), 24*time.Hour)
	}

	s.records[name] = record
	return record, nil
}

func (s *ENSService) ReverseResolve(address string) (*ReverseRecord, error) {
	// Check cache first
	cached, err := s.redis.Get(context.Background(), "reverse:"+address).Result()
	if err == nil {
		var record ReverseRecord
		if json.Unmarshal([]byte(cached), &record) == nil {
			return &record, nil
		}
	}

	// Simplified - would query reverse registrar
	// Convert address to reverse name
	reverseName := address[2:10] + ".addr.reverse"

	record := &ReverseRecord{
		Address:    address,
		Name:       reverseName,
		ResolvedAt: time.Now().Unix(),
	}

	// Cache result
	if data, err := json.Marshal(record); err == nil {
		s.redis.Set(context.Background(), "reverse:"+address, string(data), 24*time.Hour)
	}

	s.reverse[address] = record
	return record, nil
}

func (s *ENSService) SetAddress(name, address string) error {
	record, err := s.Resolve(name)
	if err != nil {
		return err
	}

	record.Address = address
	record.ResolvedAt = time.Now().Unix()

	// Update cache
	if data, err := json.Marshal(record); err == nil {
		s.redis.Set(context.Background(), "ens:"+name, string(data), 24*time.Hour)
	}

	return nil
}

func (s *ENSService) SetTextRecord(name, key, value string) error {
	record, err := s.Resolve(name)
	if err != nil {
		return err
	}

	if record.TextRecords == nil {
		record.TextRecords = make(map[string]string)
	}

	record.TextRecords[key] = value
	record.ResolvedAt = time.Now().Unix()

	// Update cache
	if data, err := json.Marshal(record); err == nil {
		s.redis.Set(context.Background(), "ens:"+name, string(data), 24*time.Hour)
	}

	return nil
}

func (s *ENSService) SetContentHash(name, contentHash string) error {
	record, err := s.Resolve(name)
	if err != nil {
		return err
	}

	record.ContentHash = contentHash
	record.ResolvedAt = time.Now().Unix()

	// Update cache
	if data, err := json.Marshal(record); err == nil {
		s.redis.Set(context.Background(), "ens:"+name, string(data), 24*time.Hour)
	}

	return nil
}

func (s *ENSService) GetTextRecord(name, key string) (string, error) {
	record, err := s.Resolve(name)
	if err != nil {
		return "", err
	}

	if record.TextRecords == nil {
		return "", fmt.Errorf("no text records")
	}

	value, ok := record.TextRecords[key]
	if !ok {
		return "", fmt.Errorf("text record not found: %s", key)
	}

	return value, nil
}

func (s *ENSService) GetOwner(name string) (string, error) {
	record, err := s.Resolve(name)
	if err != nil {
		return "", err
	}

	return record.Owner, nil
}

func (s *ENSService) GetResolver(name string) (string, error) {
	record, err := s.Resolve(name)
	if err != nil {
		return "", err
	}

	return record.Resolver, nil
}

// ============================================================================
// Batch Resolution
// ============================================================================

func (s *ENSService) BatchResolve(names []string) ([]*ENSRecord, error) {
	results := make([]*ENSRecord, len(names))

	for i, name := range names {
		record, err := s.Resolve(name)
		if err != nil {
			results[i] = nil
			continue
		}
		results[i] = record
	}

	return results, nil
}

// ============================================================================
// Subgraph Query (Simplified)
// ============================================================================

func (s *ENSService) GetDomains(owner string) ([]*ENSRecord, error) {
	// Simplified - would query ENS subgraph
	// For now, return empty list
	return []*ENSRecord{}, nil
}

func (s *ENSService) GetDomain(name string) (*ENSRecord, error) {
	return s.Resolve(name)
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *ENSService) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "ens-service"})
	})

	api := r.Group("/api/v1/ens")
	{
		// Resolution
		api.GET("/resolve/:name", s.handleResolve)
		api.GET("/reverse/:address", s.handleReverseResolve)

		// Records
		api.GET("/owner/:name", s.handleGetOwner)
		api.GET("/resolver/:name", s.handleGetResolver)
		api.GET("/text/:name/:key", s.handleGetTextRecord)
		api.GET("/content/:name", s.handleGetContentHash)

		// Updates (would require auth in production)
		api.POST("/address", s.handleSetAddress)
		api.POST("/text", s.handleSetTextRecord)
		api.POST("/content", s.handleSetContentHash)

		// Batch
		api.POST("/batch", s.handleBatchResolve)

		// Domains
		api.GET("/domains/:owner", s.handleGetDomains)
		api.GET("/domain/:name", s.handleGetDomain)
	}
}

func (s *ENSService) handleResolve(c *gin.Context) {
	name := c.Param("name")

	record, err := s.Resolve(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, record)
}

func (s *ENSService) handleReverseResolve(c *gin.Context) {
	address := c.Param("address")

	// Ensure address has 0x prefix
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
	}

	record, err := s.ReverseResolve(address)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, record)
}

func (s *ENSService) handleGetOwner(c *gin.Context) {
	name := c.Param("name")

	owner, err := s.GetOwner(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"owner": owner})
}

func (s *ENSService) handleGetResolver(c *gin.Context) {
	name := c.Param("name")

	resolver, err := s.GetResolver(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"resolver": resolver})
}

func (s *ENSService) handleGetTextRecord(c *gin.Context) {
	name := c.Param("name")
	key := c.Param("key")

	value, err := s.GetTextRecord(name, key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"key": key, "value": value})
}

func (s *ENSService) handleGetContentHash(c *gin.Context) {
	name := c.Param("name")

	record, err := s.Resolve(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"content_hash": record.ContentHash})
}

func (s *ENSService) handleSetAddress(c *gin.Context) {
	var req struct {
		Name    string `json:"name" binding:"required"`
		Address string `json:"address" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.SetAddress(req.Name, req.Address); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "address set"})
}

func (s *ENSService) handleSetTextRecord(c *gin.Context) {
	var req struct {
		Name  string `json:"name" binding:"required"`
		Key   string `json:"key" binding:"required"`
		Value string `json:"value" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.SetTextRecord(req.Name, req.Key, req.Value); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "text record set"})
}

func (s *ENSService) handleSetContentHash(c *gin.Context) {
	var req struct {
		Name       string `json:"name" binding:"required"`
		ContentHash string `json:"content_hash" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.SetContentHash(req.Name, req.ContentHash); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "content hash set"})
}

func (s *ENSService) handleBatchResolve(c *gin.Context) {
	var req struct {
		Names []string `json:"names" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	records, err := s.BatchResolve(req.Names)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"records": records})
}

func (s *ENSService) handleGetDomains(c *gin.Context) {
	owner := c.Param("owner")

	domains, err := s.GetDomains(owner)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"domains": domains})
}

func (s *ENSService) handleGetDomain(c *gin.Context) {
	name := c.Param("name")

	domain, err := s.GetDomain(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, domain)
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()
	service := NewENSService(config)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	service.RegisterRoutes(r)

	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: r,
	}

	go func() {
		log.Printf("ENS Service starting on port %s", config.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
