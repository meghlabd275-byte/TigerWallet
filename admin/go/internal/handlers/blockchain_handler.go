/**
 * TigerWallet Admin - Blockchain Handler
 * CRUD for blockchain registry (GORM + PostgreSQL).
 * Matches the admin/web frontend contract: GET /api/v1/blockchains returns a
 * bare JSON array; create/update accept camelCase field names.
 */

package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tigerwallet/admin/internal/models"
	"github.com/tigerwallet/admin/pkg/database"
)

// BlockchainHandler handles blockchain registry requests
type BlockchainHandler struct {
	db *database.PostgresDB
}

// NewBlockchainHandler creates a new blockchain handler
func NewBlockchainHandler(db *database.PostgresDB) *BlockchainHandler {
	return &BlockchainHandler{db: db}
}

// ListBlockchains returns all blockchains as a bare JSON array (the admin
// frontend's getBlockchains expects the response body to be an array, not an
// envelope object).
func (h *BlockchainHandler) ListBlockchains(c *gin.Context) {
	var blockchains []models.Blockchain
	if err := h.db.Find(&blockchains).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch blockchains"})
		return
	}
	c.JSON(http.StatusOK, blockchains)
}

// GetBlockchain returns a single blockchain by id.
func (h *BlockchainHandler) GetBlockchain(c *gin.Context) {
	id := c.Param("id")
	var blockchain models.Blockchain
	if err := h.db.First(&blockchain, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blockchain not found"})
		return
	}
	c.JSON(http.StatusOK, blockchain)
}

// createBlockchainRequest is permissive: it accepts both snake_case and
// camelCase keys (the admin frontend sends camelCase).
type createBlockchainRequest struct {
	Name          string `json:"name"`
	Symbol        string `json:"symbol"`
	ChainID       int64  `json:"chainId"`
	ChainIDSnake  int64  `json:"chain_id"`
	Type          string `json:"type"`
	RPCURL        string `json:"rpcUrl"`
	RPCURLSnake   string `json:"rpc_url"`
	WSRPCURL      string `json:"wsRpcUrl"`
	WSRPCURLSnake string `json:"ws_rpc_url"`
	ExplorerURL   string `json:"explorerUrl"`
	ExplorerURLSnake string `json:"explorer_url"`
	ExplorerAPI   string `json:"explorerApi"`
	ExplorerAPISnake string `json:"explorer_api"`
	NativeToken   string `json:"nativeToken"`
	NativeTokenSnake string `json:"native_token"`
	Decimals      int    `json:"decimals"`
	LogoURL       string `json:"logoUrl"`
	LogoURLSnake  string `json:"logo_url"`
	IsTestnet     bool   `json:"isTestnet"`
	IsActive      bool   `json:"isActive"`
	Status        string `json:"status"`
	GasToken      string `json:"gasToken"`
}

// CreateBlockchain creates a new blockchain record.
func (h *BlockchainHandler) CreateBlockchain(c *gin.Context) {
	adminID := c.GetUint("admin_id")

	var req createBlockchainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	name := firstNonEmpty(req.Name)
	symbol := firstNonEmpty(req.Symbol)
	if name == "" || symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and symbol are required"})
		return
	}

	chainID := req.ChainID
	if chainID == 0 {
		chainID = req.ChainIDSnake
	}
	if chainID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chainId is required"})
		return
	}

	status := req.Status
	if status == "" {
		if req.IsActive {
			status = "active"
		} else {
			status = "inactive"
		}
	}

	blockchain := models.Blockchain{
		Name:         name,
		Symbol:       symbol,
		ChainID:      chainID,
		Type:         req.Type,
		RPCURL:       firstNonEmpty(req.RPCURL, req.RPCURLSnake),
		WSRPCURL:     firstNonEmpty(req.WSRPCURL, req.WSRPCURLSnake),
		ExplorerURL:  firstNonEmpty(req.ExplorerURL, req.ExplorerURLSnake),
		ExplorerAPI:  firstNonEmpty(req.ExplorerAPI, req.ExplorerAPISnake),
		NativeToken:  firstNonEmpty(req.NativeToken, req.NativeTokenSnake),
		Decimals:     req.Decimals,
		LogoURL:      firstNonEmpty(req.LogoURL, req.LogoURLSnake),
		IsTestnet:    req.IsTestnet,
		IsActive:     req.IsActive,
		Status:       status,
		GasToken:     req.GasToken,
		CreatedBy:    adminID,
	}

	if err := h.db.Create(&blockchain).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create blockchain"})
		return
	}

	logAdminActivity(h.db, adminID, "create_blockchain", "blockchain", strconv.FormatUint(uint64(blockchain.ID), 10), "Created blockchain: "+blockchain.Name, c.ClientIP(), c.Request.UserAgent())
	c.JSON(http.StatusCreated, blockchain)
}

