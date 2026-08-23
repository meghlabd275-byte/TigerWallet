# 🔐 WL-Shared — Shared White-Label Library (`wlgate` + `wlcrypto`)

The shared Go library imported by **every** standalone white-label product
(`wl_master_wallet`, `wl_user_wallet`, `wl_bots`, `wl_card`,
`wl_liquidity`, `wl_project_party`). It provides two packages:

- **`wlgate`** — the fail-closed license gate + heartbeat client.
- **`wlcrypto`** — local key custody: BIP-39/32/44 derivation, EIP-1559
  signing, AES-GCM at-rest encryption.

Module path: `github.com/tigerwallet/wl_shared/go` (`wl_shared/go/go.mod`).

## `wlgate` — fail-closed license gate

- **Starts dead**: the gate denies protected requests until the first
  successful heartbeat validates the license against the TigerWallet control
  plane.
- **Heartbeat client**: each WL product posts heartbeats
  (`TWO_PARTY_GATE_URL`, `WL_CLIENT_ID`, `WL_LICENSE_KEY` are mandatory at
  boot). Responses carry the license status, kill switch, per-category
  fetcher disable lists, and the **feature-flag policy snapshot**, which are
  cached locally.
- **`gate.Middleware(product, category)`**: Gin middleware that enforces
  JWT auth, live license, kill switch, fetcher-category kill, and policy
  features on every protected request — a suspended/revoked license or stale
  heartbeat 403s/503s the API.
- The TTL is deliberately shorter than the license's 5-minute token window,
  so a halted or disconnected product stops serving traffic on its own.

## `wlcrypto` — local key custody

Verified functions (`wl_shared/go/wlcrypto/crypto.go`):

| Function | Purpose |
|---|---|
| `GenerateMnemonic` | BIP-39 mnemonic generation |
| `MnemonicToSeed` | BIP-39 seed derivation (PBKDF2, optional passphrase) |
| `DeriveEVMPrivateKey` | BIP-32/BIP-44 EVM key derivation (`m/44'/60'/0'/0/i`), verified against the canonical BIP-44 test vector |
| `AddressFromPrivateKey` | EVM address from key |
| `SignTransaction` | **EIP-1559** (type-2) transaction signing |
| `SignMessage` | EIP-191 personal-sign |
| `EncryptSeedAtRest` / `DecryptSeedAtRest` | scrypt(passphrase) → **AES-256-GCM** at-rest encryption |
| `RandomBytes` | CSPRNG helper |

**Key custody is local**: WL products derive, sign, and encrypt keys inside
the client's own cloud. They **never delegate signing or seed custody to the
TigerWallet cloud** — the control plane only sees license heartbeats.

Tests: `crypto_test.go` (BIP-44 vector, AES-GCM roundtrip, mnemonic
generation, message signing).

## Run

It is a library; run its tests:

```bash
cd wl_shared/go
go test ./...
```

## Architecture role

Per `ADMIN_ARCHITECTURE.md` (§3): `wl_shared` is the common denominator of
the WL fleet. `wlgate` is what makes every self-hosted product **fail-closed
under license control** (heartbeat, kill switch, category kill, feature
flags), while `wlcrypto` guarantees that **private keys never leave the WL
client's infrastructure** — TigerWallet can halt a product but can never
touch its keys.
