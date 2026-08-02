# TigerWallet Paymaster SDK

## Overview

The Paymaster SDK enables developers to sponsor gas fees for their users, creating gasless transaction experiences. This is essential for user onboarding and dApp engagement.

## Features

- **Gasless Transactions**: Sponsor user transactions without them paying gas
- **Token Payment**: Accept ERC-20 tokens as payment for gas
- **Whitelist Management**: Control which dApps/users get sponsored
- **Rate Limiting**: Prevent abuse with configurable limits
- **Analytics**: Track sponsored transaction metrics

## Quick Start

```typescript
import { Paymaster, PaymasterConfig } from '@tigerwallet/paymaster-sdk';

const paymaster = new Paymaster({
  entryPointAddress: '0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3',
  paymasterAddress: '0xPaymasterAddress...',
  privateKey: process.env.PAYMASTER_PRIVATE_KEY,
});

// Sponsor a user operation
const paymasterData = await paymaster.sponsorUserOp({
  sender: '0xUserAddress...',
  nonce: '0x0',
  initCode: '0x',
  callData: '0x...',
  callGasLimit: '21000',
  verificationGasLimit: '100000',
  preVerificationGas: '21000',
  maxFeePerGas: '1000000000',
  maxPriorityFeePerGas: '1000000000',
  signature: '0x',
});
```

## Installation

```bash
npm install @tigerwallet/paymaster-sdk
```

## Usage Examples

### 1. Basic Gasless Transaction

```typescript
import { Paymaster, UserOperation } from '@tigerwallet/paymaster-sdk';

const paymaster = await Paymaster.init({
  // ... config
});

// For a user's UserOperation
const userOp = {
  sender: '0x742d35Cc6634C0532925a3b844Bc9e7595f1234',
  nonce: '0x1',
  initCode: '0x',
  callData: '0x',
  callGasLimit: '0x5208', // 21000
  verificationGasLimit: '0x186A0', // 100000
  preVerificationGas: '0x5208',
  maxFeePerGas: '0x3B9ACA00', // 1 gwei
  maxPriorityFeePerGas: '0x3B9ACA00',
  signature: '0x',
};

// Get paymaster data to include in userOp
const paymasterData = await paymaster.sponsorUserOp(userOp);

console.log('Paymaster Data:', paymasterData.paymasterAndData);
// Include this in your userOp before submitting to bundler
```

### 2. Token Payment for Gas

```typescript
// Accept USDC as payment for gas
const tokenPayment = await paymaster.setPaymentToken({
  tokenAddress: '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48', // USDC
  exchangeRate: '1000000', // 1 USDC = 1e6 gas units equivalent
  decimals: 6,
});

// Now users can pay with USDC
const userOpWithToken = await paymaster.sponsorUserOp(userOp, {
  gasToken: '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48',
});
```

### 3. Whitelist Management

```typescript
// Whitelist a dApp
await paymaster.whitelistDApp({
  dAppAddress: '0xDappAddress...',
  sponsorLimit: '1000000000000000000', // 1 ETH max
  expiry: Math.floor(Date.now() / 1000) + 86400 * 30, // 30 days
});

// Check whitelist status
const status = await paymaster.getWhitelistStatus('0xDappAddress...');
console.log('Whitelist status:', status);
```

### 4. Rate Limiting

```typescript
// Configure rate limits
await paymaster.setRateLimit({
  maxPerMinute: 100,
  maxPerHour: 1000,
  maxPerDay: 10000,
  perUserPerMinute: 10,
});

// Check current limits
const limits = await paymaster.getRateLimits();
console.log('Rate limits:', limits);
```

### 5. Analytics

```typescript
// Get sponsored transaction stats
const stats = await paymaster.getStats({
  startDate: '2024-01-01',
  endDate: '2024-01-31',
  groupBy: 'day',
});

console.log('Total sponsored:', stats.totalSponsored);
console.log('Total gas used:', stats.totalGasUsed);
console.log('Unique users:', stats.uniqueUsers);
```

## API Reference

### Paymaster Class

```typescript
class Paymaster {
  // Initialize paymaster
  static async init(config: PaymasterConfig): Promise<Paymaster>;

  // Sponsor a user operation
  async sponsorUserOp(
    userOp: UserOperation,
    options?: SponsorOptions
  ): Promise<PaymasterData>;

  // Set payment token
  async setPaymentToken(config: PaymentTokenConfig): Promise<void>;

  // Whitelist dApp
  async whitelistDApp(config: WhitelistConfig): Promise<void>;

  // Get whitelist status
  async getWhitelistStatus(address: string): Promise<WhitelistStatus>;

  // Set rate limits
  async setRateLimit(config: RateLimitConfig): Promise<void>;

  // Get rate limits
  async getRateLimits(): Promise<RateLimitConfig>;

  // Get statistics
  async getStats(config: StatsConfig): Promise<Stats>;

  // Withdraw funds
  async withdraw(amount: string, recipient: string): Promise<string>;
}
```

### Types

```typescript
interface PaymasterConfig {
  entryPointAddress: string;
  paymasterAddress: string;
  privateKey: string;
  rpcUrl: string;
  chainId: number;
}

interface UserOperation {
  sender: string;
  nonce: string;
  initCode: string;
  callData: string;
  callGasLimit: string;
  verificationGasLimit: string;
  preVerificationGas: string;
  maxFeePerGas: string;
  maxPriorityFeePerGas: string;
  signature: string;
}

interface SponsorOptions {
  gasToken?: string;
  forceNonZero?: boolean;
}

interface PaymasterData {
  paymasterAndData: string;
  preVerificationGas?: string;
  verificationGasLimit?: string;
  callGasLimit?: string;
}
```

## Smart Contract

The paymaster smart contract handles:
1. Validation of sponsorship requests
2. Gas accounting
3. Token payment settlement
4. Whitelist checking
5. Rate limiting

### Deployment

```bash
# Deploy paymaster contract
npx hardhat run scripts/deploy-paymaster.ts --network mainnet
```

### Contract Addresses

| Network | Address |
|---------|---------|
| Ethereum Mainnet | 0xPaymasterAddress... |
| Polygon | 0xPaymasterAddress... |
| Arbitrum | 0xPaymasterAddress... |
| Optimism | 0xPaymasterAddress... |
| Base | 0xPaymasterAddress... |

## Security

- **Signature Verification**: All sponsorship requests are cryptographically signed
- **Reentrancy Protection**: Contract uses nonReentrant modifiers
- **Access Control**: Only owner can whitelist and configure
- **Event Monitoring**: All key actions emit events for monitoring

## Gas Estimation

The SDK automatically estimates gas for sponsored transactions:

```typescript
const estimate = await paymaster.estimateGas(userOp);
console.log('Estimated gas:', estimate.totalGas);
console.log('Sponsor coverage:', estimate.sponsoredAmount);
```

## Best Practices

1. **Set Appropriate Limits**: Don't sponsor unlimited transactions
2. **Monitor Usage**: Track sponsored transactions regularly
3. **Use Token Payment**: Accept tokens to offset costs
4. **Implement Fallback**: Have backup if paymaster is down
5. **Validate Users**: Check user eligibility before sponsoring
