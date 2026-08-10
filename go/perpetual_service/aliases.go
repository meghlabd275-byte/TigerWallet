package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// listUserPositions is a path-parameter-friendly variant of listPositions.
// The frontend terminal routes call GET /api/v1/perpetual/users/:userId/positions
// (the user id is in the path, optionally mirrored via the user_id query param),
// whereas listPositions reads the caller's user_id from the JWT context. This
// handler accepts both so admin/terminal views can inspect any user.
func (s *service) listUserPositions(c *gin.Context) {
	user := c.Param("userId")
	if user == "" {
		user = c.Query("user_id")
	}
	status := c.Query("status")
	q := `SELECT id,pair_id,side,size::text,entry_price::text,leverage::text,margin::text,status,pnl::text,liq_price::text,extract(epoch from created_at)::bigint FROM perpetual_positions WHERE user_id=$1`
	args := []interface{}{user}
	if status != "" {
		q += ` AND status=$2`
		args = append(args, status)
	}
	q += ` ORDER BY created_at DESC LIMIT 200`
	rows, err := s.pg.Query(c, q, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []gin.H{}
	for rows.Next() {
		var p struct {
			ID, PID, Size, Entry, Lev, Margin, Status, PnL, Liq string
			Side int
			Ts   int64
		}
		if err := rows.Scan(&p.ID, &p.PID, &p.Side, &p.Size, &p.Entry, &p.Lev, &p.Margin, &p.Status, &p.PnL, &p.Liq, &p.Ts); err != nil {
			continue
		}
		out = append(out, gin.H{
			"id": p.ID, "pair_id": p.PID, "side": p.Side, "size": p.Size,
			"entry_price": p.Entry, "leverage": p.Lev, "margin": p.Margin,
			"status": p.Status, "pnl": p.PnL, "liq_price": p.Liq, "created_at": p.Ts,
		})
	}
	c.JSON(http.StatusOK, gin.H{"positions": out, "count": len(out)})
}

// openOrder is an alias for openPosition exposed at /api/v1/perpetual/order to
// match the frontend terminal POST route naming.
func (s *service) openOrder(c *gin.Context) {
	s.openPosition(c)
}
