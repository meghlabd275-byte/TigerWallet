// TigerWallet Extension - Background Service Worker
// Complete wallet management and DApp interaction

// ============================================================================
// CONSTANTS & CONFIGURATION
// ============================================================================

const WALLET_STATE_KEY = 'tigerwallet_state';
const RPC_CONFIG_KEY = 'tigerwallet_rpc';
const CHAIN_CONFIG_KEY = 'tigerwallet_chains';

// Go wallet-api backend (real BIP-39/32, secp256k1, keccak256, ECDSA signing,
// broadcasting). Default port matches wallet_api's WALLET_API_PORT (8443).
const WALLET_API_URL = 'http://localhost:8443';

// Auth state (JWT issued by the backend's /api/v1/auth/* endpoints). Cached in
// memory + chrome.storage.local so it survives service-worker restarts.
let authToken = null;
const AUTH_TOKEN_KEY = 'tigerwallet_auth_token';
const AUTH_USER_KEY = 'tigerwallet_auth_user';

// In-memory cache of the wallet password for the duration of a signing request.
// The password is never persisted; it is only used to unlock the server-side
// encrypted seed for /api/v1/send and /api/v1/sign.
let cachedWalletPassword = null;

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
  
  // Address derivation from a public key (secp256k1 + keccak256) is performed
  // server-side by the wallet-api. There is no synchronous client-side path
  // that can do real secp256k1/keccak256; callers must use the async backend
  // wallet creation flow (WalletManager.createWallet) instead.
  static publicKeyToAddress(_publicKey) {
    throw new Error('Address derivation is performed by the wallet-api backend. Create or import the wallet via WalletManager to obtain a real address.');
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
  // BIP-39 seed derivation (PBKDF2-HMAC-SHA512 over the mnemonic) happens
  // server-side in the Go wallet-api (go-bip39). The browser cannot reproduce it
  // reliably without shipping a WASM crypto bundle, so callers must delegate to
  // WalletManager.createWallet/importWallet, which call POST /api/v1/wallets.
  static async mnemonicToSeed(_mnemonic, _password = '') {
    throw new Error('BIP-39 seed derivation is performed by the wallet-api backend. Use WalletManager.createWallet/importWallet, which call POST /api/v1/wallets.');
  }

  // BIP-32 HD key derivation is performed server-side by the wallet-api
  // (go-ethereum/accounts DerivationPath). Callers obtain a real derived
  // address via the backend wallet creation flow.
  static async deriveKey(_seed, _path) {
    throw new Error('BIP-32 HD key derivation is performed by the wallet-api backend. Use WalletManager.createWallet/importWallet, which call POST /api/v1/wallets.');
  }

  // Generates a real BIP-39 mnemonic by creating a wallet on the backend with
  // no supplied mnemonic (the Go server calls bip39.NewEntropy/NewMnemonic).
  // Returns the mnemonic string (returned once on creation).
  static async generateMnemonic() {
    const password = await generateSecurePassword();
    const result = await createWalletViaBackend('Generated Wallet', password, 1);
    return result.mnemonic;
  }
}

// ============================================================================
// WALLET MANAGEMENT
// ============================================================================

class WalletManager {
  // Create a new wallet via the Go wallet-api backend (real BIP-39 mnemonic,
  // real BIP-32/44 HD derivation, real secp256k1 key, keccak256 address,
  // encrypted seed stored server-side). The mnemonic is returned once.
  static async createWallet(password) {
    if (!password || password.length < 8) {
      throw new Error('Password must be at least 8 characters');
    }
    cachedWalletPassword = password;

    const result = await createWalletViaBackend('TigerWallet', password, 1);
    const address = result.address;

    // The same EVM private key drives the address on every EVM-compatible
    // chain we support, so mirror the real derived address across them.
    const addresses = {};
    for (const chain of ['ethereum', 'bsc', 'polygon', 'arbitrum', 'optimism', 'base', 'avalanche', 'fantom']) {
      addresses[chain] = address;
    }

    walletState = {
      isUnlocked: true,
      currentChain: 'ethereum',
      walletId: result.id,
      derivationPath: result.derivation_path,
      addresses,
      balances: {},
      transactions: [],
      // The mnemonic is sensitive; keep it in memory only long enough to show
      // it once in the popup, then it must be cleared by the caller.
      mnemonic: result.mnemonic,
    };

    await saveState();
    return walletState;
  }

