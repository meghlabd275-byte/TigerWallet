package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"tigerwallet/backend/go/api/models"
	"tigerwallet/backend/go/api/services"
)

type SwapHandler struct {
	swapService *services.SwapService
}

func NewSwapHandler() *SwapHandler {
	return &SwapHandler{
		swapService: services.NewSwapService(),
	}
}

func (h *SwapHandler) GetSwapQuote(c *gin.Context) {
	var req models.SwapRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INVALID_REQUEST", Message: err.Error()},
		})
		return
	}

	quote, err := h.swapService.GetQuote(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "QUOTE_ERROR", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    quote,
	})
}

func (h *SwapHandler) ExecuteSwap(c *gin.Context) {
	var req struct {
		WalletID string `json:"wallet_id" binding:"required"`
		QuoteID  string `json:"quote_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INVALID_REQUEST", Message: err.Error()},
		})
		return
	}

	swap, err := h.swapService.ExecuteSwap(c.Request.Context(), req.WalletID, req.QuoteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "SWAP_ERROR", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Data:    swap,
	})
}

func (h *SwapHandler) GetSwaps(c *gin.Context) {
	walletID := c.Query("wallet_id")
	if walletID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INVALID_REQUEST", Message: "wallet_id required"},
		})
		return
	}

	swaps, err := h.swapService.GetSwapsByWalletID(c.Request.Context(), walletID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INTERNAL_ERROR", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    swaps,
	})
}

// Staking Handler
type StakingHandler struct {
	stakingService *services.StakingService
}

func NewStakingHandler() *StakingHandler {
	return &StakingHandler{
		stakingService: services.NewStakingService(),
	}
}

func (h *StakingHandler) GetStakingPools(c *gin.Context) {
	blockchainID := c.Query("blockchain_id")

	pools, err := h.stakingService.GetPools(c.Request.Context(), blockchainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INTERNAL_ERROR", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    pools,
	})
}

func (h *StakingHandler) Stake(c *gin.Context) {
	var req struct {
		WalletID string `json:"wallet_id" binding:"required"`
		PoolID  string `json:"pool_id" binding:"required"`
		Amount  string `json:"amount" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INVALID_REQUEST", Message: err.Error()},
		})
		return
	}

	position, err := h.stakingService.Stake(c.Request.Context(), req.WalletID, req.PoolID, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "STAKE_ERROR", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Data:    position,
	})
}

func (h *StakingHandler) Unstake(c *gin.Context) {
	var req struct {
		PositionID string `json:"position_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INVALID_REQUEST", Message: err.Error()},
		})
		return
	}

	err := h.stakingService.Unstake(c.Request.Context(), req.PositionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "UNSTAKE_ERROR", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
	})
}

func (h *StakingHandler) ClaimRewards(c *gin.Context) {
	var req struct {
		PositionID string `json:"position_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INVALID_REQUEST", Message: err.Error()},
		})
		return
	}

	rewards, err := h.stakingService.ClaimRewards(c.Request.Context(), req.PositionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "CLAIM_ERROR", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    gin.H{"rewards": rewards},
	})
}

func (h *StakingHandler) GetStakingPositions(c *gin.Context) {
	walletID := c.Query("wallet_id")
	if walletID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INVALID_REQUEST", Message: "wallet_id required"},
		})
		return
	}

	positions, err := h.stakingService.GetPositions(c.Request.Context(), walletID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INTERNAL_ERROR", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    positions,
	})
}

// Perpetual Handler
type PerpetualHandler struct {
	perpService *services.PerpetualService
}

func NewPerpetualHandler() *PerpetualHandler {
	return &PerpetualHandler{
		perpService: services.NewPerpetualService(),
	}
}

func (h *PerpetualHandler) GetMarkets(c *gin.Context) {
	markets, err := h.perpService.GetMarkets(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INTERNAL_ERROR", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    markets,
	})
}

func (h *PerpetualHandler) OpenPosition(c *gin.Context) {
	var req services.OpenPositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INVALID_REQUEST", Message: err.Error()},
		})
		return
	}

	position, err := h.perpService.OpenPosition(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "POSITION_ERROR", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Data:    position,
	})
}

func (h *PerpetualHandler) ClosePosition(c *gin.Context) {
	var req struct {
		PositionID string `json:"position_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INVALID_REQUEST", Message: err.Error()},
		})
		return
	}

	err := h.perpService.ClosePosition(c.Request.Context(), req.PositionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "CLOSE_ERROR", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
	})
}

func (h *PerpetualHandler) GetPositions(c *gin.Context) {
	walletID := c.Query("wallet_id")
	if walletID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INVALID_REQUEST", Message: "wallet_id required"},
		})
		return
	}

	positions, err := h.perpService.GetPositions(c.Request.Context(), walletID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INTERNAL_ERROR", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    positions,
	})
}

func (h *PerpetualHandler) CreateOrder(c *gin.Context) {
	var req services.CreatePerpetualOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "INVALID_REQUEST", Message: err.Error()},
		})
		return
	}

	order, err := h.perpService.CreateOrder(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "ORDER_ERROR", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Data:    order,
	})
}

func (h *PerpetualHandler) CancelOrder(c *gin.Context) {
	orderID := c.Param("id")

	err := h.perpService.CancelOrder(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{Code: "CANCEL_ERROR", Message: err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
	})
}
