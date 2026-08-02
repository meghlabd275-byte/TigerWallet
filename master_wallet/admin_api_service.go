package master_wallet

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// API Types
// ============================================================================

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Code    int         `json:"code"`
}

// ============================================================================
// Admin API Service
// ============================================================================

// AdminAPIService provides REST API endpoints for admin management
type AdminAPIService struct {
	mu            sync.RWMutex
	masterService *MasterWalletService
	tigerService  *TigerWalletService
	brandService  *CustomBrandingService
}

var (
	adminAPIService     *AdminAPIService
	adminAPIServiceOnce sync.Once
)

// GetAdminAPIService returns the singleton admin API service
func GetAdminAPIService() *AdminAPIService {
	adminAPIServiceOnce.Do(func() {
		adminAPIService = &AdminAPIService{
			masterService: GetMasterWalletService(),
			tigerService:  GetTigerWalletService(),
			brandService:  GetCustomBrandingService(),
		}
	})
	return adminAPIService
}

// ============================================================================
// Router Setup
// ============================================================================

// SetupRouter sets up the Gin router with all API routes
func (s *AdminAPIService) SetupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Health check
	router.GET("/health", s.healthCheck)

	// API v1
	v1 := router.Group("/api/v1")
	{
		// Master Wallet endpoints
		v1.GET("/master-wallets", s.getMasterWallets)
		v1.GET("/master-wallets/:id", s.getMasterWallet)
		v1.POST("/master-wallets", s.createMasterWallet)
		v1.PUT("/master-wallets/:id", s.updateMasterWallet)
		v1.DELETE("/master-wallets/:id", s.deleteMasterWallet)
		v1.GET("/master-wallets/:id/statistics", s.getMasterWalletStatistics)

		// Network endpoints
		v1.GET("/networks", s.getAllNetworks)
		v1.GET("/networks/:id", s.getNetwork)
		v1.GET("/networks/type/:type", s.getNetworksByType)
		v1.POST("/networks", s.addNetwork)
		v1.PUT("/networks/:id", s.updateNetwork)
		v1.DELETE("/networks/:id", s.deleteNetwork)

		// Token endpoints
		v1.GET("/tokens", s.getAllTokens)
		v1.GET("/tokens/:id", s.getToken)
		v1.GET("/tokens/chain/:chainID", s.getTokensByChain)
		v1.POST("/tokens", s.addToken)
		v1.PUT("/tokens/:id", s.updateToken)
		v1.DELETE("/tokens/:id", s.deleteToken)
		v1.GET("/tokens/stablecoins", s.getStableCoins)
		v1.GET("/tokens/verified", s.getVerifiedTokens)

		// User Wallet endpoints
		v1.GET("/user-wallets", s.getUserWallets)
		v1.GET("/user-wallets/:id", s.getUserWallet)
		v1.POST("/user-wallets", s.createUserWallet)
		v1.PUT("/user-wallets/:id", s.updateUserWallet)
		v1.DELETE("/user-wallets/:id", s.deleteUserWallet)
		v1.GET("/user-wallets/:id/balance", s.getUserWalletBalance)
		v1.GET("/user-wallets/:id/transactions", s.getUserWalletTransactions)

		// Custom Branding endpoints
		v1.GET("/brands", s.getBrands)
		v1.GET("/brands/:id", s.getBrand)
		v1.POST("/brands", s.createBrand)
		v1.PUT("/brands/:id", s.updateBrand)
		v1.DELETE("/brands/:id", s.deleteBrand)
		v1.GET("/brands/:id/users", s.getBrandUsers)
		v1.GET("/brands/:id/admins", s.getBrandAdmins)

		// Statistics
		v1.GET("/statistics/global", s.getGlobalStatistics)
		v1.GET("/statistics/wallets", s.getWalletStatistics)
	}

	return router
}

// ============================================================================
// Health Check
// ============================================================================

func (s *AdminAPIService) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"status":    "healthy",
			"networks":  s.masterService.networkRegistry.GetSupportedChains(),
			"tokens":    s.masterService.tokenRegistry.GetTokenCount(),
			"timestamp": time.Now().Unix(),
		},
		Code: 200,
	})
}

// ============================================================================
// Master Wallet Handlers
// ============================================================================

func (s *AdminAPIService) getMasterWallets(c *gin.Context) {
	wallets := s.masterService.GetAllMasterWallets()
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    wallets,
		Code:    200,
	})
}

