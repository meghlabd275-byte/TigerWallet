# TigerWallet — Admin Products & Management (Documentation Index)

This folder contains detailed documentation for the admin-controlled products and
management surfaces of the TigerWallet ecosystem.

## Documents

| File | Covers |
|------|--------|
| [`PROJECT_PARTY.md`](PROJECT_PARTY.md) | ProjectParty — token/coin listing, trading launch (launchpad), MM bot services, pricing, analytics, compliance/KYC, fees, white-label integration |
| [`BOTS_CLIENTS.md`](BOTS_CLIENTS.md) | Bots & BotsClients platform — 18 bot types, strategy engines, Solidity admin/strategies, bot API/tiers, exchange integrations, roles, admin CRUD |
| [`LIQUIDITY_TRADING_PAIRS.md`](LIQUIDITY_TRADING_PAIRS.md) | Admin-controlled liquidity & trading-pair management — own liquidity system, external liquidity import, own pair launch, full pair management, import from external systems |

## Related Platform Docs

- [`BOT_PLATFORM.md`](../BOT_PLATFORM.md) — the 9 standard trading bots, tiers, roles, endpoints.
- [`ADMIN_ARCHITECTURE.md`](../ADMIN_ARCHITECTURE.md) — unified admin architecture, tech stack, roles & permissions.

## Quick Orientation

- **ProjectParty** = coin/token listing + launch + MM services product.
- **BotsClients** = white-label wrapper around the bot/MM-bot platform under admin control.
- **Liquidity & Trading Pairs** = fully **admin-controlled** pools, pair lifecycle, and
  external import (both liquidity and pairs) from any external system.

> Note: Several handlers in `project_party` and `super_admin` expose complete route/model
> surfaces but currently return static/sample data; the `admin` panel, `mm_bot_platform`
> engine, and the white-label/API-gateway import surfaces are the most complete
> implementations.