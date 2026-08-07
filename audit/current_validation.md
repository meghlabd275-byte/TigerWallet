# TigerWallet Current Validation

## Extension package

The canonical Chrome Manifest V3 package now references `src/popup/popup.html`, `dist/service-worker.js`, and `dist/inject.js`. The compiled service worker and content injector pass Node syntax validation, all three manifest references exist, and the stale duplicate `browser_extension/chrome/manifest/manifest.json` was removed. The generated `dist/` directory remains ignored by the repository and must be addressed before a distributable extension can be reproduced from a clean checkout.

## Security hardening completed in this pass

The user-wallet Go registration path now stores bcrypt password hashes and login verifies them with bcrypt. Hard-coded token prices, network-health responses, and gas prices now fail closed on cache misses instead of returning fabricated values. The swap service no longer creates simulated routes, fabricated transaction hashes, or completed fake swaps; it returns explicit provider-unavailable errors until a live quote and signed execution provider is configured. The React hardware-wallet service no longer derives fabricated public keys or signatures and rejects device initialization when a real transport cannot provide key material.

## Validation results

`node audit/validate_extension.js` passes. `git diff --check` passes. The user-wallet Go module cannot currently run as a module-wide test because it has mixed packages at `user_wallet/go` and lacks committed `go.sum` entries for its declared dependencies. An attempted `go mod tidy` was reverted because it tried to introduce SQLite dependencies and a Go 1.25 toolchain requirement, contrary to the PostgreSQL/Redis-only requirement. The React subproject has no committed TypeScript project configuration at its inspected root; a package-manager validation attempt created temporary lock/workspace files, which were removed.

## Repository hygiene blocker

The working tree contains a large inherited set of modified files and generated/untracked artifacts, including `target/` directories, package-manager lock/workspace files, and many backend changes. These must be classified before any commit to `main`; generated build outputs should not be committed unless the release packaging process explicitly requires them.
