# 🐯 Multi-Level White Label — Hierarchical WL System

A standalone service implementing **hierarchical white-label structures**
with parent–child relationships: a root white label can create sub white
labels, which can themselves create sub white labels, up to **4 levels**
(levels 0–3). Commissions and revenue share flow upward through the tree.

## Port

`8088` (env `PORT`).

## Tech stack

- Go, Gin
- PostgreSQL (`pgx/v4` pool) — `hierarchical_white_labels`,
  `commission_records` tables (auto-created at boot)
- Redis (`go-redis/v8`)
- zerolog structured logging

## Key features (verified in `go/main.go`)

- **Hierarchical WL creation** (`POST /api/v1/white-label/hierarchy`):
  each WL record carries `parent_id`, `root_id`, `level`, branding, features,
  chain access, `commission_rate` (default `0.10`) and `revenue_share`
  (default `0.80`), plus sub-account caps (`max_sub_accounts`,
  `current_sub_accounts`).
- **4-level nesting limit**: level 0 = root; creating a child under a level-3
  parent is rejected with `Maximum hierarchy level reached` (levels 0, 1, 2,
  3 are the four allowed levels).
- **Commission / revenue-share splits**: on a revenue event the service walks
  the ancestor chain and writes `commission_records` rows
  (`from_white_label` = child that earned, `to_white_label` = parent that
  receives), applying each ancestor's `commission_rate` / `revenue_share`.
  Per-WL aggregates (`total_revenue`, `total_commission`) are tracked on the
  WL row.
- **Tree & subs introspection**:
  `GET /hierarchy/:id/tree` (recursive tree),
  `GET /hierarchy/:id/subs` (direct sub-accounts).
- **Commission reporting**: `GET /hierarchy/:id/commissions`.
- **Analytics**: `GET /hierarchy/:id/analytics`.
- **Re-parenting**: `PUT /hierarchy/move` moves a WL subtree under a new
  parent (with level validation).

## API summary

| Method | Path | Purpose |
|---|---|---|
| GET | `/health` | Health check |
| POST | `/api/v1/white-label/hierarchy` | Create hierarchical WL |
| GET | `/api/v1/white-label/hierarchy/:id/tree` | Full subtree |
| GET | `/api/v1/white-label/hierarchy/:id/subs` | Direct children |
| GET | `/api/v1/white-label/hierarchy/:id/commissions` | Commission records |
| GET | `/api/v1/white-label/hierarchy/:id/analytics` | Hierarchy analytics |
| PUT | `/api/v1/white-label/hierarchy/move` | Move a WL to a new parent |

## Environment variables

| Variable | Purpose |
|---|---|
| `PORT` | Listen port (default `8088`) |
| `DATABASE_URL` | PostgreSQL |
| `REDIS_URL` | Redis |

## Run

```bash
cd multi_level_white_label/go
go run .
```

## Architecture role

Per `ADMIN_ARCHITECTURE.md`: `multi_level_white_label` models **reseller /
sub-brand hierarchies** on top of the flat white-label system in
`white_label/` — a WL client can itself act as a provider for its own sub
white labels, with commission and revenue-share settling upward toward the
root, while TigerWallet's SuperAdmin remains the ultimate root of control.
