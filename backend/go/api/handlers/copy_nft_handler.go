package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"tigerwallet/backend/go/api/models"
	"tigerwallet/backend/go/api/services"
)

type CopyTradingHandler struct {
	copyService *services.CopyTradingService
}

func NewCopyTradingHandler() *CopyTradingHandler {
	return &CopyTradingHandler{copyService: services.NewCopyTradingService()}
}

func (h *CopyTradingHandler) GetTraders(c *gin.Context) {
	sortBy := c.DefaultQuery("sortBy", "followers")
	traders, _ := h.copyService.GetTraders(c.Request.Context(), sortBy)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: traders})
}

func (h *CopyTradingHandler) FollowTrader(c *gin.Context) {
	var req struct {
		FollowerID  string  `json:"follower_id" binding:"required"`
		TraderID   string  `json:"trader_id" binding:"required"`
		Allocation string  `json:"allocation" binding:"required"`
		MaxSlippage float64 `json:"max_slippage"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Error: &models.APIError{Code: "INVALID_REQUEST", Message: err.Error()}})
		return
	}
	maxSlippage := req.MaxSlippage
	if maxSlippage == 0 { maxSlippage = 1.0 }
	follower, err := h.copyService.FollowTrader(c.Request.Context(), req.FollowerID, req.TraderID, req.Allocation, maxSlippage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Error: &models.APIError{Code: "FOLLOW_ERROR", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: follower})
}

func (h *CopyTradingHandler) UnfollowTrader(c *gin.Context) {
	id := c.Param("id")
	err := h.copyService.UnfollowTrader(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Error: &models.APIError{Code: "UNFOLLOW_ERROR", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true})
}

func (h *CopyTradingHandler) GetCopyTrades(c *gin.Context) {
	followerID := c.Query("follower_id")
	trades, _ := h.copyService.GetCopyTrades(c.Request.Context(), followerID)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: trades})
}

type NFTHandler struct {
	nftService *services.NFTService
}

func NewNFTHandler() *NFTHandler {
	return &NFTHandler{nftService: services.NewNFTService()}
}

func (h *NFTHandler) GetCollections(c *gin.Context) {
	blockchainID := c.Query("blockchain_id")
	collections, _ := h.nftService.GetCollections(c.Request.Context(), blockchainID)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: collections})
}

func (h *NFTHandler) GetNFTs(c *gin.Context) {
	walletID := c.Query("wallet_id")
	collectionAddress := c.Query("collection_address")
	nfts, _ := h.nftService.GetNFTs(c.Request.Context(), walletID, collectionAddress)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: nfts})
}

func (h *NFTHandler) TransferNFT(c *gin.Context) {
	var req struct {
		NFTID     string `json:"nft_id" binding:"required"`
		ToAddress string `json:"to_address" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Error: &models.APIError{Code: "INVALID_REQUEST", Message: err.Error()}})
		return
	}
	err := h.nftService.TransferNFT(c.Request.Context(), req.NFTID, req.ToAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Error: &models.APIError{Code: "TRANSFER_ERROR", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true})
}

type TokenHandler struct {
	tokenService *services.TokenService
}

func NewTokenHandler() *TokenHandler {
	return &TokenHandler{tokenService: services.NewTokenService()}
}

func (h *TokenHandler) GetTokens(c *gin.Context) {
	blockchainID := c.Query("blockchain_id")
	isPopular := c.Query("is_popular") == "true"
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	tokens, total, _ := h.tokenService.GetAll(c.Request.Context(), blockchainID, isPopular, page, limit)
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: tokens, Meta: &models.APIMeta{Page: page, Limit: limit, Total: total}})
}

func (h *TokenHandler) GetToken(c *gin.Context) {
	tokenID := c.Param("id")
	token, err := h.tokenService.GetByID(c.Request.Context(), tokenID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{Success: false, Error: &models.APIError{Code: "NOT_FOUND", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: token})
}

func (h *TokenHandler) AddToken(c *gin.Context) {
	var req models.Token
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Error: &models.APIError{Code: "INVALID_REQUEST", Message: err.Error()}})
		return
	}
	token, err := h.tokenService.AddToken(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Error: &models.APIError{Code: "INTERNAL_ERROR", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, models.APIResponse{Success: true, Data: token})
}

func (h *TokenHandler) UpdateToken(c *gin.Context) {
	tokenID := c.Param("id")
	var req models.Token
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Error: &models.APIError{Code: "INVALID_REQUEST", Message: err.Error()}})
		return
	}
	token, err := h.tokenService.UpdateToken(c.Request.Context(), tokenID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Error: &models.APIError{Code: "INTERNAL_ERROR", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: token})
}

func (h *TokenHandler) DeleteToken(c *gin.Context) {
	tokenID := c.Param("id")
	err := h.tokenService.DeleteToken(c.Request.Context(), tokenID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Error: &models.APIError{Code: "INTERNAL_ERROR", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true})
}
