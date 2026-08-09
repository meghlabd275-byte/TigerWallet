# TigerWallet — BotsClients & Bots Platform

> Full feature & functionality reference for **Bots** and **BotsClients** (incl. MM bots).

## Overview

**BotsClients** is the white-label relationship/product that wraps the **Bot Platform**
(`mm_bot_platform`) so that external and white-label operators can run trading bots —
including **market-making (MM)** bots — under a fully **admin-controlled, role-gated**
access model. It is distinct from the plain "Bots" product.

- **Platform docs:** `BOT_PLATFORM.md`
- **Admin (Solidity):** `mm_bot_platform/bot_admin/TigerBotPlatform.sol`
- **Strategies (Solidity):** `mm_bot_platform/strategies/TigerBotStrategies.sol`
- **API server (Go):** `mm_bot_platform/bot_api/bot_api_server.go`
- **Core engine (Rust):** `mm_bot_platform/bot_core` (`bot_types.rs`, `strategies/mod.rs`)
- **Multi-platform clients:** `bots/` (web, desktop, android, ios, extension)

---

## 1. Products & Access Model

`WhiteLevelProduct` values (defined in `white_level_sdk/rust/src/types.rs` /
`connection_api` / `permission_service`):

`MasterWallet`, `UserWallet`, `Bots`, `BotsClients`, `ProjectParty`

**BotsClients specifics:**
- Registered as `ProductBotsClients = "bots_clients"`.
- **Granted fetchers:** `Prices`, `MarketData`, `Blockchain` (permission service).
- Access: API key → connect → session token → heartbeat; rate-limited.
- Per-fetcher permission levels: `none | read | write | execute | admin | super_admin`.
- Remote admin commands (kill-switch style): `Disable`, `Enable`, `UpdateConfig`,
  `Restart`, `Shutdown`, `ClearCache`, `ForceSync`, `UpdateFetcher`.

---

## 2. Bot Types (18 total)

From `mm_bot_platform/bot_core/src/bot_types.rs`:

| Category | Bot Types |
|----------|-----------|
| Classic MM | `MarketMaker`, `Liquidity` (Liquidity Provider) |
| MEV / Advanced | `Sniper`, `FrontRun`, `MevBot`, `Sandwich`, `FlashLoan`, `CrossChain`, `PerpHedge` |
| Standard Retail | `GridTrading`, `DcaBot`, `MomentumBot`, `MeanReversion`, `ScalpingBot` |
| AI / Signals | `AiTradingBot`, `SignalBot`, `CustomBot` |

### Bot descriptions (as documented)
- **Market Maker** — provide liquidity and earn spread.
- **Arbitrage** — profit from price differences across exchanges.
- **Sniper** — execute trades with minimal latency.
- **Liquidity** — deepen order books and earn fees.
- **Front Run / MEV / Sandwich** — extract MEV from the mempool.
- **Flash Loan** — use flash loans for risk-managed trades.
- **Cross Chain** — bridge assets for cross-chain arbitrage.
- **Perp Hedge** — hedge positions with perpetuals.
- **Grid Trading** — buy/sell at grid levels for steady profit.
- **DCA** — dollar-cost averaging at regular intervals.
- **Momentum** — follow market trends.
- **Mean Reversion** — trade price returning to the mean.
- **Scalping** — quick small profits.
- **AI Trading** — ML-based decisions.
- **Signal** — execute trades from custom signals.
- **Custom** — user-defined strategy.

`is_enabled_by_default`: `MarketMaker`, `Arbitrage`, `Sniper`, `GridTrading`, `DcaBot`.

---

## 3. Strategy Engines (Rust)

`mm_bot_platform/bot_core/src/strategies/mod.rs` — every strategy exposes
`generate_signal(price) -> Option<TradingSignal>`:

- **GridTradingStrategy** — grid_count, grid_spacing_pct, order_size_usd, base_price;
  generates buy/sell signal grids, `update_levels` to re-center.
- **DcaStrategy** — buy_interval_hours, buy_amount_usd, max_positions; `should_buy`,
  `execute_buy`, average-cost tracking.
- **MomentumStrategy** — lookback_period, entry/exit thresholds; RSI/MACD/MA-style
  momentum computation.
