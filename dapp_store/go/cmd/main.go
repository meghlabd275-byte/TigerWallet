package main

import (
	"context"
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
	"github.com/google/uuid"
)

// ============ Configuration ============

type Config struct {
	Port          string
	RedisURL      string
	IPFSGateway   string
	FeaturedDays int
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============ Models ============

type DApp struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	IconURL     string   `json:"iconUrl"`
	WebsiteURL  string   `json:"websiteUrl"`
	AppURL      string   `json:"appUrl"`
	Chains      []string `json:"chains"`
	Contracts   []string `json:"contracts"`
	Tags        []string `json:"tags"`
	Developer   Developer `json:"developer"`
	Metrics     Metrics  `json:"metrics"`
	Status      string   `json:"status"`
	Verified    bool     `json:"verified"`
	Featured    bool     `json:"featured"`
	CreatedAt   int64    `json:"createdAt"`
	UpdatedAt   int64    `json:"updatedAt"`
}

type Developer struct {
	Name        string `json:"name"`
	Website     string `json:"website"`
	Email       string `json:"email"`
	Twitter     string `json:"twitter"`
	GitHub      string `json:"github"`
}

type Metrics struct {
	Users        int64   `json:"users"`
	Transactions int64   `json:"transactions"`
	Volume24h    float64 `json:"volume24h"`
	Volume7d    float64 `json:"volume7d"`
	Rating       float64 `json:"rating"`
	Reviews      int     `json:"reviews"`
}

type Review struct {
	ID          string   `json:"id"`
	DAppID      string   `json:"dappId"`
	UserID      string   `json:"userId"`
	Rating      int      `json:"rating"`
	Comment     string   `json:"comment"`
	CreatedAt   int64    `json:"createdAt"`
}

type Category struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	AppCount    int    `json:"appCount"`
}

type DAppSubmission struct {
	ID          string    `json:"id"`
	DApp        DApp      `json:"dapp"`
	SubmittedBy string    `json:"submittedBy"`
	Status      string    `json:"status"` // pending, approved, rejected
	SubmittedAt int64     `json:"submittedAt"`
	ReviewedAt  *int64   `json:"reviewedAt"`
	Reviewer    string    `json:"reviewer"`
}

// ============ Service ============

type DAppStoreService struct {
	config *Config
	dapps  map[string]*DApp
}

func NewDAppStoreService(config *Config) *DAppStoreService {
	return &DAppStoreService{
		config: config,
		dapps:  make(map[string]*DApp),
	}
}