  // Import an existing wallet from a mnemonic via the backend, which validates
  // the BIP-39 checksum and derives the real address server-side.
  static async importWallet(mnemonic, password) {
    if (!mnemonic) {
      throw new Error('Mnemonic is required');
    }
    const words = mnemonic.trim().split(/\s+/);
    if (words.length !== 12 && words.length !== 15 && words.length !== 18 && words.length !== 21 && words.length !== 24) {
      throw new Error('Invalid mnemonic length');
    }
    if (!password || password.length < 8) {
      throw new Error('Password must be at least 8 characters');
    }
    cachedWalletPassword = password;

    const result = await importWalletViaBackend(mnemonic, 'TigerWallet', password, 1);
    const address = result.address;

    const addresses = {};
    for (const chain of ['ethereum', 'bsc', 'polygon', 'arbitrum', 'optimism', 'base', 'avalanche', 'fantom']) {
      addresses[chain] = address;
    }

    walletState = {
      isUnlocked: true,
      currentChain: 'ethereum',
      walletId: result.id,
      derivationPath: result.derivation_path,
      addresses,
      balances: {},
      transactions: [],
    };

    await saveState();
    return walletState;
  }

  // Import a wallet from a raw private key. The wallet-api backend persists
  // wallets by BIP-39 mnemonic + HD derivation path (real BIP-32/44), so it has
  // no endpoint that accepts a raw secp256k1 private key directly. We surface a
  // clear, actionable error rather than fabricating a fake address/signature.
  static async importPrivateKey(privateKey, password) {
    const key = privateKey.startsWith('0x') ? privateKey.slice(2) : privateKey;
    if (!/^[a-fA-F0-9]{64}$/.test(key)) {
      throw new Error('Invalid private key');
    }
    if (!password || password.length < 8) {
      throw new Error('Password must be at least 8 characters');
    }
    throw new Error(
      'Raw private key import is not supported by the wallet-api backend, which persists wallets by BIP-39 mnemonic + HD path. Import the mnemonic that generated this key via WalletManager.importWallet, or add a private-key import endpoint to the Go backend.'
    );
  }