- **MeanReversionStrategy** — lookback_period, std-dev threshold; z/reversion signal.
- **ScalpingStrategy** — profit_target_pct, stop_loss_pct, max_spread_pct; order-book
  spread monitoring.
- **AiTradingStrategy** — model_path, prediction_threshold; ML inference.
- **SignalStrategy** — signal_source/endpoint; follows external signals.
- **CustomStrategy** — strategy_code + params; user-defined logic.
- Factory `create_strategy(BotType)` returns the boxed `TradingStrategy`.

---

## 4. Solidity — TigerBotStrategies (on-chain strategies)

`mm_bot_platform/strategies/TigerBotStrategies.sol`

Bot creation per type:
- MarketMaker, Arbitrage, Sniper, Liquidity, MEV, FlashLoan, CrossChain, PerpHedge
- **Grid**, **DCA**, **Rebalance**, **StopLoss**, **TrailingStop**

On-chain features:
- Role assignment (`assignRole`, `removeRole`, `getUserRole`, `hasRole`)
- Per-bot-type default risk levels (`_getDefaultRiskLevel`)
- Subscriptions: `subscribeToBot` (payable), `extendSubscription`
- Lifecycle: `startBot(investment)`, `stopBot`, `pauseBot(reason)`
- Trade execution: `executeTrade`
- Exchange management: `addExchange`, `updateExchange`
- Admin: `enable/disableEmergencyMode`, `pauseAllBots`/`resumeAllBots`,
  `updatePlatformFee`, `updateBotTypeFee`, `withdrawFees`
- Views: bot, instance, strategy, performance, subscriptions, exchange listing, fees

---

## 5. Solidity — TigerBotPlatform (bot admin, role-gated)

`mm_bot_platform/bot_admin/TigerBotPlatform.sol`

Core structs: `Bot`, `BotStats`, `Exchange`, `BotRequest`, `Trade`.

**Roles & governance:**
- `grantRole`, `revokeRole`, `hasRole`, `getUserRole`
- Governance transfer: `transferGovernance`, `acceptGovernance`

**Bot lifecycle (admin/owner):**
- `createBot`, `startBot`, `stopBot`, `pauseBot`, `deleteBot`
- `executeTrade`, `executeBatchTrades`

**Exchange management:**
- `addExchange`, `removeExchange`, `connectToExchange`, `canTradeOnExchange`
- Exchange fields: router, gasLimit, min/max trade size, feeBps

**Platform / financial admin:**
- `updateProtocolFee` (bps)
- `withdrawFees`, `distributeProfit`
- `enableEmergencyMode` / `disableEmergencyMode`
- `pauseAllTrading` / `resumeAllTrading`
- `pauseNewBotsCreation` / `resumeNewBotsCreation`

**Statistics:**
- `getBot`, `getBotStats`, `getUserBots`, `getAllExchanges`, `getPlatformStats`,
  `canTradeOnExchange`

**Events:** `RoleAssigned`, `RoleRevoked`, `BotCreated`, `BotStarted`, `BotStopped`,
`BotPaused`, `TradeExecuted`, `FeeCollected`, `EmergencyMode`, `ExchangeAdded`,
`ExchangeRemoved`, `ProfitDistributed`.

---

## 6. Bot API Server (Go)

`mm_bot_platform/bot_api/bot_api_server.go` — JWT-authenticated REST API.

