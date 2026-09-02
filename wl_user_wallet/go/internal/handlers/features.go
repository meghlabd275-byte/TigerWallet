// Real handler implementations for the flat-route feature entities.
// Each handler hits the store helpers in wl/store/features.go.
package handlers

import (
        "net/http"

        "github.com/gin-gonic/gin"
        "github.com/google/uuid"
        "github.com/tigerwallet/wl-user-wallet/internal/middleware"
)

// ==================== Price alerts ====================

func (s *Svc) ListPriceAlerts(c *gin.Context) {
        list, err := s.store.ListPriceAlerts(c.Request.Context(), middleware.UserID(c))
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusOK, gin.H{"alerts": list})
}

func (s *Svc) CreatePriceAlert(c *gin.Context) {
        var req struct {
                Symbol    string `json:"symbol" binding:"required"`
                Target    string `json:"target_price" binding:"required"`
                Direction string `json:"direction"`
        }
        if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
        dir := req.Direction
        if dir == "" { dir = "above" }
        id, err := s.store.CreatePriceAlert(c.Request.Context(), middleware.UserID(c), req.Symbol, req.Target, dir)
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusCreated, gin.H{"id": id, "symbol": req.Symbol, "target_price": req.Target, "direction": dir})
}

func (s *Svc) DeletePriceAlert(c *gin.Context) {
        id, err := uuid.Parse(c.Param("id"))
        if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
        _, _ = s.store.DeletePriceAlert(c.Request.Context(), middleware.UserID(c), id)
        c.Status(http.StatusNoContent)
}

// ==================== P2P ====================

func (s *Svc) ListP2PAdverts(c *gin.Context) {
        list, err := s.store.ListP2PAdverts(c.Request.Context())
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusOK, gin.H{"adverts": list})
}

func (s *Svc) CreateP2POrder(c *gin.Context) {
        var req struct {
                AdvertID string `json:"advert_id" binding:"required"`
                Amount   string `json:"amount" binding:"required"`
        }
        if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
        advertID, err := uuid.Parse(req.AdvertID)
        if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid advert id"}); return }
        id, err := s.store.CreateP2POrder(c.Request.Context(), advertID, middleware.UserID(c), req.Amount)
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusCreated, gin.H{"id": id, "order_id": id})
}

// ==================== DAO ====================

func (s *Svc) ListDaoProposals(c *gin.Context) {
        list, err := s.store.ListDaoProposals(c.Request.Context())
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusOK, gin.H{"proposals": list})
}

func (s *Svc) CreateDaoProposal(c *gin.Context) {
        var req struct {
                Title       string `json:"title" binding:"required"`
                Description string `json:"description"`
        }
        if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
        id, err := s.store.CreateDaoProposal(c.Request.Context(), req.Title, req.Description)
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (s *Svc) VoteDaoProposal(c *gin.Context) {
        proposalID, err := uuid.Parse(c.Param("id"))
        if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid proposal id"}); return }
        var req struct {
                Support bool `json:"support"`
        }
        if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
        if err := s.store.VoteDaoProposal(c.Request.Context(), proposalID, middleware.UserID(c), req.Support); err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }
        c.JSON(http.StatusOK, gin.H{"voted": true, "proposal_id": proposalID.String()})
}

// ==================== Launchpool ====================

func (s *Svc) LaunchpoolPools(c *gin.Context) {
        list, err := s.store.ListLaunchpoolStakes(c.Request.Context(), middleware.UserID(c))
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusOK, gin.H{"pools": list})
}

func (s *Svc) LaunchpoolStakes(c *gin.Context) {
        list, err := s.store.ListLaunchpoolStakes(c.Request.Context(), middleware.UserID(c))
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusOK, gin.H{"stakes": list})
}

func (s *Svc) LaunchpoolStake(c *gin.Context) {
        var req struct {
                Amount string `json:"amount" binding:"required"`
        }
        if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
        id, err := s.store.LaunchpoolStake(c.Request.Context(), middleware.UserID(c), req.Amount)
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusCreated, gin.H{"id": id})
}

// ==================== Token sales ====================

func (s *Svc) ListTokenSales(c *gin.Context) {
        list, err := s.store.ListTokenSales(c.Request.Context())
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusOK, gin.H{"sales": list})
}

func (s *Svc) ParticipateTokenSale(c *gin.Context) {
        saleID, err := uuid.Parse(c.Param("id"))
        if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sale id"}); return }
        var req struct {
                Amount string `json:"amount" binding:"required"`
        }
        if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
        id, err := s.store.ParticipateTokenSale(c.Request.Context(), saleID, middleware.UserID(c), req.Amount)
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusCreated, gin.H{"id": id})
}

// ==================== Token approvals ====================

func (s *Svc) ListTokenApprovals(c *gin.Context) {
        list, err := s.store.ListTokenApprovals(c.Request.Context(), middleware.UserID(c))
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusOK, gin.H{"approvals": list})
}

