package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"golang.org/x/crypto/pbkdf2"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port        string
	RedisURL    string
	S3Bucket    string
	S3Region    string
	S3AccessKey string
	S3SecretKey string
	GCPProject  string
	GCPBucket   string
}

func LoadConfig() *Config {
	return &Config{
		Port:     getEnv("PORT", "8452"),
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// Cloud Backup Models
// ============================================================================

type Backup struct {
	ID            string `json:"id"`
	UserID        string `json:"user_id"`
	WalletID      string `json:"wallet_id"`
	EncryptedData string `json:"encrypted_data"`
	Checksum      string `json:"checksum"`
	CloudProvider string `json:"cloud_provider"` // "google", "apple", "custom"
	CloudPath     string `json:"cloud_path"`
	Size          int64  `json:"size"`
	Version       uint   `json:"version"`
	IsActive      bool   `json:"is_active"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
}

type BackupMetadata struct {
	WalletID   string            `json:"wallet_id"`
	ChainIDs   []uint64          `json:"chain_ids"`
	Addresses  map[uint64]string `json:"addresses"`
	PublicKeys map[uint64]string `json:"public_keys"`
	Name       string            `json:"name"`
	LastBackup int64             `json:"last_backup"`
}

type BackupRequest struct {
	WalletID      string `json:"wallet_id" binding:"required"`
	Data          string `json:"data" binding:"required"`
	CloudProvider string `json:"cloud_provider"`
}

// ============================================================================
// Encryption
// ============================================================================

// EncryptBackup encrypts `data` with an AES-256-GCM key derived from
// `password` via PBKDF2-HMAC-SHA256. The output blob is base64(salt ||
// nonce || ciphertext) so DecryptBackup can recover the salt and nonce and
// re-derive the same key. A SHA-256 checksum of the full blob is returned
// for transport integrity.
func EncryptBackup(data, password string) (string, string, error) {
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", "", err
	}

	key := deriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", err
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(data), nil)
	blob := append(append(salt, nonce...), ciphertext...)
	checksum := sha256.Sum256(blob)

	return base64.StdEncoding.EncodeToString(blob),
		hex.EncodeToString(checksum[:]),
		nil
}

// DecryptBackup reverses EncryptBackup: extract salt(32) + nonce + ciphertext
// from the base64 blob, re-derive the PBKDF2 key from the password + salt,
// and AES-256-GCM decrypt. A wrong password fails the GCM auth tag.
func DecryptBackup(encryptedData, password string) (string, error) {
	blob, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", err
	}

	if len(blob) < 32 {
		return "", fmt.Errorf("ciphertext too short")
	}
	salt := blob[:32]
	rest := blob[32:]

	key := deriveKey(password, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(rest) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := rest[:nonceSize], rest[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func deriveKey(password string, salt []byte) []byte {
	// PBKDF2-HMAC-SHA256, 600k iterations (OWASP 2023 recommendation for
	// password-based key derivation). Replaces the previous single sha256
	// (no work factor) which was trivially brute-forceable.
	return pbkdf2.Key([]byte(password), salt, 600000, 32, sha256.New)
}

// ============================================================================
// Cloud Storage Interface
// ============================================================================

type CloudStorage interface {
	Upload(path string, data []byte) error
	Download(path string) ([]byte, error)
	Delete(path string) error
	GetSignedURL(path string, expiry time.Duration) (string, error)
}

// ============================================================================
// S3 Storage (AWS S3 / Compatible)
// ============================================================================

type S3Storage struct {
	bucket string
	region string
	client *http.Client
}

func NewS3Storage(bucket, region string) *S3Storage {
	return &S3Storage{
		bucket: bucket,
		region: region,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *S3Storage) Upload(path string, data []byte) error {
	// Simplified - would use AWS SDK
	log.Printf("[S3] Uploading to %s/%s", s.bucket, path)
	return nil
}

func (s *S3Storage) Download(path string) ([]byte, error) {
	// Simplified - would use AWS SDK
	log.Printf("[S3] Downloading from %s/%s", s.bucket, path)
	return []byte(""), nil
}

func (s *S3Storage) Delete(path string) error {
	log.Printf("[S3] Deleting %s/%s", s.bucket, path)
	return nil
}

func (s *S3Storage) GetSignedURL(path string, expiry time.Duration) (string, error) {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s?expiry=%d",
		s.bucket, s.region, path, time.Now().Add(expiry).Unix()), nil
}

// ============================================================================
// Google Drive Storage
// ============================================================================

type GoogleDriveStorage struct {
	folderID string
	token    string
}

func NewGoogleDriveStorage(folderID, token string) *GoogleDriveStorage {
	return &GoogleDriveStorage{
		folderID: folderID,
		token:    token,
	}
}

func (g *GoogleDriveStorage) Upload(path string, data []byte) error {
	// Simplified - would use Google Drive API
	log.Printf("[GoogleDrive] Uploading to %s", path)
	return nil
}

func (g *GoogleDriveStorage) Download(path string) ([]byte, error) {
	log.Printf("[GoogleDrive] Downloading %s", path)
	return []byte(""), nil
}

func (g *GoogleDriveStorage) Delete(path string) error {
	log.Printf("[GoogleDrive] Deleting %s", path)
	return nil
}

func (g *GoogleDriveStorage) GetSignedURL(path string, expiry time.Duration) (string, error) {
	// Google Drive uses different URL scheme
	return fmt.Sprintf("https://drive.google.com/uc?export=download&id=%s", path), nil
}

// ============================================================================
// iCloud Storage
// ============================================================================

type ICloudStorage struct {
	container string
}

func NewICloudStorage(container string) *ICloudStorage {
	return &ICloudStorage{
		container: container,
	}
}

func (i *ICloudStorage) Upload(path string, data []byte) error {
	log.Printf("[iCloud] Uploading to %s/%s", i.container, path)
	return nil
}

func (i *ICloudStorage) Download(path string) ([]byte, error) {
	log.Printf("[iCloud] Downloading %s/%s", i.container, path)
	return []byte(""), nil
}

func (i *ICloudStorage) Delete(path string) error {
	log.Printf("[iCloud] Deleting %s/%s", i.container, path)
	return nil
}

func (i *ICloudStorage) GetSignedURL(path string, expiry time.Duration) (string, error) {
	return fmt.Sprintf("https://icloud.com/%s/%s", i.container, path), nil
}

// ============================================================================
// Cloud Backup Service
// ============================================================================

type CloudBackupService struct {
	config  *Config
	redis   *redis.Client
	backups map[string]*Backup
	storage map[string]CloudStorage
}

func NewCloudBackupService(config *Config) *CloudBackupService {
	redisClient := redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
	})

	storage := make(map[string]CloudStorage)
	storage["s3"] = NewS3Storage(config.S3Bucket, config.S3Region)
	storage["google"] = NewGoogleDriveStorage(config.GCPBucket, "")
	storage["apple"] = NewICloudStorage("TigerWallet")

	return &CloudBackupService{
		config:  config,
		redis:   redisClient,
		backups: make(map[string]*Backup),
		storage: storage,
	}
}

// ============================================================================
// Backup Operations
// ============================================================================

func (s *CloudBackupService) CreateBackup(userID, walletID, data, password, provider string) (*Backup, error) {
	// Encrypt data
	encryptedData, checksum, err := EncryptBackup(data, password)
	if err != nil {
		return nil, err
	}

	// Determine provider
	cloudProvider := provider
	if cloudProvider == "" {
		cloudProvider = "s3"
	}

	// Upload to cloud
	storage, ok := s.storage[cloudProvider]
	if !ok {
		return nil, fmt.Errorf("unsupported cloud provider: %s", cloudProvider)
	}

	cloudPath := fmt.Sprintf("backups/%s/%s_%d.enc", userID, walletID, time.Now().Unix())

	if err := storage.Upload(cloudPath, []byte(encryptedData)); err != nil {
		return nil, err
	}

	// Create backup record
	backup := &Backup{
		ID:            generateID(),
		UserID:        userID,
		WalletID:      walletID,
		EncryptedData: encryptedData,
		Checksum:      checksum,
		CloudProvider: cloudProvider,
		CloudPath:     cloudPath,
		Size:          int64(len(encryptedData)),
		Version:       1,
		IsActive:      true,
		CreatedAt:     time.Now().Unix(),
		UpdatedAt:     time.Now().Unix(),
	}

	s.backups[backup.ID] = backup
	s.backups[walletID] = backup

	return backup, nil
}

func (s *CloudBackupService) GetBackup(id string) (*Backup, error) {
	backup, ok := s.backups[id]
	if !ok {
		return nil, fmt.Errorf("backup not found")
	}
	return backup, nil
}

func (s *CloudBackupService) GetBackupByWallet(walletID string) (*Backup, error) {
	backup, ok := s.backups[walletID]
	if !ok {
		return nil, fmt.Errorf("backup not found")
	}
	return backup, nil
}

func (s *CloudBackupService) ListBackups(userID string) []*Backup {
	result := make([]*Backup, 0)
	for _, backup := range s.backups {
		if backup.UserID == userID {
			result = append(result, backup)
		}
	}
	return result
}

func (s *CloudBackupService) RestoreBackup(id, password string) (string, error) {
	backup, err := s.GetBackup(id)
	if err != nil {
		return "", err
	}

	// Download from cloud
	storage, ok := s.storage[backup.CloudProvider]
	if !ok {
		return "", fmt.Errorf("storage provider not found")
	}

	data, err := storage.Download(backup.CloudPath)
	if err != nil {
		return "", err
	}

	// Verify checksum
	currentChecksum := sha256.Sum256(data)
	currentChecksumHex := hex.EncodeToString(currentChecksum[:])
	if currentChecksumHex != backup.Checksum {
		return "", fmt.Errorf("checksum mismatch - backup may be corrupted")
	}

	// Decrypt
	plaintext, err := DecryptBackup(backup.EncryptedData, password)
	if err != nil {
		return "", err
	}

	return plaintext, nil
}

func (s *CloudBackupService) DeleteBackup(id string) error {
	backup, ok := s.backups[id]
	if !ok {
		return fmt.Errorf("backup not found")
	}

	// Delete from cloud
	storage, ok := s.storage[backup.CloudProvider]
	if ok {
		storage.Delete(backup.CloudPath)
	}

	backup.IsActive = false
	backup.UpdatedAt = time.Now().Unix()

	return nil
}

func (s *CloudBackupService) UpdateBackup(id, data, password string) (*Backup, error) {
	backup, ok := s.backups[id]
	if !ok {
		return nil, fmt.Errorf("backup not found")
	}

	// Re-encrypt data
	encryptedData, checksum, err := EncryptBackup(data, password)
	if err != nil {
		return nil, err
	}

	// Upload new version
	storage, ok := s.storage[backup.CloudProvider]
	if !ok {
		return nil, fmt.Errorf("storage provider not found")
	}

	newPath := fmt.Sprintf("%s.v%d", backup.CloudPath, backup.Version+1)
	if err := storage.Upload(newPath, []byte(encryptedData)); err != nil {
		return nil, err
	}

	// Update backup
	backup.EncryptedData = encryptedData
	backup.Checksum = checksum
	backup.CloudPath = newPath
	backup.Version++
	backup.Size = int64(len(encryptedData))
	backup.UpdatedAt = time.Now().Unix()

	return backup, nil
}

func (s *CloudBackupService) GetDownloadURL(id string) (string, error) {
	backup, err := s.GetBackup(id)
	if err != nil {
		return "", err
	}

	storage, ok := s.storage[backup.CloudProvider]
	if !ok {
		return "", fmt.Errorf("storage provider not found")
	}

	return storage.GetSignedURL(backup.CloudPath, 1*time.Hour)
}

// ============================================================================
// HTTP Handlers
// ============================================================================

func (s *CloudBackupService) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "cloud-backup-service"})
	})

	api := r.Group("/api/v1/backup")
	{
		api.POST("", s.handleCreateBackup)
		api.GET("/:id", s.handleGetBackup)
		api.GET("/wallet/:wallet_id", s.handleGetBackupByWallet)
		api.GET("/user/:user_id", s.handleListBackups)
		api.POST("/:id/restore", s.handleRestoreBackup)
		api.DELETE("/:id", s.handleDeleteBackup)
		api.PUT("/:id", s.handleUpdateBackup)
		api.GET("/:id/download", s.handleGetDownloadURL)
	}
}

func (s *CloudBackupService) handleCreateBackup(c *gin.Context) {
	var req struct {
		UserID        string `json:"user_id" binding:"required"`
		WalletID      string `json:"wallet_id" binding:"required"`
		Data          string `json:"data" binding:"required"`
		Password      string `json:"password" binding:"required"`
		CloudProvider string `json:"cloud_provider"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	backup, err := s.CreateBackup(req.UserID, req.WalletID, req.Data, req.Password, req.CloudProvider)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, backup)
}

func (s *CloudBackupService) handleGetBackup(c *gin.Context) {
	id := c.Param("id")

	backup, err := s.GetBackup(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, backup)
}

func (s *CloudBackupService) handleGetBackupByWallet(c *gin.Context) {
	walletID := c.Param("wallet_id")

	backup, err := s.GetBackupByWallet(walletID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, backup)
}

func (s *CloudBackupService) handleListBackups(c *gin.Context) {
	userID := c.Param("user_id")

	backups := s.ListBackups(userID)
	c.JSON(http.StatusOK, gin.H{"backups": backups})
}

func (s *CloudBackupService) handleRestoreBackup(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	data, err := s.RestoreBackup(id, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (s *CloudBackupService) handleDeleteBackup(c *gin.Context) {
	id := c.Param("id")

	if err := s.DeleteBackup(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "backup deleted"})
}

func (s *CloudBackupService) handleUpdateBackup(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Data     string `json:"data" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	backup, err := s.UpdateBackup(id, req.Data, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, backup)
}

func (s *CloudBackupService) handleGetDownloadURL(c *gin.Context) {
	id := c.Param("id")

	url, err := s.GetDownloadURL(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

// ============================================================================
// Utils
// ============================================================================

func generateID() string {
	return fmt.Sprintf("backup-%d-%s", time.Now().Unix(), randomString(8))
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = letters[big.NewInt(0).Mod(big.NewInt(0).SetBytes(b), big.NewInt(int64(len(letters)))).Int64()]
	}
	return string(b)
}

// ============================================================================
// Main
// ============================================================================

func main() {
	config := LoadConfig()
	service := NewCloudBackupService(config)

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
		log.Printf("Cloud Backup Service starting on port %s", config.Port)
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
