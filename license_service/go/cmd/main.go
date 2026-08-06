package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func main() {
	cfg := loadConfig()

	router := gin.Default()
	router.Use(corsMiddleware())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "tiger-license"})
	})

	api := router.Group("/api/v1")
	{
		api.POST("/licenses/generate", generateLicenseHandler)
		api.POST("/licenses/validate", validateLicenseHandler)
		api.POST("/licenses/revoke", revokeLicenseHandler)
		api.GET("/licenses/:license_id", getLicenseHandler)
		api.GET("/tenants/:tenant_id/license", getTenantLicenseHandler)

		// Super Admin
		superAdmin := api.Group("/super-admin")
		{
			superAdmin.GET("/licenses", listLicensesHandler)
			superAdmin.POST("/licenses/:license_id/suspend", suspendLicenseHandler)
		}
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: router,
	}

	go func() {
		log.Printf("License service starting on port %s", cfg.Port)
		srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}

type Config struct {
	Port        string
	PrivateKey  string
	PublicKey   string
}

func loadConfig() *Config {
	privateKey, _ := generateRSAKeyPair()
	return &Config{
		Port:        getEnv("LICENSE_PORT", "9008"),
		PrivateKey: privateKey,
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

type License struct {
	ID              uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	LicenseKey    string    `json:"license_key"`
	Product        string    `json:"product"` // master_wallet, user_wallet, bots, project_party, all
	Plan           string    `json:"plan"` // free, basic, pro, enterprise
	Status         string    `json:"status"` // active, suspended, expired
	ValidFrom     time.Time `json:"valid_from"`
	ValidUntil    time.Time `json:"valid_until"`
	MaxUsers      int       `json:"max_users"`
	MaxWallets   int       `json:"max_wallets"`
	MaxBots      int       `json:"max_bots"`
	Features     []string  `json:"features"`
	HardwareID   string    `json:"hardware_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func generateLicenseHandler(c *gin.Context) {
	var req struct {
		TenantID   string   `json:"tenant_id" binding:"required"`
		Product   string   `json:"product" binding:"required"`
		Plan      string   `json:"plan" binding:"required"`
		Duration  int      `json:"duration"` // days
		MaxUsers  int      `json:"max_users"`
		MaxWallets int     `json:"max_wallets"`
		MaxBots   int      `json:"max_bots"`
		Features  []string `json:"features"`
	}
	c.ShouldBindJSON(&req)

	tenantID, _ := uuid.Parse(req.TenantID)
	
	// Generate license key
	licenseKey := generateLicenseKey(tenantID.String(), req.Product, req.Plan)
	
	// Generate hardware ID
	hardwareID := generateHardwareID()

	duration := 365 // default 1 year
	if req.Duration > 0 {
		duration = req.Duration
	}

	license := map[string]interface{}{
		"id":           uuid.New().String(),
		"tenant_id":    req.TenantID,
		"license_key":  licenseKey,
		"product":      req.Product,
		"plan":         req.Plan,
		"status":        "active",
		"valid_from":   time.Now().Unix(),
		"valid_until":   time.Now().AddDate(0, 0, duration).Unix(),
		"max_users":    req.MaxUsers,
		"max_wallets": req.MaxWallets,
		"max_bots":    req.MaxBots,
		"features":     req.Features,
		"hardware_id": hardwareID,
		"created_at":  time.Now().Unix(),
	}

	c.JSON(http.StatusCreated, gin.H{
		"license":  license,
		"message": "License generated successfully",
	})
}

func validateLicenseHandler(c *gin.Context) {
	var req struct {
		LicenseKey string `json:"license_key" binding:"required"`
		HardwareID string `json:"hardware_id" binding:"required"`
		Product    string `json:"product" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	// In production, validate against database
	// For now, return mock validation
	now := time.Now()
	
	license := map[string]interface{}{
		"valid":        true,
		"license_key":  req.LicenseKey,
		"product":      req.Product,
		"plan":         "pro",
		"valid_until":   now.Add(365 * 24 * time.Hour).Unix(),
		"max_users":    100,
		"max_wallets": 500,
		"max_bots":     50,
		"features":     []string{"wallet.create", "bot.create", "token.list"},
	}

	c.JSON(http.StatusOK, license)
}

func revokeLicenseHandler(c *gin.Context) {
	var req struct {
		LicenseKey string `json:"license_key" binding:"required"`
		Reason     string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	c.JSON(http.StatusOK, gin.H{
		"license_key": req.LicenseKey,
		"status":      "revoked",
		"message":     "License revoked",
	})
}

func getLicenseHandler(c *gin.Context) {
	licenseID := c.Param("license_id")

	license := map[string]interface{}{
		"id":           licenseID,
		"tenant_id":    uuid.New().String(),
		"license_key":  "LW-XXXX-XXXX-XXXX",
		"product":      "all",
		"plan":         "pro",
		"status":       "active",
		"valid_from":   time.Now().Unix(),
		"valid_until":   time.Now().Add(365 * 24 * time.Hour).Unix(),
	}

	c.JSON(http.StatusOK, license)
}

func getTenantLicenseHandler(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	license := map[string]interface{}{
		"tenant_id":   tenantID,
		"license_key": "LW-XXXX-XXXX-XXXX",
		"product":     "all",
		"plan":        "enterprise",
		"status":      "active",
	}

	c.JSON(http.StatusOK, license)
}

func listLicensesHandler(c *gin.Context) {
	licenses := []map[string]interface{}{
		{
			"id":          uuid.New().String(),
			"tenant_id":   uuid.New().String(),
			"license_key": "LW-XXXX-XXXX-XXXX",
			"product":     "all",
			"plan":        "enterprise",
			"status":      "active",
		},
	}

	c.JSON(http.StatusOK, gin.H{"licenses": licenses})
}

func suspendLicenseHandler(c *gin.Context) {
	licenseID := c.Param("license_id")

	c.JSON(http.StatusOK, gin.H{
		"license_id": licenseID,
		"status":     "suspended",
		"message":    "License suspended",
	})
}

func generateLicenseKey(tenantID, product, plan string) string {
	data := fmt.Sprintf("%s-%s-%s-%d", tenantID, product, plan, time.Now().Unix())
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("LW-%s", hex.EncodeToString(hash[:])[:16])
}

func generateHardwareID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateRSAKeyPair() (string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}

	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return string(privatePEM), nil
}
