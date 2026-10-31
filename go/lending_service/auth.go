package main

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	sharedauth "github.com/tigerwallet/sharedauth"
)

// requireCaller authenticates the request and records the caller's identity.
// Lending positions belong to one account, so the address in the request body
// must be one the authenticated caller actually owns.
func requireCaller() gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "authentication is not configured"})
			return
		}
		h := c.GetHeader("Authorization")
		if len(h) < 8 || !strings.EqualFold(h[:7], "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		id, err := sharedauth.Parse(secret, strings.TrimSpace(h[7:]))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.Set("callerID", id.UserID)
		c.Set("callerIsAdmin", id.IsAdmin())
		c.Next()
	}
}

func callerID(c *gin.Context) string {
	v, _ := c.Get("callerID")
	s, _ := v.(string)
	return s
}

func callerIsAdmin(c *gin.Context) bool {
	v, _ := c.Get("callerIsAdmin")
	b, _ := v.(bool)
	return b
}

// ownedAddresses returns the wallet addresses belonging to the caller, primary
// wallet first. An empty result means the caller has no wallet yet.
func (ls *LendingService) ownedAddresses(c *gin.Context) []string {
	if ls.pg == nil {
		return nil
	}
	rows, err := ls.pg.Query(context.Background(),
		`SELECT address FROM wallets WHERE user_id = $1 ORDER BY is_primary DESC, created_at ASC`,
		callerID(c))
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err == nil && a != "" {
			out = append(out, a)
		}
	}
	return out
}

// resolveAddress validates a requested address against the caller's wallets.
// An admin may act on any address (support workflow). A non-admin may only use
// an address they own; when none is supplied the primary wallet is used. It
// writes the error response and returns false when the address is not allowed.
func (ls *LendingService) resolveAddress(c *gin.Context, requested string) (string, bool) {
	if callerIsAdmin(c) && requested != "" {
		return requested, true
	}
	owned := ls.ownedAddresses(c)
	if requested != "" {
		for _, a := range owned {
			if strings.EqualFold(a, requested) {
				return a, true
			}
		}
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "address does not belong to this account"})
		return "", false
	}
	if len(owned) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "no wallet found for this account"})
		return "", false
	}
	return owned[0], true
}
