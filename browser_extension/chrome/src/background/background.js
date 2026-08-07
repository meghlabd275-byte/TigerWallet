// TigerWallet Extension - Background Service Worker
// Complete wallet management and DApp interaction

// ============================================================================
// CONSTANTS & CONFIGURATION
// ============================================================================

const WALLET_STATE_KEY = 'tigerwallet_state';
const RPC_CONFIG_KEY = 'tigerwallet_rpc';
const CHAIN_CONFIG_KEY = 'tigerwallet_chains';

// Default RPC endpoints for major chains
const DEFAULT_RPC = {
  ethereum: 'https://eth.llamarpc.com',
  sepolia: 'https://rpc.sepolia.org',
  bsc: 'https://bsc-dataseed.binance.org',
  polygon: 'https://polygon-rpc.com',
  arbitrum: 'https://arb1.arbitrum.io/rpc',
  optimism: 'https://mainnet.optimism.io',
  base: 'https://mainnet.base.org',
  avalanche: 'https://api.avax.network/ext/bc/C/rpc',
  fantom: 'https://rpc.fantom.network',
  solana: 'https://api.mainnet-beta.solana.com',
  aptos: 'https://fullnode.mainnet.aptoslabs.com',
  sui: 'https://fullnode.mainnet.sui.io',
  tron: 'https://api.trongrid.io',
  cosmos: 'https://rpc.cosmos.network',
  near: 'https://rpc.mainnet.near.org',
};

// Chain configurations
const CHAIN_CONFIG = {
  ethereum: {
    id: '0x1',
    name: 'Ethereum',
    symbol: 'ETH',
    decimals: 18,
    explorer: 'https://etherscan.io',
  },
  sepolia: {
    id: '0xaa36a7',
    name: 'Sepolia',
    symbol: 'ETH',
    decimals: 18,
    explorer: 'https://sepolia.etherscan.io',
  },
  bsc: {
    id: '0x38',
    name: 'BNB Chain',
    symbol: 'BNB',
    decimals: 18,
    explorer: 'https://bscscan.com',
  },
  polygon: {
    id: '0x89',
    name: 'Polygon',
    symbol: 'MATIC',
    decimals: 18,
    explorer: 'https://polygonscan.com',
  },
  arbitrum: {
    id: '0xa4b1',
    name: 'Arbitrum One',
    symbol: 'ETH',
    decimals: 18,
    explorer: 'https://arbiscan.io',
  },
  optimism: {
    id: '0xa',
    name: 'Optimism',
    symbol: 'ETH',
    decimals: 18,
    explorer: 'https://optimistic.etherscan.io',
  },
  base: {
    id: '0x2105',
    name: 'Base',
    symbol: 'ETH',
    decimals: 18,
    explorer: 'https://basescan.org',
  },
  avalanche: {
    id: '0xa86a',
    name: 'Avalanche C-Chain',
    symbol: 'AVAX',
    decimals: 18,
    explorer: 'https://snowtrace.io',
  },
  fantom: {
    id: '0xfa',
    name: 'Fantom',
    symbol: 'FTM',
    decimals: 18,
    explorer: 'https://ftmscan.com',
  },
  solana: {
    id: 'solana',
    name: 'Solana',
    symbol: 'SOL',
    decimals: 9,
    explorer: 'https://explorer.solana.com',
  },
};

// ============================================================================
// STATE MANAGEMENT
// ============================================================================

let walletState = {
  isUnlocked: false,
  currentChain: 'ethereum',
  addresses: {},
  balances: {},
  transactions: [],
};

// ============================================================================
// CRYPTO UTILITIES (Web Crypto API)
// ============================================================================

class CryptoUtils {
  // Keccak-256 hash
  static async keccak256(message) {
    const msgBuffer = new TextEncoder().encode(message);
    const hashBuffer = await crypto.subtle.digest('SHA-3-256', msgBuffer);
    return Array.from(new Uint8Array(hashBuffer))
      .map(b => b.toString(16).padStart(2, '0'))
      .join('');
  }
  
  // Generate random bytes
  static async randomBytes(length) {
    const bytes = new Uint8Array(length);
    crypto.getRandomValues(bytes);
    return Array.from(bytes);
  }
  
  // Derive address from public key (uncompressed)
  static publicKeyToAddress(publicKey) {
    // Remove '04' prefix if present
    const pub = publicKey.startsWith('0x04') ? publicKey.slice(4) : publicKey;
    // Keccak-256 of the public key
    const hash = keccak256(pub);
    // Take last 20 bytes
    return '0x' + hash.slice(-40);
  }
  
