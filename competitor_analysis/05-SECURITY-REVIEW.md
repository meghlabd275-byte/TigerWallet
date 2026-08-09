# TIGERWALLET — SECURITY REVIEW (High-grade Storage / Signing / Threats)

Security-focused companion to the audit (`03-TIGERWALLET-AUDIT.md`). The goal: determine whether the repo's security posture meets the "no fake logic, no demo, no mock, high-grade security" bar, per capability.

---

## 1. Where security is genuinely real ✅

### 1.1 Seed generation — REAL
- `go/wallet_api/wallet_engine.go`: `bip39.NewEntropy(128|256)` + `bip39.NewMnemonic` (cryptographically-secure entropy via tyler-smith; Math/rand **not** used at the core).
- `frontend/web_nextjs/app/master_wallet/page.tsx`: `@scure/bip39` `generateMnemonic(wordlist, 256)` (256-bit + checksum), WebCrypto-backed. ❌ Contrast: `app/wallet/page.tsx` uses **`Math.random()`** word-picking — **not secure** (see §2.2).

### 1.2 HD key derivation — REAL
- `go/wallet_api/hd_derive.go`: real BIP-32 CKD via HMAC-SHA512, hardened index encoding (`>= 0x80000000`), secp256k1 scalar reduction; BIP-44 path parsing with `'`/`h` hardened suffixes; BIP-44 test vector passes (`abandon…¹ about` m/44'/60'/0'/0/0 → `0x9858…Eda94`).
- Seed → key via standard `bip39.NewSeed` (PBKDF2-HMAC-SHA512).

### 1.3 Signing — REAL
- `go/wallet_api`: EIP-1559 `DynamicFeeTx` + legacy, `NewLondonSigner`, `types.SignTx`; personal-sign with `\x19Ethereum Signed Message:\n` prefix, recovery byte 27/28; real `MarshalBinary` RLP. No fabricated signatures.
- `go/signature_service`: `crypto.Sign(accounts.TextHash/Hash(...), privKey)` — correct EIP-191/ETHSign path.
- `dapp_browser/go/walletconnect.go`: `handlePersonalSign` / `handleEthSignTypedData_v4` do real ECDSA; **reject with JSON-RPC `-32000`** ("Signing not available") when no key configured — do not fabricate.

### 1.4 Seed-at-rest encryption — REAL
- `go/wallet_api`: AES-256-GCM (JSON `{v,salt,iv,ciphertext}`), KDF = scrypt (N=32768, r=8, p=1) and PBKDF2. Wrong password fails on GCM authentication tag (no password stored). This matches the top-wallet storage bar.
- Browser (`master_wallet/page.tsx`): WebCrypto AES-GCM + PBKDF2 600k iters; only the encrypted blob is persisted, mnemonic held in memory only.

### 1.5 WalletConnect security — REAL
- `go/walletconnect`: real WC v2 registration/relay over TLS WebSocket to `relay.walletconnect.com`, **AES-256-GCM** payload sealing with the session key, topic-based envelopes. No plaintext fallback.

### 1.6 ZK proof — REAL
- `core/rust/zk_infrastructure`: real Fiat-Shamir Schnorr (Ristretto255), verifier recomputes the challenge and rejects the identity point / malformed encodings; domain-separated. No always-true verification.

### 1.7 Scam/honeypot/transaction screening — REAL logic
- `security_center/honeypot_detector`: real `eth_call`/`eth_getCode` module audits (`transfer`, `_transfer`, fee/whitelist/burn checks), liquidity reads, risk scoring → "SUSPICIOUS_CONTRACT" etc.
- `security_center/wallet_guardian`: transaction rule engine, address blacklist/pattern risk scoring.
- `sends`-side: scan-before-send hooks structurally present (wire-up incomplete — see 4).

---

## 2. Security anti-patterns / deal-breakers (where "high-grade" is NOT met) ❌

### 2.1 FAKE crypto that must not ship
- **P-256 used where secp256k1 is required** across `go/wallet_service`, `go/multisig_service`, `go/listing_service`, `go/mpc`, `go/mpc/threshold`, `backend/` `generateMPCSignature()`-as-sha256, `rust/mpc/signing.rs` (SHA-256 "signature"). These are not just stubs — they are **unsound cryptography** (wrong curve / non-ECDSA sigs).
- **MPC "verify" is accept-anything** (`verifySignature` returns `r>0 && s>0`), **"Simplified Lagrange"** returns a constant.
- **Fabricated transaction hashes** (`return "txhash", nil` in `defi_service`; `"In production, this would broadcast"` in payment/ens/staking; random `0x…` in the web wallet) would make a user believe a tx succeeded when none was broadcast — a **critical correctness + liquidity-risk bug**, not just cosmetics.
- **XOR "encryption"** of mnemonics in `mobile/flutter`, `master_wallet/flutter`, and **sha256-as-KDF** entropy in mobile Flutter are not secure key storage.

### 2.2 Insecure/incomplete browser flow
- `app/wallet/page.tsx` derives seeds from **`Math.random()`** over a hardcoded word array (no checksum, weak PRNG) — a weak-seed vulnerability if ever shipped, and it fabricates addresses/hashes. This path must be deleted/rewired to the real backend.

### 2.3 No hardware-backed key storage anywhere user-facing
- No TEE / Secure Enclave / HSM / hardware-wallet integration actually executes key ops in a secure environment. `hardware_wallet/`, `hardware_backend`, `hsm_integration` are **stubs**. Private keys live in process memory / DB (encrypted at rest only).

### 2.4 Authentication / secrets hygiene risks
- **JWT is HS256** (symmetric, server-secret) for admin/master APIs — fine for admin; should be asym (ES256/RS256) + short-lived + rotation if exposed broadly; unverified issuance in several stub modules.
- Committed binary `master_wallet/go/services/services` (~22 MB ELF) in git — **should be removed from VCS** (supply-chain / drift risk).
- Granting the stated "no fake logic, high-grade security" goal requires removing ALL of §2.1–§2.2 before any of these modules could be called production-safe.

---

## 3. Tiered security posture summary

| Area | Status | Notes |
|---|---|---|
| Mnemonic generation | 🟡 Real in backend + master-web; **`app/wallet` uses `Math.random()`** | Fix/delete the bad path |
| BIP-32/44 derivation | 🟢 REAL | `go/wallet_api` |
| ECDSA signing (EVM, EIP-191/1559/712) | 🟢 REAL | `go/wallet_api`, `signature_service`, `dapp_browser` |
| Seed at rest | 🟢 REAL AES-GCM + scrypt/PBKDF2 | compliant with the top wallets |
| WalletConnect payload sealing | 🟢 REAL AES-GCM | |
| ZK proof | 🟢 REAL Schnorr | |
| Scam/honeypot/token scanning | 🟢 REAL (RPC) | honeypot + wallet_guardian |
| Mobile (incl. seed/KDF/signing) | 🔴 FAKE | XOR, sha256-KDF, static wordlists, zero keys, fabricated hashes |
| Go service layer (swap/staking/defi/mpc/multisig/payment/ens/listing) | 🔴 FAKE | P-256, accept-anything, `"txhash",nil`, "In production…" |
| Rust MPC/HFT + C++ stubs | 🔴 STUB | header-only, SHA-256 "sig" |
| Hardware wallet / TEE / HSM | 🔴 Not implemented | stub dirs only |
| Smart-contract AA (paymaster/factory) | 🔴 `signature.length==64` fake | port to `PackedUserOperation`/BaseAccount |
| Admin/DB/monitoring | 🟢 REAL | GORM/pgx, monitoring, compose |

---

## 4. Security roadmap (aligns with `04-GAP-ANALYSIS.md`)

1. **Wire one real end-to-end flow (backend→web)** and delete `Math.random()`-seed + random `0x…` fallbacks (critical).
2. **Replace P-256 "MPC"/multisig/listing/payment faux-crypto** with real secp256k1 + a sound threshold scheme (GG18/WalletConnect C14) — never ship the current accept-anything path.
3. **Fix mobile** to real BIP-39 + real key ops (delegate to backend or a secure lib) — remove XOR/sha256 keys.
4. Make **ERC-4337 real** (working Account/Paymaster/Factory) and put seed/key ops in a **hardware/secure-enclave** boundary.
5. **Hygiene**: remove the committed 22 MB ELF, add missing `go.mod`s, tighten JWT (asym, expiry, rotation), ensure `forge build` passes before any contract is treated as deployable.
6. **Add a formal threat model + audit trail** for signing and spend paths, and CI gating on "no stubs allowed" markers before a security grade of "high" can honestly be claimed.

---

**Security bottom line:** The *core* (key mgmt, signing, seed-at-rest, WC2, ZK, scam-scanning, fetchers) is genuinely real and comparable to competitor secure-enclave practice. **But the repo as a whole does NOT meet "no fake logic, high-grade security"** because the mobile paths, the broad Go/Rust service layer, the "MPC"/multisig/AA, hardware-wallet, and the web `app/wallet` path all contain **fabricated, weak, or unsound** crypto that must be fixed/removed before anything there can be called shipping-grade. The safest statement today: **the plumbing is real; the product's security surface is largely unimplemented or fake.**