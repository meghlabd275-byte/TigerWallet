package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Port       string
	RedisURL   string
	FeePercent float64
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type Project struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description  string    `json:"description"`
	Token        string    `json:"token"`
	TokenAddress string    `json:"tokenAddress"`
	SoftCap      float64   `json:"softCap"`
	HardCap      float64   `json:"hardCap"`
	MinBuy       float64   `json:"minBuy"`
	MaxBuy       float64   `json:"maxBuy"`
	Price        float64   `json:"price"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	Status      string    `json:"status"`
	TotalRaised float64   `json:"totalRaised"`
	Participants int     `json:"participants"`
	Logo        string    `json:"logo"`
	Website     string    `json:"website"`
}

type Allocation struct {
	ID         string     `json:"id"`
	ProjectID  string    `json:"projectId"`
	User       string    `json:"user"`
	Amount     float64   `json:"amount"`
	Tokens     float64   `json:"tokens"`
	Status     string    `json:"status"`
	ClaimedAt  *time.Time `json:"claimedAt"`
	CreatedAt  time.Time `json:"createdAt"`
}

type LaunchpadService struct {
	config *Config
	redis  *redis.Client
}

func NewLaunchpadService(cfg *Config) (*LaunchpadService, error) {
	r := redis.NewClient(&redis.Options{Addr: cfg.RedisURL, Password: "", DB: 0})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &LaunchpadService{config: cfg, redis: r}, nil
}

func (s *LaunchpadService) CreateProject(ctx context.Context, p Project) (*Project, error) {
	p.ID = uuid.New().String()
	p.Status = "upcoming"
	p.TotalRaised = 0
	p.Participants = 0
	data, _ := json.Marshal(p)
	s.redis.Set(ctx, "launchpad:project:"+p.ID, data, 0)
	return &p, nil
}

func (s *LaunchpadService) GetProjects(ctx context.Context, status string) []Project {
	keys, _ := s.redis.Keys(ctx, "launchpad:project:*").Result()
	var projects []Project
	for _, key := range keys {
		data, _ := s.redis.Get(ctx, key).Bytes()
		var p Project
		json.Unmarshal(data, &p)
		if status == "" || p.Status == status {
			projects = append(projects, p)
		}
	}
	return projects
}

func (s *LaunchpadService) Participate(ctx context.Context, projectID, user string, amount float64) (*Allocation, error) {
	pData, _ := s.redis.Get(ctx, "launchpad:project:"+projectID).Bytes()
	var project Project
	json.Unmarshal(pData, &project)

	if amount < project.MinBuy || amount > project.MaxBuy {
		return nil, fmt.Errorf("invalid amount")
	}

	tokens := amount / project.Price
	alloc := &Allocation{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		User:      user,
		Amount:    amount,
		Tokens:    tokens,
		Status:    "confirmed",
		CreatedAt: time.Now(),
	}

	data, _ := json.Marshal(alloc)
	s.redis.Set(ctx, "launchpad:allocation:"+alloc.ID, data, 0)

	project.TotalRaised += amount
	project.Participants++
	projData, _ := json.Marshal(project)
	s.redis.Set(ctx, "launchpad:project:"+projectID, projData, 0)

	return alloc, nil
}

func main() {
	cfg := &Config{
		Port:       getEnv("PORT", "8080"),
		RedisURL:   getEnv("REDIS_URL", "localhost:6379"),
		FeePercent: 2.0,
	}

	svc, _ := NewLaunchpadService(cfg)

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	api := r.Group("/api/v1")
	{
		api.POST("/projects", func(c *gin.Context) {
			var p Project
			c.ShouldBindJSON(&p)
			result, _ := svc.CreateProject(c.Request.Context(), p)
			c.JSON(200, result)
		})
		api.GET("/projects", func(c *gin.Context) {
			status := c.Query("status")
			c.JSON(200, svc.GetProjects(c.Request.Context(), status))
		})
		api.POST("/participate", func(c *gin.Context) {
			var req struct {
				ProjectID string  `json:"projectId"`
				User      string  `json:"user"`
				Amount    float64 `json:"amount"`
			}
			c.ShouldBindJSON(&req)
			result, _ := svc.Participate(c.Request.Context(), req.ProjectID, req.User, req.Amount)
			c.JSON(200, result)
		})
		api.POST("/claim/:id", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "claimed"})
		})
	}

	log.Printf("Starting Launchpad on %s", cfg.Port)
	r.Run(":" + cfg.Port)
}
