package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
)

// ============================================================================
// TIGERWALLET MULTI-LEVEL WHITE LABEL SYSTEM
// Hierarchical white label structure with parent-child relationships
// ============================================================================

var (
	logger     zerolog.Logger
	redisClient *redis.Client
	dbPool     *pgxpool.Pool
)

// Configuration
type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	MaxLevels   int    // Maximum nesting depth
	JWTSecret   string // SuperAdmin control-plane secret (shared) for auth
}

// Hierarchical White Label
type HierarchicalWhiteLabel struct {
	ID               string    `json:"id"`
	ParentID         *string   `json:"parentId,omitempty"` // Pointer to handle null
	RootID           string    `json:"rootId"` // Top-level parent ID
	Level            int       `json:"level"` // 0 = root, 1 = child, 2 = grandchild, etc.
	Name             string    `json:"name"`
	Domain           string    `json:"domain"`
	Slug             string    `json:"slug"`
	Status           string    `json:"status"` // active, suspended, halted
	CommissionRate   float64   `json:"commissionRate"` // Commission to parent
	RevenueShare     float64   `json:"revenueShare"` // Share from transactions
	MaxSubAccounts   int       `json:"maxSubAccounts"`
	CurrentSubAccounts int     `json:"currentSubAccounts"`
	TotalUsers       int       `json:"totalUsers"`
	TotalRevenue     float64   `json:"totalRevenue"`
	TotalCommission  float64   `json:"totalCommission"`
	ChainAccess      []string  `json:"chainAccess"`
	Features         []string  `json:"features"`
	Branding         Branding  `json:"branding"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// Branding
type Branding struct {
	PrimaryColor   string `json:"primaryColor"`
	SecondaryColor string `json:"secondaryColor"`
	LogoURL        string `json:"logoUrl"`
	AppName        string `json:"appName"`
}

// White Label Hierarchy Node (for tree representation)
type HierarchyNode struct {
	WhiteLabel     *HierarchicalWhiteLabel `json:"whiteLabel"`
	Children       []*HierarchyNode        `json:"children"`
	Parent         *HierarchyNode          `json:"parent,omitempty"`
	Path           []string                 `json:"path"` // Path from root to this node
	Depth          int                      `json:"depth"`
}

// Commission Record
type CommissionRecord struct {
	ID            string    `json:"id"`
	FromWhiteLabel string  `json:"fromWhiteLabel"` // Child who earned commission
	ToWhiteLabel   string  `json:"toWhiteLabel"`   // Parent who receives commission
	Amount         float64  `json:"amount"`
	Currency       string    `json:"currency"`
	Type           string    `json:"type"` // transaction, subscription, upgrade
	ReferenceID    string    `json:"referenceId"`
	Status         string    `json:"status"` // pending, paid, cancelled
	PaidAt         *time.Time `json:"paidAt,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

// Revenue Share Distribution
type RevenueDistribution struct {
	WhiteLabelID  string  `json:"whiteLabelId"`
	TotalRevenue  float64 `json:"totalRevenue"`
	ParentShare   float64 `json:"parentShare"`
	RootShare     float64 `json:"rootShare"`
	PlatformShare float64 `json:"platformShare"`
	NetRevenue    float64 `json:"netRevenue"`
	Period        string  `json:"period"`
	CalculatedAt time.Time `json:"calculatedAt"`
}

// ============================================================================
// Database Functions
// ============================================================================

func createHierarchySchema() error {
	schema := `
	-- Hierarchical White Labels
	CREATE TABLE IF NOT EXISTS hierarchical_white_labels (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		parent_id UUID REFERENCES hierarchical_white_labels(id) ON DELETE SET NULL,
		root_id UUID NOT NULL,
		level INTEGER NOT NULL DEFAULT 0,
		name VARCHAR(255) NOT NULL,
		domain VARCHAR(255) UNIQUE NOT NULL,
		slug VARCHAR(100) UNIQUE NOT NULL,
		status VARCHAR(20) DEFAULT 'active',
		commission_rate DECIMAL(5,4) DEFAULT 0.10,
		revenue_share DECIMAL(5,4) DEFAULT 0.80,
		max_sub_accounts INTEGER DEFAULT 10,
		current_sub_accounts INTEGER DEFAULT 0,
		total_users INTEGER DEFAULT 0,
		total_revenue DECIMAL(20,8) DEFAULT 0,
		total_commission DECIMAL(20,8) DEFAULT 0,
		chain_access JSONB DEFAULT '[]',
		features JSONB DEFAULT '[]',
		branding JSONB DEFAULT '{}',
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	-- Commission Records
	CREATE TABLE IF NOT EXISTS commission_records (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		from_white_label UUID NOT NULL REFERENCES hierarchical_white_labels(id) ON DELETE CASCADE,
		to_white_label UUID NOT NULL REFERENCES hierarchical_white_labels(id) ON DELETE CASCADE,
		amount DECIMAL(20,8) NOT NULL,
		currency VARCHAR(10) NOT NULL DEFAULT 'USD',
		type VARCHAR(50) NOT NULL,
		reference_id VARCHAR(255),
		status VARCHAR(20) DEFAULT 'pending',
		paid_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT NOW()
	);

	-- Revenue Distributions
	CREATE TABLE IF NOT EXISTS revenue_distributions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		white_label_id UUID NOT NULL REFERENCES hierarchical_white_labels(id) ON DELETE CASCADE,
		total_revenue DECIMAL(20,8) NOT NULL,
		parent_share DECIMAL(20,8) DEFAULT 0,
		root_share DECIMAL(20,8) DEFAULT 0,
		platform_share DECIMAL(20,8) DEFAULT 0,
		net_revenue DECIMAL(20,8) NOT NULL,
		period VARCHAR(20) NOT NULL,
		calculated_at TIMESTAMP DEFAULT NOW()
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_hwl_parent ON hierarchical_white_labels(parent_id);
	CREATE INDEX IF NOT EXISTS idx_hwl_root ON hierarchical_white_labels(root_id);
	CREATE INDEX IF NOT EXISTS idx_hwl_level ON hierarchical_white_labels(level);
	CREATE INDEX IF NOT EXISTS idx_hwl_slug ON hierarchical_white_labels(slug);
	CREATE INDEX IF NOT EXISTS idx_comm_from ON commission_records(from_white_label);
	CREATE INDEX IF NOT EXISTS idx_comm_to ON commission_records(to_white_label);
	CREATE INDEX IF NOT EXISTS idx_rev_white_label ON revenue_distributions(white_label_id);
	`

	_, err := dbPool.Exec(context.Background(), schema)
	return err
}

// ============================================================================
// Core Functions
// ============================================================================

// Create new white label with hierarchy
func CreateHierarchicalWhiteLabel(c *gin.Context) {
	var wl HierarchicalWhiteLabel
	if err := c.ShouldBindJSON(&wl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check max levels
	if wl.ParentID != nil {
		var parentLevel int
		err := dbPool.QueryRow(context.Background(), 
			"SELECT level FROM hierarchical_white_labels WHERE id = $1", wl.ParentID).Scan(&parentLevel)
		
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Parent white label not found"})
			return
		}

		if parentLevel >= 3 { // Max level 3 (0, 1, 2, 3)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum hierarchy level reached"})
			return
		}

		// Check parent has capacity
		var maxSubs, currentSubs int
		err = dbPool.QueryRow(context.Background(), `
			SELECT max_sub_accounts, current_sub_accounts FROM hierarchical_white_labels WHERE id = $1
		`, wl.ParentID).Scan(&maxSubs, &currentSubs)

		if err != nil || currentSubs >= maxSubs {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parent white label has reached maximum sub-accounts"})
			return
		}

		// Update parent sub-account count
		dbPool.Exec(context.Background(), `
			UPDATE hierarchical_white_labels SET current_sub_accounts = current_sub_accounts + 1 
			WHERE id = $1
		`, wl.ParentID)

		// Get root ID from parent
		var rootID string
		dbPool.QueryRow(context.Background(), 
			"SELECT root_id FROM hierarchical_white_labels WHERE id = $1", wl.ParentID).Scan(&rootID)
		
		wl.RootID = rootID
		wl.Level = parentLevel + 1
	} else {
		// Root level white label
		wl.RootID = wl.ID
		wl.Level = 0
		wl.CommissionRate = 0
	}

	// Generate slug from name
	wl.Slug = generateSlug(wl.Name)
	wl.Status = "active"
	wl.CurrentSubAccounts = 0
	wl.TotalUsers = 0
	wl.TotalRevenue = 0
	wl.TotalCommission = 0
	wl.CreatedAt = time.Now()
	wl.UpdatedAt = time.Now()

	brandingJSON, _ := json.Marshal(wl.Branding)
	chainsJSON, _ := json.Marshal(wl.ChainAccess)
	featuresJSON, _ := json.Marshal(wl.Features)

	_, err := dbPool.Exec(context.Background(), `
		INSERT INTO hierarchical_white_labels (
			id, parent_id, root_id, level, name, domain, slug, status, 
			commission_rate, revenue_share, max_sub_accounts, current_sub_accounts,
			total_users, total_revenue, total_commission, chain_access, features, branding,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`, wl.ID, wl.ParentID, wl.RootID, wl.Level, wl.Name, wl.Domain, wl.Slug, wl.Status,
		wl.CommissionRate, wl.RevenueShare, wl.MaxSubAccounts, wl.CurrentSubAccounts,
		wl.TotalUsers, wl.TotalRevenue, wl.TotalCommission, chainsJSON, featuresJSON, brandingJSON,
		wl.CreatedAt, wl.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"whiteLabel": wl})
}

