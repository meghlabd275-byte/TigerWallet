package main

// Portfolio feature handlers: token approvals, perpetual positions, margin
// trading pairs, token sales, DAO proposals, and launchpool projects.
//
// All data is persisted in PostgreSQL (tables created by portfolioSchemaSQL,
// appended to schemaSQL in store.go). No mock/fabricated data — every list
// endpoint reads real rows; every create/act endpoint writes a real row.
// Lists start empty and populate as the user interacts.

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// portfolioSchemaSQL is appended to schemaSQL so the tables auto-migrate.
const portfolioSchemaSQL = `

CREATE TABLE IF NOT EXISTS token_approvals (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    wallet_id UUID,
    token TEXT NOT NULL,
    token_symbol TEXT NOT NULL,
    spender TEXT NOT NULL,
    spender_name TEXT NOT NULL,
    chain_id BIGINT NOT NULL,
    chain_name TEXT NOT NULL,
    amount TEXT NOT NULL,
    allowance TEXT NOT NULL,
    is_unlimited BOOLEAN NOT NULL DEFAULT FALSE,
    risk TEXT NOT NULL DEFAULT 'low',
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    date_approved BIGINT NOT NULL,
    revoked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_approvals_user ON token_approvals(user_id);
CREATE INDEX IF NOT EXISTS idx_approvals_wallet ON token_approvals(wallet_id);

CREATE TABLE IF NOT EXISTS perpetual_positions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    wallet_id UUID,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    size TEXT NOT NULL,
    entry_price TEXT NOT NULL,
    mark_price TEXT NOT NULL,
    leverage TEXT NOT NULL,
    margin TEXT NOT NULL,
    liq_price TEXT NOT NULL,
    unrealized_pnl TEXT NOT NULL DEFAULT '0',
    status TEXT NOT NULL DEFAULT 'open',
    chain_id BIGINT NOT NULL DEFAULT 1,
    opened_at BIGINT NOT NULL,
    closed_at BIGINT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_perp_user ON perpetual_positions(user_id);
CREATE INDEX IF NOT EXISTS idx_perp_status ON perpetual_positions(status);

CREATE TABLE IF NOT EXISTS margin_positions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    wallet_id UUID,
    pair_id TEXT NOT NULL,
    pair_symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    borrowed TEXT NOT NULL,
    collateral TEXT NOT NULL,
    leverage TEXT NOT NULL,
    entry_price TEXT NOT NULL,
    liq_price TEXT NOT NULL,
    interest_rate TEXT NOT NULL DEFAULT '0',
    unrealized_pnl TEXT NOT NULL DEFAULT '0',
    status TEXT NOT NULL DEFAULT 'open',
    chain_id BIGINT NOT NULL DEFAULT 1,
    opened_at BIGINT NOT NULL,
    closed_at BIGINT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_margin_user ON margin_positions(user_id);
CREATE INDEX IF NOT EXISTS idx_margin_status ON margin_positions(status);

CREATE TABLE IF NOT EXISTS token_sales (
    id UUID PRIMARY KEY,
    creator_id UUID,
    token_name TEXT NOT NULL,
    token_symbol TEXT NOT NULL,
    contract_address TEXT NOT NULL,
    chain_id BIGINT NOT NULL,
    price_per_token TEXT NOT NULL,
    total_supply TEXT NOT NULL,
    sold_amount TEXT NOT NULL DEFAULT '0',
    min_allocation TEXT NOT NULL DEFAULT '0',
    max_allocation TEXT NOT NULL DEFAULT '0',
    start_time BIGINT NOT NULL,
    end_time BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'upcoming',
    description TEXT,
    website TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sales_status ON token_sales(status);
CREATE INDEX IF NOT EXISTS idx_sales_chain ON token_sales(chain_id);

CREATE TABLE IF NOT EXISTS token_sale_participations (
    id UUID PRIMARY KEY,
    sale_id UUID NOT NULL,
    user_id UUID NOT NULL,
    amount TEXT NOT NULL,
    cost TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'participated',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(sale_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_sale_part_user ON token_sale_participations(user_id);

CREATE TABLE IF NOT EXISTS dao_proposals (
    id UUID PRIMARY KEY,
    creator_id UUID,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    proposer TEXT NOT NULL,
    proposer_name TEXT NOT NULL,
    for_votes TEXT NOT NULL DEFAULT '0',
    against_votes TEXT NOT NULL DEFAULT '0',
    abstain_votes TEXT NOT NULL DEFAULT '0',
    status TEXT NOT NULL DEFAULT 'active',
    start_time BIGINT NOT NULL,
    end_time BIGINT NOT NULL,
    executed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_proposals_status ON dao_proposals(status);

CREATE TABLE IF NOT EXISTS dao_delegates (
    id UUID PRIMARY KEY,
    address TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    voting_power TEXT NOT NULL DEFAULT '0',
    proposals_count BIGINT NOT NULL DEFAULT 0,
    delegated_to TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dao_votes (
    id UUID PRIMARY KEY,
    proposal_id UUID NOT NULL,
    voter_id UUID NOT NULL,
    choice TEXT NOT NULL,
    voting_power TEXT NOT NULL DEFAULT '0',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(proposal_id, voter_id)
);
CREATE INDEX IF NOT EXISTS idx_dao_votes_prop ON dao_votes(proposal_id);

CREATE TABLE IF NOT EXISTS launchpool_projects (
    id UUID PRIMARY KEY,
    token_name TEXT NOT NULL,
    token_symbol TEXT NOT NULL,
    contract_address TEXT NOT NULL,
    chain_id BIGINT NOT NULL,
    total_rewards TEXT NOT NULL,
    distributed_rewards TEXT NOT NULL DEFAULT '0',
    apy TEXT NOT NULL DEFAULT '0',
    start_time BIGINT NOT NULL,
    end_time BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'upcoming',
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_launchpool_status ON launchpool_projects(status);

CREATE TABLE IF NOT EXISTS launchpool_stakes (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL,
    user_id UUID NOT NULL,
    amount TEXT NOT NULL,
    rewards TEXT NOT NULL DEFAULT '0',
    start_time BIGINT NOT NULL,
    unstaked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(project_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_lp_stakes_user ON launchpool_stakes(user_id);
`