// updateBlockchainRequest is permissive on field naming and accepts partial
// updates (the admin frontend sends only the fields it wants to change).
type updateBlockchainRequest struct {
	Name             *string `json:"name"`
	Symbol           *string `json:"symbol"`
	ChainID          *int64  `json:"chainId"`
	ChainIDSnake     *int64  `json:"chain_id"`
	Type             *string `json:"type"`
	RPCURL           *string `json:"rpcUrl"`
	RPCURLSnake      *string `json:"rpc_url"`
	WSRPCURL         *string `json:"wsRpcUrl"`
	WSRPCURLSnake    *string `json:"ws_rpc_url"`
	ExplorerURL      *string `json:"explorerUrl"`
	ExplorerURLSnake *string `json:"explorer_url"`
	ExplorerAPI      *string `json:"explorerApi"`
	ExplorerAPISnake *string `json:"explorer_api"`
	NativeToken      *string `json:"nativeToken"`
	NativeTokenSnake *string `json:"native_token"`
	Decimals         *int    `json:"decimals"`
	LogoURL          *string `json:"logoUrl"`
	LogoURLSnake     *string `json:"logo_url"`
	IsTestnet        *bool   `json:"isTestnet"`
	IsActive         *bool   `json:"isActive"`
	Status           *string `json:"status"`
	GasToken         *string `json:"gasToken"`
}

// UpdateBlockchain updates an existing blockchain.
func (h *BlockchainHandler) UpdateBlockchain(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	id := c.Param("id")

	var blockchain models.Blockchain
	if err := h.db.First(&blockchain, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blockchain not found"})
		return
	}

	var req updateBlockchainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Symbol != nil {
		updates["symbol"] = *req.Symbol
	}
	if req.ChainID != nil {
		updates["chain_id"] = *req.ChainID
	} else if req.ChainIDSnake != nil {
		updates["chain_id"] = *req.ChainIDSnake
	}
	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.RPCURL != nil {
		updates["rpc_url"] = *req.RPCURL
	} else if req.RPCURLSnake != nil {
		updates["rpc_url"] = *req.RPCURLSnake
	}
	if req.WSRPCURL != nil {
		updates["ws_rpc_url"] = *req.WSRPCURL
	} else if req.WSRPCURLSnake != nil {
		updates["ws_rpc_url"] = *req.WSRPCURLSnake
	}
	if req.ExplorerURL != nil {
		updates["explorer_url"] = *req.ExplorerURL
	} else if req.ExplorerURLSnake != nil {
		updates["explorer_url"] = *req.ExplorerURLSnake
	}
	if req.ExplorerAPI != nil {
		updates["explorer_api"] = *req.ExplorerAPI
	} else if req.ExplorerAPISnake != nil {
		updates["explorer_api"] = *req.ExplorerAPISnake
	}
	if req.NativeToken != nil {
		updates["native_token"] = *req.NativeToken
	} else if req.NativeTokenSnake != nil {
		updates["native_token"] = *req.NativeTokenSnake
	}
	if req.Decimals != nil {
		updates["decimals"] = *req.Decimals
	}
	if req.LogoURL != nil {
		updates["logo_url"] = *req.LogoURL
	} else if req.LogoURLSnake != nil {
		updates["logo_url"] = *req.LogoURLSnake
	}
	if req.IsTestnet != nil {
		updates["is_testnet"] = *req.IsTestnet
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.GasToken != nil {
		updates["gas_token"] = *req.GasToken
	}

	if len(updates) > 0 {
		if err := h.db.Model(&models.Blockchain{}).Where("id = ?", blockchain.ID).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update blockchain"})
			return
		}
	}

	if err := h.db.First(&blockchain, blockchain.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reload blockchain"})
		return
	}

	logAdminActivity(h.db, adminID, "update_blockchain", "blockchain", strconv.FormatUint(uint64(blockchain.ID), 10), "Updated blockchain: "+blockchain.Name, c.ClientIP(), c.Request.UserAgent())
	c.JSON(http.StatusOK, blockchain)
}

