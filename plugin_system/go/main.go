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
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog"
)

// ============================================================================
// TIGERWALLET PLUGIN SYSTEM (Like MetaMask Snaps)
// Production-ready extensibility system for plugins/snaps
// ============================================================================

var (
	logger      zerolog.Logger
	redisClient *redis.Client
	pluginStore *PluginStore
)

func main() {
	// Initialize logger
	output := zerolog.ConsoleWriter{Output: os.Stdout}
	logger = zerolog.New(output).With().Timestamp().Logger()

	// Load configuration
	cfg := loadConfig()

	// Initialize Redis
	redisClient = redis.NewClient(&redis.Options{
		Addr:     cfg.RedisURL,
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Warn().Err(err).Msg("Redis connection failed, using in-memory")
	}

	// Initialize plugin store
	pluginStore = NewPluginStore(cfg)

	// Setup router
	router := setupRouter(cfg)

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	logger.Info().Str("port", cfg.Port).Msg("Plugin System service started")

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	logger.Info().Msg("Server exited")
}

// Configuration
type Config struct {
	Port            string
	RedisURL        string
	PluginDirectory string
	MaxPluginSize   int64
	AllowedOrigins  []string
}

func loadConfig() *Config {
	return &Config{
		Port:            getEnv("PLUGIN_PORT", "9215"),
		RedisURL:        getEnv("REDIS_URL", "localhost:6379"),
		PluginDirectory: getEnv("PLUGIN_DIR", "./plugins"),
		MaxPluginSize:   10 * 1024 * 1024, // 10MB
		AllowedOrigins:  strings.Split(getEnv("ALLOWED_ORIGINS", "*"), ","),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ============================================================================
// DATA MODELS
// ============================================================================

// Plugin represents a plugin/snap
type Plugin struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Description     string            `json:"description"`
	Author          string            `json:"author"`
	AuthorURL       string            `json:"authorUrl"`
	HomepageURL     string            `json:"homepageUrl"`
	RepositoryURL   string            `json:"repositoryUrl"`
	License         string            `json:"license"`
	Category        string            `json:"category"` // defi, nft, security, utility, game, social
	Tags            []string          `json:"tags"`
	IconURL         string            `json:"iconUrl"`
	Screenshots     []string          `json:"screenshots"`
	Manifest        PluginManifest    `json:"manifest"`
	Permissions     []string          `json:"permissions"`
	RequiredPermissions []string      `json:"requiredPermissions"`
	Status          string            `json:"status"` // draft, pending, published, deprecated, removed
	Downloads       int               `json:"downloads"`
	Ratings         float64           `json:"ratings"`
	RatingCount    int               `json:"ratingCount"`
	ChainIDs        []uint64          `json:"chainIds"` // Supported chains
	VersionHistory  []VersionEntry   `json:"versionHistory"`
	CreatedAt       int64             `json:"createdAt"`
	UpdatedAt       int64             `json:"updatedAt"`
	PublishedAt     int64             `json:"publishedAt"`
}

// PluginManifest defines plugin capabilities
type PluginManifest struct {
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Description string         `json:"description"`
	Main        string         `json:"main"` // Entry point
	Scripts     map[string]string `json:"scripts,omitempty"`
	Permissions []string       `json:"permissions"`
	ContentScripts []ContentScript `json:"contentScripts,omitempty"`
	Background   *BackgroundScript `json:"background,omitempty"`
	Icon         string        `json:"icon,omitempty"`
}

type ContentScript struct {
	Matches []string `json:"matches"`
	Scripts []string `json:"scripts"`
	RunAt   string   `json:"runAt"`
}

type BackgroundScript struct {
	ServiceWorker string `json:"serviceWorker"`
	Scripts       []string `json:"scripts"`
}

// VersionEntry represents a version release
type VersionEntry struct {
	Version   string `json:"version"`
	Changelog string `json:"changelog"`
	ReleasedAt int64 `json:"releasedAt"`
}

// PluginInstance represents an installed plugin
type PluginInstance struct {
	InstanceID  string    `json:"instanceId"`
	PluginID    string    `json:"pluginId"`
	UserID      string    `json:"userId"`
	Enabled     bool      `json:"enabled"`
	Permissions []string  `json:"permissions"` // Granted permissions
	Config      map[string]interface{} `json:"config"`
	CreatedAt   int64     `json:"createdAt"`
	UpdatedAt   int64     `json:"updatedAt"`
}

// PluginRequest represents a request to a plugin
type PluginRequest struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

// PluginResponse represents response from plugin
type PluginResponse struct {
	Success bool                   `json:"success"`
	Result  map[string]interface{} `json:"result,omitempty"`
	Error   string                `json:"error,omitempty"`
}

// PluginStore manages plugins
type PluginStore struct {
	config *Config
	mu     sync.RWMutex
	plugins map[string]*Plugin
}

func NewPluginStore(cfg *Config) *PluginStore {
	store := &PluginStore{
		config:   cfg,
		plugins:  make(map[string]*Plugin),
	}

	// Load built-in plugins
	store.loadBuiltInPlugins()

	return store
}

func (s *PluginStore) loadBuiltInPlugins() {
	// Add built-in plugins
	builtInPlugins := []*Plugin{
		{
			ID:          "tiger-defi",
			Name:        "DeFi Dashboard",
			Version:     "1.0.0",
			Description: "Advanced DeFi portfolio tracking and analytics",
			Author:      "TigerWallet",
			Category:   "defi",
			Tags:       []string{"defi", "portfolio", "analytics"},
			Status:     "published",
			ChainIDs:   []uint64{1, 56, 137, 42161},
			Permissions: []string{"eth_accounts", "eth_chainId", "eth_blockNumber"},
			CreatedAt:  time.Now().Unix(),
			UpdatedAt:  time.Now().Unix(),
		},
		{
			ID:          "tiger-nft",
			Name:        "NFT Manager",
			Version:     "1.0.0",
			Description: "Manage and view NFT collections across chains",
			Author:      "TigerWallet",
			Category:   "nft",
			Tags:       []string{"nft", "collection", "gallery"},
			Status:     "published",
			ChainIDs:   []uint64{1, 137, 42161},
			Permissions: []string{"eth_accounts", "eth_chainId"},
			CreatedAt:  time.Now().Unix(),
			UpdatedAt:  time.Now().Unix(),
		},
		{
			ID:          "tiger-security",
			Name:        "Security Scanner",
			Version:     "1.0.0",
			Description: "Scan transactions and addresses for security risks",
			Author:      "TigerWallet",
			Category:   "security",
			Tags:       []string{"security", "scanner", "protection"},
			Status:     "published",
			ChainIDs:   []uint64{1, 56, 137, 42161, 10},
			Permissions: []string{"eth_accounts", "eth_chainId", "eth_blockNumber"},
			CreatedAt:  time.Now().Unix(),
			UpdatedAt:  time.Now().Unix(),
		},
		{
			ID:          "tiger-stake",
			Name:        "Liquid Staking",
			Version:     "1.0.0",
			Description: "Stake tokens and earn yields with liquid staking",
			Author:      "TigerWallet",
			Category:   "defi",
			Tags:       []string{"staking", "yield", "defi"},
			Status:     "published",
			ChainIDs:   []uint64{1, 56},
			Permissions: []string{"eth_accounts", "eth_chainId"},
			CreatedAt:  time.Now().Unix(),
			UpdatedAt:  time.Now().Unix(),
		},
	}

	for _, p := range builtInPlugins {
		s.plugins[p.ID] = p
	}
}

// ============================================================================
// API HANDLERS
// ============================================================================

func setupRouter(cfg *Config) *gin.Engine {
	r := gin.Default()

	r.Use(corsMiddleware())

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// API v1
	v1 := r.Group("/api/v1")
	{
		// Plugin registry
		plugins := v1.Group("/plugins")
		{
			plugins.GET("", listPlugins)
			plugins.GET("/featured", getFeaturedPlugins)
			plugins.GET("/categories", getCategories)
			plugins.GET("/search", searchPlugins)
			plugins.GET("/:id", getPlugin)
			plugins.POST("", createPlugin)
			plugins.PUT("/:id", updatePlugin)
			plugins.DELETE("/:id", deletePlugin)
			plugins.POST("/:id/publish", publishPlugin)
			plugins.POST("/:id/deprecate", deprecatePlugin)
		}

		// User plugin instances
		instances := v1.Group("/instances")
		{
			instances.POST("", installPlugin)
			instances.DELETE("/:id", uninstallPlugin)
			instances.GET("/user/:userId", getUserInstances)
			instances.PUT("/:id/enable", enablePlugin)
			instances.PUT("/:id/disable", disablePlugin)
			instances.PUT("/:id/config", updatePluginConfig)
		}

		// Plugin execution
		execute := v1.Group("/execute")
		{
			execute.POST("/:instanceId", executePlugin)
			execute.POST("/:instanceId/rpc", pluginRPC)
		}

		// Permissions
		permissions := v1.Group("/permissions")
		{
			permissions.GET("", listPermissions)
			permissions.GET("/:pluginId", getPluginPermissions)
		}

		// Updates
		updates := v1.Group("/updates")
		{
			updates.GET("/check", checkUpdates)
		}
	}

	return r
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Plugin-Id")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// ============================================================================
// PLUGIN REGISTRY HANDLERS
// ============================================================================

func listPlugins(c *gin.Context) {
	category := c.Query("category")
	chainID := getUint64Param(c, "chainId", 0)
	limit := getIntParam(c, "limit", 50)
	offset := getIntParam(c, "offset", 0)

	var plugins []*Plugin
	for _, p := range pluginStore.plugins {
		if p.Status != "published" {
			continue
		}
		if category != "" && p.Category != category {
			continue
		}
		if chainID > 0 {
			found := false
			for _, id := range p.ChainIDs {
				if id == chainID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		plugins = append(plugins, p)
	}

	// Apply pagination
	start := offset
	end := offset + limit
	if start > len(plugins) {
		plugins = []*Plugin{}
	} else {
		if end > len(plugins) {
			end = len(plugins)
		}
		plugins = plugins[start:end]
	}

	c.JSON(http.StatusOK, gin.H{
		"plugins":   plugins,
		"total":     len(plugins),
		"limit":     limit,
		"offset":    offset,
	})
}

func getFeaturedPlugins(c *gin.Context) {
	var featured []*Plugin
	for _, p := range pluginStore.plugins {
		if p.Status == "published" && p.Downloads > 1000 {
			featured = append(featured, p)
		}
	}
	c.JSON(http.StatusOK, featured)
}

func getCategories(c *gin.Context) {
	categories := []string{"defi", "nft", "security", "utility", "game", "social"}
	c.JSON(http.StatusOK, categories)
}

func searchPlugins(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query required"})
		return
	}

	query = strings.ToLower(query)
	var results []*Plugin

	for _, p := range pluginStore.plugins {
		if p.Status != "published" {
			continue
		}

		match := strings.Contains(strings.ToLower(p.Name), query) ||
			strings.Contains(strings.ToLower(p.Description), query) ||
			strings.Contains(strings.ToLower(p.Author), query)

		for _, tag := range p.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				match = true
				break
			}
		}

		if match {
			results = append(results, p)
		}
	}

	c.JSON(http.StatusOK, results)
}

func getPlugin(c *gin.Context) {
	id := c.Param("id")

	plugin, exists := pluginStore.plugins[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}

	c.JSON(http.StatusOK, plugin)
}

func createPlugin(c *gin.Context) {
	var plugin Plugin
	if err := c.ShouldBindJSON(&plugin); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	plugin.ID = generateID()
	plugin.Status = "draft"
	plugin.CreatedAt = time.Now().Unix()
	plugin.UpdatedAt = time.Now().Unix()

	pluginStore.plugins[plugin.ID] = &plugin

	c.JSON(http.StatusCreated, plugin)
}

func updatePlugin(c *gin.Context) {
	id := c.Param("id")

	plugin, exists := pluginStore.plugins[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		plugin.Name = name
	}
	if desc, ok := updates["description"].(string); ok {
		plugin.Description = desc
	}
	if version, ok := updates["version"].(string); ok {
		plugin.Version = version
	}
	plugin.UpdatedAt = time.Now().Unix()

	c.JSON(http.StatusOK, plugin)
}

func deletePlugin(c *gin.Context) {
	id := c.Param("id")

	if _, exists := pluginStore.plugins[id]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}

	delete(pluginStore.plugins, id)
	c.JSON(http.StatusOK, gin.H{"message": "Plugin deleted"})
}

func publishPlugin(c *gin.Context) {
	id := c.Param("id")

	plugin, exists := pluginStore.plugins[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}

	plugin.Status = "published"
	plugin.PublishedAt = time.Now().Unix()
	plugin.UpdatedAt = time.Now().Unix()

	c.JSON(http.StatusOK, plugin)
}

func deprecatePlugin(c *gin.Context) {
	id := c.Param("id")

	plugin, exists := pluginStore.plugins[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}

	plugin.Status = "deprecated"
	plugin.UpdatedAt = time.Now().Unix()

	c.JSON(http.StatusOK, plugin)
}

// ============================================================================
// INSTANCE HANDLERS
// ============================================================================

func installPlugin(c *gin.Context) {
	var req struct {
		UserID   string `json:"userId" binding:"required"`
		PluginID string `json:"pluginId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify plugin exists
	plugin, exists := pluginStore.plugins[req.PluginID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}

	instance := PluginInstance{
		InstanceID:  generateID(),
		PluginID:    req.PluginID,
		UserID:      req.UserID,
		Enabled:     true,
		Permissions: plugin.RequiredPermissions,
		Config:      make(map[string]interface{}),
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}

	// Save to Redis
	instanceKey := fmt.Sprintf("instance:%s", instance.InstanceID)
	if data, err := json.Marshal(instance); err == nil {
		redisClient.Set(context.Background(), instanceKey, data, 365*24*time.Hour)
	}

	// Increment download count
	plugin.Downloads++
	pluginStore.plugins[req.PluginID] = plugin

	c.JSON(http.StatusCreated, instance)
}

func uninstallPlugin(c *gin.Context) {
	id := c.Param("id")

	instanceKey := fmt.Sprintf("instance:%s", id)
	redisClient.Del(context.Background(), instanceKey)

	c.JSON(http.StatusOK, gin.H{"message": "Plugin uninstalled"})
}

func getUserInstances(c *gin.Context) {
	userID := c.Param("userId")
	// In production: query from Redis/DB
	c.JSON(http.StatusOK, []PluginInstance{})
}

func enablePlugin(c *gin.Context) {
	id := c.Param("id")
	instanceKey := fmt.Sprintf("instance:%s", id)

	data, err := redisClient.Get(context.Background(), instanceKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Instance not found"})
		return
	}

	var instance PluginInstance
	json.Unmarshal(data, &instance)

	instance.Enabled = true
	instance.UpdatedAt = time.Now().Unix()

	if data, err := json.Marshal(instance); err == nil {
		redisClient.Set(context.Background(), instanceKey, data, 365*24*time.Hour)
	}

	c.JSON(http.StatusOK, instance)
}

func disablePlugin(c *gin.Context) {
	id := c.Param("id")
	instanceKey := fmt.Sprintf("instance:%s", id)

	data, err := redisClient.Get(context.Background(), instanceKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Instance not found"})
		return
	}

	var instance PluginInstance
	json.Unmarshal(data, &instance)

	instance.Enabled = false
	instance.UpdatedAt = time.Now().Unix()

	if data, err := json.Marshal(instance); err == nil {
		redisClient.Set(context.Background(), instanceKey, data, 365*24*time.Hour)
	}

	c.JSON(http.StatusOK, instance)
}

func updatePluginConfig(c *gin.Context) {
	id := c.Param("id")
	var config map[string]interface{}

	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	instanceKey := fmt.Sprintf("instance:%s", id)

	data, err := redisClient.Get(context.Background(), instanceKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Instance not found"})
		return
	}

	var instance PluginInstance
	json.Unmarshal(data, &instance)

	instance.Config = config
	instance.UpdatedAt = time.Now().Unix()

	if data, err := json.Marshal(instance); err == nil {
		redisClient.Set(context.Background(), instanceKey, data, 365*24*time.Hour)
	}

	c.JSON(http.StatusOK, instance)
}

// ============================================================================
// EXECUTION HANDLERS
// ============================================================================

func executePlugin(c *gin.Context) {
	instanceID := c.Param("instanceId")
	var req PluginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get instance
	instanceKey := fmt.Sprintf("instance:%s", instanceID)
	data, err := redisClient.Get(context.Background(), instanceKey).Bytes()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}

	var instance PluginInstance
	json.Unmarshal(data, &instance)

	if !instance.Enabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "Plugin is disabled"})
		return
	}

	// Get plugin
	plugin, exists := pluginStore.plugins[instance.PluginID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}

	// Execute plugin (simulated)
	response := &PluginResponse{
		Success: true,
		Result: map[string]interface{}{
			"executed":   true,
			"method":     req.Method,
			"plugin":     plugin.Name,
			"timestamp":  time.Now().Unix(),
		},
	}

	c.JSON(http.StatusOK, response)
}

func pluginRPC(c *gin.Context) {
	instanceID := c.Param("instanceId")
	var req struct {
		Method string                 `json:"method" binding:"required"`
		Params map[string]interface{} `json:"params"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Route to plugin RPC handler
	response := &PluginResponse{
		Success: true,
		Result: map[string]interface{}{
			"method": req.Method,
		},
	}

	c.JSON(http.StatusOK, response)
}

// ============================================================================
// PERMISSIONS HANDLERS
// ============================================================================

func listPermissions(c *gin.Context) {
	permissions := []map[string]string{
		{"name": "eth_accounts", "description": "View account addresses"},
		{"name": "eth_chainId", "description": "View current chain ID"},
		{"name": "eth_blockNumber", "description": "View block number"},
		{"name": "eth_call", "description": "Call smart contracts"},
		{"name": "eth_sendTransaction", "description": "Send transactions"},
		{"name": "eth_sign", "description": "Sign messages"},
		{"name": "personal_sign", "description": "Sign personal messages"},
		{"name": "eth_getBalance", "description": "Get account balance"},
		{"name": "eth_getCode", "description": "Get contract code"},
		{"name": "eth_getLogs", "description": "Get event logs"},
	}

	c.JSON(http.StatusOK, permissions)
}

func getPluginPermissions(c *gin.Context) {
	pluginID := c.Param("pluginId")

	plugin, exists := pluginStore.plugins[pluginID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}

	c.JSON(http.StatusOK, plugin.Permissions)
}

// ============================================================================
// UPDATE HANDLERS
// ============================================================================

func checkUpdates(c *gin.Context) {
	userID := c.Query("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID required"})
		return
	}

	// Get user's installed plugins
	// Check for updates
	updates := []map[string]interface{}{}

	c.JSON(http.StatusOK, gin.H{
		"hasUpdates": len(updates) > 0,
		"updates":     updates,
	})
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func generateID() string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d%s", time.Now().UnixNano(), "tiger")))
	return hex.EncodeToString(hash[:])[:16]
}

func getUint64Param(c *gin.Context, name string, def uint64) uint64 {
	if val := c.Query(name); val != "" {
		var parsed uint64
		if _, err := fmt.Sscanf(val, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return def
}

func getIntParam(c *gin.Context, name string, def int) int {
	if val := c.Query(name); val != "" {
		var parsed int
		if _, err := fmt.Sscanf(val, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return def
}
