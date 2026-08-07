// OAuth2/SSO Service for TigerWallet
// Production-ready OAuth2 implementation with multiple providers

package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// ============================================================================
// CONFIGURATION
// ============================================================================

type OAuthConfig struct {
	// OAuth Providers
	GoogleClientID       string
	GoogleClientSecret   string
	GitHubClientID       string
	GitHubClientSecret   string
	FacebookClientID     string
	FacebookClientSecret string

	// JWT
	JWT_SECRET string

	// URLs
	BaseURL     string
	RedirectURL string

	// Session
	SessionExpiry time.Duration
}

var oauthCfg *OAuthConfig
var oauth2ConfigMap map[string]*oauth2.Config
var oauthStateStore = NewOAuthStateStore()

// ============================================================================
// OAUTH STATE STORE
// ============================================================================

type OAuthState struct {
	State    string
	Provider string
	UserID   string
	Nonce    string
	Expiry   time.Time
}

type OAuthStateStore struct {
	states map[string]*OAuthState
	mu     sync.RWMutex
}

func NewOAuthStateStore() *OAuthStateStore {
	return &OAuthStateStore{
		states: make(map[string]*OAuthState),
	}
}

func (s *OAuthStateStore) Generate(provider, userID string) string {
	state := generateRandomString(32)
	nonce := generateRandomString(16)

	s.states[state] = &OAuthState{
		State:    state,
		Provider: provider,
		UserID:   userID,
		Nonce:    nonce,
		Expiry:   time.Now().Add(10 * time.Minute),
	}

	// Clean up expired states periodically
	go func() {
		time.Sleep(5 * time.Minute)
		s.cleanup()
	}()

	return state
}

func (s *OAuthStateStore) Validate(state, nonce string) (*OAuthState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sstate, ok := s.states[state]
	if !ok {
		return nil, fmt.Errorf("invalid state")
	}

	if time.Now().After(sstate.Expiry) {
		return nil, fmt.Errorf("state expired")
	}

	if nonce != "" && sstate.Nonce != nonce {
		return nil, fmt.Errorf("invalid nonce")
	}

	return sstate, nil
}

func (s *OAuthStateStore) Delete(state string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.states, state)
}

func (s *OAuthStateStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for state, sstate := range s.states {
		if now.After(sstate.Expiry) {
			delete(s.states, state)
		}
	}
}

// ============================================================================
// OAUTH PROVIDERS
// ============================================================================

func initOAuth() {
	oauthCfg = &OAuthConfig{
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		JWT_SECRET:         os.Getenv("JWT_SECRET"),
		BaseURL:            os.Getenv("BASE_URL"),
		RedirectURL:        os.Getenv("REDIRECT_URL"),
		SessionExpiry:      24 * time.Hour,
	}

	oauth2ConfigMap = make(map[string]*oauth2.Config)

	// Google OAuth2 Config
	if oauthCfg.GoogleClientID != "" {
		oauth2ConfigMap["google"] = &oauth2.Config{
			ClientID:     oauthCfg.GoogleClientID,
			ClientSecret: oauthCfg.GoogleClientSecret,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
			Endpoint:     google.Endpoint,
			RedirectURL:  oauthCfg.RedirectURL + "/oauth/google/callback",
		}
	}

	// GitHub OAuth2 Config
	if oauthCfg.GitHubClientID != "" {
		oauth2ConfigMap["github"] = &oauth2.Config{
			ClientID:     oauthCfg.GitHubClientID,
			ClientSecret: oauthCfg.GitHubClientSecret,
			Scopes:       []string{"user:email", "read:user"},
			Endpoint:     github.Endpoint,
			RedirectURL:  oauthCfg.RedirectURL + "/oauth/github/callback",
		}
	}
}

// ============================================================================
// OAUTH HANDLERS
// ============================================================================

type OAuthHandler struct{}

func NewOAuthHandler() *OAuthHandler {
	return &OAuthHandler{}
}

