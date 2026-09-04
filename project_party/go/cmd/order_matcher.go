package main

// order_matcher.go — real continuous matching engine for market_maker_orders.
//
// A background goroutine matches crossing buy/sell orders per token
// (buy.price >= sell.price) atomically in a single PostgreSQL transaction per
// cycle: both order rows are locked FOR UPDATE, the settle quantity and price
// are computed in SQL (numeric, never float), both remaining quantities are
// decremented, orders reaching zero remaining are marked filled (real
// filled_at), and a market_maker_trades row records the real settlement.
// Expired pending orders are cancelled each cycle. No simulated fills.

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// startOrderMatcher launches the continuous matching loop. The interval is
// configurable via PP_MATCH_INTERVAL_MS (default 5000ms, min 500ms).
func startOrderMatcher(db *pgxpool.Pool) {
	intervalMS := 5000
	if v := os.Getenv("PP_MATCH_INTERVAL_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 500 {
			intervalMS = n
		}
	}
	go func() {
		ticker := time.NewTicker(time.Duration(intervalMS) * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			settled, err := matchMarketMakerOrders(ctx, db)
			cancel()
			if err != nil {
				log.Printf("order matcher: %v", err)
			} else if settled > 0 {
				log.Printf("order matcher: settled %d trade(s)", settled)
			}
		}
	}()
	log.Printf("order matcher started: interval=%dms", intervalMS)
}

// matchMarketMakerOrders settles one batch of crossing orders atomically.
// Returns the number of real trades settled.
func matchMarketMakerOrders(ctx context.Context, db *pgxpool.Pool) (int, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Expire stale pending orders first (real cancellation).
	if _, err := tx.Exec(ctx,
		`UPDATE market_maker_orders SET status='cancelled' WHERE status='pending' AND expires_at < NOW()`); err != nil {
		return 0, err
	}

	// Find crossing pairs with open interest on both sides. The regex guard
	// keeps non-numeric legacy rows out of the numeric casts (fail-closed:
	// they simply never match). Highest bid first / lowest ask first —
	// price priority, then time priority.
	rows, err := tx.Query(ctx, `
		SELECT b.id, s.id, b.token_id,
		       LEAST(b.remaining::numeric, s.remaining::numeric)::text,
		       (CASE WHEN b.created_at <= s.created_at THEN b.price ELSE s.price END)::numeric::text
		FROM market_maker_orders b
		JOIN market_maker_orders s
		  ON s.token_id = b.token_id AND s.id <> b.id
		WHERE b.status='pending' AND s.status='pending'
		  AND b.side='buy' AND s.side='sell'
		  AND b.price ~ '^\d+(\.\d+)?$' AND s.price ~ '^\d+(\.\d+)?$'
		  AND b.remaining ~ '^\d+(\.\d+)?$' AND s.remaining ~ '^\d+(\.\d+)?$'
		  AND b.remaining::numeric > 0 AND s.remaining::numeric > 0
		  AND b.price::numeric >= s.price::numeric
		ORDER BY b.price::numeric DESC, s.price::numeric ASC, b.created_at, s.created_at
		LIMIT 200
		FOR UPDATE OF b, s`)
	if err != nil {
		return 0, err
	}
	type cross struct {
		buyID, sellID, tokenID uuid.UUID
		qty, price             string
	}
	var pairs []cross
	for rows.Next() {
		var p cross
		if err := rows.Scan(&p.buyID, &p.sellID, &p.tokenID, &p.qty, &p.price); err != nil {
			rows.Close()
			return 0, err
		}
		pairs = append(pairs, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	settle := `
		UPDATE market_maker_orders
		SET remaining = (remaining::numeric - $2::numeric)::text,
		    status = CASE WHEN remaining::numeric - $2::numeric <= 0 THEN 'filled' ELSE status END,
		    filled_at = CASE WHEN remaining::numeric - $2::numeric <= 0 THEN NOW() ELSE filled_at END
		WHERE id=$1 AND status='pending'`

	settled := 0
	for _, p := range pairs {
		// Settle the buy side: decrement remaining; mark filled at zero.
		ct, err := tx.Exec(ctx, settle, p.buyID, p.qty)
		if err != nil {
			return settled, err
		}
		if ct.RowsAffected() == 0 {
			continue // already settled by another pair this cycle
		}
		// Settle the sell side identically.
		if _, err := tx.Exec(ctx, settle, p.sellID, p.qty); err != nil {
			return settled, err
		}
		// Record the real settlement.
		if _, err := tx.Exec(ctx, `
			INSERT INTO market_maker_trades (id, token_id, buy_order_id, sell_order_id, price, quantity, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,NOW())`,
			uuid.New(), p.tokenID, p.buyID, p.sellID, p.price, p.qty); err != nil {
			return settled, err
		}
		settled++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return settled, nil
}

// getMakerTradesHandler lists real settled trades (public, optionally filtered
// by token_id).
func getMakerTradesHandler(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		var rows pgx.Rows
		var err error
		if tokenID := c.Query("token_id"); tokenID != "" {
			rows, err = db.Query(ctx,
				`SELECT id, token_id, buy_order_id, sell_order_id, price, quantity, created_at
				 FROM market_maker_trades WHERE token_id=$1 ORDER BY created_at DESC LIMIT 100`, tokenID)
		} else {
			rows, err = db.Query(ctx,
				`SELECT id, token_id, buy_order_id, sell_order_id, price, quantity, created_at
				 FROM market_maker_trades ORDER BY created_at DESC LIMIT 100`)
		}
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
			return
		}
		defer rows.Close()
		trades := []gin.H{}
		for rows.Next() {
			var id, tokenID, buyID, sellID uuid.UUID
			var price, qty string
			var createdAt time.Time
			if err := rows.Scan(&id, &tokenID, &buyID, &sellID, &price, &qty, &createdAt); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database scan failed"})
				return
			}
			trades = append(trades, gin.H{
				"id": id, "token_id": tokenID, "buy_order_id": buyID, "sell_order_id": sellID,
				"price": price, "quantity": qty, "created_at": createdAt,
			})
		}
		c.JSON(http.StatusOK, gin.H{"trades": trades})
	}
}
