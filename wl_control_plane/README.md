# WL Control Plane (C++ / Rust)

The ultra-low-latency license + feature-flag + auto-approval control plane
that runs **inside every white-label product process**. It is split into a
wait-free C++ hot path (`cpp/include/`) and a Rust verification/authority
layer (`rust/src/`), wired together through a stable C ABI so any product
backend (Go/Rust/Node) can FFI in without recompiling.

## Why Two Layers

Every request and every outgoing transaction inside a WL product must be
gated, but a network round-trip per request is unacceptable. The design:

- **Rust does the crypto** — Ed25519 license-token verification
  (`rust/src/license.rs`) at boot and on each heartbeat tick.
- **C++ answers in microseconds** — the verified liveness flag + feature-flag
  snapshot + auto-sign rule cache are pushed into wait-free C++ atomics
  (`cpp/include/wl_gate.hpp`, `cpp/include/wl_auto_approver.hpp`) and read
  per-request/per-tx in O(1) with no network call.

## C++ Hot Path

### `cpp/include/wl_gate.hpp` — wait-free request gate

Runs on **every request** before handler dispatch; must answer in
single-digit microseconds:

- `WlGate::instance()` — one process-wide instance. `set_alive()` /
  `is_alive()` are lock-free atomics (`std::memory_order_release/acquire`);
  the reason string is under a `shared_mutex`.
- Flag snapshot: `(product, fetcher) -> enabled` map consulted by
  `fetcher_enabled()` under a read lock — contended only during the rare
  per-heartbeat refresh.
- Does **no crypto** itself; the Rust client pushes snapshots via
  `wl_gate_set_alive` / `wl_gate_set_flags`.
- **C ABI for FFI** (`extern "C"`): `wl_gate_is_alive()`, `wl_gate_reason()`,
  `wl_gate_set_alive()`, `wl_gate_fetcher_enabled(product, fetcher)`,
  `wl_gate_set_flags(json_array)`.

### `cpp/include/wl_auto_approver.hpp` — the AUTO-APPROVE hot path

Called **before sign/broadcast** on every outgoing transaction. Two approval
modes reconcile the platform's two invariants:

1. **AUTO mode** — user-initiated outgoing txs (swap/stake/send/sign) are
   approved **in-process, wait-free** when: license alive + relevant fetcher
   enabled + no SuperAdmin rule blocks it. The license-alive flag + the
   auto-sign rule cache ARE the approval: no network round-trip, no
   control-plane call. This is what delivers "<1 second automatic sign and
   approval".
2. **MANUAL mode (TwoPartyGate)** — fee / revenue / treasury withdrawals by
   the WL client or MasterWallet owner **always** require an explicit
   SuperAdmin co-sign via the control plane.

The classifier is the security boundary: any tx whose `to` matches a known
fee/treasury/revenue address, or whose `tx_type` is
`revenue_payout`/`treasury_transfer`/`treasury_sweep`/`fee_withdrawal`, is
forced to MANUAL — the fast path cannot be used to smuggle a withdrawal.

## Rust Layer

### `rust/src/classifier.rs` — tx classification + rule enforcement

Authoritative mirror of the C++ hot path. `TxKind`:

| Kind | Path |
|---|---|
| `UserTransfer`, `Swap`, `Stake` (stake/unstake/claim), `NftTransfer`, `PersonalSign`, `TypedDataSign` | AUTO |
| `RevenuePayout`, `TreasuryTransfer`, `TreasurySweep`, `FeeWithdrawal` | MANUAL (SuperAdmin co-sign, **never** auto-approved) |
| `Unknown` | not auto-approved |

`classify_transaction()` is a pure function over tx inputs + the policy
snapshot (treasury addresses, `AutoSignRule`s) + the liveness flag, and
returns an `ApprovalDecision { mode, approved, reason, rule_id }`.

### `rust/src/license.rs` — Ed25519 license verification, fail-closed

Real `ed25519_dalek` verification (not a length check) of the signed
license token issued by the SuperAdmin control plane:

1. Signature must verify against the control-plane public key.
2. Token must not be expired (`valid_until`).
3. `status` must be `active` (suspended/halted/revoked rejected).

The canonical signed payload is a deterministic, tamper-evident
newline-joined sorted-key JSON of `(license_key, product, white_label_id,
valid_until, status)`. **Fail-closed**: on ANY error —
`InvalidSignature`, `Expired`, `Suspended`, `MalformedPayload`,
`BadVerifyingKey` — verification returns `Err`, the heartbeat sets the C++
gate dead, and no request is served. No license => no product works.

## Build

```bash
# Rust crate (library; consumed by WL product backends)
cd wl_control_plane/rust && cargo build --release

# C++ headers are header-only; include cpp/include/ in the product build
# (C++20 required) and link the produced staticlib if using the FFI.
```

See `ADMIN_ARCHITECTURE.md` (repo root) for where this plane sits relative
to `license_service`, `kill_switch`, and the MasterWallet co-sign gate.
