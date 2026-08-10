/**
 * TigerWallet CDN Service
 * 
 * CDN service for static asset delivery, caching, and content distribution.
 * Built with Go for high-load distributed operations.
 */

package cdn

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Types
// ============================================================================

// Asset represents a CDN asset
type Asset struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"`
	ContentType string   `json:"content_type"`
	Size        int64     `json:"size"`
	Hash        string    `json:"hash"`
	URL         string    `json:"url"`
	CacheTTL    int64     `json:"cache_ttl"`
	CreatedAt   int64     `json:"created_at"`
	ExpiresAt   int64     `json:"expires_at"`
}

// AssetMetadata represents asset metadata
type AssetMetadata struct {
	Name        string            `json:"name"`
	ContentType string            `json:"content_type"`
	Size        int64             `json:"size"`
	Tags        []string          `json:"tags"`
	Metadata   map[string]string `json:"metadata"`
}

// CDNConfig represents CDN configuration
type CDNConfig struct {
	BaseURL       string
	StoragePath   string
	MaxFileSize   int64
	AllowedTypes []string
	CacheTTL     int64
	EnableGzip   bool
}

// UploadRequest represents an upload request
type UploadRequest struct {
	Content     []byte
	ContentType string
	Name        string
	Tags        []string
	Metadata    map[string]string
	TTL         int64
}

// CDNService manages CDN operations
type CDNService struct {
	config    *CDNConfig
	assets   map[string]*Asset
	mu       sync.RWMutex
	storage  *MemoryStorage
}

// MemoryStorage represents in-memory cache storage
type MemoryStorage struct {
	data map[string][]byte
	mu   sync.RWMutex
}

// ============================================================================
// Service
// ============================================================================

var (
	cdnService     *CDNService
	cdnServiceOnce sync.Once
)

// GetCDNService returns the singleton CDN service
func GetCDNService(config *CDNConfig) *CDNService {
	cdnServiceOnce.Do(func() {
		if config == nil {
			config = &CDNConfig{
				BaseURL:     "https://cdn.tigerwallet.com",
				StoragePath: "./cdn-storage",
				MaxFileSize: 100 * 1024 * 1024, // 100MB
				CacheTTL:    86400, // 24 hours
				EnableGzip: true,
			}
		}
		
		cdnService = &CDNService{
			config:   config,
			assets:  make(map[string]*Asset),
			storage: &MemoryStorage{
				data: make(map[string][]byte),
			},
		}
	})
	return cdnService
}

// ============================================================================
// Upload Operations
// ============================================================================

// Upload uploads an asset to CDN
func (s *CDNService) Upload(ctx context.Context, req *UploadRequest) (*Asset, error) {
	// Validate content type
	if !s.isAllowedContentType(req.ContentType) {
		return nil, fmt.Errorf("content type not allowed: %s", req.ContentType)
	}

	// Validate file size
	if int64(len(req.Content)) > s.config.MaxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed: %d bytes", s.config.MaxFileSize)
	}

	// Generate asset ID and key
	id := "asset_" + uuid.New().String()
	key := s.generateKey(req.Name)

	// Calculate hash
	hash := md5.Sum(req.Content)
	hashStr := hex.EncodeToString(hash[:])

	// Set expiry
	now := time.Now().Unix()
	ttl := req.TTL
	if ttl == 0 {
		ttl = s.config.CacheTTL
	}

	asset := &Asset{
		ID:          id,
		Key:         key,
		ContentType: req.ContentType,
		Size:        int64(len(req.Content)),
		Hash:        hashStr,
		URL:         fmt.Sprintf("%s/%s", s.config.BaseURL, key),
		CacheTTL:    ttl,
		CreatedAt:   now,
		ExpiresAt:   now + ttl,
	}

	// Store in memory (in production, use file storage or object storage)
	s.storage.mu.Lock()
	s.storage.data[key] = req.Content
	s.storage.mu.Unlock()

	s.mu.Lock()
	s.assets[id] = asset
	s.mu.Unlock()

	return asset, nil
}

// UploadFromReader uploads an asset from a reader
func (s *CDNService) UploadFromReader(ctx context.Context, reader io.Reader, filename string, contentType string) (*Asset, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	req := &UploadRequest{
		Content:     content,
		ContentType: contentType,
		Name:        filename,
	}

	return s.Upload(ctx, req)
}

// ============================================================================
// Download Operations
// ============================================================================