// ---- Token Approvals ----

type TokenApproval struct {
	ID           uuid.UUID  `json:"id"`
	UserID       uuid.UUID  `json:"user_id"`
	WalletID     *uuid.UUID `json:"wallet_id,omitempty"`
	Token        string     `json:"token"`
	TokenSymbol  string     `json:"token_symbol"`
	Spender      string     `json:"spender"`
	SpenderName  string     `json:"spender_name"`
	ChainID      int64      `json:"chain_id"`
	ChainName    string     `json:"chain_name"`
	Amount       string     `json:"amount"`
	Allowance    string     `json:"allowance"`
	IsUnlimited  bool       `json:"is_unlimited"`
	Risk         string     `json:"risk"`
	Verified     bool       `json:"verified"`
	DateApproved int64      `json:"date_approved"`
	Revoked      bool       `json:"revoked"`
}

func handleListApprovals(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, token, token_symbol, spender, spender_name, chain_id, chain_name,
		        amount, allowance, is_unlimited, risk, verified, date_approved, revoked, wallet_id
		 FROM token_approvals WHERE user_id=$1 AND revoked=FALSE ORDER BY date_approved DESC`,
		userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []TokenApproval{}
	for rows.Next() {
		var a TokenApproval
		var walletID *string
		if err := rows.Scan(&a.ID, &a.Token, &a.TokenSymbol, &a.Spender, &a.SpenderName,
			&a.ChainID, &a.ChainName, &a.Amount, &a.Allowance, &a.IsUnlimited, &a.Risk,
			&a.Verified, &a.DateApproved, &a.Revoked, &walletID); err != nil {
			continue
		}
		if walletID != nil {
			if wid, e := uuid.Parse(*walletID); e == nil {
				a.WalletID = &wid
			}
		}
		out = append(out, a)
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func handleRevokeApproval(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ct, err := store.PG.Exec(c.Request.Context(),
		`UPDATE token_approvals SET revoked=TRUE WHERE id=$1 AND user_id=$2`, id, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "revoke failed"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "approval not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"revoked": true}})
}

// ---- Perpetual Positions ----

type PerpetualPosition struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	WalletID      *uuid.UUID `json:"wallet_id,omitempty"`
	Symbol        string     `json:"symbol"`
	Side          string     `json:"side"`
	Size          string     `json:"size"`
	EntryPrice    string     `json:"entry_price"`
	MarkPrice     string     `json:"mark_price"`
	Leverage      string     `json:"leverage"`
	Margin        string     `json:"margin"`
	LiqPrice      string     `json:"liq_price"`
	UnrealizedPnL string     `json:"unrealized_pnl"`
	Status        string     `json:"status"`
	ChainID       int64      `json:"chain_id"`
	OpenedAt      int64      `json:"opened_at"`
	ClosedAt      *int64     `json:"closed_at,omitempty"`
}

func handleListPerpetualPositions(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	status := c.DefaultQuery("status", "open")
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, symbol, side, size, entry_price, mark_price, leverage, margin,
		        liq_price, unrealized_pnl, status, chain_id, opened_at, closed_at, wallet_id
		 FROM perpetual_positions WHERE user_id=$1 AND status=$2 ORDER BY opened_at DESC`,
		userUUID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []PerpetualPosition{}
	for rows.Next() {
		var p PerpetualPosition
		var walletID *string
		if err := rows.Scan(&p.ID, &p.Symbol, &p.Side, &p.Size, &p.EntryPrice, &p.MarkPrice,
			&p.Leverage, &p.Margin, &p.LiqPrice, &p.UnrealizedPnL, &p.Status, &p.ChainID,
			&p.OpenedAt, &p.ClosedAt, &walletID); err != nil {
			continue
		}
		if walletID != nil {
			if wid, e := uuid.Parse(*walletID); e == nil {
				p.WalletID = &wid
			}
		}
		out = append(out, p)
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func handleCreatePerpetualPosition(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var req struct {
		Symbol     string `json:"symbol" binding:"required"`
		Side       string `json:"side" binding:"required"`
		Size       string `json:"size" binding:"required"`
		EntryPrice string `json:"entry_price" binding:"required"`
		MarkPrice  string `json:"mark_price" binding:"required"`
		Leverage   string `json:"leverage" binding:"required"`
		Margin     string `json:"margin" binding:"required"`
		LiqPrice   string `json:"liq_price" binding:"required"`
		ChainID    int64  `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Trading control-plane: perpetual/futures vertical halt + contract stop
	// + operator leverage cap. Builtin enforcement, no external dependency.
	if !tradingGuard(c, "perpetual", "contract", req.Symbol) {
		return
	}
	if maxLev, status, found := tradingContractFor(c.Request.Context(), "perpetual", req.Symbol); found {
		if status != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "contract is stopped by the platform operator"})
			return
		}
		if lev, err := strconv.Atoi(req.Leverage); err == nil && maxLev > 0 && lev > maxLev {
			c.JSON(http.StatusBadRequest, gin.H{"error": "leverage exceeds contract max_leverage"})
			return
		}
	}
	id := uuid.New()
	now := time.Now().Unix()
	if _, err := store.PG.Exec(c.Request.Context(),
		`INSERT INTO perpetual_positions
		 (id, user_id, symbol, side, size, entry_price, mark_price, leverage, margin, liq_price, chain_id, opened_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		id, userUUID, req.Symbol, req.Side, req.Size, req.EntryPrice, req.MarkPrice,
		req.Leverage, req.Margin, req.LiqPrice, req.ChainID, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": id, "status": "open"}})
}

func handleClosePerpetualPosition(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	now := time.Now().Unix()
	ct, err := store.PG.Exec(c.Request.Context(),
		`UPDATE perpetual_positions SET status='closed', closed_at=$1 WHERE id=$2 AND user_id=$3 AND status='open'`,
		now, id, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "close failed"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found or already closed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"closed": true}})
}

// ---- Margin Positions ----

type MarginPosition struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	WalletID      *uuid.UUID `json:"wallet_id,omitempty"`
	PairID        string     `json:"pair_id"`
	PairSymbol    string     `json:"pair_symbol"`
	Side          string     `json:"side"`
	Borrowed      string     `json:"borrowed"`
	Collateral    string     `json:"collateral"`
	Leverage      string     `json:"leverage"`
	EntryPrice    string     `json:"entry_price"`
	LiqPrice      string     `json:"liq_price"`
	InterestRate  string     `json:"interest_rate"`
	UnrealizedPnL string     `json:"unrealized_pnl"`
	Status        string     `json:"status"`
	ChainID       int64      `json:"chain_id"`
	OpenedAt      int64      `json:"opened_at"`
	ClosedAt      *int64     `json:"closed_at,omitempty"`
}

func handleListMarginPositions(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	status := c.DefaultQuery("status", "open")
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, pair_id, pair_symbol, side, borrowed, collateral, leverage,
		        entry_price, liq_price, interest_rate, unrealized_pnl, status, chain_id, opened_at, closed_at, wallet_id
		 FROM margin_positions WHERE user_id=$1 AND status=$2 ORDER BY opened_at DESC`,
		userUUID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []MarginPosition{}
	for rows.Next() {
		var m MarginPosition
		var walletID *string
		if err := rows.Scan(&m.ID, &m.PairID, &m.PairSymbol, &m.Side, &m.Borrowed, &m.Collateral,
			&m.Leverage, &m.EntryPrice, &m.LiqPrice, &m.InterestRate, &m.UnrealizedPnL,
			&m.Status, &m.ChainID, &m.OpenedAt, &m.ClosedAt, &walletID); err != nil {
			continue
		}
		if walletID != nil {
			if wid, e := uuid.Parse(*walletID); e == nil {
				m.WalletID = &wid
			}
		}
		out = append(out, m)
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func handleCreateMarginPosition(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var req struct {
		PairID     string `json:"pair_id" binding:"required"`
		PairSymbol string `json:"pair_symbol" binding:"required"`
		Side       string `json:"side" binding:"required"`
		Borrowed   string `json:"borrowed" binding:"required"`
		Collateral string `json:"collateral" binding:"required"`
		Leverage   string `json:"leverage" binding:"required"`
		EntryPrice string `json:"entry_price" binding:"required"`
		LiqPrice   string `json:"liq_price" binding:"required"`
		ChainID    int64  `json:"chain_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Trading control-plane: margin vertical halt + market stop + leverage cap.
	if !tradingGuard(c, "margin", "margin_market", req.PairSymbol) {
		return
	}
	if maxLev, status, found := marginMarketFor(c.Request.Context(), req.PairSymbol); found {
		if status != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "margin market is stopped by the platform operator"})
			return
		}
		if lev, err := strconv.Atoi(req.Leverage); err == nil && maxLev > 0 && lev > maxLev {
			c.JSON(http.StatusBadRequest, gin.H{"error": "leverage exceeds margin market max_leverage"})
			return
		}
	}
	id := uuid.New()
	now := time.Now().Unix()
	if _, err := store.PG.Exec(c.Request.Context(),
		`INSERT INTO margin_positions
		 (id, user_id, pair_id, pair_symbol, side, borrowed, collateral, leverage, entry_price, liq_price, chain_id, opened_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		id, userUUID, req.PairID, req.PairSymbol, req.Side, req.Borrowed, req.Collateral,
		req.Leverage, req.EntryPrice, req.LiqPrice, req.ChainID, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": id, "status": "open"}})
}

func handleCloseMarginPosition(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	now := time.Now().Unix()
	ct, err := store.PG.Exec(c.Request.Context(),
		`UPDATE margin_positions SET status='closed', closed_at=$1 WHERE id=$2 AND user_id=$3 AND status='open'`,
		now, id, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "close failed"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "position not found or already closed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"closed": true}})
}

// ---- Token Sales ----

type TokenSale struct {
	ID              uuid.UUID  `json:"id"`
	CreatorID       *uuid.UUID `json:"creator_id,omitempty"`
	TokenName       string     `json:"token_name"`
	TokenSymbol     string     `json:"token_symbol"`
	ContractAddress string     `json:"contract_address"`
	ChainID         int64      `json:"chain_id"`
	PricePerToken   string     `json:"price_per_token"`
	TotalSupply     string     `json:"total_supply"`
	SoldAmount      string     `json:"sold_amount"`
	MinAllocation   string     `json:"min_allocation"`
	MaxAllocation   string     `json:"max_allocation"`
	StartTime       int64      `json:"start_time"`
	EndTime         int64      `json:"end_time"`
	Status          string     `json:"status"`
	Description     string     `json:"description,omitempty"`
	Website         string     `json:"website,omitempty"`
}

func handleListTokenSales(c *gin.Context) {
	status := c.DefaultQuery("status", "")
	var rows pgx.Rows
	var err error
	if status != "" {
		rows, err = store.PG.Query(c.Request.Context(),
			`SELECT id, token_name, token_symbol, contract_address, chain_id, price_per_token,
			        total_supply, sold_amount, min_allocation, max_allocation, start_time, end_time,
			        status, COALESCE(description,''), COALESCE(website,'')
			 FROM token_sales WHERE status=$1 ORDER BY start_time DESC`, status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
			return
		}
	} else {
		rows, err = store.PG.Query(c.Request.Context(),
			`SELECT id, token_name, token_symbol, contract_address, chain_id, price_per_token,
			        total_supply, sold_amount, min_allocation, max_allocation, start_time, end_time,
			        status, COALESCE(description,''), COALESCE(website,'')
			 FROM token_sales ORDER BY start_time DESC`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
			return
		}
	}
	defer rows.Close()
	out := []TokenSale{}
	for rows.Next() {
		var s TokenSale
		var creatorID *string
		_ = creatorID
		if err := rows.Scan(&s.ID, &s.TokenName, &s.TokenSymbol, &s.ContractAddress, &s.ChainID,
			&s.PricePerToken, &s.TotalSupply, &s.SoldAmount, &s.MinAllocation, &s.MaxAllocation,
			&s.StartTime, &s.EndTime, &s.Status, &s.Description, &s.Website); err != nil {
			continue
		}
		out = append(out, s)
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func handleParticipateTokenSale(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	saleIDStr := c.Param("id")
	saleID, err := uuid.Parse(saleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sale id"})
		return
	}
	var req struct {
		Amount string `json:"amount" binding:"required"`
		Cost   string `json:"cost" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx := c.Request.Context()
	id := uuid.New()
	_, err = store.PG.Exec(ctx,
		`INSERT INTO token_sale_participations (id, sale_id, user_id, amount, cost)
		 VALUES ($1,$2,$3,$4,$5) ON CONFLICT (sale_id, user_id) DO UPDATE SET amount=$4, cost=$5`,
		id, saleID, userUUID, req.Amount, req.Cost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "participate failed"})
		return
	}
	_, _ = store.PG.Exec(ctx,
		`UPDATE token_sales SET sold_amount = (CAST(sold_amount AS NUMERIC) + $1)::TEXT WHERE id=$2`,
		req.Amount, saleID)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"participated": true}})
}

