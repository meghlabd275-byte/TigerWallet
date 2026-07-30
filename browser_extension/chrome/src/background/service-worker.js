/**
 * TigerWallet Browser Extension - Background Service Worker
 * Handles all background operations for the extension
 */

// ============================================================================
// Types
// ============================================================================

interface WalletState {
  isUnlocked: boolean;
  currentChain: string;
  addresses: Record<string, string>;
  balance: Record<string, string>;
}

interface DAppConnection {
  origin: string;
  chainId: string;
  connected: boolean;
  accounts: string[];
}

interface TransactionRequest {
  id: string;
  method: string;
  params: any;
  origin: string;
  chainId: string;
  timestamp: number;
}

// ============================================================================
// State Management
// ============================================================================

class ExtensionState {
  private wallet: WalletState = {
    isUnlocked: false,
    currentChain: 'ethereum',
    addresses: {},
    balance: {}
  };
  
  private connections: Map<string, DAppConnection> = new Map();
  private transactions: Map<string, TransactionRequest> = new Map();
  private pendingRequests: Map<string, any> = new Map();
  
  constructor() {
    this.loadState();
  }
  
  private async loadState() {
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
      await chrome.storage.local.set({
        wallet: this.wallet,
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
  
  setUnlocked(unlocked: boolean) {
    this.wallet.isUnlocked = unlocked;
    this.saveState();
  }
  
  setAddress(chain: string, address: string) {
    this.wallet.addresses[chain] = address;
    this.saveState();
  }
  
  setBalance(chain: string, balance: string) {
    this.wallet.balance[chain] = balance;
    this.saveState();
  }
  
  setCurrentChain(chain: string) {
    this.wallet.currentChain = chain;
    this.saveState();
  }
  
  // Connection methods
  addConnection(origin: string, connection: DAppConnection) {
    this.connections.set(origin, connection);
    this.saveState();
  }
  
  removeConnection(origin: string) {
    this.connections.delete(origin);
    this.saveState();
  }
  
  getConnection(origin: string): DAppConnection | undefined {
    return this.connections.get(origin);
  }
  
  getAllConnections(): DAppConnection[] {
    return Array.from(this.connections.values());
  }
  
  // Transaction methods
  addTransaction(id: string, tx: TransactionRequest) {
    this.transactions.set(id, tx);
    this.saveState();
  }
  
  getTransaction(id: string): TransactionRequest | undefined {
    return this.transactions.get(id);
  }
  
  removeTransaction(id: string) {
    this.transactions.delete(id);
    this.saveState();
  }
  
  getAllTransactions(): TransactionRequest[] {
    return Array.from(this.transactions.values());
  }
}

const state = new ExtensionState();

// ============================================================================
// Message Handling
// ============================================================================

interface Message {
  type: string;
  payload?: any;
}

interface Response {
  success: boolean;
  data?: any;
  error?: string;
}

async function handleMessage(message: Message, sender: chrome.runtime.MessageSender): Promise<Response> {
  try {
    switch (message.type) {
      // Wallet state
      case 'GET_WALLET_STATE':
        return { success: true, data: state.getWallet() };
        
      case 'UNLOCK_WALLET':
        state.setUnlocked(true);
        return { success: true };
        
      case 'LOCK_WALLET':
        state.setUnlocked(false);
        return { success: true };
        
      case 'SET_CHAIN':
        state.setCurrentChain(message.payload.chain);
        return { success: true };
        
      // Connection management
      case 'CONNECT_DAPP':
        const connection: DAppConnection = {
          origin: message.payload.origin,
          chainId: message.payload.chainId || 'ethereum',
          connected: true,
          accounts: [state.getWallet().addresses['ethereum'] || '']
        };
        state.addConnection(message.payload.origin, connection);
        
        // Notify content script
        notifyContentScript(message.payload.origin, {
          type: 'CONNECTION_CHANGED',
          payload: { connected: true, accounts: connection.accounts }
        });
        return { success: true, data: connection };
        
      case 'DISCONNECT_DAPP':
        state.removeConnection(message.payload.origin);
        
        notifyContentScript(message.payload.origin, {
          type: 'CONNECTION_CHANGED',
          payload: { connected: false, accounts: [] }
        });
        return { success: true };
        
      case 'GET_CONNECTIONS':
        return { success: true, data: state.getAllConnections() };
        
      // Transaction signing
      case 'SIGN_TRANSACTION':
        const txId = generateId();
        const txRequest: TransactionRequest = {
          id: txId,
          method: message.payload.method,
          params: message.payload.params,
          origin: message.payload.origin,
          chainId: message.payload.chainId,
          timestamp: Date.now()
        };
        
        state.addTransaction(txId, txRequest);
        
        // Show approval popup
        await showTransactionApproval(txRequest);
        
        return { success: true, data: { id: txId } };
        
      case 'APPROVE_TRANSACTION':
        const approvedTx = state.getTransaction(message.payload.id);
        if (approvedTx) {
          // In real implementation, this would sign and broadcast
          state.removeTransaction(message.payload.id);
          
          notifyContentScript(approvedTx.origin, {
            type: 'TRANSACTION_RESULT',
            payload: { id: approvedTx.id, status: 'approved', hash: '0x...' }
          });
        }
        return { success: true };
        
      case 'REJECT_TRANSACTION':
        const rejectedTx = state.getTransaction(message.payload.id);
        if (rejectedTx) {
          state.removeTransaction(message.payload.id);
          
          notifyContentScript(rejectedTx.origin, {
            type: 'TRANSACTION_RESULT',
            payload: { id: rejectedTx.id, status: 'rejected' }
          });
        }
        return { success: true };
        
      // Network switching
      case 'SWITCH_CHAIN':
        state.setCurrentChain(message.payload.chainId);
        
        // Notify all connected DApps
        for (const conn of state.getAllConnections()) {
          notifyContentScript(conn.origin, {
            type: 'CHAIN_CHANGED',
            payload: { chainId: message.payload.chainId }
          });
        }
        return { success: true };
        
      // Personal signing
      case 'SIGN_MESSAGE':
        return { success: true, data: { id: generateId() } };
        
      // Token/balance queries
      case 'GET_BALANCE':
        const wallet = state.getWallet();
        return { 
          success: true, 
          data: wallet.balance[message.payload.chain] || '0' 
        };
        
      case 'GET_ADDRESS':
        const addr = state.getWallet();
        return { 
          success: true, 
          data: addr.addresses[message.payload.chain] || '' 
        };
        
      default:
        return { success: false, error: `Unknown message type: ${message.type}` };
    }
  } catch (error) {
    console.error('Message handling error:', error);
    return { success: false, error: String(error) };
  }
}

function notifyContentScript(origin: string, message: Message) {
  chrome.tabs.query({ url: origin + '*' }, (tabs) => {
    for (const tab of tabs) {
      if (tab.id) {
        chrome.tabs.sendMessage(tab.id, message).catch(() => {});
      }
    }
  });
}

async function showTransactionApproval(tx: TransactionRequest) {
  // Create a popup or notification for transaction approval
  // For now, just log
  console.log('Transaction approval requested:', tx);
}

// ============================================================================
// Utility Functions
// ============================================================================

function generateId(): string {
  return `tx_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
}

// ============================================================================
// Event Listeners
// ============================================================================

// Handle messages from popup and content scripts
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  handleMessage(message, sender).then(sendResponse);
  return true; // Keep channel open for async response
});

// Handle tab updates - inject content script
chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
  if (changeInfo.status === 'complete' && tab.url) {
    // Check if this is a DApp we should inject into
    checkAndInjectDApp(tabId, tab.url);
  }
});

// Handle navigation - update connection status
chrome.webNavigation.onCompleted.addListener((details) => {
  if (details.frameId === 0) {
    // Top-level navigation completed
    console.log('Navigation completed:', details.url);
  }
});

// Handle install/update
chrome.runtime.onInstalled.addListener((details) => {
  if (details.reason === 'install') {
    // First install - initialize default state
    console.log('TigerWallet extension installed');
    
    // Set default chain
    state.setCurrentChain('ethereum');
    
  } else if (details.reason === 'update') {
    // Extension was updated
    console.log('TigerWallet extension updated to version:', chrome.runtime.getManifest().version);
  }
});

// Handle startup
chrome.runtime.onStartup.addListener(() => {
  console.log('TigerWallet extension starting...');
  state.loadState();
});

// ============================================================================
// Context Menu
// ============================================================================

function setupContextMenus() {
  // Create context menu items
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
  
  // Handle context menu clicks
  chrome.contextMenus?.onClicked.addListener((info, tab) => {
    if (info.menuItemId === 'tiger-send' && info.selectionText) {
      // Open send dialog with selected text as amount/address
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

// Update balances periodically
chrome.alarms.create('updateBalances', { periodInMinutes: 5 });

chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === 'updateBalances') {
    // In real implementation, fetch latest balances from RPC
    console.log('Updating balances...');
  }
});

// Cleanup expired transactions
chrome.alarms.create('cleanupTransactions', { periodInMinutes: 10 });

chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === 'cleanupTransactions') {
    const now = Date.now();
    const maxAge = 5 * 60 * 1000; // 5 minutes
    
    for (const [id, tx] of state.getAllTransactions()) {
      if (now - tx.timestamp > maxAge) {
        state.removeTransaction(id);
      }
    }
  }
});

// ============================================================================
// Export
// ============================================================================

export {};
