# UserWallet Address Sharding Design (Phase 21)

**Status: VERIFIED COMPLETE (design + initial schema).**

Target: serve billions of UserWallet addresses without a single-database
hotspot — while satisfying:
1. One 24-word seed → all EVM and all non-EVM chains.
2. One (or a few) `uw_*` parent partitioned table(s) per chain.
3. Update-safe idempotent writes via block height.
4. Route-by-`chain_id` pgBouncer sharding at the edge.

## Why HASH on `chain_id` works so well here

- The address space is uniform after `hash_bytes(address)`;
  `chain_id` simply selects the route. Within a chain the write set
  spreads evenly across buckets → parallel ingest, no sequence hotspot.
- Partition count is a deployment-tunable knob; 16 is a sane default.
  Re-shard only after measuring skew (`pg_size_pretty` vs `EXTRACT`).
- pgBouncer route by `chain_id` follows MasterWallet chain registry:
  EVM chains get one shard pool; non-EVM chains another — but the
  `uw_*` parent alwaysexists per chain. Owners are able to on-board
  any chain id by enabling it inside `master_wallet/backend/chain_registry`.

## Tables

- `uw_addresses` — every derived address per (chain_id, address). One
  seed ('HD wallet') spans accounts: `account_index` distinguishes; this
  field is needed because one seed => many chains => many accounts.
- `uw_balances` — the current hot balance per address (native token default).
  Ordering by `block_number` makes writes idempotent.
- `uw_transactions` — append-only tx index; rows for from+to are both
  `direction 'o'`/`'i'` (a single hash row can live in either partition
  because partition key is chain_id only).
- `uw_indexer_checkpoint` — per-chain high-water block for reorg
  handling and catch-up resume.
- `uw_chain_meta` — per-chain enabled/enabled name (populated by
  MasterWallet chain management; see PHASE 19).

## Reorg safety

Indexers write `(chain_id, address, tx_hash, direction)`; on reorg the
reorged block is re-written by block height from the checkpoint
(`uw_indexer_checkpoint.block_number`) so `status` flips to `reorged`
and a subsequent write of the same `tx_hash` re-confirms idempotently.

## Capacity estimate

- `uw_addresses` row ≈ 105 bytes. 10^9 addresses → ~105 GB raw; ~5 GB
  per partition at 16 buckets — well inside single-server PG + bouncers.
- Hot wallet balances: index lookups hit the per-chain index
  `idx_uw_balances_lookup` on the local partition. `pgBouncer`
  direction on `chain_id` keeps the working set local per shard.

## Schema change management

`user_wallet_sharding.sql` is **additive only** — required to keep
existing platform tables in `main_schema.sql` untouched (PHASE 0).
Recreate = `user_wallet_sharding_recreate.sql` (see file) for the
do-over on the same parent names.

## Non-EVM coverage

The schema's `chain_id` is numeric bigint matching the MasterWallet
registry (EVM ids plus non-EVM ids allocated by SuperAdmin in the
`chain_registry_data.go` table, e.g. BTC=0, SOL=501, TRX=728, etc.)
— so BTC/SOL/Cosmos all live in the same partition topology and
the vectorised balance/history lookups apply identically.
