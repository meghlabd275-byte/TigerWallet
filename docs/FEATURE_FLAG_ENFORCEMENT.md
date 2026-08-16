# Feature-Flag Enforcement Layer

TigerWallet uses a **shared Redis store** as the feature-flag source of truth
(LaunchDarkly-style). Admin backends **write** flag state to Redis; downstream
services **read** it and gate behavior on it. This gives SuperAdmin the ability
to *behaviorally* halt/pause/start/resume a feature — not just flip a DB row.

This respects app separation: there are **no code imports** between admin apps
and wallet apps. The only coupling is the shared Redis namespace.

## Redis key format + states

| Key | Value (string) | TTL |
| --- | --- | --- |
| `tigerwallet:feature:<name>` | `enabled` \| `disabled` \| `paused` | none (persistent; admin-controlled) |

- `enabled` — feature is live; gated endpoints serve normally.
- `disabled` — feature is off; gated endpoints return `423 Locked`.
- `paused` — feature is temporarily suspended. Resume → `enabled`. Gated
  endpoints return `423 Locked` (same observable behavior as `disabled`, but the
  state is distinct so admins can tell "halted" from "permanently off").

A missing key, an unknown value, or a Redis error is treated as `disabled`
(**fail-closed**): a downstream service never lets a gated operation through when
it cannot confirm the feature is `enabled`.

## How admin backends publish to Redis

Each admin backend (`admin/go`, `super_admin/go`, `white_label_admin/go`) owns a
Redis client and writes the flag state on every toggle/update:

- On **Toggle / SetStatus / SetFeature**: after updating the DB row, the backend
  calls `PublishFeatureState(name, state)` → `SET tigerwallet:feature:<name> <state>`.
  - `is_enabled=true` → `enabled`
  - `is_enabled=false` → `disabled`
  - a `status`/`paused` field → `paused`
- On **Delete**: the backend calls `DeleteFeatureState(name)` →
  `DEL tigerwallet:feature:<name>` (so the downstream service fails closed).
- **Live check endpoint**: `GET /features/:id/check` (admin/go and
  white_label_admin) and `GET /features/:name/check` (super_admin) read the
  **live Redis state**, not just the DB row, so operators can confirm what
  downstream services actually observe.

## How a downstream service implements the checker

Any downstream service that already has a Redis client implements the same
pattern (see `go/wallet_api/feature_flags.go` for the reference implementation):

1. Read `GET tigerwallet:feature:<name>` (string).
2. Map the result:
   - `enabled` → allow.
   - `disabled` / `paused` / missing / unknown / error → deny (fail-closed).
3. Cache the result **in-memory for 5 seconds** (map + mutex) so hot paths do
   not hammer Redis on every request, while still converging to the live state
   within a few seconds of an admin toggle.
4. At the top of each gated handler, call the checker; if the feature is not
   `enabled`, return:

   ```
   HTTP 423 Locked
   {"error":"feature <name> is currently <state>"}
   ```

Reference API (wallet_api):

```go
IsFeatureEnabled(featureName string) bool   // true only if state == "enabled"
FeatureState(featureName string) string     // "enabled" | "disabled" | "paused"
enforceFeature(c *gin.Context, name string) bool // writes 423 + returns false when not enabled
```

## Gated features

| Feature flag name | wallet_api operations gated |
| --- | --- |
| `swap_trading` | `/swap/quote`, `/swap/execute`, `/amm/quote`, `/amm/swap` |
| `send_transactions` | `/send` (EVM broadcast) |
| `staking` | `/staking/quote`, `/staking/stake`, `/staking/unstake`, `/staking/claim` |
| `lending` | `/lending/*` (gated when lending routes are added) |
| `nft_transfer` | `/nft/transfer` |
| `account_abstraction` | `/aa/*` (gated when AA routes are added) |
| `bridge` | `/bridge/*` (gated when bridge routes are added) |
| `fiat_onramp` | `/ramp/*` onramp (gated when ramp routes are added) |
| `fiat_offramp` | `/ramp/*` offramp (gated when ramp routes are added) |

> wallet_api currently exposes user-facing routes for `swap_trading`,
> `send_transactions`, `staking`, and `nft_transfer`; these are enforced today.
> The remaining flag names are reserved so that when `lending`, `account_abstraction`,
> `bridge`, and `fiat_onramp`/`fiat_offramp` routes are added to wallet_api (or any
> other downstream service), the same checker gates them with no protocol change.

## App separation

- Admin backends and wallet_api are separate Go modules with **no shared code
  imports**.
- The **only** coupling is the Redis key namespace `tigerwallet:feature:<name>`
  and the canonical state strings (`enabled`/`disabled`/`paused`).
- Each backend/service uses its own existing Redis client; no new cross-service
  dependency is introduced.
- Enforcement is **governance** (halt/pause/start/resume of feature behavior),
  not fund movement. Gated endpoints return `423 Locked`; they never move funds
  while a feature is disabled/paused.
