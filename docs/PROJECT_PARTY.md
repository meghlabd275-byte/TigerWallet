# TigerWallet — ProjectParty (Tokens, Coins, Launches & MM Services)

> Full feature & functionality reference for the **ProjectParty** product.

## Overview

> ✅ **All gaps are now RESOLVED (see [`GAPS.md`](GAPS.md)).** This document now reflects
> the complete, production-ready implementation. The ProjectParty backend uses real
> PostgreSQL persistence (pgxpool) for all handlers — no stubs, no sample data, no
> in-memory maps.

**ProjectParty** is the token/coin listing, trading-launch, and market-making services
platform within the TigerWallet ecosystem. It lets users browse coins and tokens, submit
new tokens for listing, run IDO/launchpad fundraisers, and provides market-making,
pricing, analytics, compliance, and fee tooling. It is a **white-label product**
(`WhiteLevelProduct::ProjectParty` / `"project_party"`) and connects to the central
`connection_api`, `permission_service`, and `fetcher_gateway` services under admin control.

- **Backend:** `project_party/go` (Gin REST API, PostgreSQL via pgx)
- **Frontends:** `project_party/web` (React), `project_party/desktop` (Electron),
  `project_party/android` (Kotlin), `project_party/ios` (Swift), `project_party/extension`
  (Chrome browser extension)
- **Default port:** `9006` (env `PROJECT_PARTY_PORT`)

---

## 1. Token & Coin Management (listing coins/tokens)

### Browse & Search
- **All Coins** — market-style table with rank, name, symbol, logo, price, 24h change,
  market cap, and 24h volume; network filter (Bitcoin, Ethereum, BNB Chain).
- **Search Tokens** — by name, symbol, or contract address.
- **Featured / Trending / Market** — curated lists for discovery.
- **Favorites** — users add/remove tokens to track.

### Submit a Token
Projects (or an admin on their behalf) submit a token via `POST /tokens`:
`name, symbol, contract_address, network, decimals, total_supply, logo_url, website_url, description`.

### Token Lifecycle & Admin Operations
Admin/API full CRUD on `/api/v1/tokens`:

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/tokens` | Create a token |
| GET | `/tokens` | List tokens |
| GET | `/tokens/:id` | Get a single token |
| PUT | `/tokens/:id` | Update token |
| DELETE | `/tokens/:id` | Delete token |
| POST | `/tokens/:id/submit` | Submit for review |
| POST | `/tokens/:id/approve` | Approve listing |
| POST | `/tokens/:id/reject` | Reject with reason |

Token fields: `name, symbol, decimals, contract_address, chain, total_supply, logo_url,
description, website, whitepaper, social_links, is_featured, listing_fee_usd`.

### Token Status States
`draft → submitted → in_review → approved → rejected → listed`

Additional tracking: `submission_date`, `reviewer_id`, `reviewed_at`, `rejection_reason`,
`launchpad_id`.

---

## 2. Token Listings (trading launch)

`/api/v1/listings`

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/listings` | Create a listing |
| GET | `/listings` | List listings |
| GET | `/listings/:id` | Get a listing |
| PUT | `/listings/:id/status` | Update status |
| POST | `/listings/:id/featured` | Feature/boost a listing |

Listing fields:
- `pair_token` (e.g. `USDT`, `ETH`)
- `initial_price` / `current_price`
- `launch_type` → `fair_launch` | `presale` | `farming`
- `start_time` / `end_time`
- `status` → `upcoming | active | completed | cancelled`
- Metrics: `volume_24h`, `liquidity_usd`, `market_cap`, `price_change_24h`

---

## 3. Launchpad (IDO / Presale)