// ---- DAO ----

type DAOProposal struct {
	ID           uuid.UUID `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Proposer     string    `json:"proposer"`
	ProposerName string    `json:"proposer_name"`
	ForVotes     string    `json:"for_votes"`
	AgainstVotes string    `json:"against_votes"`
	AbstainVotes string    `json:"abstain_votes"`
	Status       string    `json:"status"`
	StartTime    int64     `json:"start_time"`
	EndTime      int64     `json:"end_time"`
	Executed     bool      `json:"executed"`
}

func handleListDAOProposals(c *gin.Context) {
	status := c.DefaultQuery("status", "")
	var rows pgx.Rows
	var err error
	if status != "" {
		rows, err = store.PG.Query(c.Request.Context(),
			`SELECT id, title, description, proposer, proposer_name, for_votes, against_votes,
			        abstain_votes, status, start_time, end_time, executed
			 FROM dao_proposals WHERE status=$1 ORDER BY start_time DESC`, status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
			return
		}
	} else {
		rows, err = store.PG.Query(c.Request.Context(),
			`SELECT id, title, description, proposer, proposer_name, for_votes, against_votes,
			        abstain_votes, status, start_time, end_time, executed
			 FROM dao_proposals ORDER BY start_time DESC`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
			return
		}
	}
	defer rows.Close()
	out := []DAOProposal{}
	for rows.Next() {
		var p DAOProposal
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Proposer, &p.ProposerName,
			&p.ForVotes, &p.AgainstVotes, &p.AbstainVotes, &p.Status, &p.StartTime,
			&p.EndTime, &p.Executed); err != nil {
			continue
		}
		out = append(out, p)
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func handleCreateDAOProposal(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var req struct {
		Title        string `json:"title" binding:"required"`
		Description  string `json:"description" binding:"required"`
		Proposer     string `json:"proposer" binding:"required"`
		ProposerName string `json:"proposer_name" binding:"required"`
		DurationSec  int64  `json:"duration_sec"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.DurationSec == 0 {
		req.DurationSec = 86400 * 3 // 3 days default
	}
	id := uuid.New()
	now := time.Now().Unix()
	if _, err := store.PG.Exec(c.Request.Context(),
		`INSERT INTO dao_proposals (id, creator_id, title, description, proposer, proposer_name, start_time, end_time)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		id, userUUID, req.Title, req.Description, req.Proposer, req.ProposerName,
		now, now+req.DurationSec); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": id, "status": "active"}})
}

func handleVoteDAOProposal(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	propIDStr := c.Param("id")
	propID, err := uuid.Parse(propIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid proposal id"})
		return
	}
	var req struct {
		Choice      string `json:"choice" binding:"required"`
		VotingPower string `json:"voting_power"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Choice != "for" && req.Choice != "against" && req.Choice != "abstain" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid choice"})
		return
	}
	if req.VotingPower == "" {
		req.VotingPower = "0"
	}
	ctx := c.Request.Context()
	voteID := uuid.New()
	_, err = store.PG.Exec(ctx,
		`INSERT INTO dao_votes (id, proposal_id, voter_id, choice, voting_power)
		 VALUES ($1,$2,$3,$4,$5) ON CONFLICT (proposal_id, voter_id) DO UPDATE SET choice=$4, voting_power=$5`,
		voteID, propID, userUUID, req.Choice, req.VotingPower)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "vote failed"})
		return
	}
	col := map[string]string{"for": "for_votes", "against": "against_votes", "abstain": "abstain_votes"}[req.Choice]
	_, _ = store.PG.Exec(ctx,
		`UPDATE dao_proposals SET `+col+` = (CAST(`+col+` AS NUMERIC) + $1)::TEXT WHERE id=$2`,
		req.VotingPower, propID)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"voted": true}})
}

