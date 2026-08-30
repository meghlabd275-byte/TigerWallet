# TigerWallet Cluster Architecture — Global-Scale MasterWallet & UserWallet

How the signing/approval engine scales to global load, and the exact
guarantees each layer provides. Everything here is implemented and
fail-closed; there are no placeholder components.

## Topology

```
                       ┌──────────── Load Balancer ────────────┐
  UserWallet clients ──┤  (TLS, any replica, stateless JWT)    ├── WebSocket /ws
                       └──────┬───────────┬───────────┬────────┘
                              ▼           ▼           ▼
                     master_wallet/backend replicas (N, stateless)
                     :8450  + auto-signer worker pool per replica
                              │           │           │
              ┌───────────────┴───────────┴───────────┴────────────┐
              ▼                                                    ▼
   PostgreSQL (source of truth)                        Redis (coordination)
   - transactions (claim via                           - mw:events pub/sub
     FOR UPDATE SKIP LOCKED)                             (WS fanout)
   - hash partitions by chain_id                       - mw:price:* shared
     (database/schemas/user_wallet_sharding.sql)         price cache
   - PgBouncer in front at scale                       - kill:global kill switch
```

## 1. Distributed auto-signer (the signing cluster engine)

Every replica runs the auto-signer daemon. Correctness under N replicas:

- **Atomic claiming** — `claimBatch` (`auto_signer.go`) picks up to
  `MASTER_AUTO_SIGN_BATCH` (default 50) pending, auto-approvable transactions
  in ONE statement: `SELECT ... FOR UPDATE SKIP LOCKED` + status flip to
  `approved` + claim marker (`auto_sign_claim` = instance id) stamped into
  metadata. Two replicas never process the same transaction: the row is no
  longer `pending`, and racing pollers skip locked rows instead of waiting.
- **Worker pool** — each claimed batch is processed by
  `MASTER_AUTO_SIGN_WORKERS` (default 4, max 32) goroutines, so one slow RPC
  does not stall the replica's batch.
- **Crash recovery (reaper)** — if a replica dies after claiming but before
  broadcasting, the row sits at `approved` with a claim marker. Any replica's
  next poll returns it to `pending` (attempts+1) once the marker is older
  than 3 minutes. Manual HTTP approvals carry no claim marker and are never
  reaped.
- **Retry with backoff** — transient sign/broadcast failures (RPC down,
  nonce fetch, relay rejection) requeue to `pending` with an incremented
  attempt counter and exponential hold (30s·2^n, capped at 15m). After
  `MASTER_AUTO_SIGN_MAX_ATTEMPTS` (default 5) the row lands in `failed` with
  the real error message — nothing strands silently, nothing loops forever.
- **Refusal holds** — policy/guard refusals (owner disabled the kind, over
  cap, guard rejected) release the row to `pending` with a 5-minute hold so
  blocked rows are not re-claimed every 100ms poll tick, and a policy change
  takes effect within minutes.
- **Security invariants unchanged at any cluster size** — the user-funds
  guard, per-kind policy, value caps, and the two-party SuperAdmin co-sign
  gate (revenue/treasury/fee withdrawals) are evaluated per transaction on
  whichever replica claims it. Keys live only in replica memory; ownership of
  the claim is what prevents two replicas from signing the same tx.

Instance identity: `MASTER_INSTANCE_ID` (pin to the k8s pod name via the
downward API); defaults to `hostname-pid`.

## 2. WebSocket fanout (Redis pub/sub)

`notifyEvent` delivers locally AND publishes to the Redis channel
`mw:events`. Every replica subscribes (`startWSFanout`, `cluster.go`) and
re-broadcasts messages from other origins to its locally connected clients.
A user connected to replica B sees approvals signed by replica A within one
broker round-trip. With Redis down each replica degrades to local-only
delivery (single-instance behavior) — no request fails.

## 3. Shared price cache

Fiat valuation (CoinGecko) goes through `FetchTokenPriceCached`
(`price_fetcher.go`): L1 in-process TTL (60s) + L2 shared Redis
(`mw:price:<coin>`, 60s) + per-coin singleflight. N replicas share ONE
upstream rate-limit budget. On upstream failure USD fields are omitted —
prices are never fabricated.

## 4. Probes & statelessness

