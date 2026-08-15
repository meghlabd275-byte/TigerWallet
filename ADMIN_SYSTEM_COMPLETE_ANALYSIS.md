# TigerWallet Admin System — Complete Fetcher & Functionality Inventory

> Complete analysis of all admin control planes (Admin, SuperAdmin, adminPanel,
> WhiteLabel Admin, MasterAdmin Management): full fetchers and functionality, what is
> real vs stubbed, what is missing, what gaps remain, and what each role CAN and
> CANNOT perform.
>
> **Isolation:** Admin apps never access MasterWallet or UserWallet fetchers.
> MasterWallet apps never access Admin or UserWallet fetchers. UserWallet apps
> never access Admin or MasterWallet fetchers. Each family is completely separated.

---

## 1. Overview — Five Admin Control Planes

| Surface | Backend | Port | Clients |
|---------|---------|------|---------|
| **Admin** | `admin/go` (Gin) | 9093 | web, flutter, android, ios, desktop, extensions, rust, cpp |
| **SuperAdmin** | `super_admin/go` (Gin) | 8082 | web, android, desktop, extensions, rust, cpp |
| **adminPanel** | `go/admin_service` + `go/rbac_admin_service` + `api_gateway/rest_api/tiger_admin_api.go` | 8080/— | React (`frontend/admin_panel`) |
| **WhiteLabel Admin** | `white_label_admin/go` (Gin) | varies | web |
| **MasterAdmin Management** | `master_admin_management/go` (Gin) | 8082 | web, android, rust, cpp |

---

## 2. ADMIN (`admin/go`) — 223 Routes, Port 9093

### 2.1 Full Route Map by Area

#### Auth / Profile
```
POST /auth/login
POST /auth/refresh
POST /auth/logout
GET  /auth/profile
PUT  /auth/profile
POST /auth/change-password
```

#### 2FA / Security
```
POST /2fa/setup            POST /2fa/verify
POST /2fa/enable           POST /2fa/disable
GET  /2fa/status           POST /2fa/backup-codes
GET  /2fa/users            GET  /2fa/stats
```

#### Admin Management ⚠️ SUPERADMIN-GATED
```
GET  /admins               POST /admins
GET  /admins/:id           PUT  /admins/:id
DELETE /admins/:id         POST /admins/:id/suspend
POST /admins/:id/activate  GET  /admins/:id/activities
```

#### Dashboard & Analytics
```
GET  /dashboard
GET  /analytics/users | /transactions | /revenue
GET  /analytics/custom
GET  /system/metrics
```

#### Users / KYC / Transactions / Tokens / Withdrawals
```
Users:   GET/PUT/DELETE /users/:id     POST /users/:id/verify-kyc
KYC:     GET/POST      /kyc            POST /kyc/:id/{approve,reject}
         GET /kyc/stats
Txns:    GET           /transactions   POST /transactions/:id/flag
Tokens:  GET/POST/PUT/DELETE /tokens  POST /tokens/:id/{activate,deactivate,verify}
         PUT  /tokens/:id/price        GET  /tokens/stats
Withdrawals: GET/POST /withdrawals     POST /withdrawals/:id/{approve,reject,process}
             POST /withdrawals/bulk-approve  GET /withdrawals/stats
```

#### White Labels
```
GET/POST       /white-labels
GET/PUT/DELETE /white-labels/:id
POST           /white-labels/:id/{approve,suspend}
GET            /white-labels/stats
```

#### Trading Pairs / Fees / API Keys
```
Pairs:   GET/POST/PUT/DELETE /pairs  PUT /pairs/:id/{status,price}  GET /pairs/stats
Fees:    GET/POST/PUT/DELETE /fees   POST /fees/calculate            GET /fees/stats
APIKeys: GET/POST/PUT/DELETE /api-keys  POST /api-keys/:id/{revoke,reactivate,regenerate}
```

#### System Config ⚠️ SUPERADMIN-GATED
```
GET/PUT /system/config        GET/PUT /system/rate-limits
GET     /system/master-wallets | /master-wallets/:id | /master-wallets/:id/balance
```

#### Feature Flags (NOT gated — open to any authenticated)
```
GET/POST /feature-flags  PUT/DELETE /feature-flags/:id
```

#### Notifications / Tickets / Integrations / Brokers / Institutional
```
Notifications: GET/DELETE /notifications  PUT /notifications/:id/read
              POST /notifications/{send,broadcast,template}  GET /notifications/stats
Tickets:      GET/POST /tickets  GET/PUT /tickets/:id  POST /tickets/:id/{messages,close}
              GET /tickets/{stats,sla-violations}
Integrations: GET/POST/PUT/DELETE /integrations  POST /integrations/:id/{test}
              POST /integrations/{slack,pagerduty,datadog,webhook}  GET /integrations/stats
Brokers:      GET/POST/PUT/DELETE /brokers  POST /brokers/:id/{approve,suspend}
              PUT /brokers/:id/commission  GET /brokers/:id/clients  GET /brokers/stats
Institutional: GET/POST/PUT/DELETE /institutional  POST /institutional/:id/{approve,suspend}
               PUT /institutional/:id/{limits,account-manager}  GET /institutional/stats
```

#### Compliance / Knowledge Base / Multisig / NFTs
```
Compliance: POST /compliance/{aml-report,tax-report,gdpr,gdpr/export,gdpr/anonymize}
            GET  /compliance/{reports,stats}
KnowledgeBase: CRUD /knowledge-base/categories  CRUD /knowledge-base/articles
Multisig:    GET/POST/PUT/DELETE /multisig
NFTs:         GET/DELETE /nfts  POST /nfts/:id/flag  GET /nfts/stats
```