// Get hierarchy tree
func GetHierarchyTree(c *gin.Context) {
	rootID := c.Param("id")

	// Get all white labels in this tree
	rows, err := dbPool.Query(context.Background(), `
		SELECT id, parent_id, root_id, level, name, domain, slug, status, 
			commission_rate, revenue_share, max_sub_accounts, current_sub_accounts,
			total_users, total_revenue, total_commission, chain_access, features, branding,
			created_at, updated_at
		FROM hierarchical_white_labels 
		WHERE root_id = $1
		ORDER BY level, name
	`, rootID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	// Build tree
	whiteLabels := make([]*HierarchicalWhiteLabel, 0)
	for rows.Next() {
		var wl HierarchicalWhiteLabel
		var parentID, rootID sql.NullString
		var brandingJSON, chainsJSON, featuresJSON []byte

		rows.Scan(
			&wl.ID, &parentID, &rootID, &wl.Level, &wl.Name, &wl.Domain, &wl.Slug, &wl.Status,
			&wl.CommissionRate, &wl.RevenueShare, &wl.MaxSubAccounts, &wl.CurrentSubAccounts,
			&wl.TotalUsers, &wl.TotalRevenue, &wl.TotalCommission, &chainsJSON, &featuresJSON,
			&brandingJSON, &wl.CreatedAt, &wl.UpdatedAt,
		)

		if parentID.Valid {
			wl.ParentID = &parentID.String
		}
		if rootID.Valid {
			wl.RootID = rootID.String
		}

		json.Unmarshal(chainsJSON, &wl.ChainAccess)
		json.Unmarshal(featuresJSON, &wl.Features)
		json.Unmarshal(brandingJSON, &wl.Branding)

		whiteLabels = append(whiteLabels, &wl)
	}

	// Build hierarchy tree
	tree := buildTree(whiteLabels)

	c.JSON(http.StatusOK, gin.H{"tree": tree})
}

// Build tree from flat list
func buildTree(whiteLabels []*HierarchicalWhiteLabel) *HierarchyNode {
	if len(whiteLabels) == 0 {
		return nil
	}

	// Find root
	var root *HierarchicalWhiteLabel
	for _, wl := range whiteLabels {
		if wl.ParentID == nil {
			root = wl
			break
		}
	}

	if root == nil {
		return nil
	}

	// Build tree recursively
	node := &HierarchyNode{
		WhiteLabel: root,
		Children:   make([]*HierarchyNode, 0),
		Depth:      0,
		Path:       []string{root.ID},
	}

	buildChildren(node, whiteLabels)

	return node
}

func buildChildren(parent *HierarchyNode, allLabels []*HierarchicalWhiteLabel) {
	for _, wl := range allLabels {
		if wl.ParentID != nil && *wl.ParentID == parent.WhiteLabel.ID {
			child := &HierarchyNode{
				WhiteLabel: wl,
				Children:   make([]*HierarchyNode, 0),
				Parent:     parent,
				Depth:      parent.Depth + 1,
				Path:       append([]string{}, append(parent.Path, wl.ID)...),
			}
			parent.Children = append(parent.Children, child)
			buildChildren(child, allLabels)
		}
	}
}

// Calculate revenue distribution for a transaction
func CalculateRevenueDistribution(whiteLabelID string, amount float64) []RevenueDistribution {
	// Get white label and its ancestors
	var wl HierarchicalWhiteLabel
	err := dbPool.QueryRow(context.Background(), `
		SELECT id, parent_id, root_id, level, revenue_share, commission_rate 
		FROM hierarchical_white_labels WHERE id = $1
	`, whiteLabelID).Scan(&wl.ID, &wl.ParentID, &wl.RootID, &wl.Level, &wl.RevenueShare, &wl.CommissionRate)

	if err != nil {
		return nil
	}

	distributions := make([]RevenueDistribution, 0)
	remainingAmount := amount

	// Calculate platform share (20%)
	platformShare := amount * 0.20
	distributions = append(distributions, RevenueDistribution{
		WhiteLabelID:  "platform",
		TotalRevenue:  amount,
		PlatformShare: platformShare,
		NetRevenue:    amount - platformShare,
		Period:        time.Now().Format("2006-01"),
		CalculatedAt:  time.Now(),
	})

	remainingAmount = amount - platformShare

	// Calculate parent shares up the chain
	currentWL := &wl
	for currentWL.ParentID != nil {
		var parentWL HierarchicalWhiteLabel
		err := dbPool.QueryRow(context.Background(), `
			SELECT id, parent_id, root_id, level, revenue_share, commission_rate 
			FROM hierarchical_white_labels WHERE id = $1
		`, currentWL.ParentID).Scan(
			&parentWL.ID, &parentWL.ParentID, &parentWL.RootID, &parentWL.Level,
			&parentWL.RevenueShare, &parentWL.CommissionRate,
		)

		if err != nil {
			break
		}

		commission := remainingAmount * parentWL.CommissionRate
		parentShare := remainingAmount * (1 - parentWL.RevenueShare)

		distributions = append(distributions, RevenueDistribution{
			WhiteLabelID:  parentWL.ID,
			TotalRevenue:   remainingAmount,
			ParentShare:    parentShare,
			NetRevenue:     remainingAmount - parentShare - commission,
			Period:         time.Now().Format("2006-01"),
			CalculatedAt:   time.Now(),
		})

		remainingAmount = remainingAmount - parentShare - commission
		currentWL = &parentWL
	}

	// Update white label revenue
	dbPool.Exec(context.Background(), `
		UPDATE hierarchical_white_labels SET total_revenue = total_revenue + $1, updated_at = NOW()
		WHERE id = $2
	`, remainingAmount, whiteLabelID)

	return distributions
}

// Record commission
func RecordCommission(fromID, toID string, amount float64, commissionType, referenceID string) error {
	record := CommissionRecord{
		ID:             generateUUID(),
		FromWhiteLabel: fromID,
		ToWhiteLabel:   toID,
		Amount:         amount,
		Currency:       "USD",
		Type:           commissionType,
		ReferenceID:   referenceID,
		Status:         "pending",
		CreatedAt:      time.Now(),
	}

	_, err := dbPool.Exec(context.Background(), `
		INSERT INTO commission_records (id, from_white_label, to_white_label, amount, currency, type, reference_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, record.ID, record.FromWhiteLabel, record.ToWhiteLabel, record.Amount, record.Currency,
		record.Type, record.ReferenceID, record.Status, record.CreatedAt)

	if err != nil {
		return err
	}

	// Update total commission
	dbPool.Exec(context.Background(), `
		UPDATE hierarchical_white_labels SET total_commission = total_commission + $1, updated_at = NOW()
		WHERE id = $2
	`, amount, toID)

	return nil
}

// Get commissions for white label
func GetCommissions(c *gin.Context) {
	whiteLabelID := c.Param("id")

	rows, err := dbPool.Query(context.Background(), `
		SELECT id, from_white_label, to_white_label, amount, currency, type, reference_id, status, paid_at, created_at
		FROM commission_records 
		WHERE to_white_label = $1 OR from_white_label = $1
		ORDER BY created_at DESC
	`, whiteLabelID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var commissions []CommissionRecord
	for rows.Next() {
		var comm CommissionRecord
		rows.Scan(&comm.ID, &comm.FromWhiteLabel, &comm.ToWhiteLabel, &comm.Amount, &comm.Currency,
			&comm.Type, &comm.ReferenceID, &comm.Status, &comm.PaidAt, &comm.CreatedAt)
		commissions = append(commissions, comm)
	}

	c.JSON(http.StatusOK, gin.H{"commissions": commissions})
}

// Get sub-accounts
func GetSubAccounts(c *gin.Context) {
	parentID := c.Param("id")

	rows, err := dbPool.Query(context.Background(), `
		SELECT id, parent_id, root_id, level, name, domain, slug, status, 
			commission_rate, revenue_share, max_sub_accounts, current_sub_accounts,
			total_users, total_revenue, total_commission, created_at, updated_at
		FROM hierarchical_white_labels 
		WHERE parent_id = $1
		ORDER BY name
	`, parentID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var subAccounts []HierarchicalWhiteLabel
	for rows.Next() {
		var wl HierarchicalWhiteLabel
		var parentID sql.NullString
		rows.Scan(
			&wl.ID, &parentID, &wl.RootID, &wl.Level, &wl.Name, &wl.Domain, &wl.Slug, &wl.Status,
			&wl.CommissionRate, &wl.RevenueShare, &wl.MaxSubAccounts, &wl.CurrentSubAccounts,
			&wl.TotalUsers, &wl.TotalRevenue, &wl.TotalCommission, &wl.CreatedAt, &wl.UpdatedAt,
		)
		if parentID.Valid {
			wl.ParentID = &parentID.String
		}
		subAccounts = append(subAccounts, wl)
	}

	c.JSON(http.StatusOK, gin.H{"subAccounts": subAccounts})
}

// Move white label to different parent
func MoveWhiteLabel(c *gin.Context) {
	var request struct {
		WhiteLabelID string `json:"whiteLabelId" binding:"required"`
		NewParentID  string `json:"newParentId"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current parent
	var oldParentID *string
	err := dbPool.QueryRow(context.Background(), 
		"SELECT parent_id FROM hierarchical_white_labels WHERE id = $1", 
		request.WhiteLabelID).Scan(&oldParentID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "White label not found"})
		return
	}

	// Update old parent count
	if oldParentID != nil {
		dbPool.Exec(context.Background(), `
			UPDATE hierarchical_white_labels SET current_sub_accounts = current_sub_accounts - 1 
			WHERE id = $1 AND current_sub_accounts > 0
		`, *oldParentID)
	}

	// Update new parent count
	if request.NewParentID != "" {
		dbPool.Exec(context.Background(), `
			UPDATE hierarchical_white_labels SET current_sub_accounts = current_sub_accounts + 1 
			WHERE id = $1
		`, request.NewParentID)
	}

	// Update the white label
	_, err = dbPool.Exec(context.Background(), `
		UPDATE hierarchical_white_labels 
		SET parent_id = $1, updated_at = NOW()
		WHERE id = $2
	`, nullString(request.NewParentID), request.WhiteLabelID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "White label moved successfully"})
}