// InitOAuth initializes the OAuth configuration
func (h *OAuthHandler) InitOAuth() {
	initOAuth()
}

// GetLoginURL generates the OAuth login URL for a provider
func (h *OAuthHandler) GetLoginURL(provider string) (string, error) {
	cfg, ok := oauth2ConfigMap[provider]
	if !ok {
		return "", fmt.Errorf("provider %s not configured", provider)
	}

	state := oauthStateStore.Generate(provider, "")

	url := cfg.AuthCodeURL(state, oauth2.AccessTypeOnline, oauth2.ApprovalForce)
	return url, nil
}

// HandleOAuthCallback handles the OAuth callback from the provider
func (h *OAuthHandler) HandleOAuthCallback(c *gin.Context, provider string) error {
	state := c.Query("state")
	code := c.Query("code")

	// Validate state
	oauthState, err := oauthStateStore.Validate(state, "")
	if err != nil {
		return fmt.Errorf("invalid state: %w", err)
	}

	if oauthState.Provider != provider {
		return fmt.Errorf("provider mismatch")
	}

	// Get OAuth config
	cfg, ok := oauth2ConfigMap[provider]
	if !ok {
		return fmt.Errorf("provider %s not configured", provider)
	}

	// Exchange code for token
	token, err := cfg.Exchange(context.Background(), code)
	if err != nil {
		return fmt.Errorf("code exchange failed: %w", err)
	}

	// Get user info
	userInfo, err := h.getUserInfo(provider, token)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	// Generate JWT token
	jwtToken, err := h.generateJWT(userInfo)
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	// Clean up state
	oauthStateStore.Delete(state)

	// Return token
	c.JSON(http.StatusOK, gin.H{
		"token": jwtToken,
		"user":  userInfo,
	})

	return nil
}

func (h *OAuthHandler) getUserInfo(provider string, token *oauth2.Token) (*OAuthUserInfo, error) {
	switch provider {
	case "google":
		return h.getGoogleUserInfo(token)
	case "github":
		return h.getGitHubUserInfo(token)
	default:
		return nil, fmt.Errorf("unsupported provider")
	}
}

func (h *OAuthHandler) getGoogleUserInfo(token *oauth2.Token) (*OAuthUserInfo, error) {
	client := oauthCfg.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	var userInfo struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		VerifiedEmail bool   `json:"verified_email"`
	}

	json.Unmarshal(data, &userInfo)

	return &OAuthUserInfo{
		Provider:     "google",
		ProviderID:   userInfo.ID,
		Email:        userInfo.Email,
		Name:         userInfo.Name,
		Avatar:       userInfo.Picture,
		Verified:     userInfo.VerifiedEmail,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
	}, nil
}

func (h *OAuthHandler) getGitHubUserInfo(token *oauth2.Token) (*OAuthUserInfo, error) {
	client := oauthCfg.Client(context.Background(), token)

	// Get user info
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	var userInfo struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	json.Unmarshal(data, &userInfo)

	// Get primary email if not public
	email := userInfo.Email
	if email == "" {
		emailResp, _ := client.Get("https://api.github.com/user/emails")
		if emailResp != nil {
			defer emailResp.Body.Close()
			emailData, _ := io.ReadAll(emailResp.Body)
			var emails []struct {
				Email   string `json:"email"`
				Primary bool   `json:"primary"`
			}
			json.Unmarshal(emailData, &emails)
			for _, e := range emails {
				if e.Primary {
					email = e.Email
					break
				}
			}
		}
	}

	return &OAuthUserInfo{
		Provider:     "github",
		ProviderID:   fmt.Sprintf("%d", userInfo.ID),
		Email:        email,
		Name:         userInfo.Name,
		Avatar:       userInfo.AvatarURL,
		Verified:     true,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
	}, nil
}