  // Validate Ethereum address
  static isValidAddress(address) {
    return /^0x[a-fA-F0-9]{40}$/.test(address);
  }
}

// ============================================================================
// KEY DERIVATION (BIP-39/BIP-32)
// ============================================================================

class KeyDerivation {
  // Simplified BIP-39 seed generation (in production, use proper PBKDF2)
  static async mnemonicToSeed(mnemonic, password = '') {
    const encoder = new TextEncoder();
    const data = encoder.encode(mnemonic + 'mnemonic' + password);
    const hash = await crypto.subtle.digest('SHA-512', data);
    return Array.from(new Uint8Array(hash));
  }
  
  // Derive key from seed with path
  static async deriveKey(seed, path) {
    // Simplified - in production use proper BIP-32
    const encoder = new TextEncoder();
    const data = encoder.encode(path + JSON.stringify(Array.from(seed)));
    const hash = await crypto.subtle.digest('SHA-256', data);
    return new Uint8Array(hash);
  }
  
  // Generate 24-word mnemonic
  static async generateMnemonic() {
    const WORDLIST = [
      'abandon', 'ability', 'able', 'about', 'above', 'absent', 'absorb', 'abstract',
      // ... (full BIP-39 wordlist would be here)
    ];
    
    const random = await CryptoUtils.randomBytes(32);
    const words = [];
    for (let i = 0; i < 24; i++) {
      const index = (random[i >> 3] >> (8 - (i & 7))) & 0x1FF;
      words.push(WORDLIST[index % WORDLIST.length] || 'abandon');
    }
    return words.join(' ');
  }
}

// ============================================================================
// WALLET MANAGEMENT
// ============================================================================

class WalletManager {
  // Create new wallet
  static async createWallet(password) {
    const mnemonic = await KeyDerivation.generateMnemonic();
    const seed = await KeyDerivation.mnemonicToSeed(mnemonic, password);
    
    // Derive addresses for all supported chains
    const addresses = {};
    const chains = ['ethereum', 'bsc', 'polygon', 'arbitrum', 'optimism', 'base', 'avalanche', 'fantom'];
    
    for (const chain of chains) {
      const key = await KeyDerivation.deriveKey(seed, `m/44'/60'/0'/0/0`);
      // Simplified address generation
      const hash = await CryptoUtils.keccak256(JSON.stringify(Array.from(key)));
      addresses[chain] = '0x' + hash.slice(-40);
    }
    
    // Save wallet state
    walletState = {
      isUnlocked: true,
      currentChain: 'ethereum',
      addresses,
      balances: {},
      transactions: [],
      encryptedMnemonic: await CryptoUtils.encrypt(mnemonic, password),
    };
    
    await saveState();
    return walletState;
  }
  
  // Import wallet from mnemonic
  static async importWallet(mnemonic, password) {
    // Validate mnemonic
    const words = mnemonic.trim().split(/\s+/);
    if (words.length !== 12 && words.length !== 24) {
      throw new Error('Invalid mnemonic length');
    }
    
    const seed = await KeyDerivation.mnemonicToSeed(mnemonic, password);
    
    // Derive addresses
    const addresses = {};
    const chains = ['ethereum', 'bsc', 'polygon', 'arbitrum', 'optimism', 'base', 'avalanche', 'fantom'];
    
    for (const chain of chains) {
      const key = await KeyDerivation.deriveKey(seed, `m/44'/60'/0'/0/0`);
      const hash = await CryptoUtils.keccak256(JSON.stringify(Array.from(key)));
      addresses[chain] = '0x' + hash.slice(-40);
    }
    
    walletState = {
      isUnlocked: true,
      currentChain: 'ethereum',
      addresses,
      balances: {},
      transactions: [],
      encryptedMnemonic: await CryptoUtils.encrypt(mnemonic, password),
    };
    
    await saveState();
    return walletState;
  }
  
  // Import wallet from private key
  static async importPrivateKey(privateKey, password) {
    // Remove 0x prefix
    const key = privateKey.startsWith('0x') ? privateKey.slice(2) : privateKey;
    
    // Validate hex
    if (!/^[a-fA-F0-9]{64}$/.test(key)) {
      throw new Error('Invalid private key');
    }
    
    // Derive address from private key
    const publicKey = await this.privateKeyToPublicKey(privateKey);
    const address = CryptoUtils.publicKeyToAddress(publicKey);
    
    walletState = {
      isUnlocked: true,
      currentChain: 'ethereum',
      addresses: { ethereum: address },
      balances: {},
      transactions: [],
      privateKey: await CryptoUtils.encrypt(privateKey, password),
    };
    
    await saveState();
    return walletState;
  }
  
