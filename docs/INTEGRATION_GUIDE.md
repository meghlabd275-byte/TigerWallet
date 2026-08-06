# TigerWallet Integration Guide

## Connecting White Label Products to Super Admin

This guide explains how to connect your independently hosted white label product to TigerWallet Super Admin.

---

## Overview

```
┌────────────────────┐     API Calls      ┌─────────────────────┐
│   White Label      │ ───────────────►  │   Super Admin       │
│   Product          │                    │                     │
│   (Your Server)   │ ◄──────────────── │   Fetcher API       │
│                   │    Data/Response   │   Permission Bridge │
│                   │                    │   License Service   │
└────────────────────┘                    └─────────────────────┘
```

---

## Step 1: Obtain Credentials

### From TigerWallet Admin Panel

1. **Login** to your White Label Admin Panel
2. Navigate to **Settings → API Configuration**
3. Generate your **API Key** and **API Secret**
4. Note your **Tenant ID**

### Environment Variables

```bash
# Required
SUPER_ADMIN_URL=https://super-admin.tigerwallet.com
API_KEY=your-api-key
API_SECRET=your-api-secret
TENANT_ID=your-tenant-id
JWT_SECRET=your-jwt-secret
```

---

## Step 2: Configure Your Product

### Docker

```yaml
# docker-compose.yml
services:
  your-product:
    environment:
      - SUPER_ADMIN_URL=https://super-admin.tigerwallet.com
      - API_KEY=${API_KEY}
      - API_SECRET=${API_SECRET}
      - TENANT_ID=${TENANT_ID}
      - JWT_SECRET=${JWT_SECRET}
```

### Kubernetes

```yaml
# deployment.yaml
env:
  - name: SUPER_ADMIN_URL
    value: "https://super-admin.tigerwallet.com"
  - name: API_KEY
    valueFrom:
      secretKeyRef:
        name: tigerwallet-secrets
        key: api-key
```

---

## Step 3: Fetcher API Integration

### Authentication

All requests must include:

```bash
curl -X GET "https://fetcher.tigerwallet.com/api/v1/fetcher/prices?symbols=BTC,ETH" \
  -H "X-API-Key: your-api-key" \
  -H "X-Timestamp: 1699123456" \
  -H "X-Signature: hmac-sha256-signature" \
  -H "X-Tenant-ID: your-tenant-id"
```

### Signature Generation

```python
import hmac
import hashlib
import time

def generate_signature(api_secret, method, path, body=""):
    timestamp = int(time.time())
    message = f"{method}\n{path}\n{timestamp}\n{body}"
    signature = hmac.new(
        api_secret.encode(),
        message.encode(),
        hashlib.sha256
    ).hexdigest()
    return timestamp, signature
```

---

## Step 4: Using Fetcher Data

### Get Token Prices

```python
import requests

def get_prices(symbols):
    url = "https://fetcher.tigerwallet.com/api/v1/fetcher/prices"
    params = {"symbols": ",".join(symbols)}
    headers = {
        "X-API-Key": os.getenv("API_KEY"),
        "X-Tenant-ID": os.getenv("TENANT_ID"),
    }
    response = requests.get(url, params=params, headers=headers)
    return response.json()
```

### Get Wallet Balance

```python
def get_balance(chain, address):
    url = f"https://fetcher.tigerwallet.com/api/v1/fetcher/wallet/{chain}/{address}"
    headers = {
        "X-API-Key": os.getenv("API_KEY"),
        "X-Tenant-ID": os.getenv("TENANT_ID"),
    }
    response = requests.get(url, headers=headers)
    return response.json()
```

---

## Step 5: Permission Sync

### Check Permissions

```python
def check_permission(feature):
    url = "https://permission-bridge.tigerwallet.com/api/v1/permissions"
    headers = {
        "X-API-Key": os.getenv("API_KEY"),
        "X-Tenant-ID": os.getenv("TENANT_ID"),
    }
    response = requests.get(url, headers=headers)
    permissions = response.json()
    return permissions.get("permissions", {}).get(feature, False)
```

### Sync Permissions

```python
def sync_permissions():
    url = "https://permission-bridge.tigerwallet.com/api/v1/permissions/sync"
    headers = {
        "X-API-Key": os.getenv("API_KEY"),
        "X-Tenant-ID": os.getenv("TENANT_ID"),
    }
    response = requests.post(url, headers=headers)
    return response.json()
```

---

## Step 6: License Validation

### Validate License

```python
def validate_license(license_key, hardware_id):
    url = "https://license.tigerwallet.com/api/v1/licenses/validate"
    payload = {
        "license_key": license_key,
        "hardware_id": hardware_id,
        "product": "master_wallet"  # or user_wallet, bots, project_party
    }
    response = requests.post(url, json=payload)
    return response.json()
```

---

## Environment Variables Reference

| Variable | Required | Description |
|----------|----------|-------------|
| `SUPER_ADMIN_URL` | Yes | Super Admin API URL |
| `API_KEY` | Yes | Your API key |
| `API_SECRET` | Yes | Your API secret |
| `TENANT_ID` | Yes | Your tenant ID |
| `JWT_SECRET` | Yes | JWT signing secret |
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `REDIS_URL` | Yes | Redis connection string |
| `KYC_ENABLED` | No | Enable KYC module |
| `LOG_LEVEL` | No | Logging level (info, debug, error) |

---

## Troubleshooting

### Connection Issues

1. **Verify API Key**: Ensure your API key is valid and not expired
2. **Check Network**: Ensure your server can reach `*.tigerwallet.com`
3. **Firewall**: Allow outbound HTTPS (port 443) to TigerWallet IPs

### Permission Denied

1. **Sync Permissions**: Call the sync endpoint to refresh
2. **Check Plan**: Verify your subscription includes the feature
3. **Contact Support**: If issues persist, contact TigerWallet support

### Rate Limiting

- **Free Plan**: 60 requests/minute
- **Basic Plan**: 300 requests/minute
- **Pro Plan**: 1,000 requests/minute
- **Enterprise**: Custom limits

Implement exponential backoff if you hit rate limits.

---

## Support

- **Email**: support@tigerwallet.com
- **Discord**: https://discord.gg/tigerwallet
- **Status**: https://status.tigerwallet.com
