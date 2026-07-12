package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"tigerwallet/backend/go/api/models"
	"tigerwallet/backend/go/api/services"
)

// BlockchainHandler handles blockchain-related requests
type BlockchainHandler struct {
	blockchainService *services.BlockchainService
}

func NewBlockchainHandler() *BlockchainHandler {
	return &BlockchainHandler{
		blockchainService: services.NewBlockchainService(),
	}
}

// GetAllBlockchains returns all supported blockchains
// @Summary Get all blockchains
// @Description Returns a list of all supported blockchains
// @Tags blockchains
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Param type query string false "Filter by type (evm, solana, cosmos, etc.)"
// @Success 200 {object} models.APIResponse
// @Router /api/v1/blockchains [get]
func (h *BlockchainHandler) GetAllBlockchains(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	chainType := c.Query("type")

	blockchains, err := h.blockchainService.GetAll(c.Request.Context(), page, limit, chainType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	total, _ := h.blockchainService.Count(c.Request.Context())

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    blockchains,
		Meta: &models.APIMeta{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	})
}

// GetBlockchainByID returns a specific blockchain by ID
// @Summary Get blockchain by ID
// @Description Returns a single blockchain by its ID
// @Tags blockchains
// @Accept json
// @Produce json
// @Param id path string true "Blockchain ID"
// @Success 200 {object} models.APIResponse
// @Router /api/v1/blockchains/{id} [get]
func (h *BlockchainHandler) GetBlockchainByID(c *gin.Context) {
	id := c.Param("id")

	blockchain, err := h.blockchainService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Error: &models.APIError{
				Code:    "NOT_FOUND",
				Message: "Blockchain not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    blockchain,
	})
}

// AddBlockchain adds a new blockchain (Admin only)
// @Summary Add new blockchain
// @Description Adds a new blockchain to the system
// @Tags blockchains
// @Accept json
// @Produce json
// @Param blockchain body models.Blockchain true "Blockchain data"
// @Success 201 {object} models.APIResponse
// @Router /api/v1/blockchains [post]
func (h *BlockchainHandler) AddBlockchain(c *gin.Context) {
	var req models.Blockchain
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error: &models.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	blockchain, err := h.blockchainService.Create(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{
		Success: true,
		Data:    blockchain,
	})
}

// UpdateBlockchain updates an existing blockchain (Admin only)
// @Summary Update blockchain
// @Description Updates an existing blockchain
// @Tags blockchains
// @Accept json
// @Produce json
// @Param id path string true "Blockchain ID"
// @Param blockchain body models.Blockchain true "Updated blockchain data"
// @Success 200 {object} models.APIResponse
// @Router /api/v1/blockchains/{id} [put]
func (h *BlockchainHandler) UpdateBlockchain(c *gin.Context) {
	id := c.Param("id")

	var req models.Blockchain
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error: &models.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	blockchain, err := h.blockchainService.Update(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    blockchain,
	})
}

// DeleteBlockchain deletes a blockchain (Admin only)
// @Summary Delete blockchain
// @Description Removes a blockchain from the system
// @Tags blockchains
// @Accept json
// @Produce json
// @Param id path string true "Blockchain ID"
// @Success 200 {object} models.APIResponse
// @Router /api/v1/blockchains/{id} [delete]
func (h *BlockchainHandler) DeleteBlockchain(c *gin.Context) {
	id := c.Param("id")

	err := h.blockchainService.Delete(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error: &models.APIError{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
	})
}
