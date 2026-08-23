# KYC & White-Label Onboarding Auto-Approval Workflow

> **Note**: this service was **renamed from `auto_approval_workflow`**. The
> old name collided conceptually with transaction auto-signing — this
> service has **nothing to do with tx signing approval** (that lives in the
> MasterWallet auto-signer / two-party co-sign path). It automates **KYC
> application review and white-label tenant onboarding**.

A Go service (`go/main.go`, Gin + PostgreSQL (pgx) + Redis, zerolog) that
runs a workflow engine with steps, conditions, retries, and timeouts to
route onboarding applications automatically.

## Risk-Based Routing

The default workflow's decision engine (`Condition`s on `riskScore`):

| Condition | Action |
|---|---|
| `riskScore <= 0.2` | **approve** (auto-approve) |
| `riskScore >= 0.8` | **reject** (auto-reject) |
| `riskScore > 0.2` (otherwise) | **review** (manual review queue) |

The workflow definition also carries `AutoApprove: true` and
`AutoApproveThreshold: 0.3`. Applications are flagged `autoApproved` /
`manualReview` accordingly; a manual-review override endpoint lets an admin
override the engine's decision.

## Data Model

- `KYCApplication`: id, userId, whiteLabelId, type
  (`identity|address|selfie|document`), status
  (`pending|processing|approved|rejected|needs_review`), riskScore,
  riskLevel (`low|medium|high|critical`), confidence, documents,
  autoApproved, verified/rejected timestamps + reason.
- `WhiteLabelApplication`: tenant onboarding applications with the same
  workflow routing.
- `Workflow` / `WorkflowExecution`: steps (e.g. document check, sanctions
  check, decision engine), per-step timeout/retry/on-failure policy.

## HTTP API (`/api/v1`, port from `PORT`, default **8087**)

- `POST /kyc/applications`, `GET /kyc/applications/:id`
- `POST /white-label/applications`
- `GET /workflow/status`, `POST /workflow/override`
- `POST /approval/rules`, `GET /approval/rules`
- `GET /queue/status`
- `GET /health`

## Environment Variables

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8087` | Listen port |
| `DATABASE_URL` | — | PostgreSQL |
| `REDIS_URL` | — | Redis (approval queue) |
| `WEBHOOK_URL` | — | Notification webhook |
| `LOG_FORMAT` | JSON | `pretty` for console logs |

## How to Run

```bash
cd kyc_onboarding_workflow/go
DATABASE_URL=... REDIS_URL=... go run .
```
