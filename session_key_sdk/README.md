# TigerWallet Session Key SDK

## Overview

The Session Key SDK enables temporary, permissioned access for dApps without compromising the user's main wallet. Keys can be scoped to specific functions, contracts, and spending limits.

## Features

- **Scoped Permissions**: Limit keys to specific dApps and functions
- **Time-Limited Access**: Auto-expiring session keys
- **Spending Limits**: Cap on total value transferable
- **Revocable**: Instantly revoke compromised keys
- **Multi-Chain**: Works across EVM chains

## Quick Start

```typescript
import { SessionKeyManager, createSessionKey } from '@tigerwallet/session-key-sdk';

const sessionManager = new SessionKeyManager({
  accountAddress: '0xUserSmartAccount...',
  signer: userSigner,
});

// Create a session key for a dApp
const sessionKey = await sessionManager.createSessionKey({
  dAppAddress: '0xDappContract...',
  validUntil: Date.now() + 86400000, // 24 hours
  allowedContracts: ['0xUniswapRouter...'],
  allowedSelectors: ['0x7ff36ab4', '0x18cbafe5'], // swap functions
  spendingLimit: '1000000000000000000', // 1 ETH max
});

// Sign the session key permission
await sessionManager.signSessionKey(sessionKey);
```

## Installation

```bash
npm install @tigerwallet/session-key-sdk
```

## Usage Examples

### 1. dApp Login with Session Key

```typescript
// User logs into dApp
const { sessionKey, sessionKeyAddress } = await sessionManager.createSessionKey({
  dAppAddress: dAppContract.address,
  validUntil: Date.now() + 7 * 24 * 60 * 60 * 1000, // 7 days
  allowedContracts: [uniswapRouter.address],
  allowedSelectors: [
    '0x7ff36ab4', // swapExactETHForTokens
    '0x18cbafe5', // swapExactTokensForETH
  ],
  spendingLimit: ethers.utils.parseEther('1').toString(),
});

// dApp stores session key for later use
await dAppContract.setSessionKey(sessionKeyAddress);
```

### 2. Execute Transactions with Session Key

```typescript
// dApp uses session key to execute
const tx = await sessionManager.executeWithSessionKey({
  to: uniswapRouter.address,
  data: swapData,
  sessionKey: sessionKeyAddress,
});
```

### 3. Multi-Contract Permissions

```typescript
// Allow interaction with multiple DeFi protocols
const multiProtocolKey = await sessionManager.createSessionKey({
  dAppAddress: aggregatorContract.address,
  validUntil: Date.now() + 30 * 24 * 60 * 60 * 1000, // 30 days
  allowedContracts: [
    uniswapRouter.address,
    aavePool.address,
    curve.address,
  ],
  allowedSelectors: [
    // Uniswap
    '0x7ff36ab4', // swapExactETHForTokens
    '0x18cbafe5', // swapExactTokensForETH
    // Aave
    '0x8b3a356d', // supply
    '0x573ade81', // borrow
    // Curve
    '0x5b60d6c4', // exchange
  ],
  spendingLimit: ethers.utils.parseEther('10').toString(), // 10 ETH total
});
```

### 4. Revoke Session Key

```typescript
// User revokes compromised key
await sessionManager.revokeSessionKey(sessionKeyAddress);

// Or revoke all keys for a dApp
await sessionManager.revokeAllForDApp(dappAddress);
```

### 5. View Active Session Keys

```typescript
// Get all active session keys
const keys = await sessionManager.getActiveSessionKeys();

console.log('Active keys:', keys.map(k => ({
  address: k.keyAddress,
  dApp: k.dAppAddress,
  expires: new Date(k.validUntil).toISOString(),
  spent: k.spentAmount,
  limit: k.spendingLimit,
})));
```

## API Reference

### SessionKeyManager

```typescript
class SessionKeyManager {
  constructor(config: SessionKeyManagerConfig);

  // Create new session key
  async createSessionKey(config: CreateSessionKeyConfig): Promise<SessionKey>;

  // Sign session key for use
  async signSessionKey(sessionKey: SessionKey): Promise<string>;

  // Execute transaction with session key
  async executeWithSessionKey(tx: SessionTransaction): Promise<string>;

  // Revoke session key
  async revokeSessionKey(keyAddress: string): Promise<void>;

  // Revoke all keys for dApp
  async revokeAllForDApp(dAppAddress: string): Promise<void>;

  // Get all active keys
  async getActiveSessionKeys(): Promise<ActiveSessionKey[]>;

  // Get key details
  async getSessionKeyDetails(keyAddress: string): Promise<SessionKeyDetails>;
}
```

### Types