### Routes
| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/public/tiers` | Public bot tiers |
| POST | `/api/v1/auth/login` | Login |
| POST | `/api/v1/auth/logout` | Logout |
| GET | `/api/v1/bots` | List bots |
| POST | `/api/v1/bots` | Create bot |
| POST | `/api/v1/bots/{id}/start` | Start bot |
| POST | `/api/v1/bots/{id}/stop` | Stop bot |
| GET | `/api/v1/subscription` | Get subscription |
| POST | `/api/v1/subscription` | Create subscription |
| GET | `/api/v1/fees` | Get fee configs (admin) |
| PUT | `/api/v1/fees` | Update fee config (admin) |
| GET | `/api/v1/admin/fee-addresses` | List admin fee addresses |
| POST | `/api/v1/admin/fee-addresses` | Set admin fee address |
| GET | `/api/v1/cex` | List CEX connections |
| POST | `/api/v1/cex` | Add CEX connection |
| GET | `/api/v1/dex` | List DEX connections |
| POST | `/api/v1/dex` | Add DEX connection |
| GET | `/api/v1/keys` | List API keys |
| POST | `/api/v1/keys` | Create API key |
| GET | `/api/v1/admin/users` | Admin: list users |
| GET | `/api/v1/admin/stats` | Admin: platform stats |

### Subscription / tier pricing
- Tier gating by `user.MaxBots`.
- Tier fields: `bot_type`, `user_id`, `monthly_fee`, `per_dex_fee`, `per_cex_fee`,
  `num_dexs`, `num_cexs`.
- Total = `monthly_fee + (num_dexs * per_dex_fee) + (num_cexs * per_cex_fee)`.

---

## 7. Bot Tiers

| Tier | Monthly Fee | Max Bots | Max DEX | Max CEX | Latency |
|------|------------|----------|---------|---------|---------|
| Free | $0 | 1 | 1 | 0 | 5s |
| Basic | $99 | 3 | 5 | 3 | 2s |
| Pro | $299 | 10 | 15 | 10 | 500ms |
| Enterprise | $999 | 50 | 50 | 30 | 100ms |

---

## 8. Bot Types per `BOT_PLATFORM.md` (9 standard bots)

1. **Grid Trading Bot** — price range, grid levels, per-grid order size, take-profit.
2. **DCA Bot** — purchase amount, interval, max positions, stop loss.
3. **Arbitrage Bot** — min spread, max trade size, allowed exchanges, slippage.
4. **Momentum Bot** — RSI/MACD/MA indicators, entry/exit thresholds.
5. **Mean Reversion Bot** — moving-average period, deviation %, position size.
6. **Scalping Bot** — profit target %, max hold, max daily trades, stop loss.
7. **AI Trading Bot** — model type, risk level, max position, training data.
8. **Signal Bot** — signal source, copy ratio, max delay, stop loss.
9. **Custom Bot** — strategy code, parameters, risk limits, execution mode.

---

## 9. Exchange Integrations

**DEX (20+):** Uniswap V2/V3, SushiSwap, PancakeSwap, QuickSwap, Curve, Balancer,
Jupiter (Solana), Raydium (Solana), Orca (Solana), and more.

**CEX (200+):** Binance, Coinbase, Kraken, KuCoin, Bybit, OKX, Huobi, Gate, Bitget,
MEXC, and more.

Connectors: encrypted API keys/secrets, rate limits, status (`active | inactive | error`),
last sync tracking.

---

## 10. Roles & Permissions (Platform)

| Role | Capabilities |
|------|--------------|
| **Admin** | View/manage all bots, configure fees, approve clients, view analytics, manage white-label |
| **Operator** | View/manage team bots, view analytics — **cannot manage fees** |
| **Client** | View/manage own bots, own analytics — cannot access admin |

---

## 11. Analytics

**Per bot:** total P&L, volume, orders, win rate, avg latency, drawdown.
**Platform:** total bots, active bots, total volume, total P&L, by bot type, by tier.

---

## 12. Admin CRUD for BotsClients (Super Admin)

`super_admin/go/main.go` — admin endpoints for BotsClients:

| Endpoint | Purpose |
|----------|---------|
| `GET /bots-clients` | List BotsClients |
| `GET /bots-clients/:id` | Get BotsClient |
| `POST /bots-clients` | Create BotsClient |
| `PUT /bots-clients/:id` | Update BotsClient |
| `DELETE /bots-clients/:id` | Delete BotsClient |
| `PUT /bots-clients/:id/status` | Update BotsClient status |

**White-label variants:** `GET/POST/PUT/DELETE /wl-bots-clients[/:id]` and
`PUT /wl-bots-clients/:id/status`.

---

## 13. Security Measures

- API key required for all bot operations.
- Rate limiting per user/connection.
- Position limits (max investment).
- Auto-stop on repeated errors.
- Monitoring & alerting.
- Encrypted connections and stored credentials.

---

## Notes / Caveats

- The Super Admin `handle*BotsClient*` handlers are currently **stubs** returning
  hardcoded responses; the **connection/permission infrastructure** (connection_api,
  permission_service, white_level_sdk) is the real access-control implementation.
- The **Rust bot_core** is the real trait/strategy engine; the Solidity contracts provide
  the on-chain, role-gated admin/authorization layer and per-bot-type fees.