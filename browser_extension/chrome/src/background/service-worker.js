/**
 * TigerWallet Browser Extension - Background Service Worker
 * Handles all background operations for the extension.
 *
 * All crypto (BIP-39/32, secp256k1, keccak256, ECDSA signing, broadcasting)
 * is performed by the Go wallet-api backend; this worker only coordinates
 * state and forwards signing/broadcast requests to it.
 */

// ============================================================================
// Wallet-API backend configuration
// ============================================================================

// Default port matches wallet_api's WALLET_API_PORT (8443).
const WALLET_API_URL = 'http://localhost:8443';
const AUTH_TOKEN_KEY = 'tigerwallet_auth_token';

// In-memory JWT cache (loaded from chrome.storage.local on startup).
let authToken = null;

async function loadAuthToken() {
  try {
    const data = await chrome.storage.local.get(AUTH_TOKEN_KEY);
    if (data[AUTH_TOKEN_KEY]) {
      authToken = data[AUTH_TOKEN_KEY];
    }
  } catch (e) { /* storage unavailable */ }
}

async function setAuthToken(token) {
  authToken = token;
  try { await chrome.storage.local.set({ [AUTH_TOKEN_KEY]: token }); } catch (e) { /* ignore */ }
}

async function clearAuthToken() {
  authToken = null;
  try { await chrome.storage.local.remove(AUTH_TOKEN_KEY); } catch (e) { /* ignore */ }
}

function authHeaders() {
  return authToken ? { Authorization: `Bearer ${authToken}` } : {};
}

// Low-level fetch wrapper. Throws with the server's `error` field on non-2xx.
async function backendFetch(path, options = {}) {
  const url = `${WALLET_API_URL}${path}`;
  const res = await fetch(url, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...authHeaders(), ...(options.headers || {}) },
  });
  let body = null;
  try { body = await res.json(); } catch (e) { /* non-JSON */ }
  if (!res.ok) {
    throw new Error((body && body.error) || `wallet-api error ${res.status}`);
  }
  return body;
}

async function requireAuthToken() {
  if (!authToken) await loadAuthToken();
  if (!authToken) {
    throw new Error('Not authenticated with the wallet-api backend. Register or log in via /api/v1/auth first.');
  }
  return authToken;
}

// Real wallet creation/import (POST /api/v1/wallets). Server generates a real
// BIP-39 mnemonic (or validates/import an supplied one) and derives the address.
async function createWalletViaBackend(label, password, chainId = 1, mnemonic = null) {
  await requireAuthToken();
  const payload = { label, password, chain_id: chainId };
  if (mnemonic) payload.mnemonic = mnemonic;
  return backendFetch('/api/v1/wallets', { method: 'POST', body: JSON.stringify(payload) });
}

// Real ECDSA personal_sign (POST /api/v1/sign). Also serves as a password
// verification check: the backend returns 401 on a wrong password before signing.
async function signMessageViaBackend(walletId, password, message) {
  await requireAuthToken();
  return backendFetch('/api/v1/sign', {
    method: 'POST',
    body: JSON.stringify({ wallet_id: walletId, password, message }),
  });
}

// Real sign + broadcast (POST /api/v1/send).
async function sendTransactionViaBackend(walletId, password, to, value, chainId, data) {
  await requireAuthToken();
  return backendFetch('/api/v1/send', {
    method: 'POST',
    body: JSON.stringify({ wallet_id: walletId, password, to, value, chain_id: chainId, data: data || '0x' }),
  });
}

// Real balance read (GET /api/v1/public/balance).
async function getBalanceViaBackend(address, chainId) {
  return backendFetch(`/api/v1/public/balance?address=${encodeURIComponent(address)}&chain_id=${chainId}`);
}

// ============================================================================
// State Management
// ============================================================================

class ExtensionState {
  wallet = {
    isUnlocked: false,
    currentChain: 'ethereum',
    walletId: null,
    addresses: {},
    balance: {}
  };

  connections = new Map();
  transactions = new Map();
  pendingRequests = new Map();
  // In-memory password cache for the signing session (never persisted).
  cachedPassword = null;

  constructor() {
    this.loadState();
  }

  async loadState() {
    try {
      const result = await chrome.storage.local.get([
        'wallet',
        'connections',
        'transactions'
      ]);

      if (result.wallet) {
        this.wallet = { ...this.wallet, ...result.wallet };
      }

      if (result.connections) {
        this.connections = new Map(Object.entries(result.connections));
      }

      if (result.transactions) {
        this.transactions = new Map(Object.entries(result.transactions));
      }
    } catch (error) {
      console.error('Failed to load state:', error);
    }
  }

