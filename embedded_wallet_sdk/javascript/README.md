# TigerWallet Embedded Wallet SDK

Production-ready SDK for dApp developers to integrate TigerWallet functionality directly into their applications.

## Features

- 🚀 **Easy Integration** - Simple API to add wallet functionality
- 🔐 **Secure** - MPC-based key management
- 🌐 **Multi-chain** - Support for 50+ blockchains
- 💳 **Smart Accounts** - Account abstraction with social login
- ⚡ **Gasless Transactions** - Paymaster integration
- 🎨 **Customizable** - White-label support

## Installation

```bash
npm install @tigerwallet/embedded-sdk
```

## Quick Start

```javascript
import { TigerWalletEmbedded } from '@tigerwallet/embedded-sdk';

// Initialize
const wallet = new TigerWalletEmbedded({
  apiKey: 'YOUR_API_KEY',
  chainId: 1, // Ethereum
  theme: 'dark' // or 'light'
});

// Login with email (Social Login)
const user = await wallet.loginWithEmail('user@example.com');

// Or connect existing wallet
await wallet.connect();

// Get user info
const address = wallet.getAddress();
const balance = await wallet.getBalance();

// Send transaction
const tx = await wallet.sendTransaction({
  to: '0x...',
  value: '0.1',
  token: 'ETH'
});

// Sign message
const signature = await wallet.signMessage('Hello World');
```

## API Reference

### Initialization

```javascript
const wallet = new TigerWalletEmbedded({
  // Required
  apiKey: string,
  
  // Optional
  chainId?: number,        // Default: 1 (Ethereum)
  theme?: 'light' | 'dark' | 'auto',
  
  // Features
  enableSocialLogin?: boolean,
  enableGasless?: boolean,
  enableSmartAccounts?: boolean,
  
  // Callbacks
  onConnect?: () => void,
  onDisconnect?: () => void,
  onAccountChange?: (address: string) => void,
  onChainChange?: (chainId: number) => void,
});
```

### Authentication

```javascript
// Email login (Social Login)
const user = await wallet.loginWithEmail(email, verificationCode);

// Phone login
const user = await wallet.loginWithPhone(phone, verificationCode);

// Social login (Google, Apple, etc.)
const user = await wallet.loginWithSocial('google');

// Connect existing wallet (MetaMask, etc.)
await wallet.connect(walletType); // 'metamask', 'coinbase', 'walletconnect'

// Connect hardware wallet
await wallet.connectHardware('ledger' | 'trezor');
```

### Wallet Operations

```javascript
// Get current address
const address = wallet.getAddress();

// Get balance
const balance = await wallet.getBalance(token?);

// Get all token balances
const tokens = await wallet.getTokenBalances();

// Get network
const chainId = wallet.getChainId();

// Switch network
await wallet.switchChain(chainId);

// Sign transaction
const txHash = await wallet.sendTransaction({
  to: string,
  value?: string,
  data?: string,
  token?: string,  // For ERC-20 transfers
});

// Sign message
const signature = await wallet.signMessage(message);
const signature = await wallet.signTypedData(typedData);

// Disconnect
await wallet.disconnect();
```

### Smart Accounts (Account Abstraction)

```javascript
// Enable smart account
await wallet.enableSmartAccount();

// Get smart account address
const smartAddress = wallet.getSmartAccountAddress();

// Deploy smart account (if not deployed)
await wallet.deploySmartAccount();

// Execute with gasless transaction
await wallet.sendUserOperation({
  to: '0x...',
  data: '0x...',
  paymaster: 'auto' // Use paymaster for gasless
});

// Session keys - allow specific dApps
await wallet.createSessionKey({
  dappAddress: '0x...',
  permissions: ['transfer', 'swap'],
  expiresIn: 86400 // 24 hours
});
```

### DeFi Integration

```javascript
// Swap tokens
const swapResult = await wallet.swap({
  fromToken: 'ETH',
  toToken: 'USDC',
  amount: '1',
  slippage: 0.5 // 0.5%
});

// Bridge cross-chain
const bridgeResult = await wallet.bridge({
  fromChain: 1,      // Ethereum
  toChain: 42161,   // Arbitrum
  token: 'ETH',
  amount: '1'
});

// Stake
const stakeResult = await wallet.stake({
  token: 'ETH',
  amount: '1',
  validator: 'lido' // or custom validator
});
```

### Events

```javascript
wallet.on('connect', () => {
  console.log('Wallet connected');
});

wallet.on('disconnect', () => {
  console.log('Wallet disconnected');
});

wallet.on('accountChanged', (address) => {
  console.log('Account changed:', address);
});

wallet.on('chainChanged', (chainId) => {
  console.log('Chain changed:', chainId);
});

wallet.on('message', (message) => {
  console.log('Message:', message);
});

// Remove listener
wallet.off('connect', callback);
```

### Configuration Options

```javascript
const wallet = new TigerWalletEmbedded({
  apiKey: 'YOUR_API_KEY',
  
  // UI Configuration
  ui: {
    theme: 'dark',
    accentColor: '#FF6B35',
    fontFamily: 'Inter, sans-serif',
    borderRadius: 8,
    showPendingTransactions: true,
  },
  
  // Features
  features: {
    socialLogin: true,
    gasless: true,
    smartAccounts: true,
    nftViewing: true,
    staking: true,
    defi: true,
  },
  
  // Networks
  networks: {
    default: [1, 56, 137, 42161], // ETH, BSC, Polygon, Arbitrum
    supported: [1, 56, 137, 42161, 10, 8453, 43114],
  },
  
  // Callbacks
  callbacks: {
    onLogin: (user) => console.log('Logged in:', user),
    onLogout: () => console.log('Logged out'),
    onError: (error) => console.error('Error:', error),
  }
});
```

## React Integration

```jsx
import { TigerWalletProvider, useTigerWallet } from '@tigerwallet/embedded-sdk-react';

function App() {
  return (
    <TigerWalletProvider apiKey="YOUR_API_KEY">
      <YourApp />
    </TigerWalletProvider>
  );
}

function ConnectButton() {
  const { connect, disconnect, address, isConnected } = useTigerWallet();
  
  return isConnected ? (
    <button onClick={disconnect}>
      Disconnect ({address.slice(0, 6)}...{address.slice(-4)})
    </button>
  ) : (
    <button onClick={connect}>Connect Wallet</button>
  );
}
```

## TypeScript Support

The SDK is written in TypeScript and provides full type definitions:

```typescript
import type { 
  WalletConfig,
  User,
  Transaction,
  TokenBalance,
  SwapParams,
  BridgeParams,
  StakeParams,
} from '@tigerwallet/embedded-sdk';
```

## Error Handling

```javascript
try {
  const tx = await wallet.sendTransaction({...});
} catch (error) {
  switch (error.code) {
    case 'USER_REJECTED':
      // User cancelled the transaction
      break;
    case 'INSUFFICIENT_FUNDS':
      // Not enough balance
      break;
    case 'NETWORK_ERROR':
      // Network issue
      break;
    default:
      // Unknown error
  }
}
```

## Security Best Practices

1. **Never expose API keys** in client-side code
2. **Use environment variables** for sensitive configuration
3. **Implement proper error handling**
4. **Validate user inputs**
5. **Use gasless transactions** for better UX (when available)

## Support

- 📧 Email: support@tigerwallet.com
- 💬 Discord: https://discord.gg/tigerwallet
- 📖 Docs: https://docs.tigerwallet.com

## License

MIT License - see LICENSE file for details
