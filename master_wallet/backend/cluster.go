package main

// cluster.go — MasterWallet CLUSTER ENGINE: everything that makes
// master_wallet/backend safe and correct to run as N replicas behind a load
// balancer serving global-scale UserWallet traffic.
//
// Building blocks (all fail-closed, all backward-compatible with a single
// instance):
//
//  1. Distributed auto-signer claiming (auto_signer.go): pending transactions
//     are claimed with SELECT ... FOR UPDATE SKIP LOCKED + an atomic status
//     flip, so N replicas each process a DISJOINT batch — no double-signing,
//     no double-broadcasting. A claim marker (instance id + timestamp) is
//     stamped into tx metadata; a reaper returns txs whose claiming instance
//     crashed mid-flight back to 'pending' (bounded by a max-attempts
//     counter). Policy/guard refusals get a metadata hold with exponential
//     backoff so blocked rows are not re-claimed every poll tick.
//
//  2. WebSocket fanout over Redis pub/sub (this file + websocket.go): events
//     (approvals, broadcasts, balance changes) are published to the shared
//     Redis channel mw:events and every replica re-broadcasts to its locally
//     connected clients. A user connected to replica B sees events produced
//     by replica A. Messages carry the origin instance id so the publisher
//     does not double-deliver to its own clients. With Redis down, each
//     replica degrades to local-only delivery (single-instance behavior).
//
//  3. Readiness probe (/readyz): reports DB + Redis health for Kubernetes
//     readiness gates; /health stays a cheap liveness probe.
//
//  4. Shared price cache (price_fetcher.go): fiat valuations are cached in
//     Redis (cluster-wide) + in-process, so N replicas share one CoinGecko
//     rate-limit budget instead of each hammering the API per request.
//
// Cluster-wide invariants:
//   - The API tier is stateless: auth is JWT, all mutable state lives in
//     PostgreSQL, all ephemeral coordination lives in Redis.
//   - Signing keys never leave the individual replica's memory; claim
//     ownership is what prevents two replicas from using them on the same tx.
//   - Revenue/treasury/fund-custody operations remain two-party SuperAdmin
//     co-signed (license_gate.go) regardless of cluster size.

import (
        "context"
        "encoding/json"
        "fmt"
        "log"
        "net/http"
        "os"
        "strconv"
        "strings"
        "time"

        "github.com/gin-gonic/gin"
)

// mwEventsChannel is the shared Redis pub/sub channel for cross-replica
// websocket fanout.
const mwEventsChannel = "mw:events"

// instanceID uniquely identifies this replica. Operators can pin it via
// MASTER_INSTANCE_ID (e.g. the k8s pod name via the downward API); otherwise
// hostname+pid is unique enough for claim markers and fanout dedup.
var instanceID = func() string {
        if v := strings.TrimSpace(os.Getenv("MASTER_INSTANCE_ID")); v != "" {
                return v
        }
        host, err := os.Hostname()
        if err != nil || host == "" {
                host = "unknown"
        }
        return fmt.Sprintf("%s-%d", host, os.Getpid())
}()

// clusterIntEnv reads a positive integer env var with a default.
func clusterIntEnv(key string, def int) int {
        v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
        if err != nil || v <= 0 {
                return def
        }
        return v
}

// ----------------------------------------------------------------------------
// WebSocket fanout over Redis pub/sub
// ----------------------------------------------------------------------------

// wsFanoutMessage is the envelope published to mw:events. Origin lets every
// replica skip messages it published itself (it already delivered locally).
type wsFanoutMessage struct {
        Origin   string                 `json:"origin"`
        MasterID string                 `json:"master_wallet_id"`
        Payload  map[string]interface{} `json:"payload"`
}

// publishWSEvent fan-outs a websocket event to all replicas via Redis. Errors
// are logged and swallowed: local delivery already happened, so a Redis
// outage degrades fanout to single-replica scope instead of failing requests.
func (svc *Service) publishWSEvent(masterID string, payload gin.H) {
        if svc.store == nil || svc.store.redis == nil {
                return
        }
        env := wsFanoutMessage{Origin: instanceID, MasterID: masterID, Payload: payload}
        data, err := json.Marshal(env)
        if err != nil {
                return
        }
        ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
        defer cancel()
        if err := svc.store.redis.Publish(ctx, mwEventsChannel, data).Err(); err != nil {
                log.Printf("ws-fanout: publish failed (replica-local delivery only): %v", err)
        }
}

// startWSFanout subscribes to mw:events and re-broadcasts events from OTHER
// replicas to this replica's locally connected websocket clients. Runs until
// ctx is cancelled; reconnects with backoff on Redis errors.
func (svc *Service) startWSFanout(ctx context.Context) {
        if svc.store == nil || svc.store.redis == nil {
                log.Println("ws-fanout: Redis unavailable — cross-replica event fanout disabled (local delivery only)")
                return
        }
        log.Printf("ws-fanout: subscribed to %s as %s", mwEventsChannel, instanceID)
        for {
                select {
                case <-ctx.Done():
                        return
                default:
                }
                func() {
                        defer func() {
                                if r := recover(); r != nil {
                                        log.Printf("ws-fanout: panic recovered: %v", r)
                                }
                        }()
                        sub := svc.store.redis.Subscribe(ctx, mwEventsChannel)
                        defer sub.Close()
                        ch := sub.Channel()
                        for {
                                select {
                                case <-ctx.Done():
                                        return
                                case msg, ok := <-ch:
                                        if !ok {
                                                return
                                        }
                                        var env wsFanoutMessage
                                        if err := json.Unmarshal([]byte(msg.Payload), &env); err != nil {
                                                continue
                                        }
                                        if env.Origin == instanceID {
                                                continue // already delivered locally
                                        }
                                        if svc.hub != nil && env.MasterID != "" {
                                                svc.hub.broadcast(env.MasterID, env.Payload)
                                        }
                                }
                        }
                }()
                // Reconnect backoff after a subscription drop.
                select {
                case <-ctx.Done():
                        return
                case <-time.After(2 * time.Second):
                }
        }
}

// ----------------------------------------------------------------------------
// Readiness probe
// ----------------------------------------------------------------------------

// readyz reports whether this replica can serve traffic: PostgreSQL reachable
// (required) and Redis reachable (reported; Redis loss degrades fanout +
// kill-switch checks but must not pull the replica out of rotation).
func (svc *Service) readyz(c *gin.Context) {
        ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
        defer cancel()
        dbOK := false
        if svc.store != nil && svc.store.db != nil {
                dbOK = svc.store.db.Ping(ctx) == nil
        }
        redisOK := false
        if svc.store != nil && svc.store.redis != nil {
                redisOK = svc.store.redis.Ping(ctx).Err() == nil
        }
        status := http.StatusOK
        state := "ready"
        if !dbOK {
                status = http.StatusServiceUnavailable
                state = "not_ready"
        }
        c.JSON(status, gin.H{
                "status":   state,
                "db":       dbOK,
                "redis":    redisOK,
                "instance": instanceID,
                "time":     time.Now().UTC(),
        })
}