  async saveState() {
    try {
      // Never persist the password cache.
      const walletPersist = { ...this.wallet, cachedPassword: undefined };
      await chrome.storage.local.set({
        wallet: walletPersist,
        connections: Object.fromEntries(this.connections),
        transactions: Object.fromEntries(this.transactions)
      });
    } catch (error) {
      console.error('Failed to save state:', error);
    }
  }

  // Wallet methods
  getWallet() {
    return this.wallet;
  }

  setUnlocked(unlocked) {
    this.wallet.isUnlocked = unlocked;
    this.saveState();
  }

  setWalletId(id) {
    this.wallet.walletId = id;
    this.saveState();
  }

  setAddress(chain, address) {
    this.wallet.addresses[chain] = address;
    this.saveState();
  }

  setBalance(chain, balance) {
    this.wallet.balance[chain] = balance;
    this.saveState();
  }

  setCurrentChain(chain) {
    this.wallet.currentChain = chain;
    this.saveState();
  }

  // Connection methods
  addConnection(origin, connection) {
    this.connections.set(origin, connection);
    this.saveState();
  }

  removeConnection(origin) {
    this.connections.delete(origin);
    this.saveState();
  }

  getConnection(origin) {
    return this.connections.get(origin);
  }

  getAllConnections() {
    return Array.from(this.connections.values());
  }

  // Transaction methods
  addTransaction(id, tx) {
    this.transactions.set(id, tx);
    this.saveState();
  }

  getTransaction(id) {
    return this.transactions.get(id);
  }

  removeTransaction(id) {
    this.transactions.delete(id);
    this.saveState();
  }

  getAllTransactions() {
    return Array.from(this.transactions.values());
  }
}

const state = new ExtensionState();

// ============================================================================
// Message Handling
// ============================================================================