  // Unlock the wallet by verifying the password against the backend. We perform
  // a personal_sign of a sentinel message: the backend returns 401 on an
  // incorrect password before it ever signs, so this is a safe password check
  // that does NOT broadcast any transaction. The verified password is then
  // cached in memory for subsequent signing/broadcast calls in this session.
  static async unlockWallet(password) {
    await loadState();
    if (!walletState.walletId) {
      throw new Error('No wallet found');
    }
    if (!password || password.length < 8) {
      throw new Error('Invalid password');
    }
    try {
      await signMessageViaBackend(walletState.walletId, password, 'TigerWallet unlock verification');
      cachedWalletPassword = password;
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
    cachedWalletPassword = null;
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

  // Sign and broadcast a real transaction via the backend's /api/v1/send
  // endpoint (real ECDSA signing + nonce/gas fetch + broadcast).
  static async signTransaction(tx) {
    if (!walletState.isUnlocked) {
      throw new Error('Wallet is locked');
    }
    if (!walletState.walletId) {
      throw new Error('No wallet loaded');
    }
    const password = cachedWalletPassword || await getWalletPassword();
    const chainId = tx.chainId ? parseInt(tx.chainId, 16) : 1;
    const result = await sendTransactionViaBackend(
      walletState.walletId,
      password,
      tx.to,
      tx.value || '0',
      chainId,
      tx.data || '0x'
    );
    return result.tx_hash;
  }

  // Sign a message via the backend's real ECDSA personal_sign endpoint
  // (keccak256("\x19Ethereum Signed Message:\n" + len + msg)).
  static async signMessage(message) {
    if (!walletState.isUnlocked) {
      throw new Error('Wallet is locked');
    }
    if (!walletState.walletId) {
      throw new Error('No wallet loaded');
    }
    const password = cachedWalletPassword || await getWalletPassword();
    const result = await signMessageViaBackend(walletState.walletId, password, message);
    return result.signature;
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
    if (!globalThis.crypto || typeof globalThis.crypto.randomUUID !== 'function') {
      throw new Error('Secure UUID generation is unavailable');
    }
    return globalThis.crypto.randomUUID();
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
// WALLET-API BACKEND WIRING
// All crypto (BIP-39/32, secp256k1, keccak256, ECDSA signing, broadcasting)
// happens server-side in the Go wallet-api. These helpers are the only way the
// extension touches keys.
// ============================================================================

// Load the cached JWT (if any) from chrome.storage.local on startup.
async function loadAuthToken() {
  try {
    const data = await chrome.storage.local.get(AUTH_TOKEN_KEY);
    if (data[AUTH_TOKEN_KEY]) {
      authToken = data[AUTH_TOKEN_KEY];
    }
  } catch (e) { /* storage may be unavailable in some contexts */ }
}

async function setAuthToken(token, user = null) {
  authToken = token;
  const payload = { [AUTH_TOKEN_KEY]: token };
  if (user) payload[AUTH_USER_KEY] = user;
  await chrome.storage.local.set(payload);
}

async function clearAuthToken() {
  authToken = null;
  await chrome.storage.local.remove([AUTH_TOKEN_KEY, AUTH_USER_KEY]);
}

function getAuthHeaders() {
  return authToken ? { Authorization: `Bearer ${authToken}` } : {};
}

// Low-level fetch wrapper for the wallet-api. Throws with the server's `error`
// field on non-2xx responses so callers can surface a meaningful message.
async function backendFetch(path, options = {}) {
  const url = `${WALLET_API_URL}${path}`;
  const res = await fetch(url, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...getAuthHeaders(), ...(options.headers || {}) },
  });
  let body = null;
  try { body = await res.json(); } catch (e) { /* non-JSON body */ }
  if (!res.ok) {
    const msg = (body && body.error) || `wallet-api error ${res.status}`;
    throw new Error(msg);
  }
  return body;
}

// Register a new wallet-api user account and cache the returned JWT. The
// wallet-api's protected routes (/api/v1/wallets, /api/v1/send, /api/v1/sign)
// all require a Bearer token, so we register/login once and reuse the token.
async function registerViaBackend(email, username, password) {
  const result = await backendFetch('/api/v1/auth/register', {
    method: 'POST',
    body: JSON.stringify({ email, username, password }),
    headers: {}, // intentionally no auth header on register/login
  });
  if (result && result.token) {
    await setAuthToken(result.token, result.user_id || result.user);
  }
  return result;
}

// Log in an existing wallet-api user and cache the returned JWT.
async function loginViaBackend(email, password) {
  const result = await backendFetch('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
    headers: {},
  });
  if (result && result.token) {
    await setAuthToken(result.token, result.user);
  }
  return result;
}

// Ensure we have an auth token before calling a protected route. If none is
// cached, throw a clear error directing the caller to register/login first.
async function requireAuthToken() {
  if (!authToken) {
    await loadAuthToken();
  }
  if (!authToken) {
    throw new Error('Not authenticated with the wallet-api backend. Register or log in via /api/v1/auth first.');
  }
  return authToken;
}

// Create a real wallet on the backend (POST /api/v1/wallets). When no mnemonic
// is supplied, the Go server generates a real BIP-39 mnemonic (256 bits of
// entropy) and derives the address via BIP-32/44 + secp256k1 + keccak256.
async function createWalletViaBackend(label, password, chainId = 1) {
  await requireAuthToken();
  return backendFetch('/api/v1/wallets', {
    method: 'POST',
    body: JSON.stringify({ label, password, chain_id: chainId }),
  });
}

// Import an existing mnemonic via the backend. The Go server validates the
// BIP-39 checksum, derives the seed (PBKDF2-HMAC-SHA512), and derives the real
// address server-side.
async function importWalletViaBackend(mnemonic, label, password, chainId = 1) {
  await requireAuthToken();
  return backendFetch('/api/v1/wallets', {
    method: 'POST',
    body: JSON.stringify({ mnemonic, label, password, chain_id: chainId }),
  });
}

// Sign + broadcast a real transaction via POST /api/v1/send. The backend fetches
// the nonce/gas price, signs with the real secp256k1 key (EIP-155), and
// broadcasts the raw tx to the chain's RPC endpoint. Returns { tx_hash, ... }.
async function sendTransactionViaBackend(walletId, password, to, value, chainId, data) {
  await requireAuthToken();
  return backendFetch('/api/v1/send', {
    method: 'POST',
    body: JSON.stringify({
      wallet_id: walletId,
      password,
      to,
      value,
      chain_id: chainId,
      data: data || '0x',
    }),
  });
}

// Sign a message via POST /api/v1/sign. The backend performs a real ECDSA
// personal_sign: keccak256("\x19Ethereum Signed Message:\n" + len + msg).
// Returns { signature: "0x..." }.
async function signMessageViaBackend(walletId, password, message) {
  await requireAuthToken();
  return backendFetch('/api/v1/sign', {
    method: 'POST',
    body: JSON.stringify({ wallet_id: walletId, password, message }),
  });
}

// Fetch the real on-chain balance for an address via the backend's read-only
// balance endpoint (no auth required). Returns { address, balance, ... }.
async function getBalanceViaBackend(address, chainId) {
  return backendFetch(`/api/v1/public/balance?address=${encodeURIComponent(address)}&chain_id=${chainId}`);
}

// Fetch real transaction history for an address via the backend's read-only
// transactions endpoint.
async function getTransactionsViaBackend(address, chainId) {
  return backendFetch(`/api/v1/public/transactions?address=${encodeURIComponent(address)}&chain_id=${chainId}`);
}

// Generate a cryptographically-secure random password (32 hex chars) using the
// Web Crypto API. Used only by KeyDerivation.generateMnemonic, which creates a
// throwaway server-side wallet purely to obtain a fresh BIP-39 mnemonic.
async function generateSecurePassword() {
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return Array.from(bytes).map(b => b.toString(16).padStart(2, '0')).join('');
}

// Prompt the user for their wallet password via the popup. The password is kept
// in memory only for the duration of the signing request and is never
// persisted. Used when no password is cached (e.g. signing after a restart).
async function getWalletPassword() {
  return new Promise((resolve, reject) => {
    chrome.windows.create({
      url: chrome.runtime.getURL('src/popup/popup.html?action=password'),
      type: 'popup',
      width: 360,
      height: 480,
    }, (win) => {
      const listener = (msg) => {
        if (msg && msg.type === 'PASSWORD_SUBMIT' && msg.password) {
          chrome.runtime.onMessage.removeListener(listener);
          resolve(msg.password);
          chrome.windows.remove(win.id).catch(() => {});
        } else if (msg && msg.type === 'PASSWORD_CANCEL') {
          chrome.runtime.onMessage.removeListener(listener);
          reject(new Error('User cancelled password entry'));
          chrome.windows.remove(win.id).catch(() => {});
        }
      };
      chrome.runtime.onMessage.addListener(listener);
    });
  });
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
        // Auth (wallet-api JWT)
        case 'REGISTER':
          result = await registerViaBackend(message.email, message.username, message.password);
          break;

        case 'LOGIN':
          result = await loginViaBackend(message.email, message.password);
          break;

        case 'LOGOUT':
          await clearAuthToken();
          result = true;
          break;

        case 'GET_AUTH_STATE':
          await loadAuthToken();
          result = { authenticated: !!authToken };
          break;

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
          
        case 'GET_BALANCE': {
          const cfg = CHAIN_CONFIG[message.chain];
          const chainId = cfg ? parseInt(cfg.id, 16) : 1;
          const address = message.address || WalletManager.getAddress(message.chain);
          result = await getBalanceViaBackend(address, chainId);
          break;
        }

        case 'GET_TRANSACTIONS': {
          const cfg = CHAIN_CONFIG[message.chain];
          const chainId = cfg ? parseInt(cfg.id, 16) : 1;
          const address = message.address || WalletManager.getAddress(message.chain);
          result = await getTransactionsViaBackend(address, chainId);
          break;
        }

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
    mnemonic: undefined,
    // Password cache lives only in memory
    cachedWalletPassword: undefined,
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
  
  // Load saved state + cached auth token
  await loadState();
  await loadAuthToken();
  
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
    // Update balances for all chains via the backend's real balance endpoint
    for (const chain of Object.keys(walletState.addresses)) {
      try {
        const cfg = CHAIN_CONFIG[chain];
        const chainId = cfg ? parseInt(cfg.id, 16) : 1;
        const result = await getBalanceViaBackend(walletState.addresses[chain], chainId);
        walletState.balances[chain] = (result && result.balance) ? result.balance : '0';
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