```typescript
interface SessionKeyManagerConfig {
  accountAddress: string;
  signer: ethers.Signer;
  entryPointAddress?: string;
  chainId?: number;
}

interface CreateSessionKeyConfig {
  dAppAddress: string;
  validUntil: number; // timestamp
  validAfter?: number; // timestamp
  allowedContracts: string[];
  allowedSelectors?: string[];
  spendingLimit: string;
  nativeTokenLimit?: string;
}

interface SessionKey {
  keyAddress: string;
  dAppAddress: string;
  validUntil: number;
  validAfter: number;
  allowedContracts: string[];
  allowedSelectors: string[];
  spendingLimit: string;
  nativeTokenLimit: string;
  salt: string;
}

interface SessionTransaction {
  to: string;
  data: string;
  value?: string;
  sessionKey: string;
}

interface ActiveSessionKey {
  keyAddress: string;
  dAppAddress: string;
  validUntil: number;
  spentAmount: string;
  spendingLimit: string;
  isRevoked: boolean;
}
```

## How It Works

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Session Key Flow                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. CREATE                                                      │
│     ┌──────────┐     ┌──────────────┐     ┌─────────────────┐  │
│     │  User   │────▶│ SessionKey  │────▶│  Smart Account │  │
│     │ Wallet  │     │   Manager   │     │   Contract     │  │
│     └──────────┘     └──────────────┘     └─────────────────┘  │
│                              │                       │          │
│                              ▼                       ▼          │
│                        ┌──────────┐           ┌──────────┐     │
│                        │ Session  │           │ Permission│     │
│                        │   Key    │           │  Storage │     │
│                        └──────────┘           └──────────┘     │
│                                                                  │
│  2. USE                                                         │
│     ┌──────────┐     ┌──────────────┐     ┌─────────────────┐  │
│     │  dApp   │────▶│  Execute     │────▶│  Validate       │  │
│     │         │     │  with Key    │     │  Permission     │  │
│     └──────────┘     └──────────────┘     └─────────────────┘  │
│                                                           │     │
│                                                           ▼     │
│                                                   ┌──────────┐ │
│                                                   │  Execute │ │
│                                                   │  TX      │ │
│                                                   └──────────┘ │
│                                                                  │
│  3. REVOKE                                                      │
│     ┌──────────┐     ┌──────────────┐     ┌─────────────────┐  │
│     │  User   │────▶│  Revoke      │────▶│  Delete         │  │
│     │         │     │  Key         │     │  Permission     │  │
│     └──────────┘     └──────────────┘     └─────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Permission Structure

```solidity
struct Permission {
    address key;           // Session key address
    address dApp;         // dApp that can use this key
    address[] allowedTargets;  // Contracts key can interact with
    bytes4[] allowedMethods;  // Function selectors allowed
    uint256 spendingLimit;    // Max value in wei
    uint256 nativeTokenLimit; // Max native token value
    uint256 validUntil;      // Expiration timestamp
    bool isRevoked;          // Revocation status
}
```

## Security Considerations

1. **Key Rotation**: Create new keys regularly
2. **Spend Limits**: Set conservative limits initially
3. **Time Limits**: Use short expiration for high-value keys
4. **Contract Whitelisting**: Only allow trusted contracts
5. **Monitor Usage**: Track session key activity
6. **Revoke Immediately**: Revoke compromised keys instantly

## Integration with Smart Account

```typescript
// Combined with Smart Account for full AA + Session Keys
import { SmartAccount } from '@tigerwallet/account-abstraction';
import { SessionKeyManager } from '@tigerwallet/session-key-sdk';

const smartAccount = await SmartAccount.init({...});
const sessionManager = new SessionKeyManager({
  accountAddress: smartAccount.address,
  signer: smartAccount.signer,
});

// Create session key
const sessionKey = await sessionManager.createSessionKey({
  dAppAddress: '0xDapp...',
  validUntil: Date.now() + 86400000,
  allowedContracts: ['0xUniswap...'],
  allowedSelectors: ['0x7ff36ab4'],
  spendingLimit: '1000000000000000000',
});

await sessionManager.signSessionKey(sessionKey);

// Now dApp can use session key - gasless!
const tx = await sessionManager.executeWithSessionKey({
  to: '0xUniswap...',
  data: swapData,
  sessionKey: sessionKey.keyAddress,
});
```

## Best Practices

1. **Default Deny**: Start with no permissions, add as needed
2. **Minimal Scope**: Only allow what's absolutely necessary
3. **User Control**: Always let users approve/revoke keys
4. **Clear UI**: Show what each key can do
5. **Expiration**: Use short times for new dApps
6. **Monitor**: Alert on unusual key usage
