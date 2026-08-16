// Middleware - Authentication and authorization middleware
package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tigerwallet/white-label-admin/internal/config"
	"github.com/tigerwallet/white-label-admin/internal/roles"
)

// Claims carried in every WL-admin JWT. The white_label_id is the tenant
// scope: every admin is bound to exactly one WL client and can only act within
// that tenancy. scopes is the set of scoped sub-admin roles the admin holds.
type Claims struct {
	AdminID       uuid.UUID  `json:"admin_id"`
	Username      string     `json:"username"`
	Email         string     `json:"email"`
	Role          string     `json:"role"`
	WhiteLabelID  *uuid.UUID `json:"white_label_id,omitempty"`
	Scopes        []string   `json:"scopes,omitempty"`
	jwt.RegisteredClaims
}

func JWTAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("admin_id", claims.AdminID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		if claims.WhiteLabelID != nil {
			c.Set("white_label_id", *claims.WhiteLabelID)
		}
		c.Set("scopes", claims.Scopes)

		c.Next()
	}
}

// RoleAuth allows any of the given coarse roles (legacy; kept for compat).
func RoleAuth(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Role not found in token"})
			c.Abort()
			return
		}

		roleStr := role.(string)
		for _, allowedRole := range allowedRoles {
			if roleStr == allowedRole {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}

// RequireScope is the per-endpoint authorization gate. The caller passes the
// scope(s) permitted to call this endpoint. The admin passes if:
//   (a) their role is 'wl_client' (the WL owner — full tenancy control), OR
//   (b) they hold any of the required scopes in their scopes array.
// Tenant isolation is enforced separately by TenantScope (below): even a
// wl_client can only touch rows in their own white_label_id.
func RequireScope(allowedScopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role == roles.WLClient {
			c.Next()
			return
		}
		scopesVal, _ := c.Get("scopes")
		held, _ := scopesVal.([]string)
		heldSet := make(map[string]bool, len(held))
		for _, s := range held {
			heldSet[s] = true
		}
		for _, allowed := range allowedScopes {
			if heldSet[allowed] {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error":          "insufficient scope",
			"required_scopes": allowedScopes,
		})
	}
}

// TenantScope enforces per-WL-client isolation. It loads the caller's
// white_label_id from the JWT and stashes it in context as 'tenant_id'. Every
// handler MUST filter its queries by this id. A caller without a white_label_id
// (e.g. a platform SuperAdmin acting cross-tenant) is rejected here — the WL
// admin panel is strictly single-tenant.
func TenantScope() gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get("white_label_id")
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "no tenant in token"})
			return
		}
		c.Set("tenant_id", v.(uuid.UUID))
		c.Next()
	}
}

// TenantID returns the caller's white_label_id from context (set by TenantScope).
func TenantID(c *gin.Context) uuid.UUID {
	if v, ok := c.Get("tenant_id"); ok {
		return v.(uuid.UUID)
	}
	return uuid.Nil
}

// AdminID returns the caller's admin id from context.
func AdminID(c *gin.Context) uuid.UUID {
	if v, ok := c.Get("admin_id"); ok {
		return v.(uuid.UUID)
	}
	return uuid.Nil
}

func RateLimiter(cfg *config.Config) gin.HandlerFunc {
	type clientInfo struct {
		requests int
		resetAt  time.Time
	}
	clients := make(map[string]*clientInfo)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		client, exists := clients[ip]
		if !exists {
			clients[ip] = &clientInfo{requests: 1, resetAt: now.Add(cfg.RateLimitWindow)}
			c.Next()
			return
		}

		if now.After(client.resetAt) {
			client.requests = 1
			client.resetAt = now.Add(cfg.RateLimitWindow)
			c.Next()
			return
		}

		if client.requests >= cfg.RateLimitRequests {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}

		client.requests++
		c.Next()
	}
}

func IPWhitelistMiddleware(cfg *config.Config) gin.HandlerFunc {
	allowedIPs := make(map[string]bool)
	for _, ip := range cfg.AllowedIPs {
		allowedIPs[ip] = true
	}

	return func(c *gin.Context) {
		if !cfg.EnableIPWhitelist {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		if allowedIPs[clientIP] {
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "IP address not allowed"})
		c.Abort()
	}
}

func GetAdminID(c *gin.Context) (uuid.UUID, bool) {
	adminID, exists := c.Get("admin_id")
	if !exists {
		return uuid.Nil, false
	}
	return adminID.(uuid.UUID), true
}

func GetAdminRole(c *gin.Context) (string, bool) {
	role, exists := c.Get("role")
	if !exists {
		return "", false
	}
	return role.(string), true
}