  // Unlock wallet with password
  static async unlockWallet(password) {
    await loadState();
    
    if (!walletState.encryptedMnemonic && !walletState.privateKey) {
      throw new Error('No wallet found');
    }
    
    try {
      if (walletState.encryptedMnemonic) {
        const mnemonic = await CryptoUtils.decrypt(walletState.encryptedMnemonic, password);
        const seed = await KeyDerivation.mnemonicToSeed(mnemonic, password);
        
        // Re-derive addresses
        const chains = ['ethereum', 'bsc', 'polygon', 'arbitrum', 'optimism', 'base', 'avalanche', 'fantom'];
        for (const chain of chains) {
          const key = await KeyDerivation.deriveKey(seed, `m/44'/60'/0'/0/0`);
          const hash = await CryptoUtils.keccak256(JSON.stringify(Array.from(key)));
          walletState.addresses[chain] = '0x' + hash.slice(-40);
        }
      } else if (walletState.privateKey) {
        const privateKey = await CryptoUtils.decrypt(walletState.privateKey, password);
        const publicKey = await this.privateKeyToPublicKey(privateKey);
        walletState.addresses.ethereum = CryptoUtils.publicKeyToAddress(publicKey);
      }
      
      walletState.isUnlocked = true;
      await saveState();
      return walletState;
    } catch (e) {
      throw new Error('Invalid password');
    }
  }
  
  // Lock wallet
  static lockWallet() {
    walletState.isUnlocked = false;
    saveState();
    return true;
  }
  
  // Get current address
  static getAddress(chain = null) {
    const targetChain = chain || walletState.currentChain;
    return walletState.addresses[targetChain] || '';
  }
  
  // Switch chain
  static switchChain(chainId) {
    if (!CHAIN_CONFIG[chainId]) {
      throw new Error('Unsupported chain');
    }
    walletState.currentChain = chainId;
    saveState();
    return walletState;
  }
  
  // Sign transaction (simplified)
  static async signTransaction(tx) {
    if (!walletState.isUnlocked) {
      throw new Error('Wallet is locked');
    }
    
    // In production, this would:
    // 1. Build proper transaction
    // 2. Sign with private key
    // 3. Return signed transaction
    
    const txHash = await CryptoUtils.keccak256(JSON.stringify(tx) + Date.now());
    return '0x' + txHash;
  }
  
  // Sign message
  static async signMessage(message) {
    if (!walletState.isUnlocked) {
      throw new Error('Wallet is locked');
    }
    
    const signature = await CryptoUtils.keccak256(message + Date.now());
    return '0x' + signature;
  }
  
  // Encrypt data
  static async encrypt(data, password) {
    const encoder = new TextEncoder();
    const dataBuffer = encoder.encode(JSON.stringify(data));
    const keyBuffer = await crypto.subtle.digest('SHA-256', encoder.encode(password));
    
    const iv = await CryptoUtils.randomBytes(16);
    const key = await crypto.subtle.importKey(
      'raw',
      keyBuffer,
      { name: 'AES-GCM' },
      false,
      ['encrypt']
    );
    
    const encrypted = await crypto.subtle.encrypt(
      { name: 'AES-GCM', iv },
      key,
      dataBuffer
    );
    
    return JSON.stringify({
      iv: Array.from(iv),
      data: Array.from(new Uint8Array(encrypted)),
    });
  }
  
  // Decrypt data
  static async decrypt(encryptedData, password) {
    const { iv, data } = JSON.parse(encryptedData);
    const encoder = new TextEncoder();
    const keyBuffer = await crypto.subtle.digest('SHA-256', encoder.encode(password));
    
    const key = await crypto.subtle.importKey(
      'raw',
      keyBuffer,
      { name: 'AES-GCM' },
      false,
      ['decrypt']
    );
    
    const decrypted = await crypto.subtle.decrypt(
      { name: 'AES-GCM', iv: new Uint8Array(iv) },
      key,
      new Uint8Array(data)
    );
    
    const decoder = new TextDecoder();
    return JSON.parse(decoder.decode(decrypted));
  }
  
  // Get private key from public key (simplified - NOT real crypto)
  static async privateKeyToPublicKey(privateKey) {
    const hash = await CryptoUtils.keccak256(privateKey);
    return '0x04' + hash.repeat(4).slice(0, 128);
  }
}

// ============================================================================
// RPC CLIENT
// ============================================================================

