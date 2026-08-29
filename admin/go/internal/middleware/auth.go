package middleware

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"github.com/tigerwallet/admin/pkg/auth"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware creates authentication middleware
func AuthMiddleware(authSvc *auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Verify token
		claims, err := authSvc.VerifyToken(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Set admin info in context
		c.Set("admin_id", uint(claims["admin_id"].(float64)))
		c.Set("admin_email", claims["email"])
		c.Set("admin_role", claims["role"])

		c.Next()
	}
}

// RoleMiddleware creates role-based authorization middleware
func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("admin_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
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

// SuperAdminMiddleware creates super admin only middleware
func SuperAdminMiddleware() gin.HandlerFunc {
	return RoleMiddleware("super_admin")
}

// AdminMiddleware creates admin or higher middleware
func AdminMiddleware() gin.HandlerFunc {
	return RoleMiddleware("super_admin", "admin")
}

// DomainScopeMiddleware enforces per-domain RBAC for the 11 governance domains.
// It grants full access to super_admin, read+write to admin, read-only to
// support/analyst/moderator on GET, and denies write to non-admin roles.
// scope is the domain name (e.g. "futures", "options", "onramp").
func DomainScopeMiddleware(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("admin_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		roleStr, ok := role.(string)
		if !ok || roleStr == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		// super_admin has unrestricted access to all domains.
		if roleStr == "super_admin" {
			c.Next()
			return
		}

		method := c.Request.Method
		isWrite := method == "POST" || method == "PUT" || method == "PATCH" || method == "DELETE"

		// admin role: full CRUD on all governance domains.
		if roleStr == "admin" {
			c.Next()
			return
		}

		// support/analyst/moderator: read-only on governance domains.
		if !isWrite && (roleStr == "support" || roleStr == "analyst" || roleStr == "moderator") {
			c.Next()
			return
		}

		if isWrite {
			c.JSON(http.StatusForbidden, gin.H{"error": "write access to " + scope + " requires admin role"})
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": "access to " + scope + " denied"})
		}
		c.Abort()
	}
}

// PermissionMiddleware creates permission checking middleware
func PermissionMiddleware(requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get admin role from context
		role, exists := c.Get("admin_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		roleStr := role.(string)

		// Super admin has all permissions
		if roleStr == "super_admin" {
			c.Next()
			return
		}

		// Check role-based permissions
		rolePermissions := map[string][]string{
			"admin": {
				"users.read", "users.write",
				"transactions.read",
				"kyc.read", "kyc.write",
				"wallets.read",
				"analytics.read",
			},
			"support": {
				"users.read",
				"transactions.read",
				"kyc.read",
			},
			"analyst": {
				"users.read",
				"transactions.read",
				"analytics.read",
			},
			"moderator": {
				"users.read", "users.write",
				"transactions.read",
				"kyc.read", "kyc.write",
			},
		}

		permissions, ok := rolePermissions[roleStr]
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		for _, perm := range permissions {
			if perm == requiredPermission {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		c.Abort()
	}
}

// RateLimitMiddleware creates rate limiting middleware
type RateLimiter struct {
	requests map[string]int
	limits   map[string]int
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		requests: make(map[string]int),
		limits:   make(map[string]int),
	}
}

func (rl *RateLimiter) Middleware(requestsPerMinute int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP
		clientIP := c.ClientIP()
		key := "rate_limit:" + clientIP

		// In production, use Redis for distributed rate limiting
		// For now, use in-memory
		rl.requests[key]++

		if rl.requests[key] > requestsPerMinute {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// CORSMiddleware creates CORS middleware
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")

		c.Next()
	}
}

// RequestIDMiddleware adds request ID to context
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// generateRequestID returns a random RFC-4122 v4 UUID. The previous
// implementation used strings.ReplaceAll on a template, which replaced every
// placeholder with the SAME computed digit — producing near-constant IDs that
// broke request tracing and audit correlation.
func generateRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failure is practically unreachable; fall back to a
		// nanotime-derived ID rather than returning a constant.
		return fmt.Sprintf("%x-%x", time.Now().UnixNano(), time.Now().Unix())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
