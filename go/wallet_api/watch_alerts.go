package main

// Watch-only wallets + price alerts — two UserWallet features that were
// declared in the audit as missing. Everything here is real: watch-only
// records cannot sign (they have no seed, and the signing paths reject them),
// and price alerts are evaluated against live CoinGecko spot prices on read.

import (
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ---- Watch-only wallets ----

type watchOnlyWalletReq struct {
	Address string `json:"address" binding:"required"`
	Label   string `json:"label"`
	ChainID int64  `json:"chain_id"`
}

// handleCreateWatchOnlyWallet records an external address the user wants to
// track. No seed is stored; the wallet can only be read (balance/tokens/tx),
// never signed. Watch-only wallets are rejected by every funds-movement path.
func handleCreateWatchOnlyWallet(c *gin.Context) {
	if store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage unavailable"})
		return
	}
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var req watchOnlyWalletReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !common.IsHexAddress(req.Address) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid address"})
		return
	}
	if req.ChainID == 0 {
		req.ChainID = 1
	}
	if req.Label == "" {
		req.Label = "Watch-only"
	}
	w := &WalletRecord{
		UserID:         userUUID,
		Label:          req.Label,
		ChainID:        req.ChainID,
		Address:        common.HexToAddress(req.Address).Hex(),
		DerivationPath: "",
		AccountIndex:   0,
		IsPrimary:      false,
		IsWatchOnly:    true,
	}
	if err := store.SaveWallet(c.Request.Context(), w); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save watch-only wallet"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":            w.ID,
		"label":         w.Label,
		"chain_id":      w.ChainID,
		"address":       w.Address,
		"is_watch_only": true,
	})
}

// ---- Price alerts ----

type priceAlertRecord struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	CoinID    string    `json:"coin_id"`
	Direction string    `json:"direction"`
	Price     float64   `json:"price"`
	Active    bool      `json:"active"`
	CreatedAt int64     `json:"created_at"`
	UpdatedAt int64     `json:"updated_at"`
	// observation of the current spot price (from CoinGecko) at read time
	CurrentPrice float64 `json:"current_price"`
	Triggered    bool    `json:"triggered"`
}

type priceAlertReq struct {
	CoinID    string  `json:"coin_id" binding:"required"`
	Direction string  `json:"direction" binding:"required"`
	Price     float64 `json:"price" binding:"required"`
}

func handleListPriceAlerts(c *gin.Context) {
	if store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage unavailable"})
		return
	}
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, user_id, coin_id, direction, CAST(price AS DOUBLE PRECISION),
		        active, extract(epoch from created_at)::bigint, extract(epoch from updated_at)::bigint
		 FROM price_alerts WHERE user_id=$1 ORDER BY created_at DESC`, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	ctx := c.Request.Context()
	out := []priceAlertRecord{}
	for rows.Next() {
		var r priceAlertRecord
		if err := rows.Scan(&r.ID, &r.UserID, &r.CoinID, &r.Direction, &r.Price, &r.Active, &r.CreatedAt, &r.UpdatedAt); err != nil {
			continue
		}
		// Evaluate against the live spot price now.
		if p, err := FetchTokenPrice(ctx, r.CoinID); err == nil && p.PriceUSD > 0 {
			r.CurrentPrice = p.PriceUSD
			r.Triggered = (r.Direction == "above" && p.PriceUSD >= r.Price) ||
				(r.Direction == "below" && p.PriceUSD <= r.Price)
		}
		out = append(out, r)
	}
	c.JSON(http.StatusOK, gin.H{"alerts": out, "count": len(out)})
}

func handleCreatePriceAlert(c *gin.Context) {
	if store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage unavailable"})
		return
	}
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var req priceAlertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Direction != "above" && req.Direction != "below" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "direction must be 'above' or 'below'"})
		return
	}
	if req.Price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "price must be positive"})
		return
	}
	id := uuid.New()
	_, err = store.PG.Exec(c.Request.Context(),
		`INSERT INTO price_alerts (id, user_id, coin_id, direction, price) VALUES ($1,$2,$3,$4,$5)`,
		id, userUUID, req.CoinID, req.Direction, req.Price)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot save alert"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "coin_id": req.CoinID, "direction": req.Direction, "price": req.Price})
}

func handleUpdatePriceAlert(c *gin.Context) {
	if store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage unavailable"})
		return
	}
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Direction *string  `json:"direction"`
		Price     *float64 `json:"price"`
		Active    *bool    `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	var (
		direction = "above"
		price     = 0.0
		active    = true
	)
	// Read existing values so partial updates don't clobber the rest.
	var curDir string
	var curPrice float64
	var curActive bool
	err = store.PG.QueryRow(ctx,
		`SELECT direction, CAST(price AS DOUBLE PRECISION), active FROM price_alerts WHERE id=$1 AND user_id=$2`,
		id, userUUID).Scan(&curDir, &curPrice, &curActive)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}
	direction, price, active = curDir, curPrice, curActive
	if req.Direction != nil {
		if *req.Direction != "above" && *req.Direction != "below" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "direction must be 'above' or 'below'"})
			return
		}
		direction = *req.Direction
	}
	if req.Price != nil {
		if *req.Price <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "price must be positive"})
			return
		}
		price = *req.Price
	}
	if req.Active != nil {
		active = *req.Active
	}
	tag, err := store.PG.Exec(ctx,
		`UPDATE price_alerts SET direction=$1, price=$2, active=$3, updated_at=NOW() WHERE id=$4 AND user_id=$5`,
		direction, price, active, id, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "updated": true})
}

func handleDeletePriceAlert(c *gin.Context) {
	if store == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "storage unavailable"})
		return
	}
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	tag, err := store.PG.Exec(c.Request.Context(), `DELETE FROM price_alerts WHERE id=$1 AND user_id=$2`, id, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "deleted": true})
}
