# Allowance Manager (ERC-20 Approval Scan & Revoke)

> **Note**: this tool was **renamed from `approval_manager`**. The old name
> was ambiguous — this has **nothing to do with transaction-signing
> approval** (that lives in the MasterWallet auto-signer / two-party co-sign
> path). "Approval" here means the **ERC-20 `approve()` allowance** a token
> owner has granted to a spender contract.

A user-facing security tool (Next.js frontend,
`frontend/src/page.tsx`, component `ApprovalManagerPage`) for inspecting and
revoking standing token allowances on a wallet address.

## Features

- **Scan all token approvals** for an address: spender, token
  (address/symbol/name/decimals/logo/price), approved amount, USD value,
  infinite-allowance flag, last-updated time, granting tx hash.
- **Risk assessment** per allowance: `low | medium | high | critical`,
  with aggregate stats (total approvals, per-risk counts, total USD value,
  infinite-approval count).
- **Revocation**, including **batch revocation** of selected allowances
  (each revoke submits an `approve(spender, 0)` transaction and returns its
  tx hash via `RevokeResult`).
- **Real-time monitoring** of allowance state.

## API

The UI talks to a backend at `NEXT_PUBLIC_API_URL` (default
`http://localhost:9098`).

## How to Run

```bash
cd allowance_manager/frontend
npm install && npm run dev
```