func (s *Svc) RevokeTokenApproval(c *gin.Context) {
        id, err := uuid.Parse(c.Param("id"))
        if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid approval id"}); return }
        _, _ = s.store.RevokeTokenApproval(c.Request.Context(), middleware.UserID(c), id)
        c.Status(http.StatusNoContent)
}

// ==================== Fees ====================

func (s *Svc) ListFees(c *gin.Context) {
        list, err := s.store.ListFees(c.Request.Context())
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusOK, gin.H{"fees": list})
}

func (s *Svc) FeeRevenue(c *gin.Context) {
        rev, err := s.store.FeeRevenue(c.Request.Context())
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusOK, gin.H{"revenue": rev})
}

// ==================== KYC ====================

func (s *Svc) KycStatus(c *gin.Context) {
        rec, err := s.store.KycRecordFor(c.Request.Context(), middleware.UserID(c))
        if err != nil {
                c.JSON(http.StatusOK, gin.H{"status": "not_submitted"})
                return
        }
        c.JSON(http.StatusOK, gin.H{"status": rec.Status, "full_name": rec.FullName, "doc_type": rec.DocType})
}

func (s *Svc) KycRegister(c *gin.Context) {
        var req struct {
                FullName string `json:"full_name" binding:"required"`
                DocType  string `json:"doc_type"`
                DocNumber string `json:"doc_number"`
        }
        if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
        id, err := s.store.UpsertKycRecord(c.Request.Context(), middleware.UserID(c), req.FullName, req.DocType, req.DocNumber)
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusCreated, gin.H{"id": id, "status": "pending"})
}

func (s *Svc) KycSubmit(c *gin.Context) {
        rec, err := s.store.KycRecordFor(c.Request.Context(), middleware.UserID(c))
        if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "no KYC record"}); return }
        c.JSON(http.StatusOK, gin.H{"status": rec.Status, "submitted": true})
}

// ==================== Cards ====================

func (s *Svc) CardBalance(c *gin.Context) {
        ca, err := s.store.CardAccount(c.Request.Context(), middleware.UserID(c))
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusOK, gin.H{"balance": ca.Balance, "account_id": ca.ID})
}

func (s *Svc) CardTransactions(c *gin.Context) {
        list, err := s.store.ListCardTransactions(c.Request.Context(), middleware.UserID(c))
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusOK, gin.H{"transactions": list})
}

// ==================== Margin/Perp ====================

func (s *Svc) ListMarginPositions(c *gin.Context) {
        list, err := s.store.ListMarginPositions(c.Request.Context(), middleware.UserID(c))
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusOK, gin.H{"positions": list})
}

func (s *Svc) CreateMarginPosition(c *gin.Context) {
        var req struct {
                Pair     string `json:"pair" binding:"required"`
                Side     string `json:"side" binding:"required"`
                Size     string `json:"size" binding:"required"`
                Leverage int    `json:"leverage" binding:"required,min=1"`
        }
        if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
        // Trading control-plane: margin vertical halt + explicit market stop.
        if s.tradingStopped(c, "margin", "margin_market", req.Pair) {
        	return
        }
        id, err := s.store.CreateMarginPosition(c.Request.Context(), middleware.UserID(c), req.Pair, req.Side, req.Size, req.Leverage)
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (s *Svc) CloseMarginPosition(c *gin.Context) {
        closePositionHandler(s, c, "margin_positions")
}

func (s *Svc) ListPerpPositions(c *gin.Context) {
        list, err := s.store.ListPerpPositions(c.Request.Context(), middleware.UserID(c))
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusOK, gin.H{"positions": list})
}

func (s *Svc) CreatePerpPosition(c *gin.Context) {
        var req struct {
                Pair     string `json:"pair" binding:"required"`
                Side     string `json:"side" binding:"required"`
                Size     string `json:"size" binding:"required"`
                Leverage int    `json:"leverage" binding:"required,min=1"`
        }
        if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
        // Trading control-plane: perpetual vertical halt + explicit contract stop.
        if s.tradingStopped(c, "perpetual", "contract", req.Pair) {
        	return
        }
        id, err := s.store.CreatePerpPosition(c.Request.Context(), middleware.UserID(c), req.Pair, req.Side, req.Size, req.Leverage)
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
        c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (s *Svc) ClosePerpPosition(c *gin.Context) {
        closePositionHandler(s, c, "perp_positions")
}

func closePositionHandler(s *Svc, c *gin.Context, table string) {
        id, err := uuid.Parse(c.Param("id"))
        if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid position id"}); return }
        var req struct {
                Pnl string `json:"pnl"`
        }
        _ = c.ShouldBindJSON(&req)
        _, err2 := s.store.ClosePosition(c.Request.Context(), middleware.UserID(c), table, id, req.Pnl)
        if err2 != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err2.Error()}); return }
        c.JSON(http.StatusOK, gin.H{"closed": true})
}

// ==================== Card account singleton ====================

// Auto-create on first access — real PG upsert at query time.
