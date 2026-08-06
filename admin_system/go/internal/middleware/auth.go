// Middleware - Authentication and authorization middleware
package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tigerwallet/admin/internal/config"
)

type Claims struct {
	AdminID   uuid.UUID `json:"admin_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
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

		c.Next()
	}
}

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
