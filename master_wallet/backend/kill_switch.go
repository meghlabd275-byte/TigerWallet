package main

// kill_switch.go — wires the canonical MasterWallet backend into the
// TigerWallet SuperAdmin kill-switch control plane.
//
// The kill_switch service (:8469) writes halts as positive Redis keys
// (kill:global, kill:client:<id>, kill:product:<id>:<product>, kill:fetcher:...)
// and republishes them from PostgreSQL on a self-heal loop. This file adds:
//
//   - Store.IsKillSwitchHalted(): checks the kill:global key in Redis (the
//     shared control-plane Redis). Returns (halted, reason). When Redis is
//     unavailable, returns (false, "") — the canonical operator backend stays
//     available rather than self-paralyzing on a Redis outage (halts are a
//     positive signal; absence of the key == not halted).
//   - KillSwitchMiddleware(): a gin middleware that 503-blocks every /api/v1/
//     request when a global halt is active (except /health, which stays up so
//     monitoring can see the halt). Fail-closed for SuperAdmin-issued halts.
//
// Only a SuperAdmin can issue a halt (via kill_switch :8469 /api/v1/kill/halt,
// superAdminAuth-gated). The MasterWallet owner cannot halt the platform.

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// killGlobalKey is the Redis key the kill_switch service sets for a global
// platform halt (value = the reason string).
const killGlobalKey = "kill:global"

// IsKillSwitchHalted reports whether a global SuperAdmin kill-switch halt is
// currently active. It consults the shared control-plane Redis. When Redis is
// nil (disabled) or unreachable, it returns (false, "") — the canonical
// operator backend does not self-paralyze on a Redis outage.
func (s *Store) IsKillSwitchHalted() (bool, string) {
	if s == nil || s.redis == nil {
		return false, ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	reason, err := s.redis.Get(ctx, killGlobalKey).Result()
	if err != nil {
		// Key missing (redis.Nil) or transient error => not halted.
		return false, ""
	}
	return true, reason
}

// KillSwitchMiddleware blocks every /api/v1/ request with 503 when a global
// kill-switch halt is active. /health and /api/v1/health stay reachable so
// monitoring can observe the halt. The check is best-effort: a Redis outage
// does not halt the canonical backend.
func KillSwitchMiddleware(store *Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/health" || path == "/api/v1/health" || path == "/ws" {
			c.Next()
			return
		}
		if halted, reason := store.IsKillSwitchHalted(); halted {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error":  "platform halted by SuperAdmin kill-switch",
				"reason": reason,
			})
			return
		}
		c.Next()
	}
}

// GetKillSwitchStatus reports the current global kill-switch state (read-only).
// Any authenticated user may read it; only a SuperAdmin may toggle it via the
// kill_switch service (:8469 /api/v1/kill/halt|resume).
func (svc *Service) GetKillSwitchStatus(c *gin.Context) {
	if svc.store == nil {
		c.JSON(http.StatusOK, gin.H{"halted": false, "reason": "", "source": "store-unavailable"})
		return
	}
	halted, reason := svc.store.IsKillSwitchHalted()
	c.JSON(http.StatusOK, gin.H{
		"halted": halted,
		"reason": reason,
		"source": "redis:kill:global",
		"note":   "halt/resume is SuperAdmin-only via kill_switch :8469",
	})
}
