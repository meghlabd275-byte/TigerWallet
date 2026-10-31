package main

// auth.go — JWT auth + bcrypt password hashing + role-based middleware for the
// MasterWallet backend. Real HMAC-SHA256 JWT signing; no plaintext passwords.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// context keys for authenticated user info.
type ctxKey string

const (
	ctxUserID ctxKey = "mw_user_id"
	ctxEmail  ctxKey = "mw_email"
	ctxRole   ctxKey = "mw_role"
)

// MasterClaims is the JWT claim set for the MasterWallet backend.
type MasterClaims struct {
	UserID string `json:"uid"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// IssueJWT issues a 24h HS256 JWT for a user.
func IssueJWT(secret, userID, email, role string) (string, error) {
	claims := MasterClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "tigerwallet-master-wallet",
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(secret))
}

// ParseJWT validates + parses a JWT.
func ParseJWT(secret, tokenStr string) (*MasterClaims, error) {
	claims := &MasterClaims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !tok.Valid {
		return nil, jwt.ErrTokenInvalidId
	}
	return claims, nil
}

// AuthMiddleware validates the Bearer JWT and loads user info into context.
func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		claims, err := ParseJWT(secret, tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set(string(ctxUserID), claims.UserID)
		c.Set(string(ctxEmail), claims.Email)
		c.Set(string(ctxRole), claims.Role)
		c.Next()
	}
}

// RequireRole returns middleware that 403-rejects users without one of the
// given roles. Used for admin/treasury endpoints.
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		role, _ := c.Get(string(ctxRole))
		roleStr, _ := role.(string)
		if !allowed[roleStr] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}
		c.Next()
	}
}

// hashPassword hashes a password with bcrypt (cost 12).
func hashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", fmt.Errorf("password must be at least 8 characters")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// verifyPassword checks a bcrypt hash.
func verifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// currentUserID extracts the authenticated user id from gin context.
func currentUserID(c *gin.Context) string {
	v, _ := c.Get(string(ctxUserID))
	s, _ := v.(string)
	return s
}

// currentRole extracts the authenticated role from gin context.
func currentRole(c *gin.Context) string {
	v, _ := c.Get(string(ctxRole))
	s, _ := v.(string)
	return s
}

// audit records an audit log entry (best-effort; never blocks the request).
func (s *Store) audit(ctx context.Context, masterWalletID, eventType, category, actorType, actorID, targetType, targetID, severity string, details map[string]interface{}) {
	if s == nil || s.db == nil {
		return
	}
	_, _ = s.db.Exec(ctx,
		`INSERT INTO audit_logs (master_wallet_id, event_type, event_category, actor_type, actor_id, target_type, target_id, severity, details)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		nilIfEmpty(masterWalletID), eventType, category, actorType, actorID, targetType, targetID, severity, detailsJSON(details))
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// ---------------------------------------------------------------------------
// Master-wallet ownership
// ---------------------------------------------------------------------------

// mayAccessMasterWallet reports whether the caller may act on the master wallet
// identified by masterID. The owner is master_wallets.created_by, which
// CreateMasterWallet sets to the authenticated user; an operator role may also
// act on any wallet. A wallet with no recorded owner is reachable only by an
// operator, so a row created outside the API cannot be driven by any tenant.
//
// This mirrors the rule websocket.go already applied to the live feed, but the
// REST handlers under /api/v1/master-wallet/:id did not apply it at all, so any
// authenticated user could read another tenant's wallet, spend from it, or
// manage its multisig and treasury.
func (svc *Service) mayAccessMasterWallet(ctx context.Context, masterID, userID, role string) bool {
	if role == "admin" || role == "super_admin" || role == "operator" {
		return true
	}
	var createdBy string
	if err := svc.store.db.QueryRow(ctx,
		`SELECT COALESCE(created_by::text, '') FROM master_wallets WHERE id = $1`, masterID).Scan(&createdBy); err != nil {
		return false
	}
	return createdBy != "" && createdBy == userID
}

// requireMasterWalletAccess resolves the :id parameter and enforces ownership.
// It writes the 403/400 response itself and returns false when the caller must
// not proceed, so handlers can `if !svc.requireMasterWalletAccess(c) { return }`.
func (svc *Service) requireMasterWalletAccess(c *gin.Context) (string, bool) {
	masterID := c.Param("id")
	if masterID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet id required"})
		return "", false
	}
	if !svc.mayAccessMasterWallet(c.Request.Context(), masterID, currentUserID(c), currentRole(c)) {
		// The same response is returned for "does not exist" and "not yours" so
		// the endpoint cannot be used to enumerate other tenants' wallet ids.
		c.JSON(http.StatusForbidden, gin.H{"error": "not your master wallet"})
		return "", false
	}
	return masterID, true
}