async function handleMessage(message, sender) {
  try {
    switch (message.type) {
      // Auth (wallet-api JWT)
      case 'REGISTER': {
        const result = await backendFetch('/api/v1/auth/register', {
          method: 'POST',
          body: JSON.stringify({
            email: message.payload.email,
            username: message.payload.username,
            password: message.payload.password
          }),
          headers: {}
        });
        if (result && result.token) await setAuthToken(result.token);
        return { success: true, data: result };
      }

      case 'LOGIN': {
        const result = await backendFetch('/api/v1/auth/login', {
          method: 'POST',
          body: JSON.stringify({ email: message.payload.email, password: message.payload.password }),
          headers: {}
        });
        if (result && result.token) await setAuthToken(result.token);
        return { success: true, data: result };
      }

      case 'LOGOUT': {
        await clearAuthToken();
        return { success: true };
      }

      // Wallet state
      case 'GET_WALLET_STATE':
        return { success: true, data: state.getWallet() };

      // Create a real wallet via the backend (real BIP-39 mnemonic, real address).
      case 'CREATE_WALLET': {
        const result = await createWalletViaBackend(
          message.payload.label || 'TigerWallet',
          message.payload.password,
          message.payload.chainId || 1
        );
        const address = result.address;
        const chains = ['ethereum', 'bsc', 'polygon', 'arbitrum', 'optimism', 'base', 'avalanche', 'fantom'];
        for (const c of chains) state.setAddress(c, address);
        state.setWalletId(result.id);
        state.cachedPassword = message.payload.password;
        state.setUnlocked(true);
        return { success: true, data: { ...result, mnemonic: result.mnemonic } };
      }

      // Import an existing wallet from a mnemonic via the backend.
      case 'IMPORT_WALLET': {
        const result = await createWalletViaBackend(
          message.payload.label || 'TigerWallet',
          message.payload.password,
          message.payload.chainId || 1,
          message.payload.mnemonic
        );
        const address = result.address;
        const chains = ['ethereum', 'bsc', 'polygon', 'arbitrum', 'optimism', 'base', 'avalanche', 'fantom'];
        for (const c of chains) state.setAddress(c, address);
        state.setWalletId(result.id);
        state.cachedPassword = message.payload.password;
        state.setUnlocked(true);
        return { success: true, data: result };
      }

      // Unlock the wallet by verifying the password against the backend via a
      // personal_sign of a sentinel message (no transaction is broadcast).
      case 'UNLOCK_WALLET': {
        const wallet = state.getWallet();
        if (!wallet.walletId) {
          return { success: false, error: 'No wallet found' };
        }
        try {
          await signMessageViaBackend(wallet.walletId, message.payload.password, 'TigerWallet unlock verification');
          state.cachedPassword = message.payload.password;
          state.setUnlocked(true);
          return { success: true };
        } catch (e) {
          return { success: false, error: 'Invalid password' };
        }
      }

      case 'LOCK_WALLET':
        state.setUnlocked(false);
        state.cachedPassword = null;
        return { success: true };

      case 'SET_CHAIN':
        state.setCurrentChain(message.payload.chain);
        return { success: true };

      // Connection management
      case 'CONNECT_DAPP': {
        const account = state.getWallet().addresses['ethereum'];
        if (!account) {
          return { success: false, error: 'No verified wallet account is available. Create or import a wallet first.' };
        }
        const connection = {
          origin: message.payload.origin,
          chainId: message.payload.chainId || 'ethereum',
          connected: true,
          accounts: [account]
        };
        state.addConnection(message.payload.origin, connection);

        notifyContentScript(message.payload.origin, {
          type: 'CONNECTION_CHANGED',
          payload: { connected: true, accounts: connection.accounts }
        });
        return { success: true, data: connection };
      }

      case 'DISCONNECT_DAPP':
        state.removeConnection(message.payload.origin);

        notifyContentScript(message.payload.origin, {
          type: 'CONNECTION_CHANGED',
          payload: { connected: false, accounts: [] }
        });
        return { success: true };

      case 'GET_CONNECTIONS':
        return { success: true, data: state.getAllConnections() };

      // Transaction signing + broadcast via the backend's real ECDSA endpoint.
      case 'SIGN_TRANSACTION': {
        const wallet = state.getWallet();
        if (!wallet.isUnlocked) {
          return { success: false, error: 'Wallet is locked' };
        }
        if (!wallet.walletId) {
          return { success: false, error: 'No wallet loaded' };
        }
        const password = state.cachedPassword;
        if (!password) {
          return { success: false, error: 'Wallet password not available; unlock the wallet first.' };
        }
        const tx = message.payload.tx || message.payload;
        const chainId = tx.chainId ? parseInt(tx.chainId, 16) : 1;
        try {
          const result = await sendTransactionViaBackend(
            wallet.walletId,
            password,
            tx.to,
            tx.value || '0',
            chainId,
            tx.data || '0x'
          );
          return { success: true, data: result.tx_hash || result };
        } catch (e) {
          return { success: false, error: e.message };
        }
      }

      // Approve a pending transaction (forwarded to SIGN_TRANSACTION flow).
      case 'APPROVE_TRANSACTION': {
        const pending = state.getTransaction(message.payload.id);
        if (!pending) {
          return { success: false, error: 'No such pending transaction' };
        }
        const wallet = state.getWallet();
        const password = state.cachedPassword;
        if (!wallet.walletId || !password) {
          return { success: false, error: 'Wallet is locked or not loaded' };
        }
        const chainId = pending.chainId ? parseInt(pending.chainId, 16) : 1;
        try {
          const result = await sendTransactionViaBackend(
            wallet.walletId,
            password,
            pending.params.to,
            pending.params.value || '0',
            chainId,
            pending.params.data || '0x'
          );
          state.removeTransaction(message.payload.id);
          notifyContentScript(pending.origin, {
            type: 'TRANSACTION_RESULT',
            payload: { id: pending.id, status: 'approved', txHash: result.tx_hash }
          });
          return { success: true, data: result.tx_hash };
        } catch (e) {
          return { success: false, error: e.message };
        }
      }

      case 'REJECT_TRANSACTION': {
        const rejectedTx = state.getTransaction(message.payload.id);
        if (rejectedTx) {
          state.removeTransaction(message.payload.id);

          notifyContentScript(rejectedTx.origin, {
            type: 'TRANSACTION_RESULT',
            payload: { id: rejectedTx.id, status: 'rejected' }
          });
        }
        return { success: true };
      }

      // Network switching
      case 'SWITCH_CHAIN':
        state.setCurrentChain(message.payload.chainId);

        for (const conn of state.getAllConnections()) {
          notifyContentScript(conn.origin, {
            type: 'CHAIN_CHANGED',
            payload: { chainId: message.payload.chainId }
          });
        }
        return { success: true };

      // Personal signing via the backend's real ECDSA personal_sign endpoint.
      case 'SIGN_MESSAGE': {
        const wallet = state.getWallet();
        if (!wallet.isUnlocked) {
          return { success: false, error: 'Wallet is locked' };
        }
        if (!wallet.walletId) {
          return { success: false, error: 'No wallet loaded' };
        }
        const password = state.cachedPassword;
        if (!password) {
          return { success: false, error: 'Wallet password not available; unlock the wallet first.' };
        }
        try {
          const result = await signMessageViaBackend(wallet.walletId, password, message.payload.message);
          return { success: true, data: result.signature };
        } catch (e) {
          return { success: false, error: e.message };
        }
      }

      // Token/balance queries (real on-chain balance via the backend)
      case 'GET_BALANCE': {
        const wallet = state.getWallet();
        const address = wallet.addresses[message.payload.chain] || wallet.addresses['ethereum'];
        if (!address) {
          return { success: true, data: '0' };
        }
        try {
          const chainId = parseInt(message.payload.chainId || '0x1', 16);
          const result = await getBalanceViaBackend(address, chainId);
          const balance = (result && result.balance) ? result.balance : '0';
          state.setBalance(message.payload.chain, balance);
          return { success: true, data: balance };
        } catch (e) {
          return { success: false, error: e.message };
        }
      }

      case 'GET_ADDRESS': {
        const addr = state.getWallet();
        return {
          success: true,
          data: addr.addresses[message.payload.chain] || ''
        };
      }

      default:
        return { success: false, error: `Unknown message type: ${message.type}` };
    }
  } catch (error) {
    console.error('Message handling error:', error);
    return { success: false, error: String(error) };
  }
}