type DAODelegate struct {
	ID             uuid.UUID `json:"id"`
	Address        string    `json:"address"`
	Name           string    `json:"name"`
	VotingPower    string    `json:"voting_power"`
	ProposalsCount int64     `json:"proposals_count"`
	DelegatedTo    string    `json:"delegated_to,omitempty"`
}

func handleListDAODelegates(c *gin.Context) {
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, address, name, voting_power, proposals_count, COALESCE(delegated_to,'')
		 FROM dao_delegates ORDER BY CAST(voting_power AS NUMERIC) DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []DAODelegate{}
	for rows.Next() {
		var d DAODelegate
		if err := rows.Scan(&d.ID, &d.Address, &d.Name, &d.VotingPower, &d.ProposalsCount, &d.DelegatedTo); err != nil {
			continue
		}
		out = append(out, d)
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

// ---- Launchpool ----

type LaunchpoolProject struct {
	ID                 uuid.UUID `json:"id"`
	TokenName          string    `json:"token_name"`
	TokenSymbol        string    `json:"token_symbol"`
	ContractAddress    string    `json:"contract_address"`
	ChainID            int64     `json:"chain_id"`
	TotalRewards       string    `json:"total_rewards"`
	DistributedRewards string    `json:"distributed_rewards"`
	APY                string    `json:"apy"`
	StartTime          int64     `json:"start_time"`
	EndTime            int64     `json:"end_time"`
	Status             string    `json:"status"`
	Description        string    `json:"description,omitempty"`
}

func handleListLaunchpool(c *gin.Context) {
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, token_name, token_symbol, contract_address, chain_id, total_rewards,
		        distributed_rewards, apy, start_time, end_time, status, COALESCE(description,'')
		 FROM launchpool_projects ORDER BY start_time DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []LaunchpoolProject{}
	for rows.Next() {
		var p LaunchpoolProject
		if err := rows.Scan(&p.ID, &p.TokenName, &p.TokenSymbol, &p.ContractAddress, &p.ChainID,
			&p.TotalRewards, &p.DistributedRewards, &p.APY, &p.StartTime, &p.EndTime,
			&p.Status, &p.Description); err != nil {
			continue
		}
		out = append(out, p)
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

type LaunchpoolStake struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	UserID    uuid.UUID `json:"user_id"`
	Amount    string    `json:"amount"`
	Rewards   string    `json:"rewards"`
	StartTime int64     `json:"start_time"`
	Unstaked  bool      `json:"unstaked"`
}

func handleListLaunchpoolStakes(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, project_id, amount, rewards, start_time, unstaked
		 FROM launchpool_stakes WHERE user_id=$1 AND unstaked=FALSE ORDER BY start_time DESC`,
		userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []LaunchpoolStake{}
	for rows.Next() {
		var s LaunchpoolStake
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.Amount, &s.Rewards, &s.StartTime, &s.Unstaked); err != nil {
			continue
		}
		out = append(out, s)
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func handleLaunchpoolStake(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var req struct {
		ProjectID string `json:"projectId" binding:"required"`
		Amount    string `json:"amount" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}
	ctx := c.Request.Context()
	id := uuid.New()
	now := time.Now().Unix()
	_, err = store.PG.Exec(ctx,
		`INSERT INTO launchpool_stakes (id, project_id, user_id, amount, start_time)
		 VALUES ($1,$2,$3,$4,$5) ON CONFLICT (project_id, user_id) DO UPDATE SET amount = (CAST(launchpool_stakes.amount AS NUMERIC) + $4)::TEXT, unstaked=FALSE`,
		id, projectID, userUUID, req.Amount, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stake failed"})
		return
	}
	_, _ = store.PG.Exec(ctx,
		`UPDATE launchpool_projects SET distributed_rewards = (CAST(distributed_rewards AS NUMERIC) + $1)::TEXT WHERE id=$2`,
		req.Amount, projectID)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"success": true}})
}

func handleLaunchpoolUnstake(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var req struct {
		ProjectID string `json:"projectId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id"})
		return
	}
	ctx := context.Background()
	ct, err := store.PG.Exec(ctx,
		`UPDATE launchpool_stakes SET unstaked=TRUE WHERE project_id=$1 AND user_id=$2 AND unstaked=FALSE`,
		projectID, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unstake failed"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active stake found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"success": true}})
}
