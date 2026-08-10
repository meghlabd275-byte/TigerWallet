package main

import (
	"context"
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

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
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
	Name        string            `json:"name"`
	Labelhash   string            `json:"labelhash"`
	Nodehash    string            `json:"nodehash"`
	Owner       string            `json:"owner"`
	Resolver    string            `json:"resolver"`
	TTL         uint64            `json:"ttl"`
	Address     string            `json:"address,omitempty"`
	ContentHash string            `json:"content_hash,omitempty"`
	TextRecords map[string]string `json:"text_records,omitempty"`
	Coins       map[string]string `json:"coins,omitempty"`
	ResolvedAt  int64             `json:"resolved_at"`
}

type ReverseRecord struct {
	Address    string `json:"address"`
	Name       string `json:"name"`
	ResolvedAt int64  `json:"resolved_at"`
}

type ENSService struct {
	config  *Config
	redis   *redis.Client
	client  *http.Client
	eth     *ethclient.Client // real Ethereum RPC client for on-chain ENS lookup
	records map[string]*ENSRecord
	reverse map[string]*ReverseRecord
}

// ENSRegistry is the canonical ENS registry deployed on Ethereum mainnet.
// See ENS deployment docs. Used to resolve resolver(bytes32 node).
var ENSRegistry = common.HexToAddress("0x00000000000C2E074eC69A0dFb2997BA6C7d2e1e")

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

	svc := &ENSService{
		config:  config,
		redis:   redisClient,
		client:  &http.Client{Timeout: 30 * time.Second},
		records: make(map[string]*ENSRecord),
		reverse: make(map[string]*ReverseRecord),
	}

	// Initialize a real Ethereum RPC client for on-chain ENS resolution.
	if config.EthRPCURL != "" {
		client, err := ethclient.Dial(config.EthRPCURL)
		if err != nil {
			log.Printf("ENS: failed to connect to Ethereum RPC %s: %v", config.EthRPCURL, err)
		} else {
			svc.eth = client
			log.Printf("ENS: connected to Ethereum RPC %s", config.EthRPCURL)
		}
	}

	return svc
}

// ============================================================================
// Name Processing
// ============================================================================

// nameHash implements the EIP-137 namehash algorithm using keccak256.
// namehash("") = 0x00...00; namehash(name) = keccak256(namehash(parent) || keccak256(label)).
// Labels are processed in reverse order (TLD last). This is the real ENS
// algorithm; the previous implementation incorrectly used SHA-256.
func (s *ENSService) nameHash(name string) string {
	node := make([]byte, 32) // namehash("") = 32 zero bytes
	if name == "" {
		return hex.EncodeToString(node)
	}
	labels := strings.Split(name, ".")
	for i := len(labels) - 1; i >= 0; i-- {
		labelHash := crypto.Keccak256([]byte(labels[i]))
		node = crypto.Keccak256(append(node, labelHash...))
	}
	return hex.EncodeToString(node)
}

// labelHash returns keccak256(label) - the ENS labelhash (EIP-137).
func (s *ENSService) labelHash(label string) string {
	return hex.EncodeToString(crypto.Keccak256([]byte(label)))
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

	nodehash := s.nameHash(name)
	node := common.HexToHash(nodehash)

	record := &ENSRecord{
		Name:        name,
		Labelhash:   s.labelHash(strings.TrimSuffix(name, ".eth")),
		Nodehash:    nodehash,
		TextRecords: make(map[string]string),
		Coins:       make(map[string]string),
		ResolvedAt:  time.Now().Unix(),
	}

	// Real on-chain ENS resolution. If no Ethereum RPC client is configured,
	// return only the computed namehash (no fabricated owner/resolver/address).
	if s.eth == nil {
		return record, fmt.Errorf("ETH_RPC_URL not configured: cannot resolve on-chain ENS record")
	}

	// resolver(bytes32) selector = 0x0178b8bf
	resolverData := append([]byte{0x01, 0x78, 0xb8, 0xbf}, node.Bytes()...)
	res, err := s.eth.CallContract(context.Background(), ethereum.CallMsg{
		To:   &ENSRegistry,
		Data: resolverData,
	}, nil)
	if err != nil || len(res) < 32 {
		return record, fmt.Errorf("no resolver set for %s: %v", name, err)
	}
	resolverAddr := common.BytesToAddress(res[12:32])
	record.Resolver = resolverAddr.Hex()
	if resolverAddr == (common.Address{}) {
		return record, fmt.Errorf("no resolver set for %s", name)
	}

	// addr(bytes32) selector = 0x3b3b57de
	addrData := append([]byte{0x3b, 0x3b, 0x57, 0xde}, node.Bytes()...)
	addrRes, err := s.eth.CallContract(context.Background(), ethereum.CallMsg{
		To:   &resolverAddr,
		Data: addrData,
	}, nil)
	if err == nil && len(addrRes) >= 32 {
		record.Address = common.BytesToAddress(addrRes[12:32]).Hex()
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

	// Real reverse resolution via the ENS reverse registrar. The reverse node
	// is namehash("<lowercase-hex-addr-without-0x>.addr.reverse"). If no RPC
	// client is configured we return an empty name (never a fabricated one).
	cleanAddr := strings.ToLower(strings.TrimPrefix(address, "0x"))
	if cleanAddr == "" {
		return nil, fmt.Errorf("invalid address")
	}

	record := &ReverseRecord{
		Address:    address,
		Name:       "",
		ResolvedAt: time.Now().Unix(),
	}

	if s.eth == nil {
		return record, fmt.Errorf("ETH_RPC_URL not configured: cannot reverse-resolve on-chain")
	}

	reverseNode := common.HexToHash(s.nameHash(cleanAddr + ".addr.reverse"))

	// resolver(bytes32) on the ENS registry
	resolverData := append([]byte{0x01, 0x78, 0xb8, 0xbf}, reverseNode.Bytes()...)
	res, err := s.eth.CallContract(context.Background(), ethereum.CallMsg{
		To:   &ENSRegistry,
		Data: resolverData,
	}, nil)
	if err != nil || len(res) < 32 {
		return record, fmt.Errorf("no reverse resolver for %s: %v", address, err)
	}
	resolverAddr := common.BytesToAddress(res[12:32])
	if resolverAddr == (common.Address{}) {
		return record, fmt.Errorf("no reverse resolver set for %s", address)
	}

	// name(bytes32) selector = 0x691f3431
	nameData := append([]byte{0x69, 0x1f, 0x34, 0x31}, reverseNode.Bytes()...)
	nameRes, err := s.eth.CallContract(context.Background(), ethereum.CallMsg{
		To:   &resolverAddr,
		Data: nameData,
	}, nil)
	if err != nil || len(nameRes) < 96 {
		return record, fmt.Errorf("no reverse name for %s: %v", address, err)
	}

	// ABI-decode a dynamic string: offset(32) + length(32) + data
	offset := new(big.Int).SetBytes(nameRes[0:32]).Int64()
	length := new(big.Int).SetBytes(nameRes[offset : offset+32]).Int64()
	end := offset + 32 + length
	if end > int64(len(nameRes)) {
		end = int64(len(nameRes))
	}
	record.Name = string(nameRes[offset+32 : end])

	// Cache result
	if data, err := json.Marshal(record); err == nil && record.Name != "" {
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
	// No ENS subgraph source is configured. Return an empty result (pending)
	// rather than fabricating domain records for the owner.
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
		Name        string `json:"name" binding:"required"`
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
