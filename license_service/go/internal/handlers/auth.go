package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/tigerwallet/license-service/internal/config"
)

// Claims carried in every control-plane JWT. The role is either 'superadmin'
// (TigerWallet-side) or 'wl_client' (a WL customer, limited actions only).
type Claims struct {
	AdminID   uuid.UUID `json:"admin_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"` // superadmin | wl_client
	WLClientID *uuid.UUID `json:"wl_client_id,omitempty"`
	jwt.RegisteredClaims
}

// IssueJWT mints a signed JWT for a control-plane operator.
func IssueJWT(cfg *config.Config, adminID uuid.UUID, email, role string, wlClientID *uuid.UUID) (string, error) {
	now := time.Now()
	claims := Claims{
		AdminID:    adminID,
		Email:      email,
		Role:       role,
		WLClientID: wlClientID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   adminID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.JWTExpiry)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(cfg.JWTSecret))
}

// AuthMiddleware validates the Bearer JWT and loads claims into context.
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}
		parts := strings.Split(h, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}
		claims := &Claims{}
		tok, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (any, error) {
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !tok.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set("admin_id", claims.AdminID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		if claims.WLClientID != nil {
			c.Set("wl_client_id", *claims.WLClientID)
		}
		c.Next()
	}
}

// RequireSuperAdmin 403-rejects any caller whose role is not 'superadmin'.
// This is the gate for every governance action: license issue/suspend/revoke,
// WL client approve/halt/resume, feature-flag CRUD, withdrawal co-sign, etc.
// A WL client (role 'wl_client') can NEVER pass this gate.
func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "superadmin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "SuperAdmin privilege required"})
			return
		}
		c.Next()
	}
}

// RequireWLClientOrSuperAdmin allows a WL client to act on its OWN tenancy
// (e.g. request a withdrawal, heartbeat) OR SuperAdmin to act on any tenancy.
func RequireWLClientOrSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "superadmin" && role != "wl_client" && role != "service" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient privileges"})
			return
		}
		c.Next()
	}
}

// ServiceOrUserAuth authenticates a two-party-gate service-to-service call
// (Authorization: Bearer <SERVICE_AUTH_TOKEN> + X-WL-Client-ID header) OR a
// normal user JWT (Bearer <jwt>). When the service token matches, it synthesizes
// role="wl_client" + wl_client_id from the X-WL-Client-ID header so the same
// RequireWLClientOrSuperAdmin gate admits the machine caller. This lets the
// wl_master_wallet / wl_user_wallet backends call /withdrawals/request,
// /withdrawals/:id/approved, /withdrawals/:id/executed WITHOUT a human JWT,
// while SuperAdmin governance (/approve, /reject, list) stays JWT-gated.
//
// Fail-closed: if SERVICE_AUTH_TOKEN is unset, the service path is disabled
// entirely (the call must then present a valid user JWT).
func ServiceOrUserAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}
		parts := strings.Split(h, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			return
		}
		// 1) Service-to-service path: the bearer token matches SERVICE_AUTH_TOKEN.
		if cfg.ServiceToken != "" && parts[1] == cfg.ServiceToken {
			wlIDStr := c.GetHeader("X-WL-Client-ID")
			if wlIDStr != "" {
				if wlID, err := uuid.Parse(wlIDStr); err == nil {
					c.Set("wl_client_id", wlID)
				}
			}
			c.Set("role", "service")
			c.Set("email", "service:"+c.GetHeader("X-Service-Product"))
			c.Next()
			return
		}
		// 2) Human path: parse + verify a normal JWT.
		claims := &Claims{}
		tok, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (any, error) {
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !tok.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set("admin_id", claims.AdminID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		if claims.WLClientID != nil {
			c.Set("wl_client_id", *claims.WLClientID)
		}
		c.Next()
	}
}

// adminID extracts the operator id from context.
func adminID(c *gin.Context) uuid.UUID {
	if v, ok := c.Get("admin_id"); ok {
		return v.(uuid.UUID)
	}
	return uuid.Nil
}