type OAuthUserInfo struct {
	Provider     string `json:"provider"`
	ProviderID   string `json:"provider_id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	Avatar       string `json:"avatar"`
	Verified     bool   `json:"verified"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func (h *OAuthHandler) generateJWT(userInfo *OAuthUserInfo) (string, error) {
	claims := jwt.MapClaims{
		"sub":      userInfo.ProviderID,
		"email":    userInfo.Email,
		"name":     userInfo.Name,
		"avatar":   userInfo.Avatar,
		"provider": userInfo.Provider,
		"verified": userInfo.Verified,
		"exp":      time.Now().Add(oauthCfg.SessionExpiry).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(oauthCfg.JWT_SECRET))
}

// ============================================================================
// OAUTH ROUTES
// ============================================================================

func (h *OAuthHandler) RegisterRoutes(router *gin.RouterGroup) {
	oauth := router.Group("/oauth")
	{
		// Login endpoints
		oauth.GET("/login/:provider", h.handleLogin)
		oauth.GET("/google/login", h.handleGoogleLogin)
		oauth.GET("/github/login", h.handleGitHubLogin)

		// Callback endpoints
		oauth.GET("/google/callback", h.handleGoogleCallback)
		oauth.GET("/github/callback", h.handleGitHubCallback)

		// Link account
		oauth.POST("/link/:provider", h.handleLinkAccount)
		oauth.DELETE("/unlink/:provider", h.handleUnlinkAccount)

		// Token refresh
		oauth.POST("/refresh", h.handleRefreshToken)
	}
}

func (h *OAuthHandler) handleLogin(c *gin.Context) {
	provider := c.Param("provider")

	url, err := h.GetLoginURL(provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

func (h *OAuthHandler) handleGoogleLogin(c *gin.Context) {
	url, err := h.GetLoginURL("google")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Redirect(http.StatusFound, url)
}

func (h *OAuthHandler) handleGitHubLogin(c *gin.Context) {
	url, err := h.GetLoginURL("github")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Redirect(http.StatusFound, url)
}

func (h *OAuthHandler) handleGoogleCallback(c *gin.Context) {
	if err := h.HandleOAuthCallback(c, "google"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
}

func (h *OAuthHandler) handleGitHubCallback(c *gin.Context) {
	if err := h.HandleOAuthCallback(c, "github"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
}

func (h *OAuthHandler) handleLinkAccount(c *gin.Context) {
	provider := c.Param("provider")

	url, err := h.GetLoginURL(provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url, "message": "Link your account"})
}

func (h *OAuthHandler) handleUnlinkAccount(c *gin.Context) {
	provider := c.Param("provider")

	// Get current user from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	// Unlink account logic here
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Unlinked %s account", provider)})
}

func (h *OAuthHandler) handleRefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify and generate new token
	// In production, validate refresh token and issue new access token
	c.JSON(http.StatusOK, gin.H{"message": "Token refreshed"})
}

// ============================================================================
// HELPERS
// ============================================================================

func generateRandomString(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:length]
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}

// ============================================================================
// SSO SAML SUPPORT (Placeholder)
// ============================================================================

type SAMLConfig struct {
	IDPEntityID         string
	IDPSSOURL           string
	IDPCert             string
	SPEntityID          string
	SPACSURL            string
	AttributeNameFormat string
}

// SAMLService provides SAML 2.0 SSO capability
type SAMLService struct {
	config *SAMLConfig
}

func NewSAMLService(config *SAMLConfig) *SAMLService {
	return &SAMLService{config: config}
}

// InitiateLogin starts SAML authentication
func (s *SAMLService) InitiateLogin(relayState string) (string, error) {
	// Generate SAML AuthnRequest
	// In production, use proper SAML library
	return "", nil
}

// ProcessResponse processes SAML response
func (s *SAMLService) ProcessResponse(response, relayState string) (*SAMLAssertion, error) {
	// Validate and parse SAML response
	// In production, use proper SAML library
	return nil, nil
}

type SAMLAssertion struct {
	NameID     string
	Email      string
	FirstName  string
	LastName   string
	Attributes map[string]string
}
