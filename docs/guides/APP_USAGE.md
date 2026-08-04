# TigerWallet Apps - Complete Usage Guide

## Table of Contents
1. [Available Apps](#available-apps)
2. [Admin Platform](#admin-platform)
3. [User App](#user-app)
4. [Web App](#web-app)
5. [Desktop App](#desktop-app)
6. [Browser Extension](#browser-extension)
7. [API Usage](#api-usage)

---

## Available Apps

### Supported Platforms

| App | Platform | Technology | Status |
|-----|----------|------------|--------|
| Admin Platform | Web | React + TypeScript | ✅ Complete |
| User App (Android) | Mobile | React Native | ✅ Complete |
| User App (iOS) | Mobile | React Native | ✅ Complete |
| Web Wallet | Web | React + TypeScript | ✅ Complete |
| Desktop App | Desktop | Electron | ✅ Complete |
| Browser Extension | Extension | React | ✅ Complete |

---

## Admin Platform

### Access
```
URL: https://admin.yourdomain.com
Port: 3000 (development)
```

### Login
1. Navigate to admin portal
2. Enter email and password
3. Complete 2FA if enabled
4. Access dashboard

### Features

#### Dashboard
- Revenue overview
- User statistics
- Transaction volume
- KYC pending count

#### User Management
```bash
# API Endpoints
GET    /api/v1/admin/users          # List all users
POST   /api/v1/admin/users          # Create user
GET    /api/v1/admin/users/:id      # Get user details
PUT    /api/v1/admin/users/:id      # Update user
DELETE /api/v1/admin/users/:id     # Delete user
POST   /api/v1/admin/users/:id/suspend   # Suspend user
POST   /api/v1/admin/users/:id/activate  # Activate user
```

#### KYC Management
```bash
# API Endpoints
GET    /api/v1/admin/kyc                    # List KYC applications
GET    /api/v1/admin/kyc/:id                # Get KYC details
POST   /api/v1/admin/kyc/:id/approve        # Approve KYC
POST   /api/v1/admin/kyc/:id/reject         # Reject KYC
POST   /api/v1/admin/kyc/:id/request-info    # Request more info
```

#### White Label Management
```bash
# API Endpoints
GET    /api/v1/admin/white-labels              # List white labels
POST   /api/v1/admin/white-labels              # Create white label
GET    /api/v1/admin/white-labels/:id          # Get white label
PUT    /api/v1/admin/white-labels/:id          # Update white label
POST   /api/v1/admin/white-labels/:id/approve  # Approve white label
POST   /api/v1/admin/white-labels/:id/suspend  # Suspend white label
DELETE /api/v1/admin/white-labels/:id           # Delete white label
```

#### Analytics
```bash
# API Endpoints
GET    /api/v1/admin/analytics/revenue         # Revenue analytics
GET    /api/v1/admin/analytics/users           # User analytics
GET    /api/v1/admin/analytics/transactions   # Transaction analytics
GET    /api/v1/admin/analytics/kyc            # KYC analytics
```

---

## User App

### Installation

#### Android
```bash
# Using Expo
cd user_app/react
npx expo install
npx expo run:android

# Or build APK
npx expo build:android -t apk
```

#### iOS
```bash
# Using Expo
cd user_app/react
npx expo install
npx expo run:ios

# Or build IPA
npx expo build:ios
```

### Features

#### Wallet Features
- Create new wallet
- Import existing wallet
- View balance
- Send transactions
- Receive transactions
- Swap tokens
- Stake tokens
- NFT management
- DeFi integration

#### Security Features
- Biometric login (fingerprint/face)
- PIN code
- Two-factor authentication
- Hardware wallet support (Ledger/Trezor)

#### Transaction Types
```bash
# Supported transactions
- Native token transfer
- ERC-20 token transfer
- NFT transfer (ERC-721/ERC-1155)
- Token swap
- Token stake
- Token unstake
- Bridge transfer
```

---

## Web App

### Access
```
URL: https://wallet.yourdomain.com
```

### Features

#### Authentication
```javascript
// Login with email
const response = await fetch('/api/v1/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    email: 'user@example.com',
    password: 'password'
  })
});

// Login with wallet
const signer = provider.getSigner();
const signature = await signer.signMessage('Login to TigerWallet');
```

#### Wallet Operations
```javascript
// Get balance
const balance = await wallet.getBalance('eth');

// Send transaction
const tx = await wallet.sendTransaction({
  to: '0x...',
  value: ethers.utils.parseEther('0.1')
});

// Swap tokens
const swap = await wallet.swap({
  fromToken: 'ETH',
  toToken: 'USDT',
  amount: '0.1'
});
```

---

## Desktop App

### Installation

#### Windows
```bash
# Build executable
cd user_app/desktop
npm run build:win

# Output in dist/
```

#### macOS
```bash
# Build executable
cd user_app/desktop
npm run build:mac

# Output in dist/
```

#### Linux
```bash
# Build executable
cd user_app/desktop
npm run build:linux

# Output in dist/
```

### Features
- Full wallet functionality
- System tray support
- Desktop notifications
- Auto-updates

---

## Browser Extension

### Installation

#### Chrome
1. Open `chrome://extensions/`
2. Enable "Developer mode"
3. Click "Load unpacked"
4. Select `user_app/extension/dist` folder

#### Firefox
1. Open `about:debugging`
2. Click "This Firefox"
3. Click "Load Temporary Add-on"
4. Select any file in extension folder

### Usage
```javascript
// Injected into web pages
if (window.tigerWallet) {
  // Request account
  const accounts = await window.tigerWallet.request({ method: 'eth_requestAccounts' });
  
  // Sign transaction
  const tx = await window.tigerWallet.sendTransaction({
    from: accounts[0],
    to: '0x...',
    value: '0x...'
  });
}
```

---

## API Usage

### Authentication
```bash
# Get JWT token
curl -X POST https://api.tigerwallet.io/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"password"}'

# Use token
curl -X GET https://api.tigerwallet.io/v1/users \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### White Label API
```bash
# Create white label
curl -X POST https://api.tigerwallet.io/v1/white-labels \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"My Wallet",
    "domain":"wallet.mydomain.com",
    "branding":{
      "primaryColor":"#000000",
      "secondaryColor":"#FFFFFF"
    }
  }'
```

### Transaction API
```bash
# Create transaction
curl -X POST https://api.tigerwallet.io/v1/transactions \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "to":"0x...",
    "amount":"0.1",
    "chain":"ethereum"
  }'
```

---

## SDK Integration

### JavaScript SDK
```javascript
import { TigerWallet } from '@tigerwallet/sdk';

const wallet = new TigerWallet({
  apiKey: 'YOUR_API_KEY',
  network: 'ethereum'
});

// Connect wallet
await wallet.connect();

// Get balance
const balance = await wallet.getBalance();

// Send transaction
const tx = await wallet.send({
  to: '0x...',
  value: '0.1'
});
```

### REST API Reference

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/auth/login` | POST | User login |
| `/v1/auth/register` | POST | User registration |
| `/v1/auth/verify-2fa` | POST | Verify 2FA |
| `/v1/wallet/balance` | GET | Get balance |
| `/v1/wallet/transactions` | GET | List transactions |
| `/v1/wallet/send` | POST | Send transaction |
| `/v1/wallet/swap` | POST | Swap tokens |
| `/v1/kyc/submit` | POST | Submit KYC |
| `/v1/white-labels` | GET | List white labels |
| `/v1/admin/users` | GET | Admin: List users |

---

## Troubleshooting

### Common Issues

#### 1. Login Failed
- Check credentials
- Verify email verification
- Check 2FA code
- Clear browser cache

#### 2. Transaction Failed
- Check network connection
- Verify sufficient balance
- Check gas price
- Verify recipient address

#### 3. Extension Not Working
- Refresh the page
- Check extension permissions
- Reinstall extension

---

## Support

- Documentation: https://docs.tigerwallet.io
- GitHub: https://github.com/meghlabd275-byte/TigerWallet
- Email: support@tigerwallet.io