#### Master Wallet / Billing / Crypto Cards
```
MasterWallet: GET /master-wallet/{stats,balances,transactions}
Billing:      CRUD /billing/plans  GET/POST/DELETE /billing/subscription
              GET /billing/{invoices,payment-methods}
              POST/DELETE /billing/payment-methods/:id  PUT /billing/payment-methods/:id/default
CryptoCards:  GET/POST /crypto-cards  GET /crypto-cards/:id
              POST /crypto-cards/:id/{block,activate}  PUT /crypto-cards/:id/limit
```

#### Features / Liquidity / Margin Trading / P2P Merchants
```
Features:     CRUD /features  POST /features/:id/{toggle}  PUT /features/:id/rollout
              GET /features/:id/check
Liquidity:    GET/POST /liquidity/pools  GET/PUT/POST /liquidity/pools/:id/{add,remove}
              GET /liquidity/stats
MarginTrading: GET/POST /margin/positions  POST /margin/positions/:id/{close,liquidation}
               GET /margin/stats
P2PMerchants:  GET/POST /p2p-merchants  GET/PUT /p2p-merchants/:id
               POST /p2p-merchants/:id/{approve,reject}  GET /p2p-merchants/:id/transactions
```

#### Audit Logs (NOT gated)
```
GET /audit-logs
```

---

## 3. SUPERADMIN (`super_admin/go`) — 186 Routes, Port 8082

### 3.1 Full Route Map by Area

#### Auth / Profile
```
POST /api/v1/auth/{login,register,refresh}
POST /api/v1/{logout,change-password}
POST /api/v1/2fa/{enable,disable}     ← stub (returns hardcoded message)
```

#### Users / KYC / Transactions / Withdrawals
```
Users:        GET /api/v1/users  GET /api/v1/users/:id
              PUT /api/v1/users/:id/status
              POST /api/v1/users/:id/{ban,unban,suspend}
KYC:          GET /api/v1/kyc  POST /api/v1/kyc/:id/{approve,reject}
Txns:         GET /api/v1/transactions  GET /api/v1/transactions/:id
              POST /api/v1/transactions/:id/{flag,unflag}
Withdrawals:  GET /api/v1/withdrawals
              POST /api/v1/withdrawals/:id/{approve,reject,process}
```

#### Tokens / Pairs / Blockchains / Fees
```
Tokens:       GET/POST /api/v1/tokens  PUT/DELETE /api/v1/tokens/:id
Pairs:        GET/POST /api/v1/pairs  PUT /api/v1/pairs/:id/status
Blockchains:  GET/POST /api/v1/blockchains  PUT /api/v1/blockchains/:id
              PUT /api/v1/blockchains/:id/status
Fees:         GET/POST /api/v1/fees  PUT /api/v1/fees/:id
```

#### Webhooks / Notifications / Audit Logs / Sessions
```
Webhooks:     GET/POST/DELETE /api/v1/webhooks  POST /api/v1/webhooks/:id/test
Notifications: GET /api/v1/notifications  PUT /api/v1/notifications/:id/read
               POST /api/v1/notifications/{send,broadcast}
Audit:        GET /api/v1/audit-logs  POST /api/v1/audit-logs/export
Sessions:     GET /api/v1/sessions  DELETE /api/v1/sessions/{:id,}
```

#### Feature Flags / IP Whitelist / White Labels / Stats
```
FeatureFlags: GET/POST/PUT/DELETE /api/v1/feature-flags  ← ALL STUB (empty data)
IPWhitelist:  GET/POST/DELETE /api/v1/ip-whitelist
WhiteLabels:  GET/POST/PUT/DELETE /api/v1/white-labels
Stats:        GET /api/v1/stats
```

#### Support Tickets
```
GET/POST /api/v1/tickets  GET /api/v1/tickets/:id
PUT /api/v1/tickets/:id/{status,messages}
PUT /api/v1/tickets/:id/assign
```

#### Bots & Bot-Tiers ⭐
```
GET/POST/PUT/DELETE /api/v1/bots
GET /api/v1/bots/:id  PUT /api/v1/bots/:id/status
GET /api/v1/bots/:id/stats
GET/POST/PUT/DELETE /api/v1/bots/tiers
```

#### Bot-Clients ⭐
```
GET/POST/PUT/DELETE /api/v1/bots-clients
GET /api/v1/bots-clients/:id
PUT /api/v1/bots-clients/:id/status
```

#### Project Teams ⭐
```
GET/POST/PUT/DELETE /api/v1/project-teams
GET /api/v1/project-teams/:id
GET/POST/DELETE /api/v1/project-teams/:id/members
```

#### WL Clients ⭐
```
GET/POST/PUT/DELETE /api/v1/wl-clients
GET /api/v1/wl-clients/:id
PUT /api/v1/wl-clients/:id/status   ← start/stop (binary)
```

#### WL Master Wallets ⭐
```
GET/POST/PUT/DELETE /api/v1/wl-master-wallets
GET /api/v1/wl-master-wallets/:id
PUT /api/v1/wl-master-wallets/:id/status   ← start/stop (binary)
```

#### WL User Wallets ⭐
```
GET/POST/PUT/DELETE /api/v1/wl-user-wallets
GET /api/v1/wl-user-wallets/:id
PUT /api/v1/wl-user-wallets/:id/status   ← start/stop (binary)
```

#### WL Bots ⭐
```
GET/POST/PUT/DELETE /api/v1/wl-bots
GET /api/v1/wl-bots/:id
PUT /api/v1/wl-bots/:id/status   ← start/stop (binary)
```

#### WL Bot-Clients ⭐
```
GET/POST/PUT/DELETE /api/v1/wl-bots-clients
GET /api/v1/wl-bots-clients/:id
PUT /api/v1/wl-bots-clients/:id/status   ← start/stop (binary)
```

#### WL Project Teams ⭐
```
GET/POST/PUT/DELETE /api/v1/wl-project-teams
GET /api/v1/wl-project-teams/:id
```