- `/health` — cheap liveness.
- `/readyz` — readiness: 503 until PostgreSQL answers (Redis reported but
  non-fatal; its loss degrades fanout/kill-switch, not correctness).
- The API tier is fully stateless: JWT auth, all state in PostgreSQL, all
  coordination in Redis → scale replicas horizontally behind any L7 LB
  (use cookie-less or consistent-hashing only for `/ws` stickiness).

## 5. Database sharding for billions of rows

`database/schemas/user_wallet_sharding.sql` adds PG hash partitions over
`chain_id` for the hot tables (additive, unchecked). Deploy with PgBouncer
(transaction pooling) in front; replicas then hold far fewer connections.

## 6. Kubernetes

`deploy/k8s/masterwallet-backend.yaml` ships a Deployment (3 replicas,
readiness/liveness probes, `MASTER_INSTANCE_ID` from pod name, PodDisruption
Budget) + Service + HPA (CPU 70%, 3–12 replicas). Required env/secrets:
`MASTER_WALLET_DATABASE_URL`, `MASTER_WALLET_REDIS_ADDR`,
`MASTER_WALLET_JWT_SECRET`, `MASTER_AUTO_SIGN_PASSWORD` (broadcast signing).

## 7. Non-EVM broadcast coverage (honest matrix)

Real, tested code paths in `master_wallet/backend`:

| Family | Chains | Path |
|---|---|---|
| EVM | all 120 seeded | `fetchers.go` + `SignEVMTransaction` + `eth_sendRawTransaction` |
| Cosmos-SDK | all 23 seeded | amino SignDoc + LCD broadcast (`cosmos_broadcast.go`), registry-routed |
| Bitcoin | 1 | legacy P2PKH + esplora (`utxo_chains.go`, blockstream.info) |
| Litecoin | 1 | same path, version 0x30, litecoinspace.org (env-overridable) |
| Solana | 1 | ed25519 transfer + JSON-RPC (`solana_broadcast.go`) |
| **Remaining 40** (Tron, NEAR, Cardano, XRP, Stellar, Tezos, TON, Sui, Aptos, Polkadot, Algorand, Hedera, Filecoin, Flow, ICP, Kaspa, Nano, Nervos, VeChain, Waves, Zilliqa, Aleo, MultiversX, Pi, and BTC-derived chains needing fork-id/different hashing: BCH, BSV, eCash, Zcash, Groestlcoin, Dogecoin, Dash, …) | 40 | **fail-closed** explicit error. Each needs a chain-specific SDK/signer — never faked. Adding one = params + signer + relay in `utxo_chains.go`-style tables or a new family file. |

## 8. Transaction history without explorer keys

`GET /api/v1/transactions/history` resolves explorers in order: curated
Etherscan-family map (key via env, e.g. `ETHERSCAN_API_KEY`) → registry
`ExplorerURL` + `/api` (Blockscout-compatible, **keyless**) → 503 with a
descriptive error. Upstream errors propagate verbatim.

---

# UserWallet Backend (go/wallet_api) — Cluster Engine & Global Expansion

How the canonical UserWallet backend (`go/wallet_api`, :8443) scales to
billions of users, transactions, swaps, and trades across regions.

## Design principle: stateless replicas + shared coordination plane

Every wallet_api replica is **stateless**:

- Auth is JWT — any replica can serve any request, no session stickiness.
- Durable state lives in PostgreSQL (sharded — see
  `docs/USER_WALLET_SHARDING.md`).
- Hot state (balances, prices, gas, rate limits, cluster registry, live-feed
  cache) lives in Redis.

Therefore the fleet scales horizontally without code changes: add replicas
behind the load balancer and every cluster mechanism below picks them up
automatically.

## Cluster mechanisms (implemented in `go/wallet_api`)

### 1. Node registry — `cluster.go`

Each replica heartbeats its identity (pod name, region, IP, version, started
time, connected WebSocket clients) into a Redis hash every 5s with a 15s TTL,
and sweeps expired members from the node set. Dead replicas disappear from the
topology within ~15s with no controller.

Operators read the live topology via the admin-gated endpoint:

```
GET /api/v1/admin/cluster/status   (JWT + admin role)
→ { self_id, node_count, nodes[], total_ws_clients, regions{} }
```

On shutdown a replica deregisters immediately (before the heartbeat TTL), so
rolling updates never show phantom nodes.

