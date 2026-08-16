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