#### Master Wallets (system-wide)
```
GET/POST/PUT/DELETE /api/v1/master-wallets
GET /api/v1/master-wallets/:id
GET /api/v1/master-wallets/:id/balance
POST /api/v1/master-wallets/:id/transfer
```

#### User Wallets
```
GET/POST/PUT/DELETE /api/v1/user-wallets
GET /api/v1/user-wallets/:id
GET /api/v1/user-wallets/:id/balance
```

#### Admins
```
GET/POST/PUT/DELETE /api/v1/admins
POST /api/v1/admins/:id/{suspend,activate}
```

#### Workflows / Approval Requests / Backups / Knowledge Base
```
Workflows:       GET/POST/PUT/DELETE /api/v1/workflows
ApprovalRequests: GET /api/v1/approval-requests
                  POST /api/v1/approval-requests/:id/{approve,reject}
Backups:         GET/POST/DELETE /api/v1/backups
                  POST /api/v1/backups/:id/restore
KnowledgeBase:   GET/POST/PUT/DELETE /api/v1/knowledge-base
```

#### Archival / Reports / SLA
```
Archival: CRUD /api/v1/archival/policies  POST /api/v1/archival/policies/:id/run
          GET /api/v1/archival/records
Reports:  GET/POST /api/v1/reports/configs  GET/POST /api/v1/reports
SLA:      CRUD /api/v1/sla/policies  GET /api/v1/sla/reports
          POST /api/v1/sla/reports/generate   ← STUB
```

---

## 4. SUPERADMIN — Full Capabilities Matrix

### 4.1 What SuperAdmin CAN Do

#### ✅ Complete WL Client Control (per requirement)
- **WL Clients**: Full CRUD + `PUT .../status` (start/stop). ✅ Implemented.
- **WL-MasterWallets**: Full CRUD + `PUT .../status` (start/stop). ✅ Implemented.
- **WL-UserWallets**: Full CRUD + `PUT .../status` (start/stop). ✅ Implemented.
- **WL-Bots**: Full CRUD + `PUT .../status` (start/stop). ✅ Implemented.
- **WL-BotClients**: Full CRUD + `PUT .../status` (start/stop). ✅ Implemented.
- **WL-ProjectTeams**: Full CRUD. ✅ Implemented.
- **WL product fetcher visibility**: Via `white_label_system` (separate service), WL admins can `GET /fetchers`, `GET /fetchers/:name/data`, `GET /fetchers/:name/stats`. ✅ Partially there.

#### ✅ Bots + Bot-Tiers + Bot-Clients
- Bot CRUD + status + stats + tiers CRUD + bot-clients CRUD + status. ✅ Full.

#### ✅ Project Teams
- CRUD + members add/remove. ✅ Full.

#### ✅ Master Wallets (system-wide)
- CRUD + balance + transfer. ✅ Full.

#### ✅ User Wallets
- CRUD + balance. ✅ Full.

#### ✅ Admin Management
- CRUD + suspend/activate. ✅ Full.
- `handleUpdateAdmin` exists — BUT no dedicated permission CRUD endpoint
  (no `PUT /admins/:id/permissions` in routes, despite `api.ts` referencing it).

#### ✅ Security (2FA, sessions, IP whitelist)
- 2FA enable/disable (stub), sessions revoke, IP whitelist CRUD. ✅ Partial (2FA stub).

### 4.2 What SuperAdmin CANNOT Do

#### ❌ No Granular WL Product Fetcher-Level Permissions
SuperAdmin can only toggle WL product entity status (binary start/stop). Cannot:
- Disable a specific fetcher within a WL product (e.g., disable only `getBalance`
  for a WL client while keeping `sendTransaction` enabled).
- Pause/resume individual fetchers (only binary on/off via `/status`).
- No `POST /wl-clients/:id/products/:pid/fetchers/:fid/enable` or similar.
- **Gap:** The `white_label_system` has `wlAdmin.GET /fetchers/:name/data` and
  `wlAdmin.GET /fetchers/:name/stats` but NO fetcher-level enable/disable.

#### ❌ No Per-WL-Client Custom Fee Config
- No route to set custom trading/withdrawal/deposit fees per WL client.
- No route to set custom RPC endpoints per WL client.

#### ❌ No WL Product Coin/Token Listing Approval Sub-Workflow
- No route for SuperAdmin to approve/reject individual coin/token listings within
  a WL product (listing is at the platform level via `listing_service`).

#### ❌ No Futures / Options / Perpetual Trading Admin Controls
- No routes for: futures positions/orders, options contracts, perpetual funding rates.
- `go/perpetual_service` exists (separate service, no admin group).
- `options_trading/go` exists (separate, no admin routes).
- SuperAdmin cannot pause/resume perpetual trading or manage futures markets.

#### ❌ No Copy Trading Admin Controls
- No routes for copy trading: leader management, follower allocation, copy settings.
- `copy_trading/go` and `copy_trading/frontend` exist (separate).

#### ❌ No Convert Trading Admin Controls
- No routes for the convert/swap aggregator admin: slippage config, routing prefs.
- `go/convert_service` exists but has no admin group.

#### ❌ No P2P User-Side Admin (only merchant admin exists in Admin)
- SuperAdmin has no P2P orders/disputes/escrow admin controls.
- `p2p_trading/go` exists with order/trade/dispute routes but no admin group.

#### ❌ No On-Ramp / Off-Ramp Admin Controls
- No routes for fiat on-ramp/off-ramp: order management, provider config, KYC queue.
- `fiat_onramp/go`, `fiat_gateway/go`, `fiat_ramp/go` exist (separate services).

#### ❌ No Dedicated Liquidity Source Management
- Admin has pool CRUD (add/remove liquidity) but no:
  - Liquidity source management (DEX connectors, LP partners, yield strategies).
  - No route to add/remove/configure liquidity sources.

