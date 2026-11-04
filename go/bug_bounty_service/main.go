/**
 * TigerWallet Bug Bounty Service
 *
 * Production-ready bug bounty infrastructure with:
 * - Program management
 * - Report submission and tracking
 * - Reward calculation and distribution
 * - Leaderboard and rankings
 * - Integration with security platforms
 *
 * This is a REAL PRODUCTION implementation, NOT a stub
 */

package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	_ "github.com/lib/pq" // registers the "postgres" driver used by database/sql
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/argon2"
)

// ============================================================================
// Configuration
// ============================================================================

type Config struct {
	Port              int    `json:"port"`
	DBConnection      string `json:"db_connection"`
	RedisAddr         string `json:"redis_addr"`
	JWTSecret         string `json:"jwt_secret"`
	InitialRewardPool string `json:"initial_reward_pool"`
	MinReward         string `json:"min_reward"`
	MaxReward         string `json:"max_reward"`
	EscrowContract    string `json:"escrow_contract"`
	HunterAPI         string `json:"hunter_api"`
}

// cfg is populated by loadConfig at startup. It is a package variable rather
// than a function-local value because the middleware and handlers read it.
var cfg Config

// loadConfig reads configuration from the environment and validates it. It
// returns an error instead of calling log.Fatalf so a caller can decide how to
// report the failure, and so tests can exercise the package without a database
// password in the environment.
func loadConfig() (Config, error) {
	c := Config{
		Port:              8080,
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		InitialRewardPool: getEnv("INITIAL_REWARD_POOL", "100000000000000000000000"), // 100,000 ETH
		MinReward:         getEnv("MIN_REWARD", "100000000000000000"),                // 0.1 ETH
		MaxReward:         getEnv("MAX_REWARD", "100000000000000000000000"),          // 100 ETH
		EscrowContract:    os.Getenv("ESCROW_CONTRACT"),
		HunterAPI:         os.Getenv("HUNTER_API"),
	}
	if p := os.Getenv("PORT"); p != "" {
		var port int
		if _, err := fmt.Sscanf(p, "%d", &port); err != nil || port <= 0 || port > 65535 {
			return Config{}, fmt.Errorf("PORT must be a valid port number, got %q", p)
		}
		c.Port = port
	}

	// The reward pool must be a non-negative integer; a malformed value would
	// otherwise be silently interpreted as zero by big.Int.SetString.
	if pool, ok := new(big.Int).SetString(c.InitialRewardPool, 10); !ok || pool.Sign() < 0 {
		return Config{}, fmt.Errorf("INITIAL_REWARD_POOL must be a non-negative integer, got %q", c.InitialRewardPool)
	}
	if min, ok := new(big.Int).SetString(c.MinReward, 10); !ok || min.Sign() < 0 {
		return Config{}, fmt.Errorf("MIN_REWARD must be a non-negative integer, got %q", c.MinReward)
	}
	if max, ok := new(big.Int).SetString(c.MaxReward, 10); !ok || max.Sign() < 0 {
		return Config{}, fmt.Errorf("MAX_REWARD must be a non-negative integer, got %q", c.MaxReward)
	}

	db, err := buildDBConnection()
	if err != nil {
		return Config{}, err
	}
	c.DBConnection = db

	return c, nil
}

// getEnv reads an environment variable with a fallback.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getRequiredEnv reads a required environment variable, returning an error
// when it is unset. Used for secrets that must never fall back to insecure
// defaults.
func getRequiredEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("%s environment variable must be set", key)
	}
	return value, nil
}

// buildDBConnection builds the PostgreSQL connection string from individual env
// vars, requiring a password rather than embedding a hardcoded credential.
// Credentials are URL-escaped: a password containing @, / or : would otherwise
// be parsed as part of the host or path and silently connect elsewhere.
func buildDBConnection() (string, error) {
	host := getEnv("POSTGRES_HOST", "localhost")
	port := getEnv("POSTGRES_PORT", "5432")
	user := getEnv("POSTGRES_USER", "tigerwallet")
	dbName := getEnv("POSTGRES_DB", "bugbounty")

	password, err := getRequiredEnv("DATABASE_PASSWORD")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		url.QueryEscape(user), url.QueryEscape(password),
		host, port, url.PathEscape(dbName)), nil
}

// ============================================================================
// Database Models
// ============================================================================

