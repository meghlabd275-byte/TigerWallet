package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// Hub fans out real-time commands to WL products via Redis pub/sub. A WL
// product subscribes to its channel on connect and also receives pending
// commands on each heartbeat (pull-based fallback). This dual delivery ensures
// commands reach the product even if the pub/sub connection is flaky.
type Hub struct {
	rdb *redis.Client
}

func NewHub(addr, password string) *Hub {
	rdb := redis.NewClient(&redis.Options{Addr: addr, Password: password})
	return &Hub{rdb: rdb}
}

func (h *Hub) Close() error { return h.rdb.Close() }

// PublishCommand pushes a command to the WL product's channel.
func (h *Hub) PublishCommand(ctx context.Context, wlClientID uuid.UUID, product string, cmd map[string]any) error {
	ch := fmt.Sprintf("wl:commands:%s:%s", wlClientID, product)
	b, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	return h.rdb.Publish(ctx, ch, b).Err()
}

// Subscribe returns a channel of command JSON for a WL product. The caller
// (WL product side, via the Rust SDK) listens and executes.
func (h *Hub) Subscribe(ctx context.Context, wlClientID uuid.UUID, product string) <-chan map[string]any {
	ch := fmt.Sprintf("wl:commands:%s:%s", wlClientID, product)
	pubsub := h.rdb.Subscribe(ctx, ch)
	out := make(chan map[string]any, 16)
	go func() {
		defer close(out)
		defer pubsub.Close()
		msgCh := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgCh:
				if !ok {
					return
				}
				var cmd map[string]any
				if err := json.Unmarshal([]byte(msg.Payload), &cmd); err == nil {
					out <- cmd
				}
			}
		}
	}()
	return out
}

// PublishFlagChange notifies a WL product that its feature-flag set changed so
// it should re-pull and refresh its in-memory flag cache immediately (rather
// than waiting for the next heartbeat TTL).
func (h *Hub) PublishFlagChange(ctx context.Context, wlClientID uuid.UUID, product string) error {
	ch := fmt.Sprintf("wl:flags:%s:%s", wlClientID, product)
	return h.rdb.Publish(ctx, ch, `{"event":"flags_changed"}`).Err()
}

// Killed reports whether the kill-switch service has halted the platform, this
// WL client, or this product. A halt is a positive signal (key exists). A
// Redis error fails OPEN here — the kill-switch republishes active halts from
// PostgreSQL every few seconds, so a Redis blip self-heals and can never be
// mistaken for a halt, and a halt is never inferred from missing data.
func (h *Hub) Killed(ctx context.Context, wlClientID uuid.UUID, product string) (bool, string) {
	keys := []string{
		"kill:global",
		"kill:client:" + wlClientID.String(),
		"kill:product:" + wlClientID.String() + ":" + product,
	}
	vals, err := h.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return false, ""
	}
	for i, v := range vals {
		if v != nil {
			reason, _ := v.(string)
			return true, fmt.Sprintf("killed:%s:%s", keys[i], reason)
		}
	}
	return false, ""
}

// KilledFetchers returns which fetchers of a product are currently halted by
// the kill-switch (keys kill:fetcher:<client>:<product>:<fetcher>). The result
// is overlaid onto the feature-flag snapshot delivered on validate/heartbeat so
// a fetcher-scope emergency halt reaches the product on the next beat — products
// only ever receive their flags on beat, so this is the single enforcement path.
// A Redis error returns nil (fail OPEN on the read): the halt still lives in
// PostgreSQL and the kill-switch self-heal loop republishes it, so a transient
// blip cannot be silently mistaken for "no halt".
func (h *Hub) KilledFetchers(ctx context.Context, wlClientID uuid.UUID, product string) map[string]string {
	fs := FetchersByProduct[product]
	if len(fs) == 0 {
		return nil
	}
	keys := make([]string, 0, len(fs))
	for _, f := range fs {
		keys = append(keys, "kill:fetcher:"+wlClientID.String()+":"+product+":"+f)
	}
	vals, err := h.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil
	}
	out := map[string]string{}
	for i, v := range vals {
		if v != nil {
			if reason, ok := v.(string); ok {
				out[fs[i]] = reason
			}
		}
	}
	return out
}
