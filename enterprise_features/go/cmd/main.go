// Enterprise Features Service - PostgreSQL Version
// Enterprise-grade features for TigerWallet ecosystem

package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Configuration
type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
}

// Enterprise Models
type Organization struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Domain      string     `json:"domain"`
	Tier        string     `json:"tier"`     // starter, business, enterprise
	Status      string     `json:"status"`   // active, suspended, trial
	Settings    string     `json:"settings"` // JSON settings
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	TrialEndsAt *time.Time `json:"trial_ends_at"`
}

type TeamMember struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	UserID         uuid.UUID  `json:"user_id"`
	Role           string     `json:"role"`   // owner, admin, member, viewer
	Status         string     `json:"status"` // active, pending, inactive
	InvitedBy      uuid.UUID  `json:"invited_by"`
	JoinedAt       *time.Time `json:"joined_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

type ApiKey struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	Name           string     `json:"name"`
	KeyHash        string     `json:"-"`
	KeyPrefix      string     `json:"key_prefix"`
	Permissions    string     `json:"permissions"` // JSON array
	RateLimit      int        `json:"rate_limit"`  // requests per minute
	IsActive       bool       `json:"is_active"`
	LastUsedAt     *time.Time `json:"last_used_at"`
	ExpiresAt      *time.Time `json:"expires_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

type WebhookConfig struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	URL            string    `json:"url"`
	Events         string    `json:"events"` // JSON array
	Secret         string    `json:"-"`
	IsActive       bool      `json:"is_active"`
	RetryPolicy    string    `json:"retry_policy"` // JSON object
	CreatedAt      time.Time `json:"created_at"`
}

type AuditTrail struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	UserID         uuid.UUID `json:"user_id"`
	Action         string    `json:"action"`
	ResourceType   string    `json:"resource_type"`
	ResourceID     string    `json:"resource_id"`
	Details        string    `json:"details"` // JSON object
	IPAddress      string    `json:"ip_address"`
	Timestamp      time.Time `json:"timestamp"`
}

type ComplianceReport struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	ReportType     string     `json:"report_type"` // kyc, aml, audit
	StartDate      time.Time  `json:"start_date"`
	EndDate        time.Time  `json:"end_date"`
	Status         string     `json:"status"` // pending, processing, completed, failed
	ResultURL      string     `json:"result_url"`
	RequestedBy    uuid.UUID  `json:"requested_by"`
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at"`
}

type SSOSetting struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Provider       string    `json:"provider"` // okta, azure, google, saml
	Config         string    `json:"config"`   // JSON config
	IsEnabled      bool      `json:"is_enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Global variables
var (
	db          *pgxpool.Pool
	redisClient *redis.Client
	config      Config
	logger      *log.Logger
)

// Initialize database
func initDatabase() error {
	var err error
	dbURL := getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet_admin")

	db, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	if err = db.Ping(context.Background()); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create tables
	_, err = db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS organizations (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			domain VARCHAR(255),
			tier VARCHAR(50) DEFAULT 'starter',
			status VARCHAR(50) DEFAULT 'active',
			settings JSONB,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			trial_ends_at TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS team_members (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			organization_id UUID REFERENCES organizations(id),
			user_id UUID NOT NULL,
			role VARCHAR(50) DEFAULT 'member',
			status VARCHAR(50) DEFAULT 'pending',
			invited_by UUID,
			joined_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS api_keys (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			organization_id UUID REFERENCES organizations(id),
			name VARCHAR(255) NOT NULL,
			key_hash VARCHAR(255) NOT NULL,
			key_prefix VARCHAR(20) NOT NULL,
			permissions JSONB,
			rate_limit INTEGER DEFAULT 60,
			is_active BOOLEAN DEFAULT true,
			last_used_at TIMESTAMP,
			expires_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS webhook_configs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			organization_id UUID REFERENCES organizations(id),
			name VARCHAR(255) NOT NULL,
			url TEXT NOT NULL,
			events JSONB,
			secret VARCHAR(255),
			is_active BOOLEAN DEFAULT true,
			retry_policy JSONB,
			created_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS audit_trails (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			organization_id UUID REFERENCES organizations(id),
			user_id UUID NOT NULL,
			action VARCHAR(255) NOT NULL,
			resource_type VARCHAR(100),
			resource_id VARCHAR(255),
			details JSONB,
			ip_address VARCHAR(45),
			timestamp TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS compliance_reports (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			organization_id UUID REFERENCES organizations(id),
			report_type VARCHAR(50) NOT NULL,
			start_date TIMESTAMP NOT NULL,
			end_date TIMESTAMP NOT NULL,
			status VARCHAR(50) DEFAULT 'pending',
			result_url TEXT,
			requested_by UUID NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			completed_at TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS sso_settings (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			organization_id UUID REFERENCES organizations(id),
			provider VARCHAR(50) NOT NULL,
			config JSONB,
			is_enabled BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_orgs_name ON organizations(name);
		CREATE INDEX IF NOT EXISTS idx_team_org ON team_members(organization_id);
		CREATE INDEX IF NOT EXISTS idx_api_keys_org ON api_keys(organization_id);
		CREATE INDEX IF NOT EXISTS idx_webhooks_org ON webhook_configs(organization_id);
		CREATE INDEX IF NOT EXISTS idx_audit_org ON audit_trails(organization_id);
		CREATE INDEX IF NOT EXISTS idx_compliance_org ON compliance_reports(organization_id);
	`)

	return err
}

