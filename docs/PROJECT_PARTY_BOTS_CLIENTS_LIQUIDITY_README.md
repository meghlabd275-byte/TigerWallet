# TigerWallet — Admin Products & Management (Documentation Index)

This folder contains detailed documentation for the admin-controlled products and
management surfaces of the TigerWallet ecosystem.

## Documents

| File | Covers |
|------|--------|
| [`PROJECT_PARTY.md`](PROJECT_PARTY.md) | ProjectParty — token/coin listing, trading launch (launchpad), MM bot services, pricing, analytics, compliance/KYC, fees, white-label integration |
| [`BOTS_CLIENTS.md`](BOTS_CLIENTS.md) | Bots & BotsClients platform — 18 bot types, strategy engines, Solidity admin/strategies, bot API/tiers, exchange integrations, roles, admin CRUD |
| [`LIQUIDITY_TRADING_PAIRS.md`](LIQUIDITY_TRADING_PAIRS.md) | Admin-controlled liquidity & trading-pair management — own liquidity system, external liquidity import, own pair launch, full pair management, import from external systems |
| [`GAPS.md`](GAPS.md) | **Resolved gaps record** — every gap is now ✅ RESOLVED with evidence (commits + verified metrics); documents the stubs-to-PostgreSQL conversion, orphan deletions, port/JSON-contract fixes, and the completed build order |

## Related Platform Docs

- [`BOT_PLATFORM.md`](../BOT_PLATFORM.md) — the 9 standard trading bots, tiers, roles, endpoints.
- [`ADMIN_ARCHITECTURE.md`](../ADMIN_ARCHITECTURE.md) — unified admin architecture, tech stack, roles & permissions.

> 📌 **All backends now use real PostgreSQL persistence — no stubs, no sample data, no
> in-memory maps.** All frontends are buildable (tsc 0 errors) with working light/dark
> theme on every page. See [`GAPS.md`](GAPS.md) for the full record of resolved gaps;
> every item is marked ✅ RESOLVED with build verification evidence.

## Quick Orientation

- **ProjectParty** = coin/token listing + launch + MM services product.
- **BotsClients** = white-label wrapper around the bot/MM-bot platform under admin control.
- **Liquidity & Trading Pairs** = fully **admin-controlled** pools, pair lifecycle, and
  external import (both liquidity and pairs) from any external system.

> Note: All backends now use real PostgreSQL persistence — no stubs, no sample data, no
> in-memory maps. The `admin` panel, `mm_bot_platform` engine, the `project_party` and
> `super_admin` Go backends, and the white-label/API-gateway import surfaces are all
> fully implemented and PG-backed. All 6 frontends (web_nextjs, admin/web,
> super_admin/web, project_party/web, white_label/frontend, bots/web) build with 0 tsc
> errors and have working light/dark theme switching on every page.