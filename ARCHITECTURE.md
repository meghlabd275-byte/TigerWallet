# TigerWallet — System Architecture

> Canonical architecture map for the TigerWallet Web3/crypto platform.
> Statuses use the Phase-56 evidence-based vocabulary
> (`VERIFIED COMPLETE` / `PARTIAL` / `BROKEN` / `MISSING` / `FAKE`).
> Last verified: 2026-08-25 against the working tree.

This document is the **authoritative** top-level reference and supersedes the
marketing-oriented numbers in `README.md` (which overstate chain counts — see
"Blockchain registry" below).

---

## 1. Security domains (Phase 9)

The repository implements **four separated product domains plus a TigerWallet
control plane**:

```
┌────────────────────────────────────────────────────────────────────────┐
│ TIGERWALLET CONTROL PLANE (operated by TigerWallet SuperAdmin)          │
│  super_admin/go        :8082   license_service/go        :8460         │
│  admin/go              :9093   permission_service        :8085         │
│  kill_switch           :8469   permission_bridge (edge)  :9007         │
│  connection_api        :8092   fetcher_gateway (Rust)    :8093         │
└────────────────────────────────────────────────────────────────────────┘
             ▲ license heartbeat (fail-closed, wl_shared/wlgate)
┌────────────┴───────────────────────────────────────────────────────────┐
│ PRODUCT DOMAINS (structurally separated)                                │
│  UserWallet:   go/wallet_api :8443        ◄── user_wallet clients      │
│  MasterWallet: master_wallet/backend :8450◄── master_wallet clients    │
│  Admin:        admin/go :9093             ◄── admin clients            │
│  SuperAdmin:   super_admin/go :8082       ◄── super_admin clients      │
│  ProjectParty: project_party/go :8106     ◄── project_party clients    │
└────────────────────────────────────────────────────────────────────────┘
┌────────────────────────────────────────────────────────────────────────┐
│ WHITE-LABEL (self-hosted; depends ONLY on the control plane)            │
│  wl_master_wallet  wl_user_wallet  wl_bots  wl_project_party           │
│  wl_card  wl_liquidity  white_label_admin                              │
└────────────────────────────────────────────────────────────────────────┘
```

### Separation rules (Phases 10–12) — VERIFIED (structural)

| Rule | Enforcement | Status |
|---|---|---|
| UserWallet clients never reach MasterWallet/Admin internals | Separate backends/ports; `user_wallet` has no import path into `master_wallet/backend` or `admin/go` | VERIFIED COMPLETE |
| MasterWallet never reaches UserWallet internals | `master_wallet/backend` is self-contained; owns its own chain registry | VERIFIED COMPLETE |
| Admin never reaches wallet internals | `admin/go` talks to its own DB/services; no import of `go/wallet_api` internals | VERIFIED COMPLETE |
| Server-side enforcement (not frontend hiding) | `permission_bridge` fail-closed (X-API-Key → enabled product; `SUPER_ADMIN_SECRET` bearer for super-admin routes); `requireScope`/`AuthMiddleware` in admin + WL-admin backends | VERIFIED COMPLETE |

---

## 2. Request path map (Phase 3)

```
Application (android / ios / desktop / extension / web / flutter)
    ↓ HTTPS/REST (thin clients; no business logic in clients)
Owning domain backend API surface
    ↓
Business logic (Go service in go/<service> or domain backend)
    ↓
Fetcher / connector (go/full_fetchers, go/blockchain_rpc,
                     go/rpc_node_manager, cex_connectors, dex_connectors)
    ↓
Blockchain node / external provider (RPC, Stripe/MoonPay/Transak, CEX APIs)
    ↓
PostgreSQL + Redis (per docker-compose)
```

Verified request paths: extension signing
(`user_wallet/extension` → `go/wallet_api` `/api/v1/sign` + `/api/v1/send`),
extension read-only RPC (public `GET /api/v1/chains` → direct node), fiat-ramp
HMAC webhooks (`go/fiat_ramp` :8451), WL license enforcement
(`wl_shared/wlgate` heartbeat → `license_service`), MasterWallet revenue ops
(two-party co-sign at `license_gate.go`), copy-trading/staking web proxies
(`frontend/web_nextjs` → `/api/v1/copytrading/*`, `/api/v1/staking`).

---

## 3. Canonical backend fleet (VERIFIED COMPLETE — entry/build present)

