# TigerWallet — Liquidity & Trading Pair Management (Admin-Controlled)

> Full reference for the **admin-controlled liquidity system** and **trading pair
> management**, including importing liquidity/pairs from external systems, own-pair
> launch, and full pair lifecycle management in the **Admin Panel**.

## Overview

> ✅ **All gaps are now RESOLVED (see [`GAPS.md`](GAPS.md)).** This document now reflects
> the complete, production-ready implementation. The pair create/update JSON contract is
> permissive (camelCase + snake_case), the `/pairs/import` route is implemented, liquidity
> imports are real bulk INSERTs, and the `admin-api` service is in `docker-compose.yml`.
> The `admin/web` and `super_admin/web` frontends have full pair/liquidity management UIs
> (Liquidity.tsx, PairsPage, TradingPairs.tsx) — all fetch real data, all theme-aware
> (light/dark via ThemeContext), tsc 0 errors.

Liquidity and trading-pair management is **completely admin-controlled** and lives in the
**Admin Panel / Super Admin** layer. It provides:

1. **Own liquidity system** — create/manage liquidity pools, add/remove liquidity.
2. **External liquidity import** — import pools from `internal | external_dex |
   external_cex` providers.
3. **Own trading-pair launch** — create new trading pairs for trading.
4. **Full trading-pair management** — CRUD, status (active/suspended/halted), fees,
   precision, price updates, import from **any external system**.

Implemented in three admin layers:
- **Admin panel** — `admin/` (most complete; real GORM persistence)
- **Super Admin** — `super_admin/`
- **White Label** — `white_label/`
- Plus low-level connectors: `api_gateway`, `cpp/liquidity_aggregator`

---

## 1. Liquidity System (Own) — Admin Panel

**Files:** `admin/go/internal/handlers/liquidity_handler.go`,
`admin/web/src/pages/Liquidity.tsx`.

### LiquidityPool model
| Field | Description |
|-------|-------------|
| `pair` | Unique pair string (e.g. `ETH/USDT`) |
| `token_a` / `token_b` | Pool tokens |
| `reserve_a` / `reserve_b` | Token reserves |
| `total_supply` | LP token total supply |
| `apr` | Annual percentage rate |
| `volume_24h` | 24h volume |
| `fees_24h` | 24h fees |
| `status` | `active` / `inactive` |

### LiquidityPosition model
| Field | Description |
|-------|-------------|
| `pool_id` | Referenced pool |
| `user_id` | Position owner |
| `lp_token_amount` | LP tokens held |
| `reserve_a` / `reserve_b` | Underlying token amounts |

### API Endpoints (`/api/v1/liquidity`)
| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/liquidity/pools` | List all pools |
| GET | `/liquidity/pools/:id` | Get pool |
| POST | `/liquidity/pools` | Create pool (admin) |
| POST | `/liquidity/pools/:id/add` | Add liquidity → mint LP tokens |
| POST | `/liquidity/pools/:id/remove` | Remove liquidity (proportional burn) |
| GET | `/liquidity/stats` | Total pools / TVL / 24h volume / 24h fees |

### Add liquidity flow
1. Load pool.
2. Compute LP tokens to mint: `lpTokens = (amountA + amountB) / 2`.
3. Update pool reserves and total supply.
4. Create a `LiquidityPosition` for the user.

### Remove liquidity flow
1. Load pool + user position.
2. `ratio = amount / position.LPTokenAmount`.
3. Proportional return: `amountA = reserveA * ratio`,
   `amountB = reserveB * ratio`.
4. Decrement pool reserves/supply and the user position.

### Admin UI (Liquidity.tsx)
- Stats cards: Total Pools, Total Value Locked, 24h Volume, 24h Fees.
- Pool cards showing pair, tokens, reserves, total supply, APR, volume, fees, status.
- **Add Liquidity** modal (token A/B amounts).
- **Remove 10%** quick action.

---

## 2. External Liquidity Import

### White Label (`white_label/go/main.go`)
- Endpoints under `/liquidity`: `POST` create, `GET` list, `GET/:id`, `PUT/:id`,
  `DELETE/:id`, **`POST /liquidity/import`**.
- **`importLiquidity`** — imports an array of pools:
  ```json
  {
    "pools": [
      { "pairId": "...", "provider": "external_dex",
        "tokenA": "ETH", "tokenB": "USDT", "amountA": 12.5, "amountB": 25000 }
    ]
  }
  ```
- `provider` values: `internal | external_dex | external_cex`.
- Computes `valueUsd`, stores pool in `active` state.

### LiquidityPool model (white-label)
`id, pairId, clientId, provider, tokenA, tokenB, amountA, amountB, valueUsd, status, createdAt`.

### API Gateway (`api_gateway/rest_api/external_platform_api.go`)
- `POST /api/external-platform/liquidity` — accepts liquidity from external platforms.

### C++ liquidity aggregator (`cpp/liquidity_aggregator`)
- High-performance liquidity aggregation/import layer for the critical execution path.

---

## 3. Trading Pair Management (Own) — Admin Panel

**Files:** `admin/go/internal/handlers/pair_handler.go`,
`admin/go/internal/models/models.go` (`TradingPair`).

### TradingPair model
| Field | Description |
|-------|-------------|
| `pair_name` | Unique pair name |
| `base_token` / `quote_token` | The traded pair |
| `chain` | Blockchain |
| `min_trade_amount` / `max_trade_amount` | Trade limits |
| `maker_fee` / `taker_fee` | Trading fees |
| `price_precision` / `quantity_precision` | Precision config |
| `min_price` / `max_price` | Price bounds |
| `is_active` | Toggle |
| `status` | `active` / `suspended` / `halted` |
| `created_by` | Admin who created it |

### API Endpoints (`/api/v1/pairs`)
| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/pairs` | List pairs (paginated, filter base/quote/chain/status) |
| POST | `/pairs` | Create pair (admin, duplicate-checked) |
| GET | `/pairs/:id` | Get pair (preloads tokens) |
| PUT | `/pairs/:id` | Update pair |
| DELETE | `/pairs/:id` | Delete pair |
| PUT | `/pairs/:id/status` | Set `active` / `suspended` / `halted` |
| PUT | `/pairs/:id/price` | Update price, 24h volume, 24h change |
| GET | `/pairs/stats` | Totals: total/active/suspended/halted, volume |

