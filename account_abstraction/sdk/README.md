# TigerWallet Account Abstraction SDK

## Overview
This SDK provides complete Account Abstraction support for TigerWallet, implementing:
- ERC-4337 Smart Account
- Paymaster System
- Session Keys
- Batched Transactions
- Gas Abstraction

## Features

### 1. Smart Account (ERC-4337)
- Native account abstraction without deploy
- Social recovery
- Multi-owner support
- Token payment for gas

### 2. Paymaster System
- Gasless transactions (sponsored txs)
- Token-based gas payment
- Custom paymaster logic
- Whitelisting

### 3. Session Keys
- Temporary key generation
- dApp-specific permissions
- Time-limited access
- Spending limits

### 4. Batched Transactions
- Multiple operations in one tx
- Atomic execution
- Conditional execution

## Installation

```bash
npm install @tigerwallet/account-abstraction
```

## Usage

### Initialize Smart Account

```typescript
import { SmartAccount, Bundler, Paymaster } from '@tigerwallet/account-abstraction';

const smartAccount = await SmartAccount.init({
  entryPointAddress: '0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3',
  bundlerUrl: 'https://bundler.tigerwallet.com',
  paymasterUrl: 'https://paymaster.tigerwallet.com',
  chainId: 1,
});

// Get account address
const accountAddress = await smartAccount.getAccountAddress();
console.log('Smart Account:', accountAddress);
```

### Send Gasless Transaction

```typescript
const tx = await smartAccount.sendTransaction({
  to: '0x742d35Cc6634C0532925a3b844Bc9e7595f1234',
  value: ethers.utils.parseEther('0.01'),
  data: '0x',
});

// Pay with ERC-20 tokens instead of native ETH
const txWithTokenGas = await smartAccount.sendTransaction({
  to: '0x...',
  value: ethers.utils.parseEther('0.01'),
  gasToken: '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48', // USDC
}, {
  paymaster: true,
});
```

### Create Session Key

```typescript
const sessionKey = await smartAccount.createSessionKey({
  dAppAddress: '0x...',
  validUntil: Date.now() + 86400000, // 24 hours
  allowedCalls: [
    {
      to: '0x...', // Uniswap
      selector: '0x7ff36ab4', // swapExactETHForTokens
    },
  ],
  spendingLimit: ethers.utils.parseEther('1'),
});
```

### Batched Transactions

```typescript
const batch = await smartAccount.executeBatch([
  {
    to: '0x...',
    data: '0x...',
    value: ethers.utils.parseEther('0.1'),
  },
  {
    to: '0x...',
    data: '0x...',
    value: 0,
  },
]);
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Account Abstraction SDK                   │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │   dApp       │───▶│   Bundler    │───▶│   EntryPoint │  │
│  │              │    │              │    │  Contract    │  │
│  └──────────────┘    └──────────────┘    └──────────────┘  │
│         │                    │                    │          │
│         │                    │                    │          │
│         ▼                    ▼                    ▼          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                  Smart Account                        │   │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐   │   │
│  │  │  Execute   │  │   Session  │  │   Fallback │   │   │
│  │  │   Handler  │  │    Keys    │  │   Handler  │   │   │
│  │  └────────────┘  └────────────┘  └────────────┘   │   │
│  │                                                      │   │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐   │   │
│  │  │   Social   │  │   Multi    │  │    Valid   │   │   │
│  │  │  Recovery  │  │    Owner   │  │    ators   │   │   │
│  │  └────────────┘  └────────────┘  └────────────┘   │   │
│  └──────────────────────────────────────────────────────┘   │
│                              │                              │
│                              ▼                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                    Paymaster                          │   │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐   │   │
│  │  │   Gasless  │  │    Token   │  │  Whitelist │   │   │
│  │  │   Sponsor  │  │    Pay     │  │  Manager   │   │   │
│  │  └────────────┘  └────────────┘  └────────────┘   │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Smart Contract Addresses

| Network | EntryPoint | SimpleAccountFactory |
|---------|------------|---------------------|
| Ethereum Mainnet | 0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3 | 0x9406Cc6185a346906296840746125a0E449c54F |
| Polygon | 0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3 | 0x9406Cc6185a346906296840746125a0E449c54F |
| Arbitrum | 0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3 | 0x9406Cc6185a346906296840746125a0E449c54F |
| Optimism | 0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3 | 0x9406Cc6185a346906296840746125a0E449c54F |

## API Reference

### SmartAccount

```typescript
class SmartAccount {
  // Initialize account
  static async init(config: AccountConfig): Promise<SmartAccount>;
  
  // Get canonical account address
  getAccountAddress(): Promise<string>;
  
  // Get nonce
  getNonce(): Promise<bigint>;
  
  // Execute single transaction
  sendTransaction(tx: Transaction, options?: SendOptions): Promise<string>;
  
  // Execute batch
  executeBatch(txs: Transaction[]): Promise<string>;
  
  // Create session key
  createSessionKey(config: SessionKeyConfig): Promise<SessionKey>;
  
  // Add owner
  addOwner(owner: string, threshold?: number): Promise<string>;
  
  // Remove owner
  removeOwner(owner: string): Promise<string>;
  
  // Social recovery
  recoverAccount(newOwner: string, guardians: string[]): Promise<string>;
  
  // Sign user operation
  signUserOp(userOp: UserOperation): Promise<string>;
}
```

### Paymaster

```typescript
class Paymaster {
  // Sponsor gas for user operation
  async sponsorUserOp(userOp: UserOperation): Promise<PaymasterData>;
  
  // Set gas token for payment
  async setGasToken(token: string): Promise<void>;
  
  // Get paymaster balance
  async getBalance(): Promise<bigint>;
  
  // Whitelist dApp
  async whitelistDApp(dApp: string): Promise<void>;
}
```

### SessionKey

```typescript
class SessionKeyManager {
  // Create session key
  createSessionKey(config: SessionKeyConfig): Promise<SessionKey>;
  
  // Remove session key
  removeSessionKey(key: string): Promise<void>;
  
  // Get all session keys
  getSessionKeys(): Promise<SessionKey[]>;
  
  // Update spending limit
  updateSpendingLimit(key: string, limit: bigint): Promise<void>;
}
```

## Examples

### Social Recovery

```typescript
// Setup guardians during account creation
const account = await SmartAccount.init({
  guardians: [
    '0xguardian1...',
    '0xguardian2...',
    '0xguardian3...',
  ],
  guardianThreshold: 2, // Need 2 of 3 to recover
});

// Recover account (called by guardians)
await account.recoverAccount(newOwner, ['0xguardian1...', '0xguardian2...']);
```

### Multi-Sig Account

```typescript
// Create multi-sig account
const multiSig = await SmartAccount.init({
  owners: [
    '0xowner1...',
    '0xowner2...',
    '0xowner3...',
  ],
  threshold: 2, // Need 2 of 3 signatures
});
```

## Security Considerations

1. **Key Management**: Private keys should be stored securely
2. **Guardian Security**: Use hardware wallets for guardians
3. **Paymaster Limits**: Set reasonable limits on sponsored txs
4. **Session Keys**: Set appropriate expiration and limits
5. **Validation**: Always validate user operations on-chain

## License

MIT
