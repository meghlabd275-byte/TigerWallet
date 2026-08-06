# TigerWallet Admin System - Unified Architecture

## Overview
This document describes the complete, unified admin architecture for TigerWallet with:
- **C++**: Ultra-low latency critical paths (trade execution, real-time monitoring)
- **Rust**: High-speed, memory-safe services (blockchain sync, transaction validation)
- **Go**: High-load distributed services (API gateway, user management, analytics)

## Technology Stack

### Backend Layers

| Layer | Technology | Purpose | Latency Target |
|-------|------------|---------|----------------|
| Trade Execution | C++ | Critical path execution | < 1ms |
| Real-time Analytics | C++ | Live data processing | < 5ms |
| Blockchain Core | Rust | Transaction validation | < 10ms |
| Wallet Core | Rust | Key management, signing | < 5ms |
| API Gateway | Go | Request routing, auth | < 50ms |
| User Management | Go | KYC, accounts | < 100ms |
| Analytics | Go | Aggregations, reports | < 500ms |

### Database
- **PostgreSQL**: Primary data store (users, transactions, KYC)
- **Redis**: Caching, sessions, real-time data
- **TimescaleDB**: Time-series data (analytics, metrics)

### Frontend
- **Web**: React + TypeScript + Material UI
- **Mobile**: React Native (iOS/Android)
- **Desktop**: Electron + React
- **Extensions**: Chrome, Firefox, Safari

## Component Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Load Balancer                            │
└─────────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│  C++ Gateway  │    │  Go API GW    │    │  Go Admin    │
│  (Trades)     │    │  (REST/WS)   │    │  Services    │
└───────────────┘    └───────────────┘    └───────────────┘
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Rust Blockchain Core                          │
│  - Transaction Validation    - Smart Contract Interaction       │
│  - Block Sync               - Multi-chain Support               │
└─────────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│  PostgreSQL   │    │    Redis      │    │ TimescaleDB  │
│  (Primary)    │    │   (Cache)    │    │  (Metrics)   │
└───────────────┘    └───────────────┘    └───────────────┘
```

## Admin Roles & Permissions

### Super Admin
- Full platform control
- Master Wallet Admin authorization
- Profit sharing configuration
- System-wide feature flags
- All audit logs access
- Database backup/restore

### Admin
- User management (KYC, suspension)
- Transaction monitoring
- Token/Pair management
- Fee configuration
- Support ticket handling
- White label oversight

### White Label Admin
- User management (own tenant)
- Transaction review
- Custom branding
- Fee customization
- Analytics access

## API Endpoints

### Authentication
- `POST /api/v1/auth/login` - Admin login
- `POST /api/v1/auth/2fa/verify` - 2FA verification
- `POST /api/v1/auth/refresh` - Token refresh

### Users
- `GET /api/v1/users` - List users
- `GET /api/v1/users/:id` - User details
- `PUT /api/v1/users/:id/status` - Update status
- `POST /api/v1/users/:id/ban` - Ban user
- `POST /api/v1/users/:id/suspend` - Suspend user

### KYC
- `GET /api/v1/kyc` - List KYC requests
- `POST /api/v1/kyc/:id/approve` - Approve KYC
- `POST /api/v1/kyc/:id/reject` - Reject KYC

### Transactions
- `GET /api/v1/transactions` - List transactions
- `POST /api/v1/transactions/:id/flag` - Flag transaction
- `GET /api/v1/transactions/:id` - Transaction details

### Withdrawals
- `GET /api/v1/withdrawals` - List withdrawals
- `POST /api/v1/withdrawals/:id/approve` - Approve
- `POST /api/v1/withdrawals/:id/reject` - Reject

### Tokens
- `GET /api/v1/tokens` - List tokens
- `POST /api/v1/tokens` - Create token
- `PUT /api/v1/tokens/:id` - Update token
- `DELETE /api/v1/tokens/:id` - Delete token

### Pairs
- `GET /api/v1/pairs` - List pairs
- `POST /api/v1/pairs` - Create pair
- `PUT /api/v1/pairs/:id/status` - Update status

### Blockchains
- `GET /api/v1/blockchains` - List blockchains
- `POST /api/v1/blockchains` - Add blockchain
- `PUT /api/v1/blockchains/:id` - Update blockchain

### Fees
- `GET /api/v1/fees` - List fees
- `POST /api/v1/fees` - Create fee
- `PUT /api/v1/fees/:id` - Update fee

### White Labels
- `GET /api/v1/whitelabels` - List white labels
- `POST /api/v1/whitelabels` - Create white label
- `PUT /api/v1/whitelabels/:id` - Update white label

### Analytics
- `GET /api/v1/analytics/dashboard` - Dashboard stats
- `GET /api/v1/analytics/users` - User analytics
- `GET /api/v1/analytics/transactions` - Transaction analytics

### Super Admin
- `POST /api/v1/superadmin/admins` - Create admin
- `GET /api/v1/superadmin/profit-sharing` - Profit config
- `POST /api/v1/superadmin/profit-sharing` - Set profit share
- `POST /api/v1/superadmin/backup` - Create backup

## Security

### Authentication
- JWT tokens with short expiry (15 min)
- Refresh tokens with long expiry (7 days)
- 2FA (TOTP) mandatory for all admins
- IP whitelist enforcement

### Authorization
- Role-based access control (RBAC)
- Permission-based API access
- Audit logging for all actions

### Data Protection
- AES-256 encryption for sensitive data
- TLS 1.3 for all communications
- Secrets rotation policy

## Deployment

### Horizontal Scaling
- C++ services: 3-5 instances per region
- Rust services: 5-10 instances per region
- Go services: 10-20 instances per region

### Geographic Distribution
- US East (primary)
- US West (failover)
- EU Central (GDPR)
- Asia Pacific (low latency)

### Monitoring
- Prometheus metrics
- Grafana dashboards
- PagerDuty alerts
- Log aggregation (ELK)