#### ❌ No Listing Manager / Partner Manager Admin
- `go/listing_service` has admin routes for listings (CRUD + status + review).
- But no dedicated partner management (referral partners, API partners, etc.).
- No partner-specific commission/fee configuration.

#### ❌ No Customer Service / Ticket Escalation Admin (beyond basic tickets)
- SuperAdmin has basic ticket CRUD + assign. No:
  - Ticket queue management, auto-assignment rules, SLA overrides.
  - CS agent management (separate from admin management).
  - Knowledge base article management for CS (has KB routes but no CS-specific).

#### ❌ No Marketing & Promotion Admin
- No routes for: campaign management, airdrop distribution, promo codes, referral
  program config, loyalty tiers.

#### ❌ No Reward System / Loyalty Admin
- No routes for: staking rewards config, yield farming pools, airdrop triggers,
  loyalty points management.

#### ❌ Feature Flag Handlers Are Stubs
- All 4 feature flag CRUD handlers return empty arrays / mock responses.
- No real flag-toggle store wired to any service.

#### ❌ 2FA Enable/Disable Are Stubs
- `handleEnable2FA` and `handleDisable2FA` return hardcoded messages.
- No real TOTP generation/verification.

---

## 5. ADMIN (`admin/go`) — Full Capabilities Matrix

### 5.1 What Admin CAN Do

| Module | Fetcher/Function | Status |
|--------|-----------------|--------|
| **2FA** | Setup/verify/enable/disable/backup-codes/stats | ✅ Real |
| **Users** | List/get/update/delete/verify-KYC | ✅ Real |
| **KYC** | List/get/approve/reject/stats | ✅ Real |
| **Transactions** | List/get/flag | ✅ Real |
| **Tokens** | Full CRUD + activate/deactivate/verify/price | ✅ Real |
| **Withdrawals** | List/approve/reject/process/bulk-approve/stats | ✅ Real |
| **White Labels** | Full CRUD + approve/suspend/stats | ✅ Real |
| **Trading Pairs** | Full CRUD + status/price/stats | ✅ Real |
| **Fees** | Full CRUD + calculate/stats | ✅ Real |
| **API Keys** | Full CRUD + revoke/reactivate/regenerate/stats | ✅ Real |
| **Notifications** | CRUD + send/broadcast/template/stats | ✅ Real |
| **Tickets** | CRUD + messages/close/SLA-violations/stats | ✅ Real |
| **Integrations** | CRUD + Slack/PagerDuty/Datadog/webhook/test | ✅ Real |
| **Brokers** | CRUD + approve/suspend/commission/clients/stats | ✅ Real |
| **Institutional** | CRUD + approve/suspend/limits/account-manager/stats | ✅ Real |
| **Compliance** | AML/tax/GDPR reports + export/anonymize/stats | ✅ Real |
| **Knowledge Base** | Categories + articles CRUD + stats | ✅ Real |
| **Multisig** | CRUD wallets | ✅ Real |
| **NFTs** | List/get/delete/flag/stats | ✅ Real |
| **Crypto Cards** | Full CRUD + block/activate/limit | ✅ Real |
| **Margin Trading** | Positions CRUD + close/liquidation/stats | ✅ Real |
| **P2P Merchants** | CRUD + approve/reject/transactions | ✅ Real |
| **Liquidity** | Pools CRUD + add/remove liquidity + stats | ✅ Partial |
| **Master Wallet** | Stats/balances/transactions (read-only) | ✅ Real |
| **Billing** | Plans/subscription/invoices/payment-methods CRUD | ⚠️ Stub |
| **Features** | CRUD + toggle + rollout + check | ✅ Real |

### 5.2 What Admin CANNOT Do (Critical Gaps)

#### ❌ No Futures / Options / Perpetual Trading Controls
- No routes for: futures contracts, options strikes, perpetual funding, liquidations.
- `perpetual_trading/go`, `options_trading/go`, `perpetuals_engine/rust` exist
  as separate services with no admin group.

#### ❌ No Copy Trading Controls
- No routes for: copy trading leaderboard, follower management, copy settings.

#### ❌ No Convert/Swap Aggregator Controls
- No routes for: swap routing, slippage limits, DEX preference, quote management.

#### ❌ No On-Ramp / Off-Ramp Admin Controls
- No routes for: fiat purchase/sell orders, provider config, payment method
  management, KYC tier enforcement.

#### ❌ No P2P User-Side Admin Controls
- Admin only has P2P merchant management. No: P2P order management, dispute
  resolution, escrow controls, payment confirmation monitoring.
- `p2p_trading/go` has order/trade/dispute routes but no admin group.

#### ❌ No Bot Management
- Admin has NO routes for bots, bot-tiers, or bot-clients.
- Bot admin routes exist only in SuperAdmin.

#### ❌ No Listing Manager / Partner Manager
- Admin has token/pair management but no: listing request queue, partner onboarding,
  partner commission config, listing tier management.
- `go/listing_service` has its own admin routes (separate service).

#### ❌ No Liquidity Source Management
- Admin has pool management but no: liquidity source connectors (which DEXs/LPs
  are connected), yield strategy config, liquidity routing preferences.

#### ❌ No Customer Service Management Beyond Tickets
- Admin has ticket CRUD but no: CS agent performance metrics, auto-assignment rules,
  escalation policies, canned response management, SLA overrides.

#### ❌ No Marketing & Promotion Controls
- No routes for: campaigns, airdrops, promo codes, referral programs, loyalty tiers.

#### ❌ No Reward System / Staking Rewards Admin
- No routes for: staking pool configuration, reward rate management, airdrop triggers,
  yield farming settings, loyalty point allocation.

#### ❌ No Referral / Affiliate Admin
- No routes for: referral program management, affiliate partner onboarding,
  commission tracking, referral payout management.