func (s *AdminAPIService) getMasterWallet(c *gin.Context) {
	id := c.Param("id")
	wallet, ok := s.masterService.GetMasterWallet(id)
	if !ok {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "Master wallet not found",
			Code:    404,
		})
		return
	}
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    wallet,
		Code:    200,
	})
}

func (s *AdminAPIService) createMasterWallet(c *gin.Context) {
	var req struct {
		Name        string          `json:"name" binding:"required"`
		Description string          `json:"description"`
		Branding    *CustomBranding `json:"branding"`
		AdminIDs    []string        `json:"admin_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    400,
		})
		return
	}

	wallet, err := s.masterService.CreateMasterWallet(req.Name, req.Description, req.Branding, req.AdminIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    500,
		})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    wallet,
		Code:    201,
	})
}

func (s *AdminAPIService) updateMasterWallet(c *gin.Context) {
	id := c.Param("id")
	var updates map[string]interface{}

	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    400,
		})
		return
	}

	wallet, err := s.masterService.UpdateMasterWallet(id, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    500,
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    wallet,
		Code:    200,
	})
}

func (s *AdminAPIService) deleteMasterWallet(c *gin.Context) {
	id := c.Param("id")

	if err := s.masterService.DeleteMasterWallet(id); err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    500,
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    "Master wallet deleted successfully",
		Code:    200,
	})
}

func (s *AdminAPIService) getMasterWalletStatistics(c *gin.Context) {
	id := c.Param("id")
	stats, err := s.masterService.GetStatistics(id)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    404,
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    stats,
		Code:    200,
	})
}

// ============================================================================
// Network Handlers
// ============================================================================

func (s *AdminAPIService) getAllNetworks(c *gin.Context) {
	networks := s.masterService.networkRegistry.GetAllNetworks()
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    networks,
		Code:    200,
	})
}

func (s *AdminAPIService) getNetwork(c *gin.Context) {
	id := c.Param("id")
	network, ok := s.masterService.networkRegistry.GetNetwork(id)
	if !ok {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "Network not found",
			Code:    404,
		})
		return
	}
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    network,
		Code:    200,
	})
}

func (s *AdminAPIService) getNetworksByType(c *gin.Context) {
	networkType := c.Param("type")
	networks := s.masterService.networkRegistry.GetNetworksByType(BlockchainType(networkType))
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    networks,
		Code:    200,
	})
}

func (s *AdminAPIService) addNetwork(c *gin.Context) {
	body, _ := io.ReadAll(c.Request.Body)
	var req NetworkAddRequest
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    400,
		})
		return
	}

	// Get master wallet ID from query or use default
	masterWalletID := c.Query("master_wallet_id")
	if masterWalletID == "" {
		masterWallet := s.masterService.GetMasterWalletByType("tiger")
		if masterWallet != nil {
			masterWalletID = masterWallet.ID
		}
	}

	network, err := s.masterService.AddNetwork(masterWalletID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    500,
		})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    network,
		Code:    201,
	})
}

func (s *AdminAPIService) updateNetwork(c *gin.Context) {
	id := c.Param("id")
	var updates map[string]interface{}

	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    400,
		})
		return
	}

	network, ok := s.masterService.networkRegistry.GetNetwork(id)
	if !ok {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "Network not found",
			Code:    404,
		})
		return
	}

	// Apply updates
	if rpcURL, ok := updates["rpc_url"].(string); ok {
		network.RPCURL = rpcURL
	}
	if explorer, ok := updates["explorer"].(string); ok {
		network.Explorer = explorer
	}
	if wssURL, ok := updates["wss_url"].(string); ok {
		network.WSSURL = wssURL
	}

	if err := s.masterService.networkRegistry.UpdateNetwork(network); err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    500,
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    network,
		Code:    200,
	})
}

func (s *AdminAPIService) deleteNetwork(c *gin.Context) {
	id := c.Param("id")

	if err := s.masterService.networkRegistry.DeleteNetwork(id); err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    500,
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    "Network deleted successfully",
		Code:    200,
	})
}

// ============================================================================
// Token Handlers
// ============================================================================

func (s *AdminAPIService) getAllTokens(c *gin.Context) {
	tokens := s.masterService.tokenRegistry.GetAllTokens()
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    tokens,
		Code:    200,
	})
}

func (s *AdminAPIService) getToken(c *gin.Context) {
	id := c.Param("id")
	token, ok := s.masterService.tokenRegistry.GetToken(id)
	if !ok {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "Token not found",
			Code:    404,
		})
		return
	}
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    token,
		Code:    200,
	})
}

func (s *AdminAPIService) getTokensByChain(c *gin.Context) {
	chainID := c.Param("chainID")
	var chainIDInt int64
	fmt.Sscanf(chainID, "%d", &chainIDInt)

	tokens := s.masterService.tokenRegistry.GetTokensByChain(chainIDInt)
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    tokens,
		Code:    200,
	})
}

func (s *AdminAPIService) addToken(c *gin.Context) {
	body, _ := io.ReadAll(c.Request.Body)
	var req TokenAddRequest
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    400,
		})
		return
	}

	// Get master wallet ID from query or use default
	masterWalletID := c.Query("master_wallet_id")
	if masterWalletID == "" {
		masterWallet := s.masterService.GetMasterWalletByType("tiger")
		if masterWallet != nil {
			masterWalletID = masterWallet.ID
		}
	}

	token, err := s.masterService.AddToken(masterWalletID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    500,
		})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    token,
		Code:    201,
	})
}

func (s *AdminAPIService) updateToken(c *gin.Context) {
	id := c.Param("id")
	var updates map[string]interface{}

	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    400,
		})
		return
	}

	token, ok := s.masterService.tokenRegistry.GetToken(id)
	if !ok {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "Token not found",
			Code:    404,
		})
		return
	}

	// Apply updates
	if logoURL, ok := updates["logo_url"].(string); ok {
		token.LogoURL = logoURL
	}
	if website, ok := updates["website"].(string); ok {
		token.Website = website
	}
	if isVerified, ok := updates["is_verified"].(bool); ok {
		token.IsVerified = isVerified
	}
	if isStableCoin, ok := updates["is_stable_coin"].(bool); ok {
		token.IsStableCoin = isStableCoin
	}

	if err := s.masterService.tokenRegistry.UpdateToken(token); err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    500,
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    token,
		Code:    200,
	})
}

func (s *AdminAPIService) deleteToken(c *gin.Context) {
	id := c.Param("id")

	if err := s.masterService.tokenRegistry.DeleteToken(id); err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    500,
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    "Token deleted successfully",
		Code:    200,
	})
}

func (s *AdminAPIService) getStableCoins(c *gin.Context) {
	tokens := s.masterService.tokenRegistry.GetStableCoins()
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    tokens,
		Code:    200,
	})
}

func (s *AdminAPIService) getVerifiedTokens(c *gin.Context) {
	tokens := s.masterService.tokenRegistry.GetVerifiedTokens()
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    tokens,
		Code:    200,
	})
}

// ============================================================================
// User Wallet Handlers
// ============================================================================

func (s *AdminAPIService) getUserWallets(c *gin.Context) {
	masterWalletID := c.Query("master_wallet_id")
	userID := c.Query("user_id")

	var wallets []*Wallet
	if masterWalletID != "" {
		wallets = s.tigerService.GetWalletsByMasterWallet(masterWalletID)
	} else if userID != "" {
		wallets = s.tigerService.GetWalletByUser(userID)
	} else {
		// Return all wallets
		s.tigerService.mu.RLock()
		for _, wallet := range s.tigerService.wallets {
			wallets = append(wallets, wallet)
		}
		s.tigerService.mu.RUnlock()
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    wallets,
		Code:    200,
	})
}

func (s *AdminAPIService) getUserWallet(c *gin.Context) {
	id := c.Param("id")
	wallet, ok := s.tigerService.GetWallet(id)
	if !ok {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "User wallet not found",
			Code:    404,
		})
		return
	}
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    wallet,
		Code:    200,
	})
}

func (s *AdminAPIService) createUserWallet(c *gin.Context) {
	var req struct {
		UserID     string `json:"user_id" binding:"required"`
		Name       string `json:"name" binding:"required"`
		WalletType string `json:"wallet_type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    400,
		})
		return
	}

	walletType := req.WalletType
	if walletType == "" {
		walletType = "tiger"
	}

	wallet, err := s.tigerService.CreateWallet(req.UserID, req.Name, walletType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    500,
		})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    wallet,
		Code:    201,
	})
}