// DeleteBlockchain deletes a blockchain record.
func (h *BlockchainHandler) DeleteBlockchain(c *gin.Context) {
	adminID := c.GetUint("admin_id")
	id := c.Param("id")

	var blockchain models.Blockchain
	if err := h.db.First(&blockchain, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blockchain not found"})
		return
	}

	if err := h.db.Delete(&models.Blockchain{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete blockchain"})
		return
	}

	logAdminActivity(h.db, adminID, "delete_blockchain", "blockchain", id, "Deleted blockchain: "+blockchain.Name, c.ClientIP(), c.Request.UserAgent())
	c.JSON(http.StatusOK, gin.H{"message": "Blockchain deleted"})
}

// TestBlockchainRpc performs a lightweight connectivity check against the
// blockchain's configured RPC URL and reports success + latency. The admin
// frontend calls POST /api/v1/blockchains/:id/test-rpc and expects
// {success, latency}.
func (h *BlockchainHandler) TestBlockchainRpc(c *gin.Context) {
	id := c.Param("id")
	var blockchain models.Blockchain
	if err := h.db.First(&blockchain, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blockchain not found"})
		return
	}

	rpcURL := blockchain.RPCURL
	if rpcURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "latency": 0, "error": "No RPC URL configured"})
		return
	}

	start := time.Now()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(rpcURL, "application/json", nil)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "latency": latency, "error": err.Error()})
		return
	}
	defer resp.Body.Close()
	c.JSON(http.StatusOK, gin.H{"success": true, "latency": latency})
}

// AdminActivityListResponse is the response for the /activities audit log.
type AdminActivityListResponse struct {
	Data     []models.AdminActivity `json:"data"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"pageSize"`
}

// ListActivities returns admin action audit logs from the admin_activities
// table. The admin frontend's getAuditLogs hits /api/v1/audit-logs (handled by
// the super admin handler using the AuditLog model); this /activities endpoint
// exposes the AdminActivity records recorded by logAdminActivity across all
// handlers, giving full coverage of admin actions.
func (h *BlockchainHandler) ListActivities(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if pageSize < 1 {
		pageSize = 50
	}
	action := c.Query("action")
	resource := c.Query("resource")
	adminID := c.Query("admin_id")

	var total int64
	var activities []models.AdminActivity

	qb := h.db.Model(&models.AdminActivity{})
	if action != "" {
		qb = qb.Where("action = ?", action)
	}
	if resource != "" {
		qb = qb.Where("resource = ?", resource)
	}
	if adminID != "" {
		qb = qb.Where("admin_id = ?", adminID)
	}

	qb.Count(&total)

	offset := (page - 1) * pageSize
	fqb := h.db.Model(&models.AdminActivity{})
	if action != "" {
		fqb = fqb.Where("action = ?", action)
	}
	if resource != "" {
		fqb = fqb.Where("resource = ?", resource)
	}
	if adminID != "" {
		fqb = fqb.Where("admin_id = ?", adminID)
	}
	if err := fqb.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&activities).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch activities"})
		return
	}

	c.JSON(http.StatusOK, AdminActivityListResponse{
		Data:     activities,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}