// Get analytics for hierarchy
func GetHierarchyAnalytics(c *gin.Context) {
	rootID := c.Param("id")

	var analytics struct {
		TotalWhiteLabels int     `json:"totalWhiteLabels"`
		TotalUsers       int     `json:"totalUsers"`
		TotalRevenue     float64 `json:"totalRevenue"`
		TotalCommissions float64 `json:"totalCommissions"`
		AverageDepth     float64 `json:"averageDepth"`
		MaxDepth         int     `json:"maxDepth"`
	}

	err := dbPool.QueryRow(context.Background(), `
		SELECT 
			COUNT(*),
			COALESCE(SUM(total_users), 0),
			COALESCE(SUM(total_revenue), 0),
			COALESCE(SUM(total_commission), 0),
			COALESCE(AVG(level), 0),
			MAX(level)
		FROM hierarchical_white_labels 
		WHERE root_id = $1
	`, rootID).Scan(
		&analytics.TotalWhiteLabels, &analytics.TotalUsers, &analytics.TotalRevenue,
		&analytics.TotalCommissions, &analytics.AverageDepth, &analytics.MaxDepth,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"analytics": analytics})
}

// Helper functions
func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func generateSlug(name string) string {
	slug := ""
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			slug += string(c)
		} else if c >= 'A' && c <= 'Z' {
			slug += string(c + 32)
		} else if c == ' ' || c == '-' {
			slug += "-"
		}
	}
	if len(slug) > 50 {
		slug = slug[:50]
	}
	return slug
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// Router setup
// superAdminOnly enforces the no-resale / no-sublicensing boundary. The
// hierarchy service (parent/child white labels, sub-accounts, commissions) is a
// TigerWallet-internal governance surface. A WL client must NEVER be able to
// mint its own child WLs (which would be reselling the white label). Only the
// SuperAdmin control-plane role may create/move/read hierarchy nodes, enforced
// with the SAME JWT secret and role claim used by license_service.
func superAdminOnly(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}
		parts := strings.Split(h, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}
		claims := &struct {
			Role string `json:"role"`
			jwt.RegisteredClaims
		}{}
		tok, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (any, error) {
			return []byte(secret), nil
		})
		if err != nil || !tok.Valid || claims.Role != "superadmin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "SuperAdmin privilege required"})
			return
		}
		c.Next()
	}
}

