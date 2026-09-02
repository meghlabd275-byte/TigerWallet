package handlers

// trading_guard.go — WL-UserWallet enforcement side of the shared trading
// control-plane. A white-label client's stop/halt decisions (published by
// white_label_admin to "trading:control:<tenant>:*") AND TigerWallet
// SuperAdmin global decisions ("trading:control:global:*") gate this
// tenancy's swap / margin / perpetual / liquidity flows.
//
// Blacklist semantics: unmanaged entities trade freely, so enabling the
// control-plane never breaks existing flows. A Redis outage fails OPEN — an
// infra blip never halts trading by itself (owner policy: every user performs
// all swap and trading continuously); explicit operator stops are enforced
// whenever Redis is reachable. Position CLOSES are never gated — a halt must
// never trap user funds.

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// tradingStopped reports whether the given vertical (and optionally a
// specific entity key under it) has been stopped by SuperAdmin (global
// namespace) or this WL client (tenant namespace). When it returns true it
// has already written the 403 response.
func (s *Svc) tradingStopped(c *gin.Context, vertical, kind, key string) bool {
	if s.rdb == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 500*time.Millisecond)
	defer cancel()
	tenants := []string{"global"}
	if s.cfg.WLClientID != "" {
		tenants = append(tenants, s.cfg.WLClientID)
	}
	for _, tenant := range tenants {
		v, err := s.rdb.Get(ctx, "trading:control:"+tenant+":vertical:"+vertical).Result()
		if err == nil && v == "stopped" {
			c.JSON(http.StatusForbidden, gin.H{"error": vertical + " trading is halted by the platform operator"})
			return true
		}
		if key == "" {
			continue
		}
		v, err = s.rdb.Get(ctx, "trading:control:"+tenant+":"+kind+":"+strings.ToUpper(key)).Result()
		if err == nil && (v == "stopped" || v == "removed" || v == "suspended") {
			c.JSON(http.StatusForbidden, gin.H{"error": kind + " " + key + " is stopped by the platform operator"})
			return true
		}
	}
	return false
}

// tradingPairStopped checks both orderings of a pair symbol against the
// control plane (either ordering may be how the operator registered it).
func (s *Svc) tradingPairStopped(c *gin.Context, a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	A, B := strings.ToUpper(a), strings.ToUpper(b)
	return s.tradingStopped(c, "spot", "pair", A+"/"+B) ||
		s.tradingStopped(c, "spot", "pair", B+"/"+A)
}