#### ❌ No NFT Marketplace Admin
- No routes for: NFT collection management, listing approval, royalty config,
  marketplace fee settings, IPFS/CID management.
- `nft_marketplace/go` and `nft_marketplace/cpp` exist (separate).

#### ❌ No Security (RBAC broken — access control hole)
- Feature flags, audit logs, master-wallet, billing, compliance GDPR, liquidity,
  margin, P2P, crypto-cards — all open to any authenticated admin.
- `RoleMiddleware`/`AdminMiddleware`/`PermissionMiddleware` are dead code.

---

## 6. Trading Admin Controls — Complete Gap Analysis

| Trading Type | Admin Has Routes? | SuperAdmin Has Routes? | Notes |
|-------------|-------------------|----------------------|-------|
| **Spot / Basic** | ✅ Tokens + Pairs CRUD | ✅ Tokens + Pairs CRUD | Both have full CRUD |
| **Margin** | ✅ Positions + close + liquidation | ❌ None | Admin has it |
| **Futures** | ❌ None | ❌ None | `perpetual_trading/go` separate |
| **Options** | ❌ None | ❌ None | `options_trading/go` separate |
| **Perpetual** | ❌ None | ❌ None | `go/perpetual_service` separate |
| **Copy Trading** | ❌ None | ❌ None | `copy_trading/go` separate |
| **Convert / Swap** | ❌ None | ❌ None | `go/convert_service` separate |
| **P2P (Merchant)** | ✅ CRUD + approve/reject | ❌ None | Admin has merchant side |
| **P2P (User Orders)** | ❌ None | ❌ None | No admin for user-side orders |
| **On-Ramp (Fiat)** | ❌ None | ❌ None | `fiat_onramp/go` separate |
| **Off-Ramp (Fiat)** | ❌ None | ❌ None | `fiat_ramp/go` separate |

### Missing: Trading Admin Needs
Both Admin and SuperAdmin are missing the following trading admin fetcher/functionality:
1. **Futures**: Market creation, contract specs (tick size, leverage, expiry), funding
   rate config, liquidation engine, insurance fund monitoring.
2. **Options**: Strike price ladder, expiry calendar, implied volatility config,
   collateral requirements.
3. **Perpetual**: Funding rate config, max leverage per pair, insurance fund,
   auto-deleveraging thresholds.
4. **Copy Trading**: Leader approval/verification, follower limit per leader,
   copy amount min/max, stop-loss per copy, profit split config.
5. **Convert/Swap**: DEX routing priority, slippage tolerance config, gas optimization
   settings, gasless swap paymaster config.
6. **P2P User-Side**: Order lifecycle management, dispute resolution, escrow
   release, payment proof verification, cancellation window management.
7. **On-Ramp**: Provider routing (Ramp/MoonPay/Transak), KYC tier thresholds,
   payment method limits, country restrictions.
8. **Off-Ramp**: Settlement bank routing, withdrawal limit tiers, fraud detection
   thresholds.

---

## 7. Bot Admin Controls — Complete Gap Analysis

| Bot Area | Admin Has? | SuperAdmin Has? | Missing |
|----------|-----------|-----------------|---------|
| **Bots CRUD** | ❌ None | ✅ Full CRUD + tiers | |
| **Bot-Tiers** | ❌ None | ✅ CRUD | |
| **Bot-Clients** | ❌ None | ✅ CRUD + status | |
| **Bot Strategies** | ❌ None | ❌ None | Strategy library management |
| **Bot Performance Metrics** | ❌ None | ✅ GET only | Admin cannot configure |
| **Bot WL Subset** | ❌ None | ✅ CRUD + status | |
| **Bot WL Client Subset** | ❌ None | ✅ CRUD + status | |
| **Bot Permission Granularity** | ❌ None | ❌ None | Cannot restrict specific bot actions |
| **Bot Audit / Trade Log** | ❌ None | ❌ None | Full trade history per bot |

Both Admin and SuperAdmin need:
- Strategy library CRUD (grid, DCA, arbitrage, TWAP, scalping, etc.).
- Bot performance metrics admin (set benchmarks, disable underperforming).
- Bot-specific permission control (can/cannot trade, max position size, max daily loss).
- Bot trade log / audit trail admin access.

---

## 8. Listing / Token Management — Complete Gap Analysis

| Area | Admin Has? | SuperAdmin Has? | Missing |
|------|-----------|-----------------|---------|
| **Tokens CRUD** | ✅ Full | ✅ Full | |
| **Trading Pairs CRUD** | ✅ Full | ✅ Full | |
| **Listing Requests** | ❌ None | ❌ None | `go/listing_service` separate |
| **Listing Tiers** | ❌ None | ❌ None | Basic/Pro/Enterprise tier config |
| **Listing Partner Manager** | ❌ None | ❌ None | Partner onboarding + commission |
| **Listing Review Queue** | ❌ None | ❌ None | Approve/reject listing applications |
| **Token Metadata** | ❌ None | ❌ None | Logo, description, social links |
| **Chain/RPC Config** | ❌ None | ❌ None | Per-chain RPC, explorer, chain ID management |
| **WL Token Allowlist** | ❌ None | ❌ None | Per-WL-client token restrictions |

Missing from both Admin and SuperAdmin:
- Listing request queue with review/approve/reject workflow.
- Listing tier management (Basic/Pro/Enterprise with different fees/features).
- Partner/affiliate onboarding for token listing referrals.
- Per-WL-client token allowlist configuration.

---

## 9. P2P / On-Ramp / Off-Ramp — Complete Gap Analysis