func (s *AdminAPIService) updateUserWallet(c *gin.Context) {
	id := c.Param("id")
	var updates map[string]interface{}

	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    400,
		})
		return
	}

	wallet, err := s.tigerService.UpdateWallet(id, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    500,
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    wallet,
		Code:    200,
	})
}

func (s *AdminAPIService) deleteUserWallet(c *gin.Context) {
	id := c.Param("id")

	if err := s.tigerService.DeleteWallet(id); err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    500,
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    "User wallet deleted successfully",
		Code:    200,
	})
}

func (s *AdminAPIService) getUserWalletBalance(c *gin.Context) {
	id := c.Param("id")
	balances := s.tigerService.GetAllBalances(id)
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    balances,
		Code:    200,
	})
}

func (s *AdminAPIService) getUserWalletTransactions(c *gin.Context) {
	id := c.Param("id")
	txs, err := s.tigerService.GetTransactions(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    500,
		})
		return
	}
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    txs,
		Code:    200,
	})
}

// ============================================================================
// Custom Branding Handlers
// ============================================================================

func (s *AdminAPIService) getBrands(c *gin.Context) {
	wallets := s.brandService.GetAllBrandWallets()
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    wallets,
		Code:    200,
	})
}

func (s *AdminAPIService) getBrand(c *gin.Context) {
	id := c.Param("id")
	wallet, ok := s.brandService.GetBrandWallet(id)
	if !ok {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "Brand wallet not found",
			Code:    404,
		})
		return
	}
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    wallet,
		Code:    200,
	})
}