### 2. Cluster-wide rate limiting — `ratelimit_redis.go`

A per-replica token bucket silently multiplies limits by the replica count.
The auth (5/min) and funds-movement signing (20/min) limiters are backed by a
**Redis Lua token bucket** (atomic check-and-consume per key), so the same
policy holds across the entire fleet. Bucket keys carry a TTL derived from
the refill horizon — idle clients cost zero memory.

Failure policy is fail-closed: if Redis is unreachable, each replica falls
back to its in-process bucket (limiting more, never less). A Redis outage can
never open the credential or funds surfaces to unlimited traffic.

### 3. Cluster-shared live price feed — `live_feed.go`

The public WebSocket feed (`GET /api/v1/ws`) batches all subscribed symbols
into ONE upstream provider call per tick. In a cluster, one replica fetches
per tick for the whole fleet:

- A Redis `SET NX` lock (`livefeed:fetchlock`, TTL = 0.8 × tick) elects the
  fetcher for each tick; lock expiry = automatic failover.
- The fetcher writes each ticker to the shared cache
  (`livefeed:ticker:<SYM>`, TTL = 2 × tick) and broadcasts locally.
- Every other replica serves its own subscribers from the shared cache
  (MGET per tick).

Result: N replicas × M clients still cost **exactly one upstream call per
tick**. Fail-closed: an upstream outage produces error frames, never a
fabricated price; a cache miss just skips a tick.

### 4. Health probes for load balancers — `handlers.go`

- `GET /health/live` — process is alive (k8s liveness; never fails on a
  dependency hiccup, so replicas are not restart-looped by a DB blip).
- `GET /health/ready` — PostgreSQL + Redis reachable (k8s readiness; the
  Service only routes to replicas that can actually serve).

### 5. Connection pool governance — `store.go`

Per-replica PG pool is env-tunable (`PG_MAX_CONNS`, `PG_MIN_CONNS`,
`PG_MAX_CONN_LIFETIME_MIN`, `PG_MAX_CONN_IDLE_MIN`). The invariant:

```
replica_count × PG_MAX_CONNS ≤ Postgres connection budget
```

Front Postgres with PgBouncer (transaction pooling) when the fleet grows
past the budget (~100 replicas at the default 25 conns).

## Kubernetes deployment — `k8s/wallet-api.yaml`

- Deployment: 4+ replicas, `maxUnavailable: 0` rolling updates, zone
  topology spread, preStop drain + 30s graceful termination.
- Service: ClusterIP, no session affinity (state is cluster-shared).
- HPA: 4–100 replicas on 65% CPU (fast scale-up, damped scale-down).
- PDB: `minAvailable: 2` so voluntary disruption never drops the fleet.
- Pod identity (`HOSTNAME`, `POD_IP`, `CLUSTER_REGION`) feeds the node
  registry via the downward API.

## Global expansion topology

```
                    geo-DNS / anycast
                 /      |        \
          us-east   eu-west    ap-southeast
          ┌──────┐   ┌──────┐   ┌──────┐
          │wallet│   │wallet│   │wallet│   stateless replicas (HPA 4..100)
          │-api ×N│  │-api ×N│  │-api ×N│
          └──┬───┘   └──┬───┘   └──┬───┘
             │          │          │
        regional Redis (coordination + hot cache)
             │          │          │
        PostgreSQL: regional primaries, hash-sharded by chain_id
        (docs/USER_WALLET_SHARDING.md); cross-region read replicas
        for balance/tx reads, single-writer per shard for sends.
```

- **Reads** (balances, prices, tx history, market data) are served from the
  nearest region: Redis hot cache + PG read replicas.
- **Writes** (send/sign/swap/trade) route to the shard primary for the
  wallet's chain; idempotency keys on POSTs make client retries safe across
  regions.
- **Cross-service back ends** (swap aggregator, bridge, perpetuals, copy
  trading, P2P) are already separate Go services proxied by wallet_api — each
  scales independently with the same stateless pattern.

## What this does NOT change

- **App separation**: UserWallet replicas never import MasterWallet/admin
  fetchers; multisig/cards remain service-token proxies.
- **Security**: JWT_SECRET fail-closed, 2FA, kill switch, encrypted seed
  storage — all unchanged; the cluster plane adds no new trust surface
  (Redis keys are internal-only, admin status endpoint is role-gated).