| Area | Admin Has? | SuperAdmin Has? | Missing |
|------|-----------|-----------------|---------|
| **P2P Merchants** | ✅ CRUD + approve/reject | ❌ None | |
| **P2P Merchant Transactions** | ✅ GET | ❌ None | |
| **P2P User Orders** | ❌ None | ❌ None | Full order lifecycle admin |
| **P2P Disputes** | ❌ None | ❌ None | Dispute resolution queue |
| **P2P Escrow** | ❌ None | ❌ None | Escrow balance monitoring |
| **On-Ramp Orders** | ❌ None | ❌ None | Fiat purchase order management |
| **On-Ramp Provider Config** | ❌ None | ❌ None | Provider routing, limits |
| **Off-Ramp Orders** | ❌ None | ❌ None | Fiat sell order management |
| **Off-Ramp Settlement** | ❌ None | ❌ None | Bank routing, settlement config |
| **P2P Merchant+Client** | ❌ None | ❌ None | Combined P2P ecosystem admin |

Both Admin and SuperAdmin need a unified P2P/On-Ramp/Off-Ramp admin panel covering:
- P2P: order management, dispute resolution, payment proof verification, escrow
  monitoring, merchant+client lifecycle.
- On-Ramp: order queue, provider routing, KYC tier management, country restrictions.
- Off-Ramp: settlement bank config, fraud thresholds, payout scheduling.

---

## 10. Customer Service & Other Services — Complete Gap Analysis

| Service | Admin Has? | SuperAdmin Has? | Missing |
|---------|-----------|-----------------|---------|
| **Support Tickets** | ✅ CRUD + assign + SLA | ✅ CRUD + assign | Advanced CS features missing |
| **Knowledge Base** | ✅ Full | ✅ Full | |
| **Notifications** | ✅ Full | ✅ Full | |
| **KYC** | ✅ Full + stats | ✅ Approve/reject | Auto-approval rules missing |
| **Crypto Cards** | ✅ Full | ❌ None | |
| **Marketing / Campaigns** | ❌ None | ❌ None | Airdrops, promo, banners |
| **Referral / Affiliate** | ❌ None | ❌ None | Referral program admin |
| **Reward System** | ❌ None | ❌ None | Staking rewards, loyalty tiers |
| **Audit Logs** | ✅ Full | ✅ Full | |
| **Sessions** | ❌ None | ✅ Revoke | |
| **IP Whitelist** | ❌ None | ✅ CRUD | |
| **2FA (Admin)** | ✅ Real | ⚠️ Stub | |
| **Security (WL)** | ❌ None | ❌ None | Per-WL security config |
| **Reporting / BI** | ❌ None | ❌ None | Advanced analytics dashboard |

---

## 11. adminPanel (`frontend/admin_panel` + Go backends)

### 11.1 Frontend Pages

Dashboard, Users, Transactions, Analytics, Fees, Integrations, Compliance,
Notifications, Security, Support, Bots, Bridges, Chains, DEXs, MarketMaker,
Pools, Treasury.

### 11.2 Endpoint Coverage vs Backend

| Page | Frontend calls | Backend serves? |
|------|--------------|----------------|
| Dashboard | `admin/analytics/{dashboard,revenue,volume}` | ❌ No route |
| Users | `admin/users` | ✅ (rbac_admin_service) |
| Transactions | `admin/transactions`, `stats` | ✅ Partial |
| Analytics | `admin/analytics/dashboard` etc | ❌ No route |
| Fees | `admin/fees` | ✅ Partial |
| Integrations | `admin/integrations` | ✅ Partial |
| Compliance | `admin/compliance/{aml,tax,reports,stats}` | ❌ No route |
| Notifications | `admin/notifications/{broadcast,read-all,stats}` | ❌ No route |
| Security | `admin/security/{ip-rules,stats}`, `2fa/*` | ❌ No route |
| Support | `admin/support/{tickets,stats}` | ❌ No route |
| Bots | `admin/bots` | ✅ (rbac_admin_service) |
| **Bridges** | `admin/bridges` | ❌ No route |
| **Chains** | `admin/chains` | ✅ Partial |
| **DEXs** | `admin/dexs` | ❌ No route |
| **MarketMaker** | `admin/market-makers` | ❌ No route |
| **Pools** | `admin/pools` | ❌ No route |
| **Treasury** | `admin/treasury`, `stats` | ❌ No route |

### 11.3 Missing / Gaps

1. **No single backend** implements the full `/api/v1/admin/*` surface.
2. **Route prefix mismatch**: `admin_service` serves `/api/v1/*` (no `/admin`);
   frontend prefixes `/api/v1/admin/*` → 404.
3. **9 of 17 pages** (Bridges, DEXs, Pools, Treasury, MarketMaker, 2FA,
   Notifications, Compliance, Support) have **no matching Go route**.
4. **No RBAC** on `admin_service` — admin vs superadmin distinction not enforced.

---

## 12. White Label Admin (`white_label_admin/go`) — 8 Pages

### 12.1 Full Route Map

