# TigerWallet Fetcher API Documentation

## Overview

The Fetcher API allows white label products (MasterWallet, UserWallet, Bots, ProjectParty) to connect to TigerWallet Super Admin for data fetching services.

## Base URL

```
Production: https://fetcher.tigerwallet.com/api/v1
Staging:    https://fetcher-staging.tigerwallet.com/api/v1
Sandbox:    https://fetcher-sandbox.tigerwallet.com/api/v1
```

## Authentication

All API requests require authentication using an API Key and Signature.

### Authentication Headers

| Header | Description | Required |
|--------|-------------|-----------|
| `X-API-Key` | Your API key | Yes |
| `X-Timestamp` | Unix timestamp (seconds) | Yes |
| `X-Signature` | HMAC-SHA256 signature | Yes |
| `X-Tenant-ID` | Your tenant ID | Yes |

### Signature Generation

```python
import hmac
import hashlib
import time

def generate_signature(api_secret: str, method: str, path: str, body: str = "") -> tuple:
    timestamp = int(time.time())
    message = f"{method}\n{path}\n{timestamp}\n{body}"
    signature = hmac.new(
        api_secret.encode(),
        message.encode(),
        hashlib.sha256
    ).hexdigest()
    return timestamp, signature

# Example
timestamp, signature = generate_signature(
    api_secret="your-api-secret",
    method="GET",
    path="/api/v1/fetcher/prices",
    body=""
)
```

### Example Request

```bash
curl -X GET "https://fetcher.tigerwallet.com/api/v1/fetcher/prices?symbols=BTC,ETH" \
  -H "X-API-Key: your-api-key" \
  -H "X-Timestamp: 1699123456" \
  -H "X-Signature: abc123..." \
  -H "X-Tenant-ID: tenant-uuid"
```

---

## Rate Limits

| Plan | Requests/Minute | Requests/Day |
|------|----------------|-------------|
| Free | 60 | 10,000 |
| Basic | 300 | 100,000 |
| Pro | 1,000 | 1,000,000 |
| Enterprise | 10,000 | Unlimited |

---

## Endpoints

### 1. Price Data

#### Get Token Prices

```
GET /fetcher/prices
```

**Query Parameters:**

| Parameter | Type | Description | Required |
|-----------|------|-------------|----------|
| symbols | string | Comma-separated token symbols | Yes |
| currencies | string | Fiat currencies (USD,EUR) | No |

**Response:**

```json
{
  "data": [
    {
      "symbol": "BTC",
      "name": "Bitcoin",
      "price": "45123.45",
      "change_24h": "2.34",
      "change_percent_24h": "5.46",
      "volume_24h": "28500000000",
      "market_cap": "882000000000",
      "high_24h": "45500.00",
      "low_24h": "44200.00",
      "timestamp": 1699123456
    }
  ],
  "cached": false,
  "latency_ms": 45
}
```

#### Get Price History

```
GET /fetcher/prices/{symbol}/history
```

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| interval | string | 1m, 5m, 15m, 1h, 4h, 1d |
| limit | int | Number of data points (max 1000) |
| start_time | int | Unix timestamp |
| end_time | int | Unix timestamp |

**Response:**

```json
{
  "symbol": "BTC",
  "interval": "1h",
  "data": [
    {
      "timestamp": 1699120000,
      "open": "45100.00",
      "high": "45200.00",
      "low": "45050.00",
      "close": "45123.45",
      "volume": "1250000"
    }
  ]
}
```

---

### 2. Blockchain Data

#### Get Blockchain Status

```
GET /fetcher/blockchain/{chain}
```

**Path Parameters:**

| Parameter | Description |
|-----------|-------------|
| chain | Chain name (ethereum, bsc, polygon, arbitrum, solana, etc.) |

**Response:**

```json
{
  "chain": "ethereum",
  "block_number": 18500000,
  "block_hash": "0xabc123...",
  "timestamp": 1699123456,
  "gas_price": "25000000000",
  "network_id": 1,
  "synced": true
}
```

#### Get Transaction Details

```
GET /fetcher/blockchain/{chain}/tx/{tx_hash}
```

**Response:**

```json
{
  "chain": "ethereum",
  "hash": "0xabc123...",
  "from": "0xfrom...",
  "to": "0xto...",
  "value": "1000000000000000000",
  "gas_limit": 21000,
  "gas_used": 21000,
  "gas_price": "25000000000",
  "nonce": 42,
  "status": "confirmed",
  "block_number": 18500000,
  "block_hash": "0xblock...",
  "timestamp": 1699123456,
  "transfers": [
    {
      "from": "0xfrom...",
      "to": "0xto...",
      "token": "ETH",
      "amount": "1.0"
    }
  ]
}
```

---

### 3. Wallet Data

#### Get Wallet Balance

```
GET /fetcher/wallet/{chain}/{address}
```

**Response:**

```json
{
  "chain": "ethereum",
  "address": "0xabc123...",
  "native": {
    "symbol": "ETH",
    "balance": "5.5",
    "balance_usd": "8250.00"
  },
  "tokens": [
    {
      "symbol": "USDT",
      "address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
      "balance": "10000.00",
      "balance_usd": "10000.00",
      "decimals": 6
    }
  ],
  "nsfts": [],
  "timestamp": 1699123456
}
```

#### Get Wallet Transactions

```
GET /fetcher/wallet/{chain}/{address}/transactions
```

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| limit | int | Max results (default 50, max 200) |
| cursor | string | Pagination cursor |
| token | string | Filter by token |

**Response:**