func (s *AdminAPIService) createBrand(c *gin.Context) {
	var req struct {
		BrandName    string `json:"brand_name" binding:"required"`
		BrandLogo    string `json:"brand_logo"`
		BrandColor   string `json:"brand_color"`
		BrandTagline string `json:"brand_tagline"`
		SupportEmail string `json:"support_email"`
		WebsiteURL   string `json:"website_url"`
		AdminEmail   string `json:"admin_email" binding:"required"`
		AdminName    string `json:"admin_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    400,
		})
		return
	}

	brand, err := s.brandService.CreateBrandWallet(
		req.BrandName,
		req.BrandLogo,
		req.BrandColor,
		req.BrandTagline,
		req.SupportEmail,
		req.WebsiteURL,
		req.AdminEmail,
		req.AdminName,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    500,
		})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    brand,
		Code:    201,
	})
}

func (s *AdminAPIService) updateBrand(c *gin.Context) {
	id := c.Param("id")
	var updates map[string]interface{}

	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    400,
		})
		return
	}

	brand, err := s.brandService.UpdateBrandWallet(id, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    500,
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    brand,
		Code:    200,
	})
}

func (s *AdminAPIService) deleteBrand(c *gin.Context) {
	id := c.Param("id")

	if err := s.brandService.DeleteBrandWallet(id); err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    500,
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    "Brand wallet deleted successfully",
		Code:    200,
	})
}

func (s *AdminAPIService) getBrandUsers(c *gin.Context) {
	id := c.Param("id")
	users, err := s.brandService.GetBrandUsers(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    500,
		})
		return
	}
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    users,
		Code:    200,
	})
}

func (s *AdminAPIService) getBrandAdmins(c *gin.Context) {
	id := c.Param("id")
	admins, err := s.brandService.GetBrandAdmins(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
			Code:    500,
		})
		return
	}
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    admins,
		Code:    200,
	})
}

// ============================================================================
// Statistics Handlers
// ============================================================================

func (s *AdminAPIService) getGlobalStatistics(c *gin.Context) {
	stats := s.masterService.GetGlobalStatistics()
	stats["tiger_wallets"] = s.tigerService.GetWalletCount()
	stats["tiger_users"] = s.tigerService.GetUserCount()
	stats["tiger_volume"] = s.tigerService.GetTotalVolume()
	stats["brand_wallets"] = s.brandService.GetBrandCount()
	stats["brand_users"] = s.brandService.GetBrandUserCount()

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    stats,
		Code:    200,
	})
}

func (s *AdminAPIService) getWalletStatistics(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"total_wallets":   s.tigerService.GetWalletCount(),
			"total_users":     s.tigerService.GetUserCount(),
			"total_volume":    s.tigerService.GetTotalVolume(),
			"total_networks":  s.masterService.networkRegistry.GetSupportedChains(),
			"total_tokens":    s.masterService.tokenRegistry.GetTokenCount(),
			"active_chains":   s.masterService.networkRegistry.GetActiveChainCount(),
		},
		Code: 200,
	})
}

// Add time import
func init() {
	_ = time.Now()
	_ = strings.ToUpper
}