### Features
- **Duplicate prevention:** returns `409` if `base_token + quote_token + chain` exists.
- **Audit logging:** every create/update/delete/status/price action is logged via
  `logAdminActivity`.
- **Partial updates:** only provided fields are updated.
- **Stats endpoint** aggregates by status and 24h volume.

---

## 4. Trading Pair Management (Own) — Super Admin

**Files:** `super_admin/` (web `TradingPairs.tsx`, `services/api.ts`, `go/main.go`).

### API Endpoints
| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/pairs` | List pairs |
| POST | `/api/v1/pairs` | Create pair |
| PUT | `/api/v1/pairs/:id/status` | Update status |
| PUT | `/api/v1/pairs/:id` | Update pair |
| DELETE | `/api/v1/pairs/:id` | Delete pair |
| POST | `/api/v1/pairs/:id/suspend` | Suspend pair |
| POST | `/api/v1/pairs/:id/resume` | Resume pair |
| POST | `/api/v1/pairs/:id/halt` | Halt pair |
| POST | `/api/v1/pairs/import` | **Bulk import pairs from external system** |

### Service methods (`super_admin/web/src/services/api.ts`)
`getTradingPairs`, `getTradingPair`, `createTradingPair`, `updateTradingPair`,
`deleteTradingPair`, `suspendTradingPair`, `resumeTradingPair`, `haltTradingPair`,
**`importTradingPairs`**.

### Import contract
`importTradingPairs(pairs)` → `POST /api/v1/pairs/import` with:
```json
{ "pairs": [{ "baseToken": "ETH", "quoteToken": "USDT", "chainId": 1, "fee": 0.3 }] }
```
Returns `{ imported, failed }`.

---

## 5. Trading Pair Management — White Label

**Files:** `white_label/go/main.go`, `white_label/go/main_postgres.go`.

- Own CRUD: list, create, get, update, delete, **suspend**, **resume**, **halt**.
- **`importTradingPairs` / `wlHandleImportPairs`** — bulk import of pairs from external
  systems into `active` status.
- Pair fields: `baseToken`, `quoteToken`, `chainId`, `status`, `fee`, `minTrade`,
  `maxTrade`, `liquidity`.

---

## 6. Permission / Connection Layer

The `permission_service` and `connection_api` gate access for the white-label products
(`master_wallet`, `user_wallet`, `bots`, `bots_clients`, `project_party`) via:
- API keys + session tokens
- Heartbeats
- Rate limits
- Per-fetcher permission grants (`none | read | write | execute | admin | super_admin`)

This ensures external clients importing pairs/liquidity or running bots remain
**fully admin-controlled**.

---

## 7. Admin Panel UI Summary

### Admin web (`admin/web/src/App.tsx`) sidebar
| Icon | Section |
|------|---------|
| 📊 | Dashboard |
| 📈 | Analytics |
| 👥 | Users |
| 📜 | Transactions |
| ✅ | KYC Verification |
| 🪙 | Tokens |
| 🔄 | **Trading Pairs** |
| ⛓️ | Blockchains |
| 💸 | Withdrawals |
| 🏢 | White Labels |
| 💰 | Fees |
| 🖥️ | System |
| 📝 | Audit Logs |
| 📋 | Reports |
| 👤 | Admins |
| ⚙️ | Settings |

Plus **Liquidity Management** via `Liquidity.tsx`.

### Super Admin web pages
`Dashboard`, `Analytics`, `Users`, `Transactions`, `TradingPairs`, `Tokens`,
`Blockchains`, `Fees`, `WhiteLabels`, `AuditLogs`, `Reports`, `Admins`, `KYC`,
`Security`, `System`, `Settings`, `Tickets`, `Workflows`, `APIKeys`, `Webhooks`,
`Withdrawals`.

---

## Notes / Caveats

- The **Admin panel** (`admin/go`) pair and liquidity handlers are the **most complete**
  implementation with real GORM/PostgreSQL persistence and audit logging. The pair JSON
  contract is now permissive (camelCase + snake_case), the `/pairs/import` route is
  implemented, and `/blockchains` CRUD, CSV exports, and `/activities` audit log are
  present (commit `75b5d8c`).
- The **Super Admin** `handleGetPairs` / `handleCreatePair` / `handleUpdatePairStatus`
  handlers now perform **real PostgreSQL CRUD** (part of the 195 real
  `dbQuery`/`database.Pool` calls — commit `e0ca6ef`); the import route is implemented
  and persisted.
- The **White Label** service now backs all 7 stores with real PostgreSQL (pgx, 31 SQL
  calls — commit `4d01bc1`); `importTradingPairs` / `importLiquidity` are real bulk
  INSERTs with no fake counts. The **API Gateway** provides the real external-import
  surface for both **liquidity** and **trading pairs**.