```
Users: GET/POST /admin/users  GET /admin/users/:id
       PUT /admin/users/:id/status
       POST /admin/users/:id/{ban,unban,suspend}
KYC:   GET/POST /admin/kyc  POST /admin/kyc/:id/{approve,reject}
Txns:  GET/POST /admin/transactions  GET /admin/transactions/:id
       POST /admin/transactions/:id/{flag,unflag}
Withdrawals: GET/POST /admin/withdrawals
             POST /admin/withdrawals/:id/{approve,reject,process}
Tokens: GET/POST /admin/tokens  PUT /admin/tokens/:id  DELETE /admin/tokens/:id
Pairs:  GET/POST /admin/pairs  PUT /admin/pairs/:id/status
Blockchains: GET/POST /admin/blockchains  PUT /admin/blockchains/:id
             PUT /admin/blockchains/:id/status
Fees:   GET/POST /admin/fees  PUT /admin/fees/:id
Webhooks: GET/POST /admin/webhooks  POST /admin/webhooks/:id/test
         DELETE /admin/webhooks/:id
Notifications: GET /admin/notifications  PUT /admin/notifications/:id/read
               POST /admin/notifications/{send,broadcast}
Tickets: GET/POST /admin/tickets  GET /admin/tickets/:id
         PUT /admin/tickets/:id/status  POST /admin/tickets/:id/{messages,assign}
AuditLogs: GET /admin/audit-logs  POST /admin/audit-logs/export
Sessions: GET /admin/sessions  DELETE /admin/sessions/{:id,}
WhiteLabels: GET/POST /admin/white-labels  PUT/DELETE /admin/white-labels/:id
Stats: GET /admin/stats
Auth: POST /admin/{logout,change-password,2fa/{enable,disable}}
Admins: GET/POST /admin/admins  GET/PUT/DELETE /admin/admins/:id
        POST /admin/admins/:id/{suspend,activate}
Workflows: GET/POST /admin/workflows  PUT/DELETE /admin/workflows/:id
Approvals: GET /admin/approval-requests
           POST /admin/approval-requests/:id/{approve,reject}
Backups: GET/POST /admin/backups  POST /admin/backups/:id/restore
         DELETE /admin/backups/:id
KnowledgeBase: GET/POST/PUT/DELETE /admin/knowledge-base
Archival: CRUD /admin/archival/policies  POST /admin/archival/policies/:id/run
          GET /admin/archival/records
Reports: GET/POST /admin/reports  POST /admin/reports/generate
         GET/POST /admin/reports/configs
SLA:    CRUD /admin/sla/policies  GET /admin/sla/reports
        POST /admin/sla/reports/generate
Integrations: GET/POST/PUT/DELETE /admin/integrations
              POST /admin/integrations/:id/test
```

### 12.2 Missing / Gaps

1. **Web pages**: Only Dashboard, Users, KYC, Transactions, Fees, Settings,
   Tokens, Withdrawals (8 pages). All other routes (Webhooks, Notifications,
   AuditLogs, Sessions, Tickets, WhiteLabels, Stats, 2FA, Admins, Workflows,
   Approvals, Backups, KnowledgeBase, Archival, Reports, SLA, Integrations)
   have **no frontend page**.
2. **No granular WL fetcher-level permissions** — cannot disable specific
   fetchers within the WL product.
3. **No per-fetcher fee config override** — cannot set custom fees per WL product.
4. **No P2P/On-Ramp/Off-Ramp/Bots/Trading/Liquidity/NFT/Reward pages** even
   though these may be needed for WL clients.

---

## 13. MasterAdmin Management (`master_admin_management/go`) — Port 8082

Routes mirror SuperAdmin with differences:
- ✅ Users/KYC/Transactions/Withdrawals/Tokens/Pairs/Blockchains/Fees/Webhooks/
  Notifications/Tickets/FeatureFlags/IPWhitelist/WhiteLabels/Stats/Auth/2FA/
  Admins/Workflows/ApprovalRequests/Backups/KnowledgeBase/Archival/Reports/SLA/
  Integrations.
- ❌ **No Bots, Bot-Tiers, Bot-Clients** (SuperAdmin has these).
- ❌ **No WL-specific routes** (WL-clients, WL-bots, WL-master-wallets, etc.).
- ❌ **No MasterWallets, UserWallets** (system-wide wallet management).
- ❌ **No WL ProjectTeams**.

**Android client** is the most complete (bottom nav, fragments, layouts for all
areas). Rust and C++ backends are scaffolding (stub handlers).

---

## 14. Complete CAN / CANNOT Matrix

### SuperAdmin (port 8082)

| Action | CAN ✅ | CANNOT ❌ | Gap |
|--------|-------|-----------|-----|
| Manage WL-clients (full CRUD + start/stop) | ✅ | | |
| Manage WL-MasterWallets (full CRUD + start/stop) | ✅ | | |
| Manage WL-UserWallets (full CRUD + start/stop) | ✅ | | |
| Manage WL-Bots + WL-BotClients (full CRUD + start/stop) | ✅ | | |
| Manage WL-ProjectTeams (full CRUD) | ✅ | | |
| Per-fetcher permission toggle per WL product | ❌ | ✅ Not implemented | No route for granular fetcher enable/disable |
| Per-fetcher pause/resume within WL product | ❌ | ✅ Not implemented | Only binary on/off |
| Per-WL-client custom fee config | ❌ | ✅ Not implemented | No route |
| Per-WL-client RPC endpoint config | ❌ | ✅ Not implemented | No route |
| Manage WL coin/token listing approval | ❌ | ✅ Not implemented | Listing is platform-level only |
| Futures admin (markets, funding, liquidation) | ❌ | ✅ No routes | `perpetual_trading/go` separate |
| Options admin (strikes, expiry, IV) | ❌ | ✅ No routes | `options_trading/go` separate |
| Perpetual admin (funding rate, insurance) | ❌ | ✅ No routes | `go/perpetual_service` separate |
| Copy trading admin (leaders, followers, copy) | ❌ | ✅ No routes | `copy_trading/go` separate |
| Convert admin (routing, slippage) | ❌ | ✅ No routes | `go/convert_service` separate |
| P2P user-side admin (orders, disputes) | ❌ | ✅ No routes | `p2p_trading/go` separate |
| On-ramp admin (provider routing, limits) | ❌ | ✅ No routes | `fiat_onramp/go` separate |
| Off-ramp admin (settlement, fraud) | ❌ | ✅ No routes | `fiat_ramp/go` separate |
| Bot management (full CRUD) | ✅ | | |
| Bot-Tiers + Bot-Clients (full CRUD) | ✅ | | |
| Liquidity source management (DEX/LP connectors) | ❌ | ✅ No routes | Only pool management |
| Listing manager (queue, tiers, partners) | ❌ | ✅ No routes | `listing_service` separate |
| Marketing/promotion admin (airdrop, campaigns) | ❌ | ✅ No routes | Not implemented |
| Reward/staking admin (reward rate, pools) | ❌ | ✅ No routes | Not implemented |
| Referral/affiliate admin | ❌ | ✅ No routes | Not implemented |
| Customer service escalation advanced (auto-assign) | ❌ | ✅ Basic tickets only | No advanced CS |
| NFT marketplace admin (collections, royalties) | ❌ | ✅ No routes | `nft_marketplace` separate |
| Admin management (CRUD + suspend/activate) | ✅ | | |
| Admin permission CRUD per admin | ❌ | ✅ No permission endpoints wired | `handleUpdateAdmin` exists but no routes |
| Admin role CRUD + assign roles | ❌ | ✅ Basic admin CRUD only | No role CRUD endpoints |
| Master wallets (system-wide CRUD + transfer) | ✅ | | |
| User wallets (system-wide CRUD + balance) | ✅ | | |
| Feature flag management | ⚠️ | ✅ Routes exist, handlers stubbed | Returns empty data |
| 2FA management | ⚠️ | ✅ Stubbed | Returns hardcoded messages |
| Security (sessions, IP whitelist) | ✅ | | |
| Backup/restore | ✅ | | |
| Archival policies | ✅ | | |
| Reports + SLA | ⚠️ | ✅ SLA report generation stubbed | |