// Initialize Redis
func initRedis() error {
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return err
	}

	redisClient = redis.NewClient(opt)
	return redisClient.Ping(context.Background()).Err()
}

// Handlers

// CreateOrganization - Create a new organization
func CreateOrganization(c *gin.Context) {
	var org Organization
	if err := c.ShouldBindJSON(&org); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	org.ID = uuid.New()
	// Status and tier are server-controlled; a caller must not be able to
	// self-provision an enterprise tier or an already-active paid plan.
	org.Tier = "starter"
	org.Status = "active"
	org.CreatedAt = time.Now()
	org.UpdatedAt = time.Now()

	settingsJSON, _ := json.Marshal(org.Settings)

	tx, err := db.Begin(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer tx.Rollback(context.Background())

	if _, err := tx.Exec(context.Background(), `
		INSERT INTO organizations (id, name, domain, tier, status, settings, created_at, updated_at, trial_ends_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, org.ID, org.Name, org.Domain, org.Tier, org.Status, settingsJSON, org.CreatedAt, org.UpdatedAt, org.TrialEndsAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// The creator becomes the organization owner, which is what the
	// authorization checks read.
	if _, err := tx.Exec(context.Background(), `
		INSERT INTO team_members (id, organization_id, user_id, role, status, joined_at, created_at)
		VALUES ($1, $2, $3, 'owner', 'active', NOW(), NOW())
	`, uuid.New(), org.ID, callerID(c)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record ownership"})
		return
	}

	if err := tx.Commit(context.Background()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit"})
		return
	}

	c.JSON(http.StatusCreated, org)
}

// GetOrganizations - Get organizations
func GetOrganizations(c *gin.Context) {
	rows, err := db.Query(context.Background(), `
		SELECT id, name, domain, tier, status, settings, created_at, updated_at, trial_ends_at
		FROM organizations
		ORDER BY created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var orgs []Organization
	for rows.Next() {
		var org Organization
		var settings []byte
		if err := rows.Scan(&org.ID, &org.Name, &org.Domain, &org.Tier, &org.Status, &settings, &org.CreatedAt, &org.UpdatedAt, &org.TrialEndsAt); err != nil {
			continue
		}
		json.Unmarshal(settings, &org.Settings)
		orgs = append(orgs, org)
	}

	c.JSON(http.StatusOK, gin.H{"organizations": orgs})
}

// AddTeamMember - Add team member
func AddTeamMember(c *gin.Context) {
	var member TeamMember
	if err := c.ShouldBindJSON(&member); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID := c.Param("org_id")
	if !memberOfOrganization(c, orgID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized for this organization"})
		return
	}

	member.ID = uuid.New()
	member.OrganizationID = uuid.MustParse(orgID)
	// New members always start as pending members; a caller cannot inject an
	// owner or pre-activated membership.
	member.Role = "member"
	member.Status = "pending"
	if inviter, err := uuid.Parse(callerID(c)); err == nil {
		member.InvitedBy = inviter
	}
	member.CreatedAt = time.Now()

	_, err := db.Exec(context.Background(), `
		INSERT INTO team_members (id, organization_id, user_id, role, status, invited_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, member.ID, member.OrganizationID, member.UserID, member.Role, member.Status, member.InvitedBy, member.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, member)
}

// GetTeamMembers - Get team members
func GetTeamMembers(c *gin.Context) {
	orgID := c.Param("org_id")
	orgUUID, _ := uuid.Parse(orgID)

	rows, err := db.Query(context.Background(), `
		SELECT id, organization_id, user_id, role, status, invited_by, joined_at, created_at
		FROM team_members
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`, orgUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var members []TeamMember
	for rows.Next() {
		var member TeamMember
		if err := rows.Scan(&member.ID, &member.OrganizationID, &member.UserID, &member.Role, &member.Status, &member.InvitedBy, &member.JoinedAt, &member.CreatedAt); err != nil {
			continue
		}
		members = append(members, member)
	}

	c.JSON(http.StatusOK, gin.H{"members": members})
}

// CreateApiKey - Create API key
func CreateApiKey(c *gin.Context) {
	var key ApiKey
	if err := c.ShouldBindJSON(&key); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orgID := c.Param("org_id")
	if !memberOfOrganization(c, orgID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not authorized for this organization"})
		return
	}

	// The secret is generated server-side. A caller-supplied hash would let
	// them register a key whose plaintext they already know.
	secret, err := generateApiKeySecret()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate key"})
		return
	}
	keyHash := hashApiKey(secret)

	if key.RateLimit <= 0 {
		key.RateLimit = 600
	}

	key.ID = uuid.New()
	key.OrganizationID = uuid.MustParse(orgID)
	key.KeyPrefix = secret[:11] // "tw_" plus 8 random characters
	key.KeyHash = keyHash
	key.IsActive = true
	key.CreatedAt = time.Now()

	permissionsJSON, _ := json.Marshal(key.Permissions)

	_, err = db.Exec(context.Background(), `
		INSERT INTO api_keys (id, organization_id, name, key_hash, key_prefix, permissions, rate_limit, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, key.ID, key.OrganizationID, key.Name, key.KeyHash, key.KeyPrefix, permissionsJSON, key.RateLimit, key.IsActive, key.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// The plaintext key is returned once and never stored; only the hash
	// remains in the database.
	c.JSON(http.StatusCreated, gin.H{
		"api_key": key,
		"secret":  secret,
		"warning": "store this secret now; it cannot be retrieved again",
	})
}

// generateApiKeySecret returns a high-entropy API key using a CSPRNG.
func generateApiKeySecret() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "tw_" + hex.EncodeToString(buf), nil
}

// hashApiKey hashes a key for storage so a database leak does not expose
// usable credentials.
func hashApiKey(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// GetApiKeys - Get API keys
func GetApiKeys(c *gin.Context) {
	orgID := c.Param("org_id")
	orgUUID, _ := uuid.Parse(orgID)

	rows, err := db.Query(context.Background(), `
		SELECT id, organization_id, name, key_prefix, permissions, rate_limit, is_active, last_used_at, expires_at, created_at
		FROM api_keys
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`, orgUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var keys []ApiKey
	for rows.Next() {
		var key ApiKey
		var permissions []byte
		if err := rows.Scan(&key.ID, &key.OrganizationID, &key.Name, &key.KeyPrefix, &permissions, &key.RateLimit, &key.IsActive, &key.LastUsedAt, &key.ExpiresAt, &key.CreatedAt); err != nil {
			continue
		}
		json.Unmarshal(permissions, &key.Permissions)
		keys = append(keys, key)
	}

	c.JSON(http.StatusOK, gin.H{"api_keys": keys})
}

// CreateWebhook - Create webhook
func CreateWebhook(c *gin.Context) {
	var webhook WebhookConfig
	if err := c.ShouldBindJSON(&webhook); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	webhook.ID = uuid.New()
	webhook.IsActive = true
	webhook.CreatedAt = time.Now()

	eventsJSON, _ := json.Marshal(webhook.Events)
	retryJSON, _ := json.Marshal(webhook.RetryPolicy)

	_, err := db.Exec(context.Background(), `
		INSERT INTO webhook_configs (id, organization_id, name, url, events, secret, is_active, retry_policy, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, webhook.ID, webhook.OrganizationID, webhook.Name, webhook.URL, eventsJSON, webhook.Secret, webhook.IsActive, retryJSON, webhook.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, webhook)
}

// GetWebhooks - Get webhooks
func GetWebhooks(c *gin.Context) {
	orgID := c.Param("org_id")
	orgUUID, _ := uuid.Parse(orgID)

	rows, err := db.Query(context.Background(), `
		SELECT id, organization_id, name, url, events, is_active, retry_policy, created_at
		FROM webhook_configs
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`, orgUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var webhooks []WebhookConfig
	for rows.Next() {
		var webhook WebhookConfig
		var events []byte
		var retry []byte
		if err := rows.Scan(&webhook.ID, &webhook.OrganizationID, &webhook.Name, &webhook.URL, &events, &webhook.IsActive, &retry, &webhook.CreatedAt); err != nil {
			continue
		}
		json.Unmarshal(events, &webhook.Events)
		json.Unmarshal(retry, &webhook.RetryPolicy)
		webhooks = append(webhooks, webhook)
	}

	c.JSON(http.StatusOK, gin.H{"webhooks": webhooks})
}

// RecordAuditTrail - Record audit trail
func RecordAuditTrail(c *gin.Context) {
	var trail AuditTrail
	if err := c.ShouldBindJSON(&trail); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	trail.ID = uuid.New()
	trail.Timestamp = time.Now()

	detailsJSON, _ := json.Marshal(trail.Details)

	_, err := db.Exec(context.Background(), `
		INSERT INTO audit_trails (id, organization_id, user_id, action, resource_type, resource_id, details, ip_address, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, trail.ID, trail.OrganizationID, trail.UserID, trail.Action, trail.ResourceType, trail.ResourceID, detailsJSON, trail.IPAddress, trail.Timestamp)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, trail)
}

// GetAuditTrail - Get audit trail
func GetAuditTrail(c *gin.Context) {
	orgID := c.Param("org_id")
	orgUUID, _ := uuid.Parse(orgID)
	limit := c.DefaultQuery("limit", "100")

	rows, err := db.Query(context.Background(), `
		SELECT id, organization_id, user_id, action, resource_type, resource_id, details, ip_address, timestamp
		FROM audit_trails
		WHERE organization_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`, orgUUID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var trails []AuditTrail
	for rows.Next() {
		var trail AuditTrail
		var details []byte
		if err := rows.Scan(&trail.ID, &trail.OrganizationID, &trail.UserID, &trail.Action, &trail.ResourceType, &trail.ResourceID, &details, &trail.IPAddress, &trail.Timestamp); err != nil {
			continue
		}
		json.Unmarshal(details, &trail.Details)
		trails = append(trails, trail)
	}

	c.JSON(http.StatusOK, gin.H{"audit_trail": trails})
}

// CreateComplianceReport - Create compliance report
func CreateComplianceReport(c *gin.Context) {
	var report ComplianceReport
	if err := c.ShouldBindJSON(&report); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report.ID = uuid.New()
	report.Status = "pending"
	report.CreatedAt = time.Now()

	_, err := db.Exec(context.Background(), `
		INSERT INTO compliance_reports (id, organization_id, report_type, start_date, end_date, status, requested_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, report.ID, report.OrganizationID, report.ReportType, report.StartDate, report.EndDate, report.Status, report.RequestedBy, report.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, report)
}

// GetComplianceReports - Get compliance reports
func GetComplianceReports(c *gin.Context) {
	orgID := c.Param("org_id")
	orgUUID, _ := uuid.Parse(orgID)

	rows, err := db.Query(context.Background(), `
		SELECT id, organization_id, report_type, start_date, end_date, status, result_url, requested_by, created_at, completed_at
		FROM compliance_reports
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`, orgUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var reports []ComplianceReport
	for rows.Next() {
		var report ComplianceReport
		if err := rows.Scan(&report.ID, &report.OrganizationID, &report.ReportType, &report.StartDate, &report.EndDate, &report.Status, &report.ResultURL, &report.RequestedBy, &report.CreatedAt, &report.CompletedAt); err != nil {
			continue
		}
		reports = append(reports, report)
	}

	c.JSON(http.StatusOK, gin.H{"reports": reports})
}

// ConfigureSSO - Configure SSO
func ConfigureSSO(c *gin.Context) {
	var sso SSOSetting
	if err := c.ShouldBindJSON(&sso); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sso.ID = uuid.New()
	sso.CreatedAt = time.Now()
	sso.UpdatedAt = time.Now()

	configJSON, _ := json.Marshal(sso.Config)

	_, err := db.Exec(context.Background(), `
		INSERT INTO sso_settings (id, organization_id, provider, config, is_enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, sso.ID, sso.OrganizationID, sso.Provider, configJSON, sso.IsEnabled, sso.CreatedAt, sso.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, sso)
}

// GetSSOSettings - Get SSO settings
func GetSSOSettings(c *gin.Context) {
	orgID := c.Param("org_id")
	orgUUID, _ := uuid.Parse(orgID)

	rows, err := db.Query(context.Background(), `
		SELECT id, organization_id, provider, config, is_enabled, created_at, updated_at
		FROM sso_settings
		WHERE organization_id = $1
	`, orgUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var settings []SSOSetting
	for rows.Next() {
		var sso SSOSetting
		var config []byte
		if err := rows.Scan(&sso.ID, &sso.OrganizationID, &sso.Provider, &config, &sso.IsEnabled, &sso.CreatedAt, &sso.UpdatedAt); err != nil {
			continue
		}
		json.Unmarshal(config, &sso.Config)
		settings = append(settings, sso)
	}

	c.JSON(http.StatusOK, gin.H{"sso_settings": settings})
}

// Health check
// callerID returns the authenticated subject.
func callerID(c *gin.Context) string {
	return c.GetString("userID")
}

// callerOrgIDs returns the organizations the caller belongs to. An admin may
// act on any organization.
func callerIsAdmin(c *gin.Context) bool {
	switch c.GetString("role") {
	case "admin", "super_admin":
		return true
	}
	return false
}

// memberOfOrganization reports whether the caller may act on the organization.
// Membership is read from the organization_members table rather than trusted
// from the request, otherwise anyone could mint keys for any tenant.
func memberOfOrganization(c *gin.Context, orgID string) bool {
	if callerIsAdmin(c) {
		return true
	}
	userID := callerID(c)
	if userID == "" || orgID == "" {
		return false
	}

	var exists bool
	err := db.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1 FROM team_members
			WHERE organization_id = $1 AND user_id = $2 AND status <> 'removed'
		)
	`, orgID, userID).Scan(&exists)
	if err != nil {
		log.Printf("enterprise: membership lookup failed: %v", err)
		return false
	}
	return exists
}

// authMiddleware verifies the bearer token. An unset secret is a
// misconfiguration, so requests are refused rather than served unauthenticated.
func authMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if secret == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "JWT_SECRET not configured; refusing unauthenticated request",
			})
			return
		}

		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		token, err := jwt.Parse(strings.TrimPrefix(header, "Bearer "), func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{"HS256", "HS384", "HS512"}))
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
			return
		}
		sub, _ := claims["sub"].(string)
		if sub == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing subject"})
			return
		}

		c.Set("userID", sub)
		if role, ok := claims["role"].(string); ok {
			c.Set("role", role)
		}
		c.Next()
	}
}

func HealthCheck(c *gin.Context) {
	ctx := context.Background()

	dbStatus := "healthy"
	if err := db.Ping(ctx); err != nil {
		dbStatus = "unhealthy"
	}

	redisStatus := "healthy"
	if err := redisClient.Ping(ctx).Err(); err != nil {
		redisStatus = "unhealthy"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"database":  dbStatus,
		"redis":     redisStatus,
		"timestamp": time.Now(),
	})
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func main() {
	// Initialize logger
	logger = log.New(os.Stdout, "Enterprise Features: ", log.LstdFlags)
	logger.Println("Starting Enterprise Features Service...")

	// Load configuration
	config.Port = getEnv("ENTERPRISE_PORT", "8095")
	config.DatabaseURL = getEnv("DATABASE_URL", "postgres://tigerwallet:tigerwallet@localhost:5432/tigerwallet_admin")
	config.RedisURL = getEnv("REDIS_URL", "redis://localhost:6379")
	config.JWTSecret = getEnv("JWT_SECRET", "")

	// Initialize database
	if err := initDatabase(); err != nil {
		logger.Fatalf("Failed to initialize database: %v", err)
	}
	logger.Println("Database connected successfully")

	// Initialize Redis
	if err := initRedis(); err != nil {
		logger.Fatalf("Failed to initialize Redis: %v", err)
	}
	logger.Println("Redis connected successfully")

	// Initialize Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// Health check
	router.GET("/health", HealthCheck)

	// Every API route requires an authenticated caller.
	api := router.Group("/api/v1")
	api.Use(authMiddleware(config.JWTSecret))

	// Organization routes
	api.POST("/organizations", CreateOrganization)
	api.GET("/organizations", GetOrganizations)

	// Team member routes
	api.POST("/organizations/:org_id/team", AddTeamMember)
	api.GET("/organizations/:org_id/team", GetTeamMembers)

	// API key routes
	api.POST("/organizations/:org_id/keys", CreateApiKey)
	api.GET("/organizations/:org_id/keys", GetApiKeys)

	// Webhook routes
	api.POST("/organizations/:org_id/webhooks", CreateWebhook)
	api.GET("/organizations/:org_id/webhooks", GetWebhooks)

	// Audit trail routes
	api.POST("/audit", RecordAuditTrail)
	api.GET("/organizations/:org_id/audit", GetAuditTrail)

	// Compliance report routes
	api.POST("/compliance/reports", CreateComplianceReport)
	api.GET("/organizations/:org_id/compliance/reports", GetComplianceReports)

	// SSO routes
	api.POST("/organizations/:org_id/sso", ConfigureSSO)
	api.GET("/organizations/:org_id/sso", GetSSOSettings)

	// Start server
	logger.Printf("Starting server on port %s", config.Port)
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	logger.Println("Server started successfully")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	db.Close()
	redisClient.Close()
	logger.Println("Server exited")
}