// setupProtectedRouter wires the hierarchy endpoints behind the SuperAdmin-only
// JWT gate. This is the no-resale/no-sublicensing enforcement: only a
// control-plane SuperAdmin (never a wl_client) may create/move/read hierarchy.
func setupProtectedRouter(secret string) *gin.Engine {
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	v1 := r.Group("/api/v1")
	v1.Use(superAdminOnly(secret))
	{
		v1.POST("/white-label/hierarchy", CreateHierarchicalWhiteLabel)
		v1.GET("/white-label/hierarchy/:id/tree", GetHierarchyTree)
		v1.GET("/white-label/hierarchy/:id/subs", GetSubAccounts)
		v1.GET("/white-label/hierarchy/:id/commissions", GetCommissions)
		v1.GET("/white-label/hierarchy/:id/analytics", GetHierarchyAnalytics)
		v1.PUT("/white-label/hierarchy/move", MoveWhiteLabel)
	}

	return r
}

func main() {
	logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	logger.Info().Msg("Starting Multi-Level White Label System")

	config := Config{
		Port:        getEnv("PORT", "8088"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://tigerwallet:tigerpass@localhost:5432/tigerwallet?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
		JWTSecret:   getEnv("JWT_SECRET", ""),
	}

	// Fail closed: without a JWT secret we cannot authenticate the SuperAdmin and
	// therefore cannot safely guard the hierarchy endpoints from WL-client resale.
	if config.JWTSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required for the multi-level white-label hierarchy service")
	}

	var err error
	dbPool, err = pgxpool.Connect(context.Background(), config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	if err := createHierarchySchema(); err != nil {
		logger.Warn().Err(err).Msg("Schema creation warning")
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
	})
	defer redisClient.Close()

	router := setupProtectedRouter(config.JWTSecret)

	srv := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		logger.Info().Str("port", config.Port).Msg("Server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down server...")
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