### Admin (port 9093)

| Action | CAN | CANNOT | Gap |
|--------|------|--------|-----|
| Margin trading (positions CRUD + close + liquidation) | ✅ | | |
| Futures admin | ❌ | ✅ No routes | |
| Options admin | ❌ | ✅ No routes | |
| Perpetual admin | ❌ | ✅ No routes | |
| Copy trading admin | ❌ | ✅ No routes | |
| Convert/swap admin | ❌ | ✅ No routes | |
| P2P merchants (CRUD + approve/reject) | ✅ | | |
| P2P user orders (disputes, escrow) | ❌ | ✅ No routes | |
| On-ramp admin | ❌ | ✅ No routes | |
| Off-ramp admin | ❌ | ✅ No routes | |
| Bot management | ❌ | ✅ No routes | SuperAdmin has it |
| Liquidity pools (CRUD + add/remove) | ✅ | | |
| Liquidity sources (DEX connectors, LP) | ❌ | ✅ No routes | |
| Listing requests queue | ❌ | ✅ No routes | `listing_service` separate |
| Token + Pair CRUD | ✅ | | |
| KYC approve/reject | ✅ | | |
| User management | ✅ | | |
| Compliance AML/tax/GDPR | ✅ | | |
| Crypto cards (full CRUD) | ✅ | | |
| Multisig wallets | ✅ | | |
| NFTs (list/delete/flag) | ✅ | | |
| Notifications | ✅ | | |
| Tickets | ✅ | | |
| Integrations | ✅ | | |
| Brokers + Institutional | ✅ | | |
| Master wallet (read-only) | ✅ | | |
| Feature flags | ✅ | | ⚠️ Not RBAC-gated |
| Audit logs | ✅ | | ⚠️ Not RBAC-gated |
| Billing | ⚠️ | ✅ Stubbed | Hardcoded plans |
| Marketing / promotion | ❌ | ✅ No routes | |
| Referral / reward system | ❌ | ✅ No routes | |
| NFT marketplace | ❌ | ✅ No routes | |
| **SECURITY: RBAC broken** | ⚠️ | ✅ 200+ endpoints open to any authenticated admin | `RoleMiddleware` dead code |

---

## 15. Summary — Critical Gaps by Priority

### Critical (must fix)
1. **Admin RBAC broken** — `RoleMiddleware`/`AdminMiddleware`/`PermissionMiddleware`
   dead code; ~200 endpoints accessible to any authenticated non-superadmin.
2. **adminPanel: 9 of 17 pages** have no backend (Bridges/DEXs/Pools/Treasury/
   MarketMaker/2FA/Notifications/Compliance/Support).
3. **admin_service route prefix mismatch** — serves `/api/v1/*`, frontend calls
   `/api/v1/admin/*` → 404.

### High (trading + bot + listing missing)
4. **No Futures/Options/Perpetual/Copy/Convert admin controls** anywhere — these
   trading modules exist as separate services with no admin group.
5. **No P2P user-side / On-ramp / Off-ramp admin controls** — merchant side only.
6. **No bot management in Admin** — only in SuperAdmin.
7. **No granular WL fetcher-level permission toggles** in SuperAdmin.
8. **No listing manager / partner manager** admin.

### Medium
9. **SuperAdmin feature flag handlers stubbed** — CRUD endpoints return empty data.
10. **SuperAdmin 2FA handlers stubbed** — returns hardcoded messages.
11. **No marketing, promotion, referral, reward system admin** anywhere.
12. **No NFT marketplace admin** — separate service not integrated.
13. **Billing handler stubbed** in Admin.
14. **WhiteLabel Admin web: 8 pages for 50+ routes**.
15. **SuperAdmin: 14 of 19 web pages are 2-line stubs**; wrong port (`:9090` vs `:8082`).
16. **Rust & C++ backends scaffolding** in all admin apps.
17. **Base-URL chaos** across all admin clients.

### Recommended Priority Fixes
1. Wire `RoleMiddleware` into `admin/go/main.go`.
2. Build missing admin_panel backend routes for all 9 missing pages.
3. Fix route prefix mismatch in admin_service.
4. Add Futures/Options/Perpetual/CopyTrading/Convert admin routes to Admin or SuperAdmin.
5. Add P2P user-side + On-ramp + Off-ramp admin controls.
6. Add bot management to Admin (currently SuperAdmin-only).
7. Implement granular WL fetcher-level permission system in SuperAdmin.
8. Add listing manager + partner manager admin.
9. Fix SuperAdmin feature flag + 2FA stub handlers.
10. Add marketing, referral, reward system admin.
11. Complete SuperAdmin web stub pages; fix port mismatch.