// Initialize with popular DApps
func (s *DAppStoreService) Initialize() {
	// DeFi
	s.AddDApp(&DApp{
		ID:          "uniswap",
		Name:        "Uniswap",
		Description: "Decentralized trading protocol",
		Category:    "DeFi",
		IconURL:     "https://uniswap.org/favicon.ico",
		WebsiteURL:  "https://uniswap.org",
		AppURL:      "https://app.uniswap.org",
		Chains:      []string{"ethereum", "arbitrum", "optimism", "polygon"},
		Tags:        []string{"swap", "defi", "dex"},
		Developer: Developer{
			Name:     "Uniswap Labs",
			Twitter:  "@uniswap",
			GitHub:   "Uniswap",
		},
		Metrics: Metrics{
			Users:        3500000,
			Transactions: 15000000,
			Volume24h:    500000000,
			Rating:       4.8,
			Reviews:      5000,
		},
		Status:   "active",
		Verified: true,
		Featured: true,
	})

	s.AddDApp(&DApp{
		ID:          "aave",
		Name:        "Aave",
		Description: "Non-custodial liquidity protocol",
		Category:    "DeFi",
		IconURL:     "https://aave.com/favicon.ico",
		WebsiteURL:  "https://aave.com",
		AppURL:      "https://app.aave.com",
		Chains:      []string{"ethereum", "polygon", "arbitrum", "optimism"},
		Tags:        []string{"lending", "borrow", "defi"},
		Developer: Developer{
			Name:     "Aave Labs",
			Twitter:  "@aaveaave",
			GitHub:   "aave",
		},
		Metrics: Metrics{
			Users:        150000,
			Transactions: 5000000,
			Volume24h:    200000000,
			Rating:       4.7,
			Reviews:      2000,
		},
		Status:   "active",
		Verified: true,
		Featured: true,
	})

	// NFTs
	s.AddDApp(&DApp{
		ID:          "opensea",
		Name:        "OpenSea",
		Description: "Digital marketplace for NFTs",
		Category:    "NFT",
		IconURL:     "https://opensea.io/favicon.ico",
		WebsiteURL:  "https://opensea.io",
		AppURL:      "https://opensea.io",
		Chains:      []string{"ethereum", "polygon", "arbitrum", "optimism", "bsc"},
		Tags:        []string{"nft", "marketplace", "collectibles"},
		Developer: OpenSeaDev{
			Foundation, "OpenSea", "@opensea", "opensea",
		},
		Metrics: Metrics{
			Users:        2000000,
			Transactions: 10000000,
			Volume24h:    50000000,
			Rating:       4.5,
			Reviews:      10000,
		},
		Status:   "active",
		Verified: true,
		Featured: true,
	})

	s.AddDApp(&DApp{
		ID:          "blur",
		Name:        "Blur",
		Description: "NFT marketplace for pro traders",
		Category:    "NFT",
		IconURL:     "https://blur.io/favicon.ico",
		WebsiteURL:  "https://blur.io",
		AppURL:      "https://blur.io",
		Chains:      []string{"ethereum"},
		Tags:        []string{"nft", "trading", "blur"},
		Developer: Developer{
			Name:     "Blur Foundation",
			Twitter:  "@blur_io",
		},
		Metrics: Metrics{
			Users:        500000,
			Transactions: 2000000,
			Volume24h:    100000000,
			Rating:       4.6,
			Reviews:      1500,
		},
		Status:   "active",
		Verified: true,
		Featured: false,
	})

	// Games
	s.AddDApp(&DApp{
		ID:          "axie-infinity",
		Name:        "Axie Infinity",
		Description: "Play-to-earn blockchain game",
		Category:    "GameFi",
		IconURL:     "https://axieinfinity.com/favicon.ico",
		WebsiteURL:  "https://axieinfinity.com",
		AppURL:      "https://app.axieinfinity.com",
		Chains:      []string{"ronin", "ethereum"},
		Tags:        []string{"game", "nft", "play2earn"},
		Developer: Developer{
			Name:     "Sky Mavis",
			Twitter:  "@SkyMavis",
			GitHub:   "axieinfinity",
		},
		Metrics: Metrics{
			Users:        1000000,
			Transactions: 5000000,
			Volume24h:    10000000,
			Rating:       4.3,
			Reviews:      3000,
		},
		Status:   "active",
		Verified: true,
		Featured: false,
	})

	// Bridges
	s.AddDApp(&DApp{
		ID:          "layerzero",
		Name:        "LayerZero",
		Description: "Cross-chain messaging protocol",
		Category:    "Bridge",
		IconURL:     "https://layerzero.network/favicon.ico",
		WebsiteURL:  "https://layerzero.network",
		AppURL:      "https://layerzero.network/omnichain",
		Chains:      []string{"ethereum", "avalanche", "polygon", "bsc", "arbitrum", "optimism"},
		Tags:        []string{"bridge", "crosschain", "omnichain"},
		Developer: Developer{
			Name:     "LayerZero Labs",
			Twitter:  "@LayerZero_Labs",
		},
		Metrics: Metrics{
			Users:        500000,
			Transactions: 3000000,
			Volume24h:    150000000,
			Rating:       4.7,
			Reviews:      800,
		},
		Status:   "active",
		Verified: true,
		Featured: true,
	})

	// Tools
	s.AddDApp(&DApp{
		ID:          "dune",
		Name:        "Dune",
		Description: "Blockchain analytics platform",
		Category:    "Analytics",
		IconURL:     "https://dune.com/favicon.ico",
		WebsiteURL:  "https://dune.com",
		AppURL:      "https://dune.com",
		Chains:      []string{"ethereum", "polygon", "bsc", "arbitrum", "optimism"},
		Tags:        []string{"analytics", "data", "dashboard"},
		Developer: Developer{
			Name:     "Dune Analytics",
			Twitter:  "@DuneAnalytics",
		},
		Metrics: Metrics{
			Users:        150000,
			Transactions: 0,
			Volume24h:    0,
			Rating:       4.9,
			Reviews:      500,
		},
		Status:   "active",
		Verified: true,
		Featured: false,
	})
}

type OpenSeaDev struct {
	Developer
	Foundation string `json:"foundation"`
}

func (s *DAppStoreService) AddDApp(dapp *DApp) {
	dapp.ID = strings.ToLower(strings.ReplaceAll(dapp.Name, " ", "-"))
	dapp.CreatedAt = time.Now().Unix()
	dapp.UpdatedAt = time.Now().Unix()
	s.dapps[dapp.ID] = dapp
}

func (s *DAppStoreService) GetDApp(id string) (*DApp, bool) {
	dapp, ok := s.dapps[id]
	return dapp, ok
}

func (s *DAppStoreService) GetAllDApps() []*DApp {
	var dapps []*DApp
	for _, dapp := range s.dapps {
		dapps = append(dapps, dapp)
	}
	return dapps
}

func (s *DAppStoreService) GetDAppsByCategory(category string) []*DApp {
	var dapps []*DApp
	for _, dapp := range s.dapps {
		if dapp.Category == category {
			dapps = append(dapps, dapp)
		}
	}
	return dapps
}

func (s *DAppStoreService) GetFeaturedDApps() []*DApp {
	var dapps []*DApp
	for _, dapp := range s.dapps {
		if dapp.Featured {
			dapps = append(dapps, dapp)
		}
	}
	return dapps
}

func (s *DAppStoreService) SearchDApps(query string) []*DApp {
	query = strings.ToLower(query)
	var dapps []*DApp
	for _, dapp := range s.dapps {
		if strings.Contains(strings.ToLower(dapp.Name), query) ||
			strings.Contains(strings.ToLower(dapp.Description), query) {
			dapps = append(dapps, dapp)
		}
	}
	return dapps
}

