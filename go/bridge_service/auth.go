package main

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	sharedauth "github.com/tigerwallet/sharedauth"
)

// requireCaller authenticates the request and records the caller's identity.
// Bridge operations move funds for one account, so a request without a valid
// token is rejected rather than attributed to a body-supplied user id.
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

func isAdmin(c *gin.Context) bool {
	v, _ := c.Get("callerIsAdmin")
	b, _ := v.(bool)
	return b
}