`/api/v1/launchpad`

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/launchpad/create` | Create a launchpad |
| GET | `/launchpad` | List launchpads |
| GET | `/launchpad/:id` | Get a launchpad |
| POST | `/launchpad/:id/contribute` | Contribute funds |
| POST | `/launchpad/:id/claim` | Claim allocated tokens |
| POST | `/launchpad/:id/cancel` | Cancel & refund |

Launchpad configuration:
- `soft_cap` / `hard_cap`
- `min_contribution` / `max_contribution`
- `start_time` / `end_time`
- `token_price`
- `accepted_payment` → `USDT` | `BNB` | `ETH`

Tracking: `total_raised`, `status` → `upcoming | active | completed | cancelled | refunded`,
per-contribution status → `pending | claimed | refunded`.

---

## 4. Market-Making Services (mm bot services)

`/api/v1/market-making` — powers MM bots for listed tokens.

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/market-making/orders` | Create maker buy/sell orders |
| GET | `/market-making/orders` | List maker orders |
| PUT | `/market-making/orders/:id/status` | Update order status |
| GET | `/market-making/status/:token_id` | MM status (spread, orders, volume) |
| POST | `/market-making/liquidity/add` | Add liquidity |
| POST | `/market-making/liquidity/remove` | Remove liquidity |

MM status metrics per token: `active`, `spread`, `total_orders`, `filled_orders`,
`volume_24h`.

This surface is the ProjectParty gateway into the broader
**MM Bot / BotsClients** platform (see `BOTS_CLIENTS.md`).

---

## 5. Pricing / Oracle

`/api/v1/pricing`

- `POST /pricing/set` — set a token price (admin).
- `GET /pricing/:token_id` — current price, `change_24h`, `volume_24h`, `market_cap`.
- `GET /pricing/history/:token_id` — price history.
- `POST /pricing/update` — update price.

---

## 6. Analytics

`/api/v1/analytics`

- `GET /analytics/volume` — 24h/7d/30d volume, breakdown by token.
- `GET /analytics/liquidity` — total liquidity, breakdown by pair.
- `GET /analytics/holders` — holder count (`total`, `new_24h`).
- `GET /analytics/transactions` — transaction count (24h/7d).

---

## 7. Compliance & KYC

`/api/v1/compliance`

- `POST /compliance/audit` — request audits (`security` | `code` | `financial`).
- `GET /compliance/audit/:token_id` — audit status + report URL.
- `POST /compliance/kyc/submit` — submit KYC.
- `GET /compliance/kyc/:token_id` — KYC status, verified/expiry timestamps.

Audit status: `requested | in_progress | completed | failed`.

---

## 8. Fees

`/api/v1/fees`

- `GET /fees` — fee schedule:
  - basic listing, featured listing
  - audit required
  - KYC verification
  - launchpad basic, launchpad premium
- `POST /fees/calculate` — compute total from listing type + features (`featured`,
  `audit`, `kyc`).
- `POST /fees/pay` — pay listing fees.

---

## 9. ProjectParty as a White-Label Product

ProjectParty is one of the **five** `WhiteLevelProduct` types managed by the
Super Admin system:
`MasterWallet`, `UserWallet`, `Bots`, `BotsClients`, `ProjectParty`.

- Registered in `connection_api` (`ProductProjectParty = "project_party"`).
- **Fetchers granted** (by `permission_service.getProductFetchers`):
  `TokenInfo`, `MarketData`, `Blockchain`, `KYC`.
- Access is API-key gated, with session tokens, heartbeats, rate limits, and per-fetcher
  permission levels (`none | read | write | execute | admin | super_admin`).
- Remote admin controls available: `Disable`, `Enable`, `UpdateConfig`, `Restart`,
  `Shutdown`, `ClearCache`, `ForceSync`, `UpdateFetcher`.

---

## 10. Supported Chains / Frontends

- **Chains:** Bitcoin, Ethereum (EVM), BNB Chain, Polygon (per submission network lists).
- **Web (React):** Dashboard, Coins, Tokens, Favorites, Submit, Settings, Login.
- **Desktop (Electron):** Dashboard, Wallets, Transactions, Settings, Login.
- **Android / iOS:** multi-tab wallet-style apps.
- **Extension (Chrome):** popup with token submission and coin browsing.

---

## Notes / Caveats

- All ProjectParty Go handlers now use **real PostgreSQL persistence** (pgxpool, 64 real
  SQL calls — commit `e0ca6ef`). Tokens, listings, launchpads, contributions, maker
  orders, pricing, analytics, KYC, and fees are all persisted to the database — no static
  or sample data is returned.
- Real trading-launch execution, liquidity placement, and MM bot execution are handled by
  the lower-level engines (see `BOTS_CLIENTS.md` and `LIQUIDITY_TRADING_PAIRS.md`), now
  wired through the PG-backed API surfaces.