func (s *DAppStoreService) GetCategories() []Category {
	categories := make(map[string]int)
	for _, dapp := range s.dapps {
		categories[dapp.Category]++
	}

	var result []Category
	for name, count := range categories {
		result = append(result, Category{
			ID:          strings.ToLower(strings.ReplaceAll(name, " ", "-")),
			Name:        name,
			Description: fmt.Sprintf("Best %s DApps", name),
			AppCount:    count,
		})
	}
	return result
}

func (s *DAppStoreService) GetDAppsByChain(chain string) []*DApp {
	var dapps []*DApp
	for _, dapp := range s.dapps {
		for _, c := range dapp.Chains {
			if c == chain {
				dapps = append(dapps, dapp)
				break
			}
		}
	}
	return dapps
}

func (s *DAppStoreService) SubmitDApp(dapp DApp, userID string) string {
	submissionID := uuid.New().String()
	// In production, would save to database
	log.Printf("DApp submission: %s by user %s", dapp.Name, userID)
	return submissionID
}

func (s *DAppStoreService) AddReview(review Review) {
	// In production, would save to database
	log.Printf("Review added for DApp: %s by user %s", review.DAppID, review.UserID)
}

func (s *DAppStoreService) GetReviews(dappID string) []Review {
	// In production, would fetch from database
	return []Review{}
}

func (s *DAppStoreService) UpdateMetrics(dappID string, metrics Metrics) {
	if dapp, ok := s.dapps[dappID]; ok {
		dapp.Metrics = metrics
		dapp.UpdatedAt = time.Now().Unix()
	}
}

// ============ Handlers ============

type Handler struct {
	service *DAppStoreService
}

func NewHandler(service *DAppStoreService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetDApps(c *gin.Context) {
	category := c.Query("category")
	chain := c.Query("chain")
	featured := c.Query("featured")
	search := c.Query("search")

	var dapps []*DApp

	if search != "" {
		dapps = h.service.SearchDApps(search)
	} else if featured == "true" {
		dapps = h.service.GetFeaturedDApps()
	} else if category != "" {
		dapps = h.service.GetDAppsByCategory(category)
	} else if chain != "" {
		dapps = h.service.GetDAppsByChain(chain)
	} else {
		dapps = h.service.GetAllDApps()
	}

	c.JSON(http.StatusOK, gin.H{
		"dapps": dapps,
		"total": len(dapps),
	})
}

func (h *Handler) GetDApp(c *gin.Context) {
	id := c.Param("id")

	dapp, ok := h.service.GetDApp(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "DApp not found"})
		return
	}

	c.JSON(http.StatusOK, dapp)
}

func (h *Handler) GetCategories(c *gin.Context) {
	categories := h.service.GetCategories()
	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

func (h *Handler) SubmitDApp(c *gin.Context) {
	var req struct {
		DApp    DApp  `json:"dapp" binding:"required"`
		Submitter string `json:"submitter" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := h.service.SubmitDApp(req.DApp, req.Submitter)
	c.JSON(http.StatusOK, gin.H{"submissionId": id, "status": "pending"})
}

func (h *Handler) AddReview(c *gin.Context) {
	var review Review
	if err := c.ShouldBindJSON(&review); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	review.ID = uuid.New().String()
	review.CreatedAt = time.Now().Unix()

	h.service.AddReview(review)

	c.JSON(http.StatusOK, gin.H{"reviewId": review.ID})
}

func (h *Handler) GetReviews(c *gin.Context) {
	dappID := c.Param("id")
	reviews := h.service.GetReviews(dappID)

	c.JSON(http.StatusOK, gin.H{"reviews": reviews})
}

func (h *Handler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query required"})
		return
	}

	dapps := h.service.SearchDApps(query)
	c.JSON(http.StatusOK, gin.H{
		"dapps": dapps,
		"total": len(dapps),
		"query": query,
	})
}

// ============ Main ============

func main() {
	config := &Config{
		Port:          getEnv("PORT", "8080"),
		RedisURL:      getEnv("REDIS_URL", "localhost:6379"),
		IPFSGateway:   getEnv("IPFS_GATEWAY", "https://ipfs.io/ipfs/"),
		FeaturedDays:  7,
	}

	service := NewDAppStoreService(config)
	service.Initialize()

	handler := NewHandler(service)

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "dapps": len(service.dapps)})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// DApps
		api.GET("/dapps", handler.GetDApps)
		api.GET("/dapps/:id", handler.GetDApp)
		api.GET("/categories", handler.GetCategories)
		api.GET("/search", handler.Search)

		// Submissions
		api.POST("/dapps/submit", handler.SubmitDApp)

		// Reviews
		api.POST("/dapps/:id/reviews", handler.AddReview)
		api.GET("/dapps/:id/reviews", handler.GetReviews)
	}

	// Start server
	addr := fmt.Sprintf(":%s", config.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("Starting DApp Store on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
