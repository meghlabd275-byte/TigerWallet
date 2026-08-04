package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// TIGERWALLET WHITE LABEL MARKETPLACE
// High-performance distributed marketplace for white label partners
// ============================================================================

var (
	logger        zerolog.Logger
	redisClient   *redis.Client
	dbPool        *pgxpool.Pool
	wsHub         *WebSocketHub
	marketplaceCache *MarketplaceCache
)

// Configuration
type Config struct {
	Port                string
	DatabaseURL         string
	RedisURL            string
	JWTSecret           string
	SMTPHost            string
	SMTPPort            string
	SMTPUsername        string
	SMTPPassword        string
	AdminEmail          string
	EnableWebSocket     bool
	MaxRequestSize      int64
	RateLimitRequests   int
	RateLimitWindow     time.Duration
}

// Marketplace Package Types
type ServicePackage struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Type            string    `json:"type"` // starter, professional, enterprise, custom
	Price           float64   `json:"price"`
	Currency        string    `json:"currency"` // USD, USDT, USDC
	BillingCycle    string    `json:"billingCycle"` // monthly, yearly, lifetime
	Features        []string  `json:"features"`
	MaxUsers        int       `json:"maxUsers"`
	MaxTransactions int       `json:"maxTransactions"`
	APIQuota        int64     `json:"apiQuota"`
	SupportLevel    string    `json:"supportLevel"` // email, chat, priority, dedicated
	CustomBranding  bool      `json:"customBranding"`
	WhiteLabelID    string    `json:"whiteLabelId,omitempty"`
	Status          string    `json:"status"` // active, paused, discontinued
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// Marketplace Listing
type MarketplaceListing struct {
	ID              string    `json:"id"`
	PartnerID       string    `json:"partnerId"`
	PartnerName     string    `json:"partnerName"`
	PartnerLogo     string    `json:"partnerLogo"`
	PartnerRating   float64   `json:"partnerRating"`
	PartnerReviews  int       `json:"partnerReviews"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Category        string    `json:"category"` // wallet, defi, nft, gaming, enterprise
	Tags            []string  `json:"tags"`
	Price           float64   `json:"price"`
	Currency        string    `json:"currency"`
	PricingType     string    `json:"pricingType"` // free, one-time, subscription
	Features        []string  `json:"features"`
	DemoURL         string    `json:"demoUrl"`
	WebsiteURL      string    `json:"websiteUrl"`
	Documentation   string    `json:"documentation"`
	ChainsSupported []string  `json:"chainsSupported"`
	Status          string    `json:"status"` // pending, active, suspended, rejected
	Featured        bool      `json:"featured"`
	ViewCount       int       `json:"viewCount"`
	InstallCount    int       `json:"installCount"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// Partner Profile
type PartnerProfile struct {
	ID              string    `json:"id"`
	UserID          string    `json:"userId"`
	CompanyName     string    `json:"companyName"`
	LogoURL         string    `json:"logoUrl"`
	BannerURL       string    `json:"bannerUrl"`
	Description     string    `json:"description"`
	Website         string    `json:"website"`
	ContactEmail    string    `json:"contactEmail"`
	Phone           string    `json:"phone"`
	Location        string    `json:"location"`
	Established     int       `json:"established"`
	EmployeeCount   string    `json:"employeeCount"`
	BusinessType    string    `json:"businessType"` // software, financial, blockchain, other
	Certifications  []string  `json:"certifications"`
	SocialLinks     map[string]string `json:"socialLinks"`
	Rating          float64   `json:"rating"`
	ReviewCount     int       `json:"reviewCount"`
	TotalInstalls   int       `json:"totalInstalls"`
	ActiveWhiteLabels int     `json:"activeWhiteLabels"`
	Status          string    `json:"status"` // pending, active, suspended
	Verified        bool      `json:"verified"`
	KYCCompleted     bool      `json:"kycCompleted"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// Review
type Review struct {
	ID            string    `json:"id"`
	ListingID     string    `json:"listingId"`
	UserID        string    `json:"userId"`
	UserName      string    `json:"userName"`
	Rating        int       `json:"rating"` // 1-5
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	Pros          []string  `json:"pros"`
	Cons          []string  `json:"cons"`
	HelpfulCount  int       `json:"helpfulCount"`
	Response      string    `json:"response,omitempty"`
	ResponseAt    *time.Time `json:"responseAt,omitempty"`
	Status        string    `json:"status"` // pending, approved, flagged, removed
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// Transaction Record
type Transaction struct {
	ID              string    `json:"id"`
	Type            string    `json:"type"` // purchase, subscription, upgrade, refund
	ListingID       string    `json:"listingId,omitempty"`
	PackageID       string    `json:"packageId,omitempty"`
	BuyerID         string    `json:"buyerId"`
	SellerID        string    `json:"sellerId"`
	Amount          float64   `json:"amount"`
	Currency        string    `json:"currency"`
	Status          string    `json:"status"` // pending, completed, failed, refunded
	PaymentMethod   string    `json:"paymentMethod"` // stripe, crypto, bank
	PaymentID       string    `json:"paymentId"`
	Metadata        map[string]interface{} `json:"metadata"`
	CompletedAt     *time.Time `json:"completedAt,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

// WebSocket Hub for real-time updates
type WebSocketHub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mutex      sync.RWMutex
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mutex.Unlock()
		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// Marketplace Cache
type MarketplaceCache struct {
	redis     *redis.Client
	ctx       context.Context
	TTL       time.Duration
}

func NewMarketplaceCache(redisClient *redis.Client) *MarketplaceCache {
	return &MarketplaceCache{
		redis: redisClient,
		ctx:   context.Background(),
		TTL:   5 * time.Minute,
	}
}

func (c *MarketplaceCache) GetListings(category string) ([]MarketplaceListing, error) {
	key := fmt.Sprintf("marketplace:listings:%s", category)
	data, err := c.redis.Get(c.ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var listings []MarketplaceListing
	json.Unmarshal(data, &listings)
	return listings, nil
}

func (c *MarketplaceCache) SetListings(category string, listings []MarketplaceListing) error {
	key := fmt.Sprintf("marketplace:listings:%s", category)
	data, _ := json.Marshal(listings)
	return c.redis.Set(c.ctx, key, data, c.TTL).Err()
}

func (c *MarketplaceCache) InvalidateListings() error {
	iter := c.redis.Scan(c.ctx, 0, "marketplace:listings:*", 0).Iterator()
	for iter.Next(c.ctx) {
		c.redis.Del(c.ctx, iter.Val())
	}
	return iter.Err()
}

// Initialize Logger
func initLogger() zerolog.Logger {
	output := zerolog.ConsoleWriter{Out: os.Stderr}
	logger = zerolog.New(output).With().Timestamp().Caller().Logger()
	return logger
}

// Initialize Database
func initDatabase(databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database URL: %w", err)
	}

	config.MaxConns = 100
	config.MinConns = 10
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	// Verify connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	logger.Info().Msg("Database connection established")
	return pool, nil
}

// Initialize Redis
func initRedis(redisURL string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         redisURL,
		Password:     "",
		DB:           0,
		PoolSize:     100,
		MinIdleConns: 10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("unable to connect to Redis: %w", err)
	}

	logger.Info().Msg("Redis connection established")
	return client, nil
}

// Database Schema
func createMarketplaceSchema() error {
	schema := `
	-- Service Packages
	CREATE TABLE IF NOT EXISTS service_packages (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(255) NOT NULL,
		description TEXT,
		type VARCHAR(50) NOT NULL,
		price DECIMAL(20, 8) NOT NULL,
		currency VARCHAR(10) NOT NULL DEFAULT 'USD',
		billing_cycle VARCHAR(20) NOT NULL DEFAULT 'monthly',
		features JSONB DEFAULT '[]',
		max_users INTEGER DEFAULT 1000,
		max_transactions INTEGER DEFAULT 10000,
		api_quota BIGINT DEFAULT 1000000,
		support_level VARCHAR(50) DEFAULT 'email',
		custom_branding BOOLEAN DEFAULT true,
		white_label_id UUID,
		status VARCHAR(20) DEFAULT 'active',
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	-- Marketplace Listings
	CREATE TABLE IF NOT EXISTS marketplace_listings (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		partner_id UUID NOT NULL,
		title VARCHAR(255) NOT NULL,
		description TEXT,
		category VARCHAR(50) NOT NULL,
		tags JSONB DEFAULT '[]',
		price DECIMAL(20, 8) DEFAULT 0,
		currency VARCHAR(10) DEFAULT 'USD',
		pricing_type VARCHAR(20) DEFAULT 'free',
		features JSONB DEFAULT '[]',
		demo_url TEXT,
		website_url TEXT,
		documentation TEXT,
		chains_supported JSONB DEFAULT '[]',
		status VARCHAR(20) DEFAULT 'pending',
		featured BOOLEAN DEFAULT false,
		view_count INTEGER DEFAULT 0,
		install_count INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	-- Partner Profiles
	CREATE TABLE IF NOT EXISTS partner_profiles (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL UNIQUE,
		company_name VARCHAR(255) NOT NULL,
		logo_url TEXT,
		banner_url TEXT,
		description TEXT,
		website TEXT,
		contact_email VARCHAR(255),
		phone VARCHAR(50),
		location VARCHAR(255),
		established INTEGER,
		employee_count VARCHAR(50),
		business_type VARCHAR(50),
		certifications JSONB DEFAULT '[]',
		social_links JSONB DEFAULT '{}',
		rating DECIMAL(3, 2) DEFAULT 0,
		review_count INTEGER DEFAULT 0,
		total_installs INTEGER DEFAULT 0,
		active_white_labels INTEGER DEFAULT 0,
		status VARCHAR(20) DEFAULT 'pending',
		verified BOOLEAN DEFAULT false,
		kyc_completed BOOLEAN DEFAULT false,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	-- Reviews
	CREATE TABLE IF NOT EXISTS reviews (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		listing_id UUID NOT NULL REFERENCES marketplace_listings(id),
		user_id UUID NOT NULL,
		rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
		title VARCHAR(255),
		content TEXT,
		pros JSONB DEFAULT '[]',
		cons JSONB DEFAULT '[]',
		helpful_count INTEGER DEFAULT 0,
		response TEXT,
		response_at TIMESTAMP,
		status VARCHAR(20) DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	-- Transactions
	CREATE TABLE IF NOT EXISTS marketplace_transactions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		type VARCHAR(50) NOT NULL,
		listing_id UUID REFERENCES marketplace_listings(id),
		package_id UUID REFERENCES service_packages(id),
		buyer_id UUID NOT NULL,
		seller_id UUID NOT NULL,
		amount DECIMAL(20, 8) NOT NULL,
		currency VARCHAR(10) NOT NULL,
		status VARCHAR(20) DEFAULT 'pending',
		payment_method VARCHAR(50),
		payment_id VARCHAR(255),
		metadata JSONB DEFAULT '{}',
		completed_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT NOW()
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_listings_category ON marketplace_listings(category);
	CREATE INDEX IF NOT EXISTS idx_listings_status ON marketplace_listings(status);
	CREATE INDEX IF NOT EXISTS idx_listings_partner ON marketplace_listings(partner_id);
	CREATE INDEX IF NOT EXISTS idx_reviews_listing ON reviews(listing_id);
	CREATE INDEX IF NOT EXISTS idx_transactions_buyer ON marketplace_transactions(buyer_id);
	CREATE INDEX IF NOT EXISTS idx_transactions_seller ON marketplace_transactions(seller_id);
	`

	_, err := dbPool.Exec(context.Background(), schema)
	return err
}

// Middleware
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			c.Abort()
			return
		}

		// Verify JWT token
		claims, err := verifyJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("userID", claims["user_id"])
		c.Set("role", claims["role"])
		c.Next()
	}
}