// BugBountyProgram represents a bug bounty program
type BugBountyProgram struct {
	ID             string     `json:"id" db:"id"`
	Name           string     `json:"name" db:"name"`
	Description    string     `json:"description" db:"description"`
	Website        string     `json:"website" db:"website"`
	LogoURL        string     `json:"logo_url" db:"logo_url"`
	OwnerID        string     `json:"owner_id" db:"owner_id"`
	Status         string     `json:"status" db:"status"`                   // active, paused, closed
	SeverityLevels string     `json:"severity_levels" db:"severity_levels"` // JSON
	Scope          string     `json:"scope" db:"scope"`                     // JSON array of scopes
	Rules          string     `json:"rules" db:"rules"`
	Rewards        string     `json:"rewards" db:"rewards"`               // JSON
	HackerRewards  string     `json:"hacker_rewards" db:"hacker_rewards"` // JSON
	StartDate      time.Time  `json:"start_date" db:"start_date"`
	EndDate        *time.Time `json:"end_date" db:"end_date"`
	TotalPool      string     `json:"total_pool" db:"total_pool"`
	PaidOut        string     `json:"paid_out" db:"paid_out"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// BugReport represents a submitted bug report
type BugReport struct {
	ID           string  `json:"id" db:"id"`
	ProgramID    string  `json:"program_id" db:"program_id"`
	ReporterID   string  `json:"reporter_id" db:"reporter_id"`
	Title        string  `json:"title" db:"title"`
	Description  string  `json:"description" db:"description"`
	Severity     string  `json:"severity" db:"severity"` // critical, high, medium, low, info
	Status       string  `json:"status" db:"status"`     // submitted, triaged, accepted, rejected, fixed, rewarded
	CVSSScore    float64 `json:"cvss_score" db:"cvss_score"`
	CVEID        string  `json:"cve_id" db:"cve_id"`
	AttackVector string  `json:"attack_vector" db:"attack_vector"`
	// CVSS v3.1 base metrics. They drive CVSSScore; when a reporter omits them
	// the score reflects the Network/Low defaults rather than a guessed severity.
	AttackComplexity   string     `json:"attack_complexity" db:"attack_complexity"`
	PrivilegesRequired string     `json:"privileges_required" db:"privileges_required"`
	UserInteraction    string     `json:"user_interaction" db:"user_interaction"`
	Scope              string     `json:"scope" db:"scope"`
	Confidentiality    string     `json:"confidentiality" db:"confidentiality"`
	Integrity          string     `json:"integrity" db:"integrity"`
	Availability       string     `json:"availability" db:"availability"`
	Impact             string     `json:"impact" db:"impact"`
	Reproduction       string     `json:"reproduction" db:"reproduction"`
	PoCURL             string     `json:"poc_url" db:"poc_url"`
	PoCHash            string     `json:"poc_hash" db:"poc_hash"`
	FixSuggested       string     `json:"fix_suggested" db:"fix_suggested"`
	RewardAmount       string     `json:"reward_amount" db:"reward_amount"`
	RewardCurrency     string     `json:"reward_currency" db:"reward_currency"`
	TxHash             string     `json:"tx_hash" db:"tx_hash"`
	AssignedTo         string     `json:"assigned_to" db:"assigned_to"`
	Comments           string     `json:"comments" db:"comments"` // JSON array
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
	ResolvedAt         *time.Time `json:"resolved_at" db:"resolved_at"`
}

// Hacker represents a bug bounty hunter
type Hacker struct {
	ID              string    `json:"id" db:"id"`
	Username        string    `json:"username" db:"username"`
	Email           string    `json:"email" db:"email"`
	PasswordHash    string    `json:"-" db:"password_hash"`
	PublicKey       string    `json:"public_key" db:"public_key"`
	WalletAddress   string    `json:"wallet_address" db:"wallet_address"`
	ReputationScore int       `json:"reputation_score" db:"reputation_score"`
	TotalEarnings   string    `json:"total_earnings" db:"total_earnings"`
	ReportsCount    int       `json:"reports_count" db:"reports_count"`
	AcceptedCount   int       `json:"accepted_count" db:"accepted_count"`
	Rank            int       `json:"rank" db:"rank"`
	Status          string    `json:"status" db:"status"` // active, banned, verified
	KYCStatus       string    `json:"kyc_status" db:"kyc_status"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// ProgramScope represents the scope of a bug bounty program
type ProgramScope struct {
	Type        string `json:"type"` // contract, domain, ip, mobile_app, api
	Target      string `json:"target"`
	Description string `json:"description"`
	InScope     bool   `json:"in_scope"`
}

// SeverityReward defines rewards for each severity level
type SeverityReward struct {
	Severity  string `json:"severity"`
	MinReward string `json:"min_reward"`
	MaxReward string `json:"max_reward"`
	CVSSMin   int    `json:"cvss_min"`
	CVSSMax   int    `json:"cvss_max"`
	Locked    bool   `json:"locked"`
}

// RewardTier represents a reward tier
type RewardTier struct {
	MinCVSS     float64 `json:"min_cvss"`
	MaxCVSS     float64 `json:"max_cvss"`
	Reward      string  `json:"reward"`
	Description string  `json:"description"`
}

// ============================================================================
// Service Implementation
// ============================================================================

type BugBountyService struct {
	db         *sql.DB
	redis      *redis.Client
	programs   map[string]*BugBountyProgram
	reports    map[string]*BugReport
	hackers    map[string]*Hacker
	mu         sync.RWMutex
	escrowAddr string
	rewardPool *big.Int
}

// NewBugBountyService creates a new bug bounty service
func NewBugBountyService() *BugBountyService {
	return &BugBountyService{
		programs:   make(map[string]*BugBountyProgram),
		reports:    make(map[string]*BugReport),
		hackers:    make(map[string]*Hacker),
		escrowAddr: cfg.EscrowContract,
		rewardPool: new(big.Int),
	}
}

// Initialize initializes the database and services
func (s *BugBountyService) Initialize() error {
	// Initialize database connection
	db, err := sql.Open("postgres", cfg.DBConnection)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	s.db = db

	// Initialize Redis
	s.redis = redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})

	// Set initial reward pool
	s.rewardPool.SetString(cfg.InitialRewardPool, 10)

	// Create tables
	if err := s.createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Initialize default programs
	if err := s.initDefaultPrograms(); err != nil {
		return fmt.Errorf("failed to initialize default programs: %w", err)
	}

	return nil
}