function notifyContentScript(origin, message) {
  chrome.tabs.query({ url: origin + '*' }, (tabs) => {
    for (const tab of tabs) {
      if (tab.id) {
        chrome.tabs.sendMessage(tab.id, message).catch(() => {});
      }
    }
  });
}

// ============================================================================
// Utility Functions
// ============================================================================

function generateId() {
  return `tx_${crypto.randomUUID()}`;
}

// ============================================================================
// Event Listeners
// ============================================================================

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  handleMessage(message, sender).then(sendResponse);
  return true; // Keep channel open for async response
});

chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
  if (changeInfo.status === 'complete' && tab.url) {
    checkAndInjectDApp(tabId, tab.url);
  }
});

chrome.webNavigation.onCompleted.addListener((details) => {
  if (details.frameId === 0) {
    console.log('Navigation completed:', details.url);
  }
});

chrome.runtime.onInstalled.addListener((details) => {
  if (details.reason === 'install') {
    console.log('TigerWallet extension installed');
    state.setCurrentChain('ethereum');
  } else if (details.reason === 'update') {
    console.log('TigerWallet extension updated to version:', chrome.runtime.getManifest().version);
  }
});

chrome.runtime.onStartup.addListener(() => {
  console.log('TigerWallet extension starting...');
  state.loadState();
  loadAuthToken();
});

function checkAndInjectDApp(tabId, url) {
  // Placeholder for DApp detection logic.
}

// ============================================================================
// Context Menu
// ============================================================================

function setupContextMenus() {
  chrome.contextMenus?.create({
    id: 'tiger-send',
    title: 'Send with TigerWallet',
    contexts: ['selection']
  });

  chrome.contextMenus?.create({
    id: 'tiger-view-address',
    title: 'View Address in TigerWallet',
    contexts: ['page']
  });

  chrome.contextMenus?.onClicked.addListener((info, tab) => {
    if (info.menuItemId === 'tiger-send' && info.selectionText) {
      chrome.runtime.sendMessage({
        type: 'OPEN_SEND_DIALOG',
        payload: { data: info.selectionText }
      });
    }
  });
}

setupContextMenus();

// ============================================================================
// Periodic Tasks
// ============================================================================

chrome.alarms.create('updateBalances', { periodInMinutes: 5 });

chrome.alarms.onAlarm.addListener(async (alarm) => {
  if (alarm.name === 'updateBalances') {
    const wallet = state.getWallet();
    if (!wallet.isUnlocked) return;
    const chains = Object.keys(wallet.addresses);
    for (const chain of chains) {
      try {
        // Balance is the same EVM address across chains; use chain 1 for the
        // read since the backend resolves the address regardless of chain_id.
        const result = await getBalanceViaBackend(wallet.addresses[chain], 1);
        state.setBalance(chain, (result && result.balance) ? result.balance : '0');
      } catch (e) {
        console.error(`Failed to update balance for ${chain}:`, e);
      }
    }
  }
});

chrome.alarms.create('cleanupTransactions', { periodInMinutes: 10 });

chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === 'cleanupTransactions') {
    const now = Date.now();
    const maxAge = 5 * 60 * 1000;

    for (const [id, tx] of state.getAllTransactions()) {
      if (now - tx.timestamp > maxAge) {
        state.removeTransaction(id);
      }
    }
  }
});