| Service | Location | Default port | Notes |
|---|---|---|---|
| MasterWallet backend | `master_wallet/backend` | 8450 | auto-signer, chain registry, treasury, multisig, license gate |
| UserWallet API | `go/wallet_api` | 8443 | real on-chain RPC, BIP-39/32/44, secp256k1 sign + broadcast, AES-256-GCM seed persistence |
| Admin API | `admin/go` | 9093 | 46 handler files, ~352 routes; GORM/PG + Redis |
| SuperAdmin API | `super_admin/go` | 8082 | licenses, profit-share, feature flags, kill switch, co-sign |
| ProjectParty API | `project_party/go` | 8106 | PG-backed; on-chain launchpad via go-ethereum |
| License service (control plane) | `license_service/go` | 8460 | Ed25519 license signing + heartbeat |
| Kill switch | `kill_switch` | 8469 | global/client/product/fetcher halt; SuperAdmin JWT only |
| Permission service | `permission_service` | 8085 | authoritative WL license/permission control plane (pgx) |
| Permission bridge | `permission_bridge` | 9007 | tenant-facing edge; fail-closed X-API-Key auth |
| Connection API | `connection_api` | 8092 | WL product connection ledger |
| Fetcher gateway | `fetcher_gateway` (Rust) | 8093 | single entry for WL data fetchers |
| Fiat ramp | `go/fiat_ramp` | 8451 | HMAC-verified Stripe/MoonPay/Transak webhooks |
| Bot API (MM platform) | `mm_bot_platform/bot_api` | 8471 | PG-backed bot platform API |

Additional services in `go/` (bridge :8007, lending :8009, copy-trading :8006,
governance :8454, prediction :8455, p2p :8475, card :8457, dapp-browser :8083,
airdrop :8465, earn :8466, coupon :8467, red-packets :8468, bots :8108, and
~70 more) — see `REPOSITORY_INVENTORY.md` for the full fleet.

Ports are the **container-internal defaults**. Host-side mappings live in
`docker-compose.yml` (see §5).

---

## 4. White-label (WL) deployables

| Product | Location | Inner port | License-gated |
|---|---|---|---|
| WL MasterWallet | `wl_master_wallet/go` | 8450 | Yes (wlgate heartbeat) |
| WL UserWallet | `wl_user_wallet/go` | 8443 | Yes |
| WL Bots | `wl_bots/go` | 8471 | Yes |
| WL ProjectParty | `wl_project_party/go` | 8106 | Yes |
| WL Card | `wl_card/go` | 8463 | Yes |
| WL Liquidity | `wl_liquidity/go` | 8462 | Yes |
| WL Admin | `white_label_admin/go` | 8082 | Yes |
| Self-hosted MasterWallet (reference) | `selfhosted_masterwallet` (Rust) | — | **NO** — reference impl only; do not ship to WL clients |

Each WL product runs **independently** in the client's cloud and depends on the
TigerWallet control plane only for the periodic license heartbeat (fail-closed
via `wl_shared/go/wlgate`). It does **not** route client traffic through
TigerWallet at runtime.

---

## 5. Docker-compose topology & WL host-port collision

`docker-compose.yml` defines ~42 services. The WL block previously reused the
host ports `8461`, `8462`, `8463` for two services each, which caused `docker
compose up` to fail with "port is already allocated". **Resolved 2026-08-25** by
changing only the host side:

| Service | Fixed host mapping | Container port |
|---|---|---|
| wl-user-wallet | 8461 | 8443 |
| wl-master-wallet | 8462 | 8450 |
| wl-bots | 8463 | 8471 |
| wl-project-party | 8464 | 8106 |
| **wl-admin** | **8456** | 8082 |
| **wl-liquidity** | **8458** | 8462 |
| **wl-card** | **8459** | 8463 |

Only the **host side** changed; container-internal listen ports (and therefore
the per-product README defaults) are unchanged. This fix is applied in
`docker-compose.yml`; `docker compose config --quiet` passes.

---

## 6. Blockchain registry (VERIFIED counts — supersedes README)

- **120 EVM mainnets** — `go/wallet_api/chains_evm_data.go`
  (`evmMainnetCount = 120`).
- **66 non-EVM chains** — `go/wallet_api/chains_nonevm_data.go`
  (`nonEVMMainnetCount = 66`).

Served to clients via public `GET /api/v1/chains`. RPC endpoints resolve from
env vars at runtime and **fail closed** when unset (no fabricated endpoints).

---

## 7. Client application matrix

| Domain | Android | iOS | Desktop | Extension | Web | Flutter |
|---|---|---|---|---|---|---|
| user_wallet | ✅ | ✅ | repo-root `desktop_app/` (Tauri) | ✅ MV3 + EIP-1193 | ✅ (19 pages) | — |
| master_wallet | ✅ | ✅ | ⚠️ C++ health-probe | ✅ single-source + per-browser manifests | ✅ (13 pages) | ✅ |
| admin | ✅ | ✅ | ✅ | ✅ per-browser | ✅ (37 pages) | ✅ |
| super_admin | ✅ | ✅ | ✅ | ✅ per-browser (compat shim) | ✅ (41 pages) | — |
| white_label_admin | ✅ | ✅ | ✅ | ✅ per-browser | ✅ (31 pages) | — |
| project_party | ✅ | ✅ | ✅ Electron | ✅ | ✅ (11 pages) | — |
| bots | ✅ | ✅ | ✅ Electron | ✅ | ✅ (9 pages) | — |

See the domain READMEs (`user_wallet/README.md`, `master_wallet/README.md`,
`admin/README.md`, `super_admin/README.md`, `white_label_admin/README.md`) for
per-domain details. See `SECURITY.md` for the signing/co-sign security model and
`ENVIRONMENT.md` for the full environment-variable reference.