class RpcClient {
  static async request(chainId, method, params = []) {
    const rpcUrl = DEFAULT_RPC[chainId] || DEFAULT_RPC.ethereum;
    
    const response = await fetch(rpcUrl, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        jsonrpc: '2.0',
        id: Date.now(),
        method,
        params,
      }),
    });
    
    const result = await response.json();
    if (result.error) {
      throw new Error(result.error.message);
    }
    return result.result;
  }
  
  // Get balance
  static async getBalance(chainId, address) {
    if (!CryptoUtils.isValidAddress(address)) {
      throw new Error('Invalid address');
    }
    return await this.request(chainId, 'eth_getBalance', [address, 'latest']);
  }
  
  // Get transaction count
  static async getTransactionCount(chainId, address) {
    return await this.request(chainId, 'eth_getTransactionCount', [address, 'latest']);
  }
  
  // Estimate gas
  static async estimateGas(chainId, tx) {
    return await this.request(chainId, 'eth_estimateGas', [tx]);
  }
  
  // Get gas price
  static async getGasPrice(chainId) {
    return await this.request(chainId, 'eth_gasPrice');
  }
  
  // Send raw transaction
  static async sendRawTransaction(chainId, signedTx) {
    return await this.request(chainId, 'eth_sendRawTransaction', [signedTx]);
  }
  
  // Get transaction receipt
  static async getTransactionReceipt(chainId, txHash) {
    return await this.request(chainId, 'eth_getTransactionReceipt', [txHash]);
  }
  
  // Call contract
  static async call(chainId, to, data) {
    return await this.request(chainId, 'eth_call', [{ to, data }, 'latest']);
  }
  
  // Get chain ID
  static async getChainId(chainId) {
    return await this.request(chainId, 'eth_chainId');
  }
}

// ============================================================================
// WALLET CONNECT (Simplified)
// ============================================================================

class WalletConnectManager {
  static sessions = new Map();
  static bridges = new Map();
  
  // Create session
  static async createSession(peerId, peerMeta) {
    const session = {
      topic: generateTopic(),
      peerId,
      peerMeta,
      accounts: [WalletManager.getAddress()],
      chainId: 1,
      created: Date.now(),
    };
    
    this.sessions.set(session.topic, session);
    return session;
  }
  
  // Approve session
  static async approveSession(topic, accounts, chainId) {
    const session = this.sessions.get(topic);
    if (!session) throw new Error('Session not found');
    
    session.accounts = accounts;
    session.chainId = chainId;
    session.approved = true;
    
    return session;
  }
  
  // Reject session
  static rejectSession(topic) {
    this.sessions.delete(topic);
  }
  
  // Get session
  static getSession(topic) {
    return this.sessions.get(topic);
  }
  
  // Disconnect
  static disconnect(topic) {
    this.sessions.delete(topic);
  }
  
  // Generate topic
  static generateTopic() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
      const r = Math.random() * 16 | 0;
      const v = c === 'x' ? r : (r & 0x3 | 0x8);
      return v.toString(16);
    });
  }
}

// ============================================================================
// DAPP PERMISSION MANAGEMENT
// ============================================================================

class DAppPermissionManager {
  static permissions = new Map();
  
  // Request permissions
  static async requestPermissions(origin, requestedPermissions) {
    const existing = this.permissions.get(origin) || { allowed: false, permissions: [] };
    
    // Always allow wallet address
    const permissions = {
      eth_accounts: {
        accounts: [WalletManager.getAddress()],
      },
      ...existing.permissions,
    };
    
    this.permissions.set(origin, {
      allowed: true,
      permissions,
      granted: Date.now(),
    });
    
    return permissions;
  }
  
  // Check permission
  static hasPermission(origin, permission) {
    const perms = this.permissions.get(origin);
    return perms && perms.allowed && permission in perms.permissions;
  }
  
  // Revoke permission
  static revokePermission(origin) {
    this.permissions.delete(origin);
  }
  
  // Get allowed origins
  static getAllowedOrigins() {
    return Array.from(this.permissions.entries())
      .filter(([_, p]) => p.allowed)
      .map(([origin, _]) => origin);
  }
}

// ============================================================================
// EVENT HANDLING
// ============================================================================

class EventEmitter {
  static listeners = new Map();
  
  static on(event, callback) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }
    this.listeners.get(event).add(callback);
    return () => this.off(event, callback);
  }
  
  static off(event, callback) {
    if (this.listeners.has(event)) {
      this.listeners.get(event).delete(callback);
    }
  }
  
  static emit(event, data) {
    if (this.listeners.has(event)) {
      this.listeners.get(event).forEach(cb => cb(data));
    }
  }
}

// ============================================================================
// MESSAGE HANDLING
// ============================================================================

// Handle messages from popup and content scripts
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  (async () => {
    try {
      let result;
      
      switch (message.type) {
        // Wallet operations
        case 'CREATE_WALLET':
          result = await WalletManager.createWallet(message.password);
          break;
          
        case 'IMPORT_WALLET':
          result = await WalletManager.importWallet(message.mnemonic, message.password);
          break;
          
        case 'IMPORT_PRIVATE_KEY':
          result = await WalletManager.importPrivateKey(message.privateKey, message.password);
          break;
          
        case 'UNLOCK_WALLET':
          result = await WalletManager.unlockWallet(message.password);
          break;
          
        case 'LOCK_WALLET':
          result = WalletManager.lockWallet();
          break;
          
        case 'GET_STATE':
          result = walletState;
          break;
          
        case 'GET_ADDRESS':
          result = WalletManager.getAddress(message.chain);
          break;
          
        case 'SWITCH_CHAIN':
          result = WalletManager.switchChain(message.chainId);
          EventEmitter.emit('chainChanged', message.chainId);
          break;
          
        // Transaction operations
        case 'SIGN_TRANSACTION':
          result = await WalletManager.signTransaction(message.tx);
          break;
          
        case 'SIGN_MESSAGE':
          result = await WalletManager.signMessage(message.message);
          break;
          
        // RPC operations
        case 'RPC_REQUEST':
          result = await RpcClient.request(message.chainId, message.method, message.params);
          break;
          
        case 'GET_BALANCE':
          result = await RpcClient.getBalance(message.chain, message.address);
          break;
          
        // WalletConnect
        case 'WC_CREATE_SESSION':
          result = await WalletConnectManager.createSession(message.peerId, message.peerMeta);
          break;
          
        case 'WC_APPROVE_SESSION':
          result = await WalletConnectManager.approveSession(message.topic, message.accounts, message.chainId);
          break;
          
        case 'WC_REJECT_SESSION':
          WalletConnectManager.rejectSession(message.topic);
          result = true;
          break;
          
        // Permissions
        case 'REQUEST_PERMISSIONS':
          result = await DAppPermissionManager.requestPermissions(message.origin, message.permissions);
          break;
          
        case 'CHECK_PERMISSION':
          result = DAppPermissionManager.hasPermission(message.origin, message.permission);
          break;
          
        // Default
        default:
          throw new Error(`Unknown message type: ${message.type}`);
      }
      
      sendResponse({ success: true, data: result });
    } catch (error) {
      sendResponse({ success: false, error: error.message });
    }
  })();
  
  return true; // Keep channel open for async response
});

// ============================================================================
// STATE PERSISTENCE
// ============================================================================

async function saveState() {
  const stateToSave = {
    ...walletState,
    // Don't save sensitive data to local storage
    encryptedMnemonic: undefined,
    privateKey: undefined,
  };
  
  await chrome.storage.local.set({ [WALLET_STATE_KEY]: stateToSave });
}

async function loadState() {
  const stored = await chrome.storage.local.get(WALLET_STATE_KEY);
  if (stored[WALLET_STATE_KEY]) {
    walletState = { ...walletState, ...stored[WALLET_STATE_KEY] };
  }
}

// ============================================================================
// INITIALIZATION
// ============================================================================

async function initialize() {
  console.log('TigerWallet Extension initialized');
  
  // Load saved state
  await loadState();
  
  // Set up chain change listener
  chrome.storage.onChanged.addListener((changes, area) => {
    if (area === 'local' && changes[WALLET_STATE_KEY]) {
      EventEmitter.emit('stateChanged', changes[WALLET_STATE_KEY].newValue);
    }
  });
  
  // Set up alarm for periodic balance updates
  chrome.alarms.create('updateBalances', { periodInMinutes: 5 });
}

chrome.alarms.onAlarm.addListener(async (alarm) => {
  if (alarm.name === 'updateBalances' && walletState.isUnlocked) {
    // Update balances for all chains
    for (const chain of Object.keys(walletState.addresses)) {
      try {
        const balance = await RpcClient.getBalance(chain, walletState.addresses[chain]);
        walletState.balances[chain] = balance;
      } catch (e) {
        console.error(`Failed to update balance for ${chain}:`, e);
      }
    }
    await saveState();
  }
});

// Start initialization
initialize();

console.log('TigerWallet Background Service Worker Ready');
