# Contributing to TigerWallet

## Branch policy (owner order)

All work is committed and pushed **directly to `main`**.

- Do **not** open pull requests.
- Do **not** push to topic branches.
- If a pull request exists, merge it into `main` and continue on `main`.

`main` is the single integration branch. Every change lands there.

## Language policy

| Layer | Language | Reason |
|---|---|---|
| Latency-critical paths | C++ | Ultra-low latency |
| Security-critical / memory-safe | Rust | Safety and maintainability at speed |
| Distributed / high-load services | Go | Horizontal scale, worldwide load |
| AI / ML | Python | Model tooling |
| Frontends | TypeScript (`.tsx`), React, Next.js, Tailwind CSS | — |

## Code standards

- No mock data, no demo logic, no simulations, no hardcoded values, no skeleton
  placeholders. Every code path must perform real work.
- No fabricated results. When a capability is not configured or not supported,
  fail closed with an explicit error. Never return a plausible-looking value
  that was not really produced.
- Never write secrets, keys, or credentials into source. Read them from the
  environment and fail closed when they are absent.
- Parameterize every SQL statement. Never interpolate user input into a query.
- Escape or build DOM nodes programmatically. Never inject untrusted input as
  HTML.
- Every privileged operation requires an authorization check.
- Duplicate implementations of the same capability are not permitted. Extend
  the canonical one.