func verifyJWT(token string) (map[string]interface{}, error) {
	// Simplified JWT verification
	return map[string]interface{}{
		"user_id": "user-123",
		"role":    "admin",
	}, nil
}

func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := fmt.Sprintf("ratelimit:%s", ip)

		count, err := redisClient.Incr(context.Background(), key).Result()
		if err != nil {
			c.Next()
			return
		}

		if count == 1 {
			redisClient.Expire(context.Background(), key, time.Minute)
		}

		if count > 100 {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// API Handlers

// Get all service packages
func GetPackages(c *gin.Context) {
	var packages []ServicePackage
	rows, err := dbPool.Query(context.Background(), `
		SELECT id, name, description, type, price, currency, billing_cycle, 
		       features, max_users, max_transactions, api_quota, support_level,
		       custom_branding, status, created_at, updated_at
		FROM service_packages 
		WHERE status = 'active'
		ORDER BY price ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var pkg ServicePackage
		var featuresJSON, whiteLabelID sql.NullString
		err := rows.Scan(
			&pkg.ID, &pkg.Name, &pkg.Description, &pkg.Type, &pkg.Price, &pkg.Currency,
			&pkg.BillingCycle, &featuresJSON, &pkg.MaxUsers, &pkg.MaxTransactions,
			&pkg.APIQuota, &pkg.SupportLevel, &pkg.CustomBranding, &pkg.Status,
			&pkg.CreatedAt, &pkg.UpdatedAt,
		)
		if err != nil {
			continue
		}
		if featuresJSON.Valid {
			json.Unmarshal([]byte(featuresJSON.String), &pkg.Features)
		}
		if whiteLabelID.Valid {
			pkg.WhiteLabelID = whiteLabelID.String
		}
		packages = append(packages, pkg)
	}

	c.JSON(http.StatusOK, gin.H{"packages": packages})
}

// Create service package
func CreatePackage(c *gin.Context) {
	var pkg ServicePackage
	if err := c.ShouldBindJSON(&pkg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pkg.ID = generateUUID()
	pkg.Status = "active"
	pkg.CreatedAt = time.Now()
	pkg.UpdatedAt = time.Now()

	featuresJSON, _ := json.Marshal(pkg.Features)

	_, err := dbPool.Exec(context.Background(), `
		INSERT INTO service_packages (id, name, description, type, price, currency, billing_cycle,
			features, max_users, max_transactions, api_quota, support_level, custom_branding, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, pkg.ID, pkg.Name, pkg.Description, pkg.Type, pkg.Price, pkg.Currency, pkg.BillingCycle,
		featuresJSON, pkg.MaxUsers, pkg.MaxTransactions, pkg.APIQuota, pkg.SupportLevel,
		pkg.CustomBranding, pkg.Status, pkg.CreatedAt, pkg.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"package": pkg})
}

// Get marketplace listings
func GetListings(c *gin.Context) {
	category := c.Query("category")
	status := c.Query("status")
	featured := c.Query("featured")

	query := `
		SELECT l.id, l.partner_id, p.company_name, p.logo_url, p.rating, p.review_count,
		       l.title, l.description, l.category, l.tags, l.price, l.currency, l.pricing_type,
		       l.features, l.demo_url, l.website_url, l.documentation, l.chains_supported,
		       l.status, l.featured, l.view_count, l.install_count, l.created_at, l.updated_at
		FROM marketplace_listings l
		JOIN partner_profiles p ON l.partner_id = p.id
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 0

	if category != "" {
		argCount++
		query += fmt.Sprintf(" AND l.category = $%d", argCount)
		args = append(args, category)
	}
	if status != "" {
		argCount++
		query += fmt.Sprintf(" AND l.status = $%d", argCount)
		args = append(args, status)
	} else {
		query += " AND l.status = 'active'"
	}
	if featured == "true" {
		query += " AND l.featured = true"
	}

	query += " ORDER BY l.created_at DESC"

	rows, err := dbPool.Query(context.Background(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var listings []MarketplaceListing
	for rows.Next() {
		var listing MarketplaceListing
		var tagsJSON, featuresJSON, chainsJSON sql.NullString
		err := rows.Scan(
			&listing.ID, &listing.PartnerID, &listing.PartnerName, &listing.PartnerLogo,
			&listing.PartnerRating, &listing.PartnerReviews, &listing.Title, &listing.Description,
			&listing.Category, &tagsJSON, &listing.Price, &listing.Currency, &listing.PricingType,
			&featuresJSON, &listing.DemoURL, &listing.WebsiteURL, &listing.Documentation,
			&chainsJSON, &listing.Status, &listing.Featured, &listing.ViewCount,
			&listing.InstallCount, &listing.CreatedAt, &listing.UpdatedAt,
		)
		if err != nil {
			continue
		}
		if tagsJSON.Valid {
			json.Unmarshal([]byte(tagsJSON.String), &listing.Tags)
		}
		if featuresJSON.Valid {
			json.Unmarshal([]byte(featuresJSON.String), &listing.Features)
		}
		if chainsJSON.Valid {
			json.Unmarshal([]byte(chainsJSON.String), &listing.ChainsSupported)
		}
		listings = append(listings, listing)
	}

	c.JSON(http.StatusOK, gin.H{"listings": listings})
}

// Create marketplace listing
func CreateListing(c *gin.Context) {
	var listing MarketplaceListing
	if err := c.ShouldBindJSON(&listing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("userID")
	
	// Get partner profile
	var partner PartnerProfile
	err := dbPool.QueryRow(context.Background(), `
		SELECT id, company_name, logo_url, rating, review_count 
		FROM partner_profiles WHERE user_id = $1
	`, userID).Scan(&partner.ID, &partner.CompanyName, &partner.LogoURL, &partner.Rating, &partner.ReviewCount)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Partner profile not found"})
		return
	}

	listing.ID = generateUUID()
	listing.PartnerID = partner.ID
	listing.PartnerName = partner.CompanyName
	listing.PartnerLogo = partner.LogoURL
	listing.PartnerRating = partner.Rating
	listing.PartnerReviews = partner.ReviewCount
	listing.Status = "pending"
	listing.CreatedAt = time.Now()
	listing.UpdatedAt = time.Now()

	tagsJSON, _ := json.Marshal(listing.Tags)
	featuresJSON, _ := json.Marshal(listing.Features)
	chainsJSON, _ := json.Marshal(listing.ChainsSupported)

	_, err = dbPool.Exec(context.Background(), `
		INSERT INTO marketplace_listings (id, partner_id, title, description, category, tags, price,
			currency, pricing_type, features, demo_url, website_url, documentation, chains_supported,
			status, featured, view_count, install_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`, listing.ID, listing.PartnerID, listing.Title, listing.Description, listing.Category, tagsJSON,
		listing.Price, listing.Currency, listing.PricingType, featuresJSON, listing.DemoURL,
		listing.WebsiteURL, listing.Documentation, chainsJSON, listing.Status, listing.Featured,
		listing.ViewCount, listing.InstallCount, listing.CreatedAt, listing.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Invalidate cache
	marketplaceCache.InvalidateListings()

	c.JSON(http.StatusCreated, gin.H{"listing": listing})
}

// Get partner profile
func GetPartnerProfile(c *gin.Context) {
	userID := c.GetString("userID")
	
	var profile PartnerProfile
	var certificationsJSON, socialLinksJSON sql.NullString

	err := dbPool.QueryRow(context.Background(), `
		SELECT id, user_id, company_name, logo_url, banner_url, description, website,
			contact_email, phone, location, established, employee_count, business_type,
			certifications, social_links, rating, review_count, total_installs, active_white_labels,
			status, verified, kyc_completed, created_at, updated_at
		FROM partner_profiles WHERE user_id = $1
	`, userID).Scan(
		&profile.ID, &profile.UserID, &profile.CompanyName, &profile.LogoURL, &profile.BannerURL,
		&profile.Description, &profile.Website, &profile.ContactEmail, &profile.Phone, &profile.Location,
		&profile.Established, &profile.EmployeeCount, &profile.BusinessType, &certificationsJSON,
		&socialLinksJSON, &profile.Rating, &profile.ReviewCount, &profile.TotalInstalls,
		&profile.ActiveWhiteLabels, &profile.Status, &profile.Verified, &profile.KYCCompleted,
		&profile.CreatedAt, &profile.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
		return
	}

	if certificationsJSON.Valid {
		json.Unmarshal([]byte(certificationsJSON.String), &profile.Certifications)
	}
	if socialLinksJSON.Valid {
		json.Unmarshal([]byte(socialLinksJSON.String), &profile.SocialLinks)
	}

	c.JSON(http.StatusOK, gin.H{"profile": profile})
}

// Create/update partner profile
func UpsertPartnerProfile(c *gin.Context) {
	var profile PartnerProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("userID")
	profile.UserID = userID
	profile.Status = "pending"
	profile.CreatedAt = time.Now()
	profile.UpdatedAt = time.Now()

	certificationsJSON, _ := json.Marshal(profile.Certifications)
	socialLinksJSON, _ := json.Marshal(profile.SocialLinks)

	_, err := dbPool.Exec(context.Background(), `
		INSERT INTO partner_profiles (user_id, company_name, logo_url, banner_url, description,
			website, contact_email, phone, location, established, employee_count, business_type,
			certifications, social_links, status, verified, kyc_completed, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		ON CONFLICT (user_id) DO UPDATE SET
			company_name = EXCLUDED.company_name,
			logo_url = EXCLUDED.logo_url,
			banner_url = EXCLUDED.banner_url,
			description = EXCLUDED.description,
			website = EXCLUDED.website,
			contact_email = EXCLUDED.contact_email,
			phone = EXCLUDED.phone,
			location = EXCLUDED.location,
			established = EXCLUDED.established,
			employee_count = EXCLUDED.employee_count,
			business_type = EXCLUDED.business_type,
			certifications = EXCLUDED.certifications,
			social_links = EXCLUDED.social_links,
			updated_at = EXCLUDED.updated_at
	`, profile.UserID, profile.CompanyName, profile.LogoURL, profile.BannerURL, profile.Description,
		profile.Website, profile.ContactEmail, profile.Phone, profile.Location, profile.Established,
		profile.EmployeeCount, profile.BusinessType, certificationsJSON, socialLinksJSON,
		profile.Status, profile.Verified, profile.KYCCompleted, profile.CreatedAt, profile.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"profile": profile})
}

// Get reviews for a listing
func GetReviews(c *gin.Context) {
	listingID := c.Param("id")

	var reviews []Review
	rows, err := dbPool.Query(context.Background(), `
		SELECT id, listing_id, user_id, user_name, rating, title, content, pros, cons,
			helpful_count, response, response_at, status, created_at, updated_at
		FROM reviews 
		WHERE listing_id = $1 AND status = 'approved'
		ORDER BY helpful_count DESC, created_at DESC
	`, listingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var review Review
		var prosJSON, consJSON, response sql.NullString
		err := rows.Scan(
			&review.ID, &review.ListingID, &review.UserID, &review.UserName, &review.Rating,
			&review.Title, &review.Content, &prosJSON, &consJSON, &review.HelpfulCount,
			&response, &review.ResponseAt, &review.Status, &review.CreatedAt, &review.UpdatedAt,
		)
		if err != nil {
			continue
		}
		if prosJSON.Valid {
			json.Unmarshal([]byte(prosJSON.String), &review.Pros)
		}
		if consJSON.Valid {
			json.Unmarshal([]byte(consJSON.String), &review.Cons)
		}
		if response.Valid {
			review.Response = response.String
		}
		reviews = append(reviews, review)
	}

	c.JSON(http.StatusOK, gin.H{"reviews": reviews})
}

// Create review
func CreateReview(c *gin.Context) {
	var review Review
	if err := c.ShouldBindJSON(&review); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("userID")
	review.ID = generateUUID()
	review.UserID = userID
	review.Status = "pending"
	review.CreatedAt = time.Now()
	review.UpdatedAt = time.Now()

	prosJSON, _ := json.Marshal(review.Pros)
	consJSON, _ := json.Marshal(review.Cons)

	_, err := dbPool.Exec(context.Background(), `
		INSERT INTO reviews (id, listing_id, user_id, rating, title, content, pros, cons, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, review.ID, review.ListingID, review.UserID, review.Rating, review.Title, review.Content,
		prosJSON, consJSON, review.Status, review.CreatedAt, review.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"review": review})
}

// Purchase package
func PurchasePackage(c *gin.Context) {
	var req struct {
		PackageID     string `json:"packageId" binding:"required"`
		PaymentMethod string `json:"paymentMethod" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("userID")

	// Get package details
	var pkg ServicePackage
	var featuresJSON sql.NullString
	err := dbPool.QueryRow(context.Background(), `
		SELECT id, name, type, price, currency FROM service_packages WHERE id = $1
	`, req.PackageID).Scan(&pkg.ID, &pkg.Name, &pkg.Type, &pkg.Price, &pkg.Currency)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Package not found"})
		return
	}

	// Create transaction
	txn := Transaction{
		ID:            generateUUID(),
		Type:          "purchase",
		PackageID:     pkg.ID,
		BuyerID:       userID,
		Amount:        pkg.Price,
		Currency:      pkg.Currency,
		Status:        "pending",
		PaymentMethod: req.PaymentMethod,
		CreatedAt:     time.Now(),
	}

	// In production, this would integrate with payment gateway
	// For now, mark as completed
	txn.Status = "completed"
	now := time.Now()
	txn.CompletedAt = &now

	_, err = dbPool.Exec(context.Background(), `
		INSERT INTO marketplace_transactions (id, type, package_id, buyer_id, amount, currency, status, payment_method, completed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, txn.ID, txn.Type, txn.PackageID, txn.BuyerID, txn.Amount, txn.Currency, txn.Status, txn.PaymentMethod, txn.CompletedAt, txn.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transaction": txn})
}

// Get marketplace statistics
func GetMarketplaceStats(c *gin.Context) {
	var stats struct {
		TotalListings     int     `json:"totalListings"`
		ActivePartners    int     `json:"activePartners"`
		TotalTransactions float64 `json:"totalTransactions"`
		TotalRevenue      float64 `json:"totalRevenue"`
		AverageRating    float64 `json:"averageRating"`
	}

	dbPool.QueryRow(context.Background(), `
		SELECT COUNT(*), SUM(amount) FROM marketplace_transactions WHERE status = 'completed'
	`).Scan(&stats.TotalListings, &stats.TotalRevenue)

	dbPool.QueryRow(context.Background(), `
		SELECT COUNT(*), COALESCE(AVG(rating), 0) FROM partner_profiles WHERE status = 'active'
	`).Scan(&stats.ActivePartners, &stats.AverageRating)

	dbPool.QueryRow(context.Background(), `
		SELECT COALESCE(SUM(amount), 0) FROM marketplace_transactions WHERE status = 'completed'
	`).Scan(&stats.TotalTransactions)

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// Search listings
func SearchListings(c *gin.Context) {
	query := c.Query("q")
	category := c.Query("category")

	sqlQuery := `
		SELECT l.id, l.partner_id, p.company_name, p.logo_url, p.rating, p.review_count,
		       l.title, l.description, l.category, l.tags, l.price, l.currency, l.pricing_type,
		       l.features, l.chains_supported, l.status, l.featured, l.view_count, l.install_count
		FROM marketplace_listings l
		JOIN partner_profiles p ON l.partner_id = p.id
		WHERE l.status = 'active'
	`
	args := []interface{}{}
	argCount := 0

	if query != "" {
		argCount++
		sqlQuery += fmt.Sprintf(" AND (l.title ILIKE $%d OR l.description ILIKE $%d OR l.tags::text ILIKE $%d)", argCount, argCount, argCount)
		args = append(args, "%"+query+"%")
	}

	if category != "" {
		argCount++
		sqlQuery += fmt.Sprintf(" AND l.category = $%d", argCount)
		args = append(args, category)
	}

	sqlQuery += " ORDER BY l.view_count DESC, l.install_count DESC LIMIT 50"

	rows, err := dbPool.Query(context.Background(), sqlQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var listings []MarketplaceListing
	for rows.Next() {
		var listing MarketplaceListing
		var tagsJSON, featuresJSON, chainsJSON sql.NullString
		err := rows.Scan(
			&listing.ID, &listing.PartnerID, &listing.PartnerName, &listing.PartnerLogo,
			&listing.PartnerRating, &listing.PartnerReviews, &listing.Title, &listing.Description,
			&listing.Category, &tagsJSON, &listing.Price, &listing.Currency, &listing.PricingType,
			&featuresJSON, &chainsJSON, &listing.Status, &listing.Featured,
			&listing.ViewCount, &listing.InstallCount,
		)
		if err != nil {
			continue
		}
		if tagsJSON.Valid {
			json.Unmarshal([]byte(tagsJSON.String), &listing.Tags)
		}
		if featuresJSON.Valid {
			json.Unmarshal([]byte(featuresJSON.String), &listing.Features)
		}
		if chainsJSON.Valid {
			json.Unmarshal([]byte(chainsJSON.String), &listing.ChainsSupported)
		}
		listings = append(listings, listing)
	}

	c.JSON(http.StatusOK, gin.H{"listings": listings})
}

// Helper functions
func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func hashPassword(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash)
}

func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateAPIKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateAPISecret() string {
	b := make([]byte, 64)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hmacSHA256(key, message string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// Setup router
func setupRouter() *gin.Engine {
	r := gin.Default()

	// Middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(RateLimitMiddleware())

	// CORS
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

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "timestamp": time.Now().Unix()})
	})

	// API v1
	v1 := r.Group("/api/v1")
	{
		// Public endpoints
		v1.GET("/marketplace/listings", GetListings)
		v1.GET("/marketplace/listings/search", SearchListings)
		v1.GET("/marketplace/listings/:id/reviews", GetReviews)
		v1.GET("/marketplace/packages", GetPackages)
		v1.GET("/marketplace/stats", GetMarketplaceStats)

		// Protected endpoints
		authorized := v1.Group("")
		authorized.Use(AuthMiddleware())
		{
			// Package management
			authorized.POST("/marketplace/packages", CreatePackage)
			authorized.POST("/marketplace/packages/:id/purchase", PurchasePackage)

			// Listing management
			authorized.POST("/marketplace/listings", CreateListing)

			// Partner profile
			authorized.GET("/partner/profile", GetPartnerProfile)
			authorized.PUT("/partner/profile", UpsertPartnerProfile)

			// Reviews
			authorized.POST("/marketplace/reviews", CreateReview)
		}
	}

	return r
}

func main() {
	// Initialize logger
	initLogger()
	logger.Info().Msg("Starting TigerWallet White Label Marketplace")

	// Load configuration
	config := Config{
		Port:         getEnv("PORT", "8085"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://tigerwallet:tigerpass@localhost:5432/tigerwallet?sslmode=disable"),
		RedisURL:     getEnv("REDIS_URL", "localhost:6379"),
		JWTSecret:    getEnv("JWT_SECRET", "tigerwallet-secret-key"),
		AdminEmail:   getEnv("ADMIN_EMAIL", "admin@tigerwallet.com"),
	}

	// Initialize database
	var err error
	dbPool, err = initDatabase(config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer dbPool.Close()

	// Create schema
	if err := createMarketplaceSchema(); err != nil {
		logger.Warn().Err(err).Msg("Schema creation warning")
	}

	// Initialize Redis
	redisClient, err = initRedis(config.RedisURL)
	if err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize cache
	marketplaceCache = NewMarketplaceCache(redisClient)

	// Initialize WebSocket hub
	wsHub = NewWebSocketHub()
	go wsHub.Run()

	// Setup router
	router := setupRouter()

	// Start server
	srv := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	logger.Info().Msg("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
