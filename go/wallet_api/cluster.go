package main

// cluster.go — cluster coordination plane for horizontally-scaled wallet_api
// replicas.
//
// Global expansion means many stateless replicas behind a load balancer in
// many regions. Because every replica is stateless (JWT auth, no in-process
// session state), the only things that must be shared cluster-wide are:
//
//   1. Node registry  — which replicas are alive, where, and how loaded.
//      Implemented here as Redis heartbeat hashes with a short TTL; dead
//      replicas disappear automatically within ~15s without any controller.
//   2. Rate limits    — per-IP/user limits must hold across ALL replicas, not
//      per replica (see ratelimit_redis.go).
//   3. Hot data fanout — the live price feed must cost ONE upstream provider
//      call per tick for the whole cluster, not one per replica (see
//      live_feed.go shared cache + fetch lock).
//
// Redis is the coordination plane because the service already depends on it
// for hot caches; no new infrastructure is introduced. The registry is
// best-effort: if Redis is down the service keeps serving (each replica is
// self-sufficient) and the cluster status endpoint reports the registry as
// unavailable instead of fabricating a topology.

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

const (
	clusterNodesSetKey    = "cluster:wallet_api:nodes"
	clusterNodeKeyPref    = "cluster:wallet_api:node:"
	clusterHeartbeatTTL   = 15 * time.Second
	clusterHeartbeatEvery = 5 * time.Second
	serviceVersion        = "1.1.0"
)

// clusterNode describes one live replica.
type clusterNode struct {
	ID          string    `json:"id"`
	Region      string    `json:"region"`
	Host        string    `json:"host"`
	Port        string    `json:"port"`
	Version     string    `json:"version"`
	StartedAt   time.Time `json:"started_at"`
	HeartbeatAt time.Time `json:"heartbeat_at"`
	WSClients   int       `json:"ws_clients"`
}

// clusterRegistry heartbeats this replica's identity into Redis and reads the
// live topology back for operators.
type clusterRegistry struct {
	rdb  *redis.Client
	node clusterNode

	stop   chan struct{}
	wg     sync.WaitGroup
	closed sync.Once
}

// clusterReg is the process-wide registry (nil when Redis is unavailable).
var clusterReg *clusterRegistry

// clusterNodeID returns a stable identity for this replica: the k8s pod name
// (HOSTNAME) when present, else hostname + a short random suffix.
func clusterNodeID() string {
	if h := os.Getenv("HOSTNAME"); h != "" {
		return h
	}
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "node-" + uuid.NewString()[:8]
}

// startClusterRegistry registers this replica and starts the heartbeat loop.
// Returns nil (cluster features disabled) when rdb is nil.
func startClusterRegistry(rdb *redis.Client, port string) *clusterRegistry {
	if rdb == nil {
		return nil
	}
	reg := &clusterRegistry{
		rdb: rdb,
		node: clusterNode{
			ID:        clusterNodeID(),
			Region:    os.Getenv("CLUSTER_REGION"),
			Host:      os.Getenv("POD_IP"),
			Port:      port,
			Version:   serviceVersion,
			StartedAt: time.Now().UTC(),
		},
		stop: make(chan struct{}),
	}
	reg.heartbeat() // register immediately so we are visible before the first tick
	reg.wg.Add(1)
	go reg.loop()
	return reg
}

func (reg *clusterRegistry) loop() {
	defer reg.wg.Done()
	ticker := time.NewTicker(clusterHeartbeatEvery)
	defer ticker.Stop()
	for {
		select {
		case <-reg.stop:
			return
		case <-ticker.C:
			reg.heartbeat()
		}
	}
}

// heartbeat refreshes this node's hash (with TTL) and sweeps expired members
// from the node set so the topology never accumulates dead replicas.
func (reg *clusterRegistry) heartbeat() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	key := clusterNodeKeyPref + reg.node.ID
	reg.node.HeartbeatAt = time.Now().UTC()
	fields := map[string]interface{}{
		"id":           reg.node.ID,
		"region":       reg.node.Region,
		"host":         reg.node.Host,
		"port":         reg.node.Port,
		"version":      reg.node.Version,
		"started_at":   reg.node.StartedAt.Format(time.RFC3339Nano),
		"heartbeat_at": reg.node.HeartbeatAt.Format(time.RFC3339Nano),
		"ws_clients":   liveFeed.clientCount(),
	}
	pipe := reg.rdb.TxPipeline()
	pipe.HSet(ctx, key, fields)
	pipe.PExpire(ctx, key, clusterHeartbeatTTL)
	pipe.SAdd(ctx, clusterNodesSetKey, reg.node.ID)
	if _, err := pipe.Exec(ctx); err != nil {
		return // best-effort; next tick retries
	}
	reg.sweep(ctx)
}

// sweep removes set members whose heartbeat hash has expired.
func (reg *clusterRegistry) sweep(ctx context.Context) {
	ids, err := reg.rdb.SMembers(ctx, clusterNodesSetKey).Result()
	if err != nil {
		return
	}
	for _, id := range ids {
		n, err := reg.rdb.Exists(ctx, clusterNodeKeyPref+id).Result()
		if err == nil && n == 0 {
			reg.rdb.SRem(ctx, clusterNodesSetKey, id) //nolint:errcheck
		}
	}
}

// stop deregisters this replica and halts the heartbeat loop.
func (reg *clusterRegistry) stopRegistry() {
	reg.closed.Do(func() {
		close(reg.stop)
		reg.wg.Wait()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		reg.rdb.SRem(ctx, clusterNodesSetKey, reg.node.ID) //nolint:errcheck
		reg.rdb.Del(ctx, clusterNodeKeyPref+reg.node.ID)   //nolint:errcheck
	})
}

// liveNodes returns every replica with an unexpired heartbeat.
func (reg *clusterRegistry) liveNodes(ctx context.Context) ([]clusterNode, error) {
	ids, err := reg.rdb.SMembers(ctx, clusterNodesSetKey).Result()
	if err != nil {
		return nil, err
	}
	nodes := make([]clusterNode, 0, len(ids))
	for _, id := range ids {
		m, err := reg.rdb.HGetAll(ctx, clusterNodeKeyPref+id).Result()
		if err != nil || len(m) == 0 {
			continue
		}
		n := clusterNode{
			ID:      m["id"],
			Region:  m["region"],
			Host:    m["host"],
			Port:    m["port"],
			Version: m["version"],
		}
		n.StartedAt, _ = time.Parse(time.RFC3339Nano, m["started_at"])
		n.HeartbeatAt, _ = time.Parse(time.RFC3339Nano, m["heartbeat_at"])
		n.WSClients, _ = strconv.Atoi(m["ws_clients"])
		nodes = append(nodes, n)
	}
	return nodes, nil
}

// handleClusterStatus reports the live cluster topology. Admin-gated: the
// topology (node addresses, regions, load) is operational metadata that must
// not be public.
func handleClusterStatus(c *gin.Context) {
	if clusterReg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "cluster registry unavailable (Redis down)",
			"nodes":   []clusterNode{},
			"self_id": clusterNodeID(),
		})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	nodes, err := clusterReg.liveNodes(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": fmt.Sprintf("cluster registry read failed: %v", err)})
		return
	}
	totalWS := 0
	regions := map[string]int{}
	for _, n := range nodes {
		totalWS += n.WSClients
		regions[n.Region]++
	}
	c.JSON(http.StatusOK, gin.H{
		"self_id":          clusterReg.node.ID,
		"node_count":       len(nodes),
		"nodes":            nodes,
		"total_ws_clients": totalWS,
		"regions":          regions,
		"time":             time.Now().UTC(),
	})
}
