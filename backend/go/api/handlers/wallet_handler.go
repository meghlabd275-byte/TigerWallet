package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"tigerwallet/backend/go/api/models"
	"tigerwallet/backend/go/api/services"
)

type WalletHandler struct {
	walletService *services.WalletService
}

func NewWalletHandler() *WalletHandler {
	return &WalletHandler{
		walletService: services.NewWalletService(),
	}
}

func (h *WalletHandler) CreateWallet(c *gin.Context) {
	var req struct {
		UserID        string `json:"user_id" binding:"required"`
		BlockchainID  string `json:"blockchain_id" binding:"required"`
		WalletType   string `json:"wallet_type"`
		DerivationPath string `json:"derivation_path"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INVALID_REQUEST", Message: err.Error()},
		})
		return
	}

	walletType := req.WalletType
	if walletType == "" {
		walletType = "user"
	}

	derivationPath := req.DerivationPath
	if derivationPath == "" {
		derivationPath = "m/44'/60'/0'/0/0"
	}

	wallet, err := h.walletService.Create(c.Request.Context(), req.UserID, req.BlockchainID, walletType, derivationPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INTERNAL_ERROR", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Data:    wallet,
	})
}

func (h *WalletHandler) GetWallets(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "UNAUTHORIZED", Message: "User not authenticated"},
		})
		return
	}

	wallets, err := h.walletService.GetByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INTERNAL_ERROR", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    wallets,
	})
}

func (h *WalletHandler) GetWallet(c *gin.Context) {
	walletID := c.Param("id")

	wallet, err := h.walletService.GetByID(c.Request.Context(), walletID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "NOT_FOUND", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    wallet,
	})
}

func (h *WalletHandler) DeleteWallet(c *gin.Context) {
	walletID := c.Param("id")

	err := h.walletService.Delete(c.Request.Context(), walletID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INTERNAL_ERROR", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
	})
}

func (h *WalletHandler) ImportWallet(c *gin.Context) {
	var req struct {
		UserID        string `json:"user_id" binding:"required"`
		BlockchainID  string `json:"blockchain_id" binding:"required"`
		PrivateKey   string `json:"private_key" binding:"required"`
		WalletType   string `json:"wallet_type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INVALID_REQUEST", Message: err.Error()},
		})
		return
	}

	walletType := req.WalletType
	if walletType == "" {
		walletType = "user"
	}

	wallet, err := h.walletService.ImportWallet(c.Request.Context(), req.UserID, req.BlockchainID, req.PrivateKey, walletType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INTERNAL_ERROR", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Data:    wallet,
	})
}

func (h *WalletHandler) ExportWallet(c *gin.Context) {
	walletID := c.Param("id")

	var req struct {
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INVALID_REQUEST", Message: err.Error()},
		})
		return
	}

	privateKey, err := h.walletService.ExportWallet(c.Request.Context(), walletID, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INTERNAL_ERROR", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    gin.H{"private_key": privateKey},
	})
}