// Get retrieves an asset by ID
func (s *CDNService) Get(ctx context.Context, id string) (*Asset, []byte, error) {
	s.mu.RLock()
	asset, exists := s.assets[id]
	s.mu.RUnlock()

	if !exists {
		return nil, nil, fmt.Errorf("asset not found")
	}

	// Check expiry
	if time.Now().Unix() > asset.ExpiresAt {
		return nil, nil, fmt.Errorf("asset expired")
	}

	// Retrieve content
	s.storage.mu.RLock()
	content, exists := s.storage.data[asset.Key]
	s.storage.mu.RUnlock()

	if !exists {
		return nil, nil, fmt.Errorf("asset content not found")
	}

	return asset, content, nil
}

// GetByKey retrieves an asset by key
func (s *CDNService) GetByKey(ctx context.Context, key string) (*Asset, []byte, error) {
	s.mu.RLock()
	var asset *Asset
	for _, a := range s.assets {
		if a.Key == key {
			asset = a
			break
		}
	}
	s.mu.RUnlock()

	if asset == nil {
		return nil, nil, fmt.Errorf("asset not found")
	}

	return s.Get(ctx, asset.ID)
}

// ============================================================================
// Delete Operations
// ============================================================================

// Delete removes an asset from CDN
func (s *CDNService) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	asset, exists := s.assets[id]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("asset not found")
	}

	// Remove from storage
	delete(s.storage.data, asset.Key)
	delete(s.assets, id)
	s.mu.Unlock()

	return nil
}

// DeleteByKey removes an asset by key
func (s *CDNService) DeleteByKey(ctx context.Context, key string) error {
	s.mu.RLock()
	var asset *Asset
	for _, a := range s.assets {
		if a.Key == key {
			asset = a
			break
		}
	}
	s.mu.RUnlock()

	if asset == nil {
		return fmt.Errorf("asset not found")
	}

	return s.Delete(ctx, asset.ID)
}

// ============================================================================
// List Operations
// ============================================================================

// List returns all assets
func (s *CDNService) List(ctx context.Context) []*Asset {
	s.mu.RLock()
	defer s.mu.RUnlock()

	assets := make([]*Asset, 0, len(s.assets))
	for _, asset := range s.assets {
		assets = append(assets, asset)
	}

	return assets
}

// ListByTag returns assets by tag
func (s *CDNService) ListByTag(ctx context.Context, tag string) []*Asset {
	// This would require storing tags - simplified for now
	return s.List(ctx)
}

// ============================================================================
// Cache Operations
// ============================================================================

// InvalidateCache invalidates asset cache
func (s *CDNService) InvalidateCache(ctx context.Context, id string) error {
	s.mu.RLock()
	asset, exists := s.assets[id]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("asset not found")
	}

	// Update expiry to now to force refresh
	s.mu.Lock()
	asset.ExpiresAt = time.Now().Unix() - 1
	s.mu.Unlock()

	return nil
}

// RefreshCache refreshes asset cache
func (s *CDNService) RefreshCache(ctx context.Context, id string, ttl int64) error {
	s.mu.RLock()
	asset, exists := s.assets[id]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("asset not found")
	}

	s.mu.Lock()
	if ttl > 0 {
		asset.CacheTTL = ttl
	}
	asset.ExpiresAt = time.Now().Unix() + asset.CacheTTL
	s.mu.Unlock()

	return nil
}

// ============================================================================
// HTTP Handler
// ============================================================================

// ServeHTTP serves CDN content over HTTP
func (s *CDNService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get key from URL path
	key := r.URL.Path[1:] // Remove leading slash

	if key == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Get asset
	asset, content, err := s.GetByKey(r.Context(), key)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Set headers
	w.Header().Set("Content-Type", asset.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", asset.Size))
	w.Header().Set("ETag", asset.Hash)
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", asset.CacheTTL))

	// Serve content
	w.Write(content)
}

// ============================================================================
// Helper Functions
// ============================================================================

func (s *CDNService) generateKey(filename string) string {
	// Generate unique key based on timestamp and random
	hash := md5.Sum([]byte(filename + fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(hash[:]) + filepath.Ext(filename)
}

func (s *CDNService) isAllowedContentType(contentType string) bool {
	if len(s.config.AllowedTypes) == 0 {
		return true // Allow all if not specified
	}

	for _, allowed := range s.config.AllowedTypes {
		if contentType == allowed {
			return true
		}
	}

	return false
}

func getContentType(filename string) string {
	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return contentType
}

// ============================================================================
// Utility
// ============================================================================

// GetStats returns CDN statistics
func (s *CDNService) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var totalSize int64
	var totalAssets int

	for _, asset := range s.assets {
		totalSize += asset.Size
		totalAssets++
	}

	return map[string]interface{}{
		"total_assets":  totalAssets,
		"total_size":    totalSize,
		"cache_ttl":     s.config.CacheTTL,
		"max_file_size": s.config.MaxFileSize,
		"base_url":       s.config.BaseURL,
	}
}
