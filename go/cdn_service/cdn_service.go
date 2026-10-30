/**
 * TigerWallet CDN Service
 *
 * CDN service for static asset delivery, caching, and content distribution.
 * Built with Go for high-load distributed operations.
 */

package cdn

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Types
// ============================================================================

// Asset represents a CDN asset
type Asset struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Hash        string `json:"hash"`
	URL         string `json:"url"`
	CacheTTL    int64  `json:"cache_ttl"`
	CreatedAt   int64  `json:"created_at"`
	ExpiresAt   int64  `json:"expires_at"`
}

// AssetMetadata represents asset metadata
type AssetMetadata struct {
	Name        string            `json:"name"`
	ContentType string            `json:"content_type"`
	Size        int64             `json:"size"`
	Tags        []string          `json:"tags"`
	Metadata    map[string]string `json:"metadata"`
}

// CDNConfig represents CDN configuration
type CDNConfig struct {
	BaseURL      string
	StoragePath  string
	MaxFileSize  int64
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
	config  *CDNConfig
	assets  map[string]*Asset
	mu      sync.RWMutex
	storage *MemoryStorage
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
				CacheTTL:    86400,             // 24 hours
				EnableGzip:  true,
			}
		}

		cdnService = &CDNService{
			config: config,
			assets: make(map[string]*Asset),
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

	// Calculate integrity hash. SHA-256, not MD5: this value is published
	// as an ETag and must not be forgeable by a collision attack.
	sum := sha256.Sum256(req.Content)
	hashStr := hex.EncodeToString(sum[:])

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

	// Set headers. nosniff stops the browser from re-interpreting stored
	// bytes as HTML/JS when the declared type would not normally execute.
	w.Header().Set("Content-Type", asset.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", asset.Size))
	w.Header().Set("ETag", asset.Hash)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", asset.CacheTTL))

	// Serve content
	w.Write(content)
}

// ============================================================================
// Helper Functions
// ============================================================================

func (s *CDNService) generateKey(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if !safeExtension(ext) {
		ext = ""
	}
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// crypto/rand is the only entropy source acceptable for a public
		// object key; never fall back to time or math/rand.
		panic("cdn: crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(buf[:]) + ext
}

// safeExtension rejects any extension that is not a short alphanumeric token.
// An attacker-supplied filename must never contribute path separators,
// dots or null bytes to a stored object key.
func safeExtension(ext string) bool {
	if ext == "" || len(ext) > 16 {
		return false
	}
	if ext[0] != '.' {
		return false
	}
	for _, r := range ext[1:] {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

func (s *CDNService) isAllowedContentType(contentType string) bool {
	// Media type parameters (charset etc.) are not part of the allowlist.
	if i := strings.IndexByte(contentType, ';'); i != -1 {
		contentType = contentType[:i]
	}
	contentType = strings.ToLower(strings.TrimSpace(contentType))

	// Fail closed: without an explicit allowlist, only known-safe web
	// asset types may be stored. Anything active (HTML, SVG, JS) is
	// rejected because it would be served same-origin.
	if len(s.config.AllowedTypes) == 0 {
		return defaultAllowedTypes[contentType]
	}

	for _, allowed := range s.config.AllowedTypes {
		if strings.EqualFold(strings.TrimSpace(allowed), contentType) {
			return true
		}
	}

	return false
}

// defaultAllowedTypes is the fail-closed baseline used when the operator has
// not configured an explicit allowlist.
var defaultAllowedTypes = map[string]bool{
	"image/png":                true,
	"image/jpeg":               true,
	"image/gif":                true,
	"image/webp":               true,
	"image/avif":               true,
	"audio/mpeg":               true,
	"audio/ogg":                true,
	"video/mp4":                true,
	"video/webm":               true,
	"application/pdf":          true,
	"application/octet-stream": true,
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
		"base_url":      s.config.BaseURL,
	}
}
