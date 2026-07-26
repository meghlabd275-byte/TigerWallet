// TigerWallet Launchpad Service
// Token launchpad and IDO platform

package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Config struct {
	Port int
}

var cfg = Config{Port: 8012}

type Project struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Symbol          string    `json:"symbol"`
	Description     string    `json:"description"`
	Logo            string    `json:"logo"`
	Website         string    `json:"website"`
	Chain           string    `json:"chain"`
	TokenAddress    string    `json:"token_address"`
	TotalSupply    string    `json:"total_supply"`
	IDOAllocation  string    `json:"ido_allocation"`
	Price           string    `json:"price"`
	SoftCap         string    `json:"soft_cap"`
	HardCap         string    `json:"hard_cap"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	Participants    int       `json:"participants"`
	RaisedAmount   string    `json:"raised_amount"`
	Status          string    `json:"status"` // upcoming, active, completed, cancelled
}

type Allocation struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	UserID      string    `json:"user_id"`
	Amount      string    `json:"amount"`
	Tokens      string    `json:"tokens"`
	Status      string    `json:"status"` // pending, claimed, refunded
	ClaimedAt  *time.Time `json:"claimed_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type LaunchpadService struct {
	projects    map[string]*Project
	allocations map[string]*Allocation
}

func NewLaunchpadService() *LaunchpadService {
	ls := &LaunchpadService{
		projects:    make(map[string]*Project),
		allocations: make(map[string]*Allocation),
	}
	ls.initData()
	return ls
}

func (ls *LaunchpadService) initData() {
	projects := []*Project{
		{
			ID: "p1", Name: "TigerDeFi", Symbol: "TIGER", Description: "Next-gen DeFi protocol",
			Chain: "ethereum", TokenAddress: "0x123", TotalSupply: "100000000",
			IDOAllocation: "5000000", Price: "0.05", SoftCap: "100000", HardCap: "500000",
			StartTime: time.Now().Add(24 * time.Hour), EndTime: time.Now().Add(72 * time.Hour),
			Participants: 0, RaisedAmount: "0", Status: "upcoming",
		},
		{
			ID: "p2", Name: "ChainLink Pro", Symbol: "CLP", Description: "Advanced oracle solutions",
			Chain: "polygon", TokenAddress: "0x456", TotalSupply: "50000000",
			IDOAllocation: "2500000", Price: "0.02", SoftCap: "50000", HardCap: "200000",
			StartTime: time.Now().Add(-24 * time.Hour), EndTime: time.Now().Add(48 * time.Hour),
			Participants: 250, RaisedAmount: "150000", Status: "active",
		},
	}
	for _, p := range projects {
		ls.projects[p.ID] = p
	}
}

func (ls *LaunchpadService) GetProjects(c *gin.Context) {
	status := c.Query("status")
	projects := make([]*Project, 0)
	for _, p := range ls.projects {
		if status == "" || p.Status == status {
			projects = append(projects, p)
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "projects": projects})
}

func (ls *LaunchpadService) GetProject(c *gin.Context) {
	id := c.Param("id")
	p, ok := ls.projects[id]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "project": p})
}

func (ls *LaunchpadService) Participate(c *gin.Context) {
	var req struct {
		UserID    string `json:"user_id" binding:"required"`
		ProjectID string `json:"project_id" binding:"required"`
		Amount    string `json:"amount" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project, ok := ls.projects[req.ProjectID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	// Calculate tokens
	tokens := fmt.Sprintf("%.0f", 0.0)

	alloc := &Allocation{
		ID:         uuid.New().String(),
		ProjectID:  req.ProjectID,
		UserID:     req.UserID,
		Amount:     req.Amount,
		Tokens:     tokens,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	ls.allocations[alloc.ID] = alloc
	project.Participants++
	project.RaisedAmount = fmt.Sprintf("%.0f", 0.0)

	c.JSON(http.StatusCreated, gin.H{"success": true, "allocation": alloc})
}

func (ls *LaunchpadService) GetUserAllocations(c *gin.Context) {
	userID := c.Param("user_id")
	allocs := make([]*Allocation, 0)
	for _, a := range ls.allocations {
		if a.UserID == userID {
			allocs = append(allocs, a)
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "allocations": allocs})
}

func (ls *LaunchpadService) ClaimTokens(c *gin.Context) {
	allocID := c.Param("id")
	alloc, ok := ls.allocations[allocID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "allocation not found"})
		return
	}

	now := time.Now()
	alloc.Status = "claimed"
	alloc.ClaimedAt = &now

	c.JSON(http.StatusOK, gin.H{"success": true, "allocation": alloc})
}

func (ls *LaunchpadService) CreateProject(c *gin.Context) {
	var req struct {
		Name           string `json:"name" binding:"required"`
		Symbol         string `json:"symbol" binding:"required"`
		Description    string `json:"description"`
		Chain          string `json:"chain" binding:"required"`
		TokenAddress   string `json:"token_address" binding:"required"`
		TotalSupply   string `json:"total_supply" binding:"required"`
		IDOAllocation  string `json:"ido_allocation" binding:"required"`
		Price          string `json:"price" binding:"required"`
		SoftCap        string `json:"soft_cap" binding:"required"`
		HardCap        string `json:"hard_cap" binding:"required"`
		StartTime     int64  `json:"start_time" binding:"required"`
		EndTime       int64  `json:"end_time" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project := &Project{
		ID:             uuid.New().String(),
		Name:           req.Name,
		Symbol:         req.Symbol,
		Description:    req.Description,
		Chain:          req.Chain,
		TokenAddress:   req.TokenAddress,
		TotalSupply:   req.TotalSupply,
		IDOAllocation:  req.IDOAllocation,
		Price:          req.Price,
		SoftCap:        req.SoftCap,
		HardCap:        req.HardCap,
		StartTime:      time.Unix(req.StartTime, 0),
		EndTime:        time.Unix(req.EndTime, 0),
		Status:         "upcoming",
	}

	ls.projects[project.ID] = project

	c.JSON(http.StatusCreated, gin.H{"success": true, "project": project})
}

func main() {
	log.Println("TigerWallet Launchpad Service")
	log.Printf("Starting on port %d", cfg.Port)

	ls := NewLaunchpadService()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "launchpad"})
	})

	api := r.Group("/api/v1/launchpad")
	{
		api.GET("/projects", ls.GetProjects)
		api.GET("/projects/:id", ls.GetProject)
		api.POST("/projects", ls.CreateProject)
		api.POST("/participate", ls.Participate)
		api.GET("/users/:user_id/allocations", ls.GetUserAllocations)
		api.POST("/allocations/:id/claim", ls.ClaimTokens)
	}

	log.Printf("Server starting on :%d", cfg.Port)
	r.Run(fmt.Sprintf(":%d", cfg.Port))
}