```json
{
  "transactions": [
    {
      "hash": "0xabc123...",
      "from": "0xfrom...",
      "to": "0xto...",
      "value": "1000000000000000000",
      "status": "confirmed",
      "timestamp": 1699123456,
      "transfers": [
        {
          "token": "ETH",
          "from": "0xfrom...",
          "to": "0xto...",
          "amount": "1.0"
        }
      ]
    }
  ],
  "cursor": "abc123...",
  "has_more": true
}
```

---

### 4. Token Data

#### Get Token Info

```
GET /fetcher/token/{chain}/{address}
```

**Response:**

```json
{
  "chain": "ethereum",
  "address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
  "name": "Tether USD",
  "symbol": "USDT",
  "decimals": 6,
  "total_supply": "410000000000000",
  "holders_count": 5000000,
  "transfer_count": 15000000,
  "verified": true,
  "price": "1.00",
  "market_cap": "41000000000"
}
```

#### Get Token Holders

```
GET /fetcher/token/{chain}/{address}/holders
```

**Response:**

```json
{
  "address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
  "total_holders": 5000000,
  "top_holders": [
    {
      "address": "0xabc...",
      "balance": "50000000000",
      "percentage": "12.2"
    }
  ],
  "timestamp": 1699123456
}
```

---

### 5. Market Data

#### Get Order Book

```
GET /fetcher/market/{symbol}/orderbook
```

**Response:**

```json
{
  "symbol": "BTC/USDT",
  "bids": [
    {"price": "45100.00", "quantity": "2.5"},
    {"price": "45090.00", "quantity": "5.0"}
  ],
  "asks": [
    {"price": "45110.00", "quantity": "1.5"},
    {"price": "45120.00", "quantity": "3.0"}
  ],
  "spread": "10.00",
  "spread_percent": "0.022"
}
```

#### Get Recent Trades

```
GET /fetcher/market/{symbol}/trades
```

---

### 6. Network Statistics

#### Get Gas Prices

```
GET /fetcher/network/{chain}/gas
```

**Response:**

```json
{
  "chain": "ethereum",
  "slow": "20000000000",
  "standard": "25000000000",
  "fast": "35000000000",
  "timestamp": 1699123456
}
```

---

## Webhooks

### Webhook Events

Products can subscribe to real-time events via webhooks.

#### Setup Webhook

```
POST /webhooks
```

**Request:**

```json
{
  "url": "https://your-product.com/webhook",
  "events": ["price.alert", "transaction.confirmed", "wallet.activity"],
  "secret": "your-webhook-secret"
}
```

#### Event Types

| Event | Description |
|-------|-------------|
| `price.alert` | Price reaches threshold |
| `transaction.confirmed` | On-chain transaction confirmed |
| `transaction.failed` | Transaction failed |
| `wallet.activity` | Wallet balance changed |
| `token.transfer` | Token transfer detected |

#### Webhook Payload

```json
{
  "event": "price.alert",
  "timestamp": 1699123456,
  "data": {
    "symbol": "BTC",
    "price": "45000.00",
    "condition": "above"
  }
}
```

---

## Error Codes

| Code | Description |
|------|-------------|
| 400 | Bad Request - Invalid parameters |
| 401 | Unauthorized - Invalid API key |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found - Resource not found |
| 429 | Too Many Requests - Rate limit exceeded |
| 500 | Internal Server Error |
| 503 | Service Unavailable |

---

## SDK Integration

### Go SDK

```go
package main

import (
    "github.com/tigerwallet/sdk/go/fetcher"
)

func main() {
    client := fetcher.NewClient(
        fetcher.Config{
            APIKey:    "your-api-key",
            APISecret: "your-api-secret",
            TenantID:  "your-tenant-id",
            BaseURL:   "https://fetcher.tigerwallet.com/api/v1",
        },
    )

    // Get prices
    prices, err := client.GetPrices([]string{"BTC", "ETH"})
    
    // Get wallet balance
    balance, err := client.GetWalletBalance("ethereum", "0xabc...")
    
    // Get transactions
    txs, err := client.GetTransactions("ethereum", "0xabc...", 50)
}
```

---

## Contract Data Structures

### PriceData

```go
type PriceData struct {
    Symbol        string  `json:"symbol"`
    Name          string  `json:"name"`
    Price         string  `json:"price"`
    Change24h     string  `json:"change_24h"`
    ChangePercent24h float64 `json:"change_percent_24h"`
    Volume24h     string  `json:"volume_24h"`
    MarketCap     string  `json:"market_cap"`
    High24h       string  `json:"high_24h"`
    Low24h        string  `json:"low_24h"`
    Timestamp     int64   `json:"timestamp"`
}
```

### WalletBalance

```go
type WalletBalance struct {
    Chain     string         `json:"chain"`
    Address   string         `json:"address"`
    Native    TokenBalance   `json:"native"`
    Tokens    []TokenBalance `json:"tokens"`
    NFTs      []NFTBalance   `json:"nfts"`
    Timestamp int64          `json:"timestamp"`
}
```

### Transaction

```go
type Transaction struct {
    Chain        string         `json:"chain"`
    Hash        string         `json:"hash"`
    From        string         `json:"from"`
    To          string         `json:"to"`
    Value       string         `json:"value"`
    Status      string         `json:"status"`
    BlockNumber uint64         `json:"block_number"`
    Timestamp   int64          `json:"timestamp"`
    Transfers   []TokenTransfer `json:"transfers"`
}
```

---

## Support

- Email: support@tigerwallet.com
- Discord: https://discord.gg/tigerwallet
- API Status: https://status.tigerwallet.com
