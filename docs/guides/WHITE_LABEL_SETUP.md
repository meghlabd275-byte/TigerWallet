# White Label Setup Complete Guide

## Table of Contents
1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Creating White Label](#creating-white-label)
4. [Configuration](#configuration)
5. [Branding](#branding)
6. [Deploying](#deploying)
7. [Management](#management)

---

## Overview

TigerWallet White Label allows you to create your own branded cryptocurrency wallet service. This guide walks through setting up a complete white label.

### Key Features
- Custom branding (logo, colors, domain)
- Revenue sharing (80/20 split)
- Full user management
- KYC integration
- Transaction fees
- API access

---

## Prerequisites

Before creating a white label, ensure you have:

1. **Domain** - Your custom domain (e.g., wallet.yourbrand.com)
2. **SSL Certificate** - For HTTPS
3. **Cloud Account** - For hosting (AWS, GCP, Azure, etc.)
4. **Payment Gateway** - For fee collection (optional)

---

## Creating White Label

### Step 1: Login as Master Admin

```
URL: https://admin.tigerwallet.io
Login: Your admin credentials
```

### Step 2: Navigate to White Labels

```
Admin Panel > White Labels > Create New
```

### Step 3: Fill Details

```json
{
  "name": "MyBrand Wallet",
  "domain": "wallet.mybrand.com",
  "email": "admin@mybrand.com",
  "description": "My custom cryptocurrency wallet",
  "contactPhone": "+1234567890",
  "businessAddress": "123 Main St, City, Country"
}
```

### Step 4: Configure Features

Select enabled features:
- [ ] Trading/Swaps
- [ ] Staking
- [ ] NFT Support
- [ ] DeFi Integration
- [ ] Bridge Support
- [ ] Hardware Wallet
- [ ] Multi-Sig

### Step 5: Set Fees

| Fee Type | Your Share | Platform Share |
|----------|-----------|----------------|
| Trading Fee | 0.1-10% | 90-99% |
| Swap Fee | 0.05-5% | 95-99.95% |
| Withdrawal Fee | Network + 0-2% | Fixed |

### Step 6: Submit Application

Click "Submit" to send for approval.

---

## Configuration

### Domain Setup

#### Option 1: Subdomain
```
DNS: Create CNAME record
Type: CNAME
Name: wallet
Value: tigerwallet.io
```

#### Option 2: Custom Domain
```
DNS: Create A record
Type: A
Name: wallet
Value: YOUR_SERVER_IP
```

#### Option 3: Cloudflare Proxy
```
DNS: Create CNAME with Cloudflare proxy enabled
```

### SSL Certificate

#### Let's Encrypt (Free)
```bash
# Using Certbot
sudo certbot --nginx -d wallet.yourdomain.com
```

#### Cloudflare Origin
1. Create certificate in Cloudflare
2. Download certificate and key
3. Install on your server

---

## Branding

### Logo Upload
```
Recommended: 512x512 PNG
Accepted: PNG, JPG, SVG
Max size: 2MB
```

### Color Scheme

| Element | Default | Customizable |
|---------|---------|--------------|
| Primary | #6366F1 | ✅ Yes |
| Secondary | #8B5CF6 | ✅ Yes |
| Success | #22C55E | ✅ Yes |
| Error | #EF4444 | ✅ Yes |
| Warning | #F59E0B | ✅ Yes |
| Background | #FFFFFF | ✅ Yes |
| Text | #1F2937 | ✅ Yes |

### Theme Configuration

```json
{
  "branding": {
    "logoUrl": "https://yourdomain.com/logo.png",
    "faviconUrl": "https://yourdomain.com/favicon.ico",
    "appName": "MyBrand Wallet",
    "primaryColor": "#your-color",
    "secondaryColor": "#your-color",
    "darkMode": {
      "primaryColor": "#your-dark-color",
      "background": "#0f172a"
    }
  }
}
```

---

## Deploying

### Self-Hosted Deployment

#### 1. Server Requirements
- CPU: 4+ cores
- RAM: 8+ GB
- Storage: 100+ GB SSD
- OS: Ubuntu 22.04 LTS

#### 2. Install Dependencies
```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com | sh

# Install PostgreSQL
sudo apt install postgresql

# Install Redis
sudo apt install redis-server
```

#### 3. Configure Environment
```bash
cat > .env << EOF
DATABASE_URL=postgres://user:pass@localhost:5432/tigerwallet
REDIS_URL=redis://localhost:6379
WHITE_LABEL_ID=your-white-label-id
API_KEY=your-api-key
EOF
```

#### 4. Start Services
```bash
# Start all services
docker-compose up -d
```

### Cloud Deployment

#### AWS
1. Launch EC2 instance
2. Configure RDS PostgreSQL
3. Configure ElastiCache Redis
4. Deploy application
5. Configure Route 53

#### Google Cloud
1. Create Compute Engine
2. Configure Cloud SQL
3. Configure Memorystore
4. Deploy application
5. Configure Cloud DNS

#### Azure
1. Create Virtual Machine
2. Configure Azure Database
3. Configure Azure Cache
4. Deploy application
5. Configure Azure DNS

---

## Management

### User Management

#### Creating Users
```bash
# Via Admin Panel
Admin > Users > Add User

# Via API
POST /api/v1/admin/users
{
  "email": "user@example.com",
  "username": "username",
  "kycLevel": "none|basic|full"
}
```

#### Managing KYC
```bash
# Review KYC
GET /api/v1/admin/kyc

# Approve KYC
POST /api/v1/admin/kyc/{id}/approve

# Reject KYC
POST /api/v1/admin/kyc/{id}/reject
{
  "reason": "Document expired"
}
```

### Transaction Management

#### Viewing Transactions
```bash
# All transactions
GET /api/v1/admin/transactions

# Filter by status
GET /api/v1/admin/transactions?status=pending

# Filter by user
GET /api/v1/admin/transactions?userId={id}
```

#### Processing Issues
```bash
# Cancel transaction
POST /api/v1/admin/transactions/{id}/cancel

# Refund transaction
POST /api/v1/admin/transactions/{id}/refund
```

### Revenue Management

#### Viewing Revenue
```bash
# Revenue dashboard
GET /api/v1/admin/analytics/revenue

# By period
GET /api/v1/admin/analytics/revenue?period=daily|monthly|yearly

# By fee type
GET /api/v1/admin/analytics/revenue?type=swap|withdrawal|trading
```

#### Withdrawing Revenue
```bash
# Request withdrawal
POST /api/v1/admin/revenue/withdraw
{
  "amount": "1000.00",
  "address": "0x..."
}
```

### API Access

#### Creating API Keys
```
Admin Panel > API Management > Create Key
```

#### API Key Permissions
- [ ] Read Users
- [ ] Write Users
- [ ] Read Transactions
- [ ] Write Transactions
- [ ] Read Analytics
- [ ] Manage KYC

---

## Best Practices

### Security
1. Enable 2FA for all admin accounts
2. Use IP whitelisting for API keys
3. Regular security audits
4. Keep software updated

### User Experience
1. Clear branding
2. Responsive support
3. Fast transactions
4. Transparent fees

### Compliance
1. KYC verification
2. Transaction monitoring
3. Audit logs
4. Data retention policies

---

## Support

- Documentation: https://docs.tigerwallet.io
- GitHub Issues: https://github.com/meghlabd275-byte/TigerWallet/issues
- Email: support@tigerwallet.io
- Discord: https://discord.gg/tigerwallet
