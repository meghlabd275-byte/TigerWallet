# WL Control Plane — C++ Hot-Path Components

Two wait-free, single-digit-microsecond components that sit on the hot path of
every white-label product backend. Verified to build and pass tests with
g++ 14.2 (Debian) and cmake 3.31.

## Components

### 1. `WlGate` — wait-free µs license + feature-flag gate
(`include/wl_gate.hpp`, `src/wl_gate.cpp`)

Runs on **every request** before dispatch. Holds an atomic liveness flag plus a
feature-flag snapshot (pushed by the Rust `LicenseClient` after validate /
heartbeat) and answers `is_alive()` / `fetcher_enabled(product, fetcher)` in
O(1) with wait-free reads (lock-free atomic for liveness, `shared_mutex` read
lock on the flag map, contended only during a heartbeat refresh). It does **no
crypto** — Ed25519 license verification is delegated to the Rust SDK
(`wl_control_plane/rust/src/license.rs`, tested separately via cargo; no
prebuilt Rust `.so`/`.a` is required to build or test the C++ side).

### 2. `WlAutoApprover` — AUTO vs MANUAL transaction classifier
(`include/wl_auto_approver.hpp`, `src/wl_auto_approver.cpp`)

Classifies every outgoing transaction before sign/broadcast:

- **AUTO mode**: user-initiated txs (transfer / swap / stake / nft_transfer /
  personal_sign / typed_data_sign) are approved in-process in <1 ms, provided
  the license is alive and no blocking auto-sign rule matches. No network
  round-trip.
- **MANUAL mode**: any fee / revenue / treasury movement
  (`revenue_payout`, `treasury_transfer`, `treasury_sweep`, `fee_withdrawal`)
  or **any tx whose recipient is a known treasury/revenue/fee address** is
  forced to the two-party path requiring an explicit SuperAdmin co-sign via
  the control plane. These are never auto-approved.

## Build (verified)

With CMake (produces shared + static libs and both unit tests):

```sh
cd wl_control_plane/cpp
cmake -B build -DCMAKE_BUILD_TYPE=Release
cmake --build build -j
ctest --test-dir build --output-on-failure
```

Without CMake (plain g++):

```sh
cd wl_control_plane/cpp
g++ -std=c++20 -O2 -Wall -Wextra -c src/wl_gate.cpp -I include
g++ -std=c++20 -O2 -Wall -Wextra -c src/wl_auto_approver.cpp -I include
# unit tests:
g++ -std=c++20 -O2 -I include tests/test_wl_gate.cpp src/wl_gate.cpp -o test_wl_gate && ./test_wl_gate
g++ -std=c++20 -O2 -I include tests/test_wl_auto_approver.cpp src/wl_auto_approver.cpp -o test_wl_auto_approver && ./test_wl_auto_approver
# fail-closed smoke test:
g++ -std=c++20 -O2 -I include tests/smoke_fail_closed.cpp src/wl_gate.cpp src/wl_auto_approver.cpp -o smoke_fail_closed && ./smoke_fail_closed
```

Or run the whole verification in one shot:

```sh
./verify_build.sh
```

Both `-std=c++17` and `-std=c++20` compile cleanly; the CMake build pins C++20.
No external dependencies (no nlohmann/json, no Qt, no Rust linkage).

## C ABI (`include/wl_gate_abi.h` / `extern "C"` block in `wl_auto_approver.hpp`)

Pure-C surface for FFI from Go (cgo) / Rust / Node backends — no C++ name
mangling, no C++ stdlib types cross the boundary:

| Function | Returns |
|---|---|
| `int wl_gate_is_alive(void)` | 1 alive, 0 dead |
| `const char* wl_gate_reason(void)` | dead reason (copy before next call) |
| `void wl_gate_set_alive(int alive, const char* reason)` | pushed by Rust SDK after validate/heartbeat |
| `int wl_gate_fetcher_enabled(const char* product, const char* fetcher)` | 1 permitted, 0 denied (null-safe: returns 0) |
| `void wl_gate_set_flags(const char* json_array)` | flag snapshot `[{"product":..,"fetcher":..,"enabled":..}]` |
| `int wl_auto_approve_classify(tx_type, to, token, amount, &reason, &rule_id)` | 0 = AUTO approved, 1 = MANUAL (two-party required), 2 = AUTO denied |
| `wl_auto_approver_set_alive` / `_add_treasury_address` / `_set_treasury_addresses_json` / `_set_rules_json` | snapshot pushers (heartbeat path) |

## Fail-closed guarantee

- `WlGate` starts **dead** (`alive_` defaults to `false`) and stays dead until
  the Rust SDK pushes a successful validation via `wl_gate_set_alive(1, ...)`.
- When dead, `fetcher_enabled()` returns `false` for **every** product/fetcher
  regardless of the flag cache, and `classify()` never auto-approves (rc 2).
- Revocation is immediate: a single atomic store flips every subsequent
  request to denied.
- An invalid/expired/revoked license therefore can never be served by the fast
  path. Verified by `tests/smoke_fail_closed.cpp`.

## Tests

- `tests/test_wl_gate.cpp` — gate semantics (default-dead, whole-product
  disable, per-fetcher disable, dead-overrides-flags, C ABI roundtrip).
- `tests/test_wl_auto_approver.cpp` — AUTO/MANUAL boundary, treasury-address
  forcing, license-denied, blocking-rule cases.
- `tests/smoke_fail_closed.cpp` — end-to-end fail-closed smoke test with an
  obviously-invalid license (never validated), C ABI denial codes, and
  immediate revocation.