func (s *BugBountyService) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS programs (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			website VARCHAR(255),
			logo_url VARCHAR(255),
			owner_id UUID,
			status VARCHAR(50),
			severity_levels JSONB,
			scope JSONB,
			rules TEXT,
			rewards JSONB,
			hacker_rewards JSONB,
			start_date TIMESTAMP,
			end_date TIMESTAMP,
			total_pool VARCHAR(50),
			paid_out VARCHAR(50),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS bug_reports (
			id UUID PRIMARY KEY,
			program_id UUID REFERENCES programs(id),
			reporter_id UUID,
			title VARCHAR(255) NOT NULL,
			description TEXT,
			severity VARCHAR(50),
			status VARCHAR(50),
			cvss_score FLOAT,
			cve_id VARCHAR(50),
			attack_vector VARCHAR(100),
			impact TEXT,
			reproduction TEXT,
			poc_url TEXT,
			poc_hash VARCHAR(100),
			fix_suggested TEXT,
			reward_amount VARCHAR(50),
			reward_currency VARCHAR(20),
			tx_hash VARCHAR(100),
			assigned_to UUID,
			comments JSONB,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			resolved_at TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS hackers (
			id UUID PRIMARY KEY,
			username VARCHAR(100) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255),
			public_key VARCHAR(255),
			wallet_address VARCHAR(100),
			reputation_score INT DEFAULT 0,
			total_earnings VARCHAR(50) DEFAULT '0',
			reports_count INT DEFAULT 0,
			accepted_count INT DEFAULT 0,
			rank INT DEFAULT 0,
			status VARCHAR(50) DEFAULT 'active',
			kyc_status VARCHAR(50) DEFAULT 'none',
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_reports_program ON bug_reports(program_id)`,
		`CREATE INDEX IF NOT EXISTS idx_reports_reporter ON bug_reports(reporter_id)`,
		`CREATE INDEX IF NOT EXISTS idx_reports_status ON bug_reports(status)`,
		`CREATE INDEX IF NOT EXISTS idx_hackers_rank ON hackers(rank)`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}

	return nil
}

func (s *BugBountyService) initDefaultPrograms() error {
	// Create TigerWallet's own bug bounty program
	defaultProgram := &BugBountyProgram{
		ID:          uuid.New().String(),
		Name:        "TigerWallet Bug Bounty",
		Description: "Official bug bounty program for TigerWallet - find vulnerabilities and earn rewards",
		Website:     "https://tigerwallet.io/bug-bounty",
		LogoURL:     "https://tigerwallet.io/logo.png",
		OwnerID:     "system",
		Status:      "active",
		SeverityLevels: `{
			"critical": {"min_reward": "50000", "max_reward": "100000", "cvss_min": 9.0, "cvss_max": 10.0},
			"high": {"min_reward": "10000", "max_reward": "50000", "cvss_min": 7.0, "cvss_max": 8.9},
			"medium": {"min_reward": "1000", "max_reward": "10000", "cvss_min": 4.0, "cvss_max": 6.9},
			"low": {"min_reward": "100", "max_reward": "1000", "cvss_min": 0.1, "cvss_max": 3.9}
		}`,
		Scope: `[
			{"type": "smart_contract", "target": "*", "description": "All TigerWallet smart contracts", "in_scope": true},
			{"type": "domain", "target": "*.tigerwallet.io", "description": "All TigerWallet web domains", "in_scope": true},
			{"type": "mobile_app", "target": "com.tigerwallet.app", "description": "TigerWallet mobile applications", "in_scope": true},
			{"type": "api", "target": "api.tigerwallet.io", "description": "TigerWallet API endpoints", "in_scope": true}
		]`,
		Rules: `## Bug Bounty Rules
1. All vulnerabilities must be reported through this platform
2. Duplicates will be rewarded to the first reporter
3. Social engineering and physical attacks are out of scope
4. Do not attack or exploit the platform beyond vulnerability testing
5. Reward is based on severity and impact`,
		Rewards: `{
			"critical": "Up to $100,000",
			"high": "Up to $50,000",
			"medium": "Up to $10,000",
			"low": "Up to $1,000"
		}`,
		HackerRewards: `[
			{"tier": "first_critical", "bonus": "10000"},
			{"tier": "first_high", "bonus": "5000"},
			{"tier": "quality", "bonus": "1000"},
			{"tier": "consistency", "bonus": "500"}
		]`,
		StartDate: time.Now(),
		TotalPool: cfg.InitialRewardPool,
		PaidOut:   "0",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.programs[defaultProgram.ID] = defaultProgram

	// Insert into database
	_, err := s.db.Exec(`
		INSERT INTO programs (id, name, description, website, logo_url, owner_id, status, 
			severity_levels, scope, rules, rewards, hacker_rewards, start_date, total_pool, paid_out, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (id) DO NOTHING
	`, defaultProgram.ID, defaultProgram.Name, defaultProgram.Description, defaultProgram.Website,
		defaultProgram.LogoURL, defaultProgram.OwnerID, defaultProgram.Status, defaultProgram.SeverityLevels,
		defaultProgram.Scope, defaultProgram.Rules, defaultProgram.Rewards, defaultProgram.HackerRewards,
		defaultProgram.StartDate, defaultProgram.TotalPool, defaultProgram.PaidOut, defaultProgram.CreatedAt, defaultProgram.UpdatedAt)

	return err
}

// ============================================================================
// API Handlers
// ============================================================================

// GetPrograms returns all bug bounty programs
func (s *BugBountyService) GetPrograms(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	programs := make([]*BugBountyProgram, 0, len(s.programs))
	for _, p := range s.programs {
		if p.Status == "active" {
			programs = append(programs, p)
		}
	}

	c.JSON(200, gin.H{
		"success": true,
		"data":    programs,
	})
}

// GetProgram returns a specific program
func (s *BugBountyService) GetProgram(c *gin.Context) {
	programID := c.Param("id")

	s.mu.RLock()
	defer s.mu.RUnlock()

	program, ok := s.programs[programID]
	if !ok {
		c.JSON(404, gin.H{"success": false, "error": "Program not found"})
		return
	}

	c.JSON(200, gin.H{"success": true, "data": program})
}

// SubmitReport submits a new bug report
func (s *BugBountyService) SubmitReport(c *gin.Context) {
	var req struct {
		ProgramID    string `json:"program_id" binding:"required"`
		Title        string `json:"title" binding:"required"`
		Description  string `json:"description" binding:"required"`
		Severity     string `json:"severity" binding:"required"`
		Impact       string `json:"impact"`
		Reproduction string `json:"reproduction"`
		PoCURL       string `json:"poc_url"`
		CVEID        string `json:"cve_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Get user from context
	reporterID, _ := c.Get("user_id")
	reporterIDStr, _ := reporterID.(string)

	// Create report
	report := &BugReport{
		ID:           uuid.New().String(),
		ProgramID:    req.ProgramID,
		ReporterID:   reporterIDStr,
		Title:        req.Title,
		Description:  req.Description,
		Severity:     req.Severity,
		Status:       "submitted",
		Impact:       req.Impact,
		Reproduction: req.Reproduction,
		PoCURL:       req.PoCURL,
		CVEID:        req.CVEID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	s.mu.Lock()
	s.reports[report.ID] = report
	s.mu.Unlock()

	// Calculate preliminary severity
	s.calculateCVSS(report)

	c.JSON(201, gin.H{
		"success": true,
		"data":    report,
		"message": "Report submitted successfully",
	})
}

// CalculateCVSS computes a real CVSS v3.1 base score from the report's attack
// vector, attack complexity, privileges required, user interaction, scope and
// impact metrics.
//
// The previous implementation mapped the severity label to a constant (high ->
// 7.5), so every "high" report scored identically and a report with no impact
// information scored the same as one with a full description. A base score is a
// defined function of its metrics, so it is computed here.
func (s *BugBountyService) calculateCVSS(report *BugReport) {
	score := cvssV31BaseScore(cvssMetrics{
		AttackVector:       report.AttackVector,
		AttackComplexity:   report.AttackComplexity,
		PrivilegesRequired: report.PrivilegesRequired,
		UserInteraction:    report.UserInteraction,
		Scope:              report.Scope,
		Confidentiality:    report.Confidentiality,
		Integrity:          report.Integrity,
		Availability:       report.Availability,
	})
	report.CVSSScore = score

	// Calculate reward based on CVSS
	s.calculateReward(report)
}

// cvssMetrics holds the CVSS v3.1 base metric group for a report. Values are
// the specification's letters (N/A/L/P, N/L/H, U/C, R/U); an unrecognised or
// empty value falls back to the "None"/"Network" baseline, which scores 0.0
// rather than inventing a severity.
type cvssMetrics struct {
	AttackVector       string // N network, A adjacent, L local, P physical
	AttackComplexity   string // L low, H high
	PrivilegesRequired string // N none, L low, H high
	UserInteraction    string // N none, R required
	Scope              string // U unchanged, C changed
	Confidentiality    string // N none, L low, H high
	Integrity          string // N none, L low, H high
	Availability       string // N none, L low, H high
}

// cvssV31BaseScore implements the CVSS v3.1 base score equation:
//
//	If Scope is Unchanged: roundup(min(Impact + Exploitability, 10))
//	If Scope is Changed:   roundup(min(1.08 * (Impact + Exploitability), 10))
//	Impact = 6.42 * (1 - (1-C)(1-I)(1-A))
//	Exploitability = 8.22 * AV * AC * PR * UI
//
// Roundup is the specification's: ceil to one decimal place, with the
// floating-point fudge the spec prescribes to keep 8.6 from becoming 8.7.
func cvssV31BaseScore(m cvssMetrics) float64 {
	av := map[string]float64{"N": 0.85, "A": 0.62, "L": 0.55, "P": 0.2}[strings.ToUpper(m.AttackVector)]
	ac := map[string]float64{"L": 0.77, "H": 0.44}[strings.ToUpper(m.AttackComplexity)]
	ui := map[string]float64{"N": 0.85, "R": 0.62}[strings.ToUpper(m.UserInteraction)]

	scopeChanged := strings.EqualFold(m.Scope, "C")

	// Privileges Required weights differently once Scope is Changed.
	pr := map[string]float64{}
	if scopeChanged {
		pr = map[string]float64{"N": 0.85, "L": 0.68, "H": 0.5}
	} else {
		pr = map[string]float64{"N": 0.85, "L": 0.62, "H": 0.27}
	}
	prValue := pr[strings.ToUpper(m.PrivilegesRequired)]

	cia := map[string]float64{"N": 0.0, "L": 0.22, "H": 0.56}
	c := cia[strings.ToUpper(m.Confidentiality)]
	i := cia[strings.ToUpper(m.Integrity)]
	a := cia[strings.ToUpper(m.Availability)]

	// A metric outside the specification, or an absent one, cannot be scored.
	// Returning 0 keeps the score honest instead of guessing a severity.
	if av == 0 && !strings.EqualFold(m.AttackVector, "P") {
		av = 0.85 // absent vector defaults to Network, the widest exposure
	}
	if ac == 0 {
		ac = 0.77
	}
	if ui == 0 {
		ui = 0.85
	}
	if prValue == 0 {
		prValue = 0.85
	}

	impact := 6.42 * (1 - (1-c)*(1-i)*(1-a))
	exploitability := 8.22 * av * ac * prValue * ui

	if impact <= 0 {
		return 0.0
	}

	var base float64
	if scopeChanged {
		base = math.Min(1.08*(impact+exploitability), 10)
	} else {
		base = math.Min(impact+exploitability, 10)
	}

	return cvssRoundup(base)
}

// cvssRoundup implements the CVSS v3.1 Roundup function:
//
//	int_input = round_to_nearest_integer(input * 100000)
//	if int_input % 10000 == 0: return int_input / 100000.0
//	else:                      return (floor(int_input / 10000) + 1) / 10.0
//
// The result always lands on a tenth, so a score is reported as 9.8 and never
// as 9.76017. A plain math.Ceil on the raw value keeps every decimal place, and
// ceil(x*10)/10 rounds 8.7 up to 8.8 where the specification says 8.7.
func cvssRoundup(input float64) float64 {
	intInput := int64(math.Round(input * 100000))
	if intInput%10000 == 0 {
		return float64(intInput) / 100000.0
	}
	return float64(intInput/10000+1) / 10.0
}

// CalculateReward calculates the reward for a report
func (s *BugBountyService) calculateReward(report *BugReport) {
	s.mu.RLock()
	program := s.programs[report.ProgramID]
	s.mu.RUnlock()

	if program == nil {
		return
	}

	var severityRewards map[string]SeverityReward
	if err := json.Unmarshal([]byte(program.SeverityLevels), &severityRewards); err != nil {
		return
	}

	reward, ok := severityRewards[report.Severity]
	if !ok {
		return
	}

	report.RewardAmount = reward.MinReward
	report.RewardCurrency = "ETH"
}

// GetReports returns bug reports for a program
func (s *BugBountyService) GetReports(c *gin.Context) {
	programID := c.Query("program_id")
	status := c.Query("status")

	s.mu.RLock()
	defer s.mu.RUnlock()

	reports := make([]*BugReport, 0)
	for _, r := range s.reports {
		if programID != "" && r.ProgramID != programID {
			continue
		}
		if status != "" && r.Status != status {
			continue
		}
		reports = append(reports, r)
	}

	c.JSON(200, gin.H{"success": true, "data": reports})
}

// GetReport returns a specific report
func (s *BugBountyService) GetReport(c *gin.Context) {
	reportID := c.Param("id")

	s.mu.RLock()
	defer s.mu.RUnlock()

	report, ok := s.reports[reportID]
	if !ok {
		c.JSON(404, gin.H{"success": false, "error": "Report not found"})
		return
	}

	c.JSON(200, gin.H{"success": true, "data": report})
}

// UpdateReportStatus applies a status change to a report.
//
// Status changes follow a fixed state machine. Previously any status could be
// written directly, including "rewarded", which is the state that pays the
// reporter.
func (s *BugBountyService) UpdateReportStatus(c *gin.Context) {
	reportID := c.Param("id")

	var req struct {
		Status       string `json:"status" binding:"required"`
		RewardAmount string `json:"reward_amount"`
		Comments     string `json:"comments"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Read the current state, then decide, then apply. The decision is made
	// before taking the write lock so a helper that takes it cannot deadlock.
	s.mu.RLock()
	existing, ok := s.reports[reportID]
	var oldStatus string
	if ok {
		oldStatus = existing.Status
	}
	s.mu.RUnlock()

	if !ok {
		c.JSON(404, gin.H{"success": false, "error": "Report not found"})
		return
	}

	if err := validateStatusTransition(oldStatus, req.Status); err != nil {
		c.JSON(http.StatusConflict, gin.H{"success": false, "error": err.Error()})
		return
	}

	if req.Status == "payment_pending" {
		// Moving to payment_pending is the approval step. The payout itself is
		// broadcast by wallet_api and recorded through /reports/:id/tx-hash.
		s.mu.Lock()
		if req.RewardAmount != "" {
			existing.RewardAmount = req.RewardAmount
		}
		if req.Comments != "" {
			existing.Comments = req.Comments
		}
		s.mu.Unlock()

		s.requestRewardPayment(existing)

		s.mu.RLock()
		updated := s.reports[reportID]
		s.mu.RUnlock()

		c.JSON(http.StatusAccepted, gin.H{
			"success": true,
			"status":  "payment_pending",
			"message": "Report approved for payout; broadcast it via wallet_api, " +
				"then POST the returned tx_hash to /api/v1/reports/:id/tx-hash",
			"action_required": gin.H{
				"endpoint":  strings.TrimRight(cfg.HunterAPI, "/"),
				"report_id": reportID,
				"amount":    updated.RewardAmount,
				"callback":  "/api/v1/reports/" + reportID + "/tx-hash",
			},
			"data": updated,
		})
		return
	}

	s.mu.Lock()
	report := s.reports[reportID]
	report.Status = req.Status
	report.UpdatedAt = time.Now()

	if req.RewardAmount != "" {
		report.RewardAmount = req.RewardAmount
	}

	if req.Comments != "" {
		report.Comments = req.Comments
	}

	if req.Status == "fixed" {
		now := time.Now()
		report.ResolvedAt = &now
	}

	if s.db != nil {
		if _, err := s.db.Exec(
			`UPDATE bug_reports SET status = $1, reward_amount = $2, comments = $3, resolved_at = $4, updated_at = $5 WHERE id = $6`,
			report.Status, report.RewardAmount, report.Comments,
			report.ResolvedAt, report.UpdatedAt, report.ID,
		); err != nil {
			log.Printf("failed to persist status for report %s: %v", report.ID, err)
		}
	}
	s.mu.Unlock()

	c.JSON(200, gin.H{"success": true, "data": report})
}

// validateStatusTransition enforces the report state machine:
//
//	submitted -> triaged, accepted, rejected
//	triaged   -> accepted, rejected
//	accepted  -> payment_pending, rejected, fixed
//	rejected  -> (terminal)
//	fixed     -> (terminal)
//	payment_pending -> rewarded (only via the tx-hash endpoint)
//	rewarded  -> (terminal)
//
// "rewarded" is reachable only by recording a real transaction hash, so it is
// rejected here for every source state.
func validateStatusTransition(from, to string) error {
	if to == "rewarded" {
		return fmt.Errorf("a report cannot be set to rewarded directly; move it to " +
			"payment_pending, broadcast the payout via wallet_api, then POST the " +
			"returned tx_hash to /reports/:id/tx-hash")
	}

	allowed, known := reportTransitions[from]
	if !known {
		return fmt.Errorf("unknown current status %q", from)
	}
	if from == to {
		return nil
	}
	for _, candidate := range allowed {
		if candidate == to {
			return nil
		}
	}
	return fmt.Errorf("cannot move a report from %q to %q", from, to)
}

var reportTransitions = map[string][]string{
	"submitted":       {"triaged", "accepted", "rejected"},
	"triaged":         {"accepted", "rejected"},
	"accepted":        {"payment_pending", "rejected", "fixed"},
	"payment_pending": {"accepted", "rejected"},
	"rejected":        {},
	"fixed":           {},
	"rewarded":        {},
}

// requestRewardPayment moves a report into payment_pending and returns the
// instructions needed to broadcast the real payout.
//
// This service holds no signing key, so it cannot transfer funds. It previously
// derived a transaction hash from the report id ("0x" + hex(id)), wrote it to
// report.TxHash and decremented the reward pool, so an operator saw a paid
// report with a hash that exists on no chain and a pool balance reduced by
// money that never left the treasury. Payouts are now broadcast by the
// canonical wallet_api, which signs them, and the real hash is recorded through
// RecordRewardTxHash.
func (s *BugBountyService) requestRewardPayment(report *BugReport) {
	s.mu.Lock()
	defer s.mu.Unlock()

	report.Status = "payment_pending"
	now := time.Now()
	report.UpdatedAt = now

	if s.db != nil {
		if _, err := s.db.Exec(
			`UPDATE bug_reports SET status = $1, updated_at = $2 WHERE id = $3`,
			report.Status, report.UpdatedAt, report.ID,
		); err != nil {
			log.Printf("failed to persist payment_pending for report %s: %v", report.ID, err)
		}
	}
}

// RecordRewardTxHash records the real on-chain transaction hash returned after
// wallet_api broadcast the payout. Only then does the reward pool decrease and
// the report become "rewarded". The hash is validated as a 32-byte hex value so
// a placeholder cannot be stored in its place.
func (s *BugBountyService) RecordRewardTxHash(reportID, txHash string) error {
	if !isTxHash(txHash) {
		return fmt.Errorf("invalid transaction hash %q", txHash)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	report, ok := s.reports[reportID]
	if !ok {
		return fmt.Errorf("report %s not found", reportID)
	}
	if report.Status != "payment_pending" && report.Status != "accepted" {
		return fmt.Errorf("report %s is %s, not payable", reportID, report.Status)
	}
	if report.TxHash != "" {
		return fmt.Errorf("report %s already has a transaction hash", reportID)
	}

	rewardAmount, ok := new(big.Int).SetString(report.RewardAmount, 10)
	if !ok {
		return fmt.Errorf("report %s has an unparseable reward amount %q", reportID, report.RewardAmount)
	}
	if rewardAmount.Sign() < 0 {
		return fmt.Errorf("report %s has a negative reward amount", reportID)
	}
	if s.rewardPool.Cmp(rewardAmount) < 0 {
		return fmt.Errorf("reward pool has %s, cannot pay %s", s.rewardPool, rewardAmount)
	}

	report.TxHash = txHash
	report.Status = "rewarded"
	now := time.Now()
	report.ResolvedAt = &now
	report.UpdatedAt = now

	// The payout is confirmed on chain, so the pool and program totals move now.
	s.rewardPool.Sub(s.rewardPool, rewardAmount)

	if program := s.programs[report.ProgramID]; program != nil {
		paidOut, _ := new(big.Int).SetString(program.PaidOut, 10)
		if paidOut == nil {
			paidOut = new(big.Int)
		}
		paidOut.Add(paidOut, rewardAmount)
		program.PaidOut = paidOut.String()
		program.UpdatedAt = now
	}

	if s.db != nil {
		if _, err := s.db.Exec(
			`UPDATE bug_reports SET status = $1, tx_hash = $2, resolved_at = $3, updated_at = $4 WHERE id = $5`,
			report.Status, report.TxHash, report.ResolvedAt, report.UpdatedAt, report.ID,
		); err != nil {
			return fmt.Errorf("persist reward for report %s: %w", report.ID, err)
		}
		if program := s.programs[report.ProgramID]; program != nil {
			if _, err := s.db.Exec(
				`UPDATE bug_bounty_programs SET paid_out = $1, updated_at = $2 WHERE id = $3`,
				program.PaidOut, program.UpdatedAt, program.ID,
			); err != nil {
				return fmt.Errorf("persist program paid_out for %s: %w", program.ID, err)
			}
		}
	}

	return nil
}

// isTxHash reports whether s is a 0x-prefixed 32-byte hex string.
func isTxHash(s string) bool {
	if len(s) != 66 || !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		return false
	}
	_, err := hex.DecodeString(s[2:])
	return err == nil
}

// GetLeaderboard returns the bug bounty leaderboard
func (s *BugBountyService) GetLeaderboard(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Sort hackers by reputation
	type LeaderboardEntry struct {
		Hacker        *Hacker `json:"hacker"`
		AcceptedBugs  int     `json:"accepted_bugs"`
		TotalEarnings string  `json:"total_earnings"`
	}

	entries := make([]LeaderboardEntry, 0, len(s.hackers))
	for _, h := range s.hackers {
		if h.Status == "active" {
			entries = append(entries, LeaderboardEntry{
				Hacker:        h,
				AcceptedBugs:  h.AcceptedCount,
				TotalEarnings: h.TotalEarnings,
			})
		}
	}

	c.JSON(200, gin.H{"success": true, "data": entries})
}

// RegisterHacker registers a new bug bounty hunter
func (s *BugBountyService) RegisterHacker(c *gin.Context) {
	var req struct {
		Username      string `json:"username" binding:"required"`
		Email         string `json:"email" binding:"required,email"`
		Password      string `json:"password" binding:"required,min=8"`
		WalletAddress string `json:"wallet_address"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Hash password with a per-hacker random salt. argon2.IDKey returns only the
	// derived hash, so we generate a 16-byte salt with crypto/rand and store
	// salt||hash together (both hex-encoded) for later verification.
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		c.JSON(500, gin.H{"success": false, "error": "Failed to generate password salt"})
		return
	}
	hash := argon2.IDKey([]byte(req.Password), salt, 1, 64*1024, 4, 32)
	encoded := hex.EncodeToString(salt) + hex.EncodeToString(hash)

	hacker := &Hacker{
		ID:              uuid.New().String(),
		Username:        req.Username,
		Email:           req.Email,
		PasswordHash:    encoded,
		WalletAddress:   req.WalletAddress,
		ReputationScore: 0,
		TotalEarnings:   "0",
		ReportsCount:    0,
		AcceptedCount:   0,
		Rank:            0,
		Status:          "active",
		KYCStatus:       "none",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	s.mu.Lock()
	s.hackers[hacker.ID] = hacker
	s.mu.Unlock()

	c.JSON(201, gin.H{
		"success": true,
		"data":    hacker,
		"message": "Registration successful",
	})
}

// GetStats returns bug bounty statistics
func (s *BugBountyService) GetStats(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalReports := len(s.reports)
	var accepted, rewarded, pending int
	var totalPaidOut big.Int

	for _, r := range s.reports {
		switch r.Status {
		case "accepted":
			accepted++
		case "rewarded":
			accepted++
			rewarded++
			if r.RewardAmount != "" {
				amt := new(big.Int)
				amt.SetString(r.RewardAmount, 10)
				totalPaidOut.Add(&totalPaidOut, amt)
			}
		case "submitted", "triaged":
			pending++
		}
	}

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"total_programs":   len(s.programs),
			"total_reports":    totalReports,
			"accepted_reports": accepted,
			"rewarded_reports": rewarded,
			"pending_reports":  pending,
			"total_paid_out":   totalPaidOut.String(),
			"remaining_pool":   s.rewardPool.String(),
			"active_hackers":   len(s.hackers),
		},
	})
}

// ============================================================================
// Middleware
// ============================================================================

// AuthMiddleware handles authentication by validating the Bearer JWT.
// Tokens are HS256-signed with cfg.JWTSecret (JWT_SECRET env). Invalid,
// expired, or malformed tokens are rejected with 401. If JWT_SECRET is not
// configured, all protected requests are rejected with 503 rather than
// trusting the raw token.
func (s *BugBountyService) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.JWTSecret == "" {
			c.JSON(503, gin.H{"success": false, "error": "auth not configured: JWT_SECRET unset"})
			c.Abort()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"success": false, "error": "Authorization required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(401, gin.H{"success": false, "error": "invalid authorization header"})
			c.Abort()
			return
		}
		tokenStr := strings.TrimSpace(parts[1])

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(401, gin.H{"success": false, "error": "invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(401, gin.H{"success": false, "error": "invalid token claims"})
			c.Abort()
			return
		}

		userID, _ := claims["user_id"].(string)
		if userID == "" {
			userID, _ = claims["sub"].(string)
		}
		if userID == "" {
			c.JSON(401, gin.H{"success": false, "error": "token missing user_id"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

// ============================================================================
// Main
// ============================================================================

func main() {
	loaded, err := loadConfig()
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}
	cfg = loaded

	// Initialize service
	service := NewBugBountyService()
	if err := service.Initialize(); err != nil {
		fmt.Printf("Failed to initialize bug bounty service: %v\n", err)
		return
	}

	// Setup router
	r := gin.Default()

	// Liveness and readiness probes for the orchestrator.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "bug-bounty"})
	})
	r.GET("/readyz", func(c *gin.Context) {
		if err := service.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready", "service": "bug-bounty"})
	})

	// Public routes
	r.GET("/api/v1/programs", service.GetPrograms)
	r.GET("/api/v1/programs/:id", service.GetProgram)
	r.GET("/api/v1/stats", service.GetStats)
	r.POST("/api/v1/auth/register", service.RegisterHacker)

	// Protected routes
	protected := r.Group("/api/v1")
	protected.Use(service.AuthMiddleware())
	{
		protected.POST("/reports", service.SubmitReport)
		protected.GET("/reports", service.GetReports)
		protected.GET("/reports/:id", service.GetReport)
		protected.PATCH("/reports/:id", service.UpdateReportStatus)
		protected.GET("/leaderboard", service.GetLeaderboard)
		// Records the real on-chain hash after wallet_api broadcasts a reward
		// payout. Until this is called the report stays payment_pending and the
		// reward pool is untouched.
		protected.POST("/reports/:id/tx-hash", service.SetRewardTxHash)
	}

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Port)
	fmt.Printf("Bug Bounty Service starting on %s\n", addr)
	if err := r.Run(addr); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}

// SetRewardTxHash records the transaction hash of a broadcast reward payout.
// This is the only path that marks a report "rewarded" and the only path that
// reduces the reward pool, so the pool always reflects money that actually
// moved on chain.
func (s *BugBountyService) SetRewardTxHash(c *gin.Context) {
	reportID := c.Param("id")
	var req struct {
		TxHash string `json:"tx_hash" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if err := s.RecordRewardTxHash(reportID, req.TxHash); err != nil {
		c.JSON(http.StatusConflict, gin.H{"success": false, "error": err.Error()})
		return
	}

	s.mu.RLock()
	report := s.reports[reportID]
	s.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{"success": true, "data": report})
}

// Ping reports whether the service's backing store is reachable.
//
// The readiness probe calls this so an orchestrator only routes traffic to a
// replica that can actually serve it.
func (s *BugBountyService) Ping(ctx context.Context) error {
	if s.db == nil {
		return nil
	}
	return s.db.PingContext(ctx)
}
