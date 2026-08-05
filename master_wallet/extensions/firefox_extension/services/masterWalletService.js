/**
 * MasterWalletService - Chrome Extension
 * Complete wallet management for Master Wallet Extension
 */

const API_BASE = 'https://master-api.tigerwallet.com/api/v1';

const CHAIN_CONFIGS = {
  1: { name: 'Ethereum', symbol: 'ETH', rpcUrl: 'https://eth.llamarpc.com', decimals: 18 },
  56: { name: 'BNB Smart Chain', symbol: 'BNB', rpcUrl: 'https://bsc-dataseed.binance.org', decimals: 18 },
  137: { name: 'Polygon', symbol: 'MATIC', rpcUrl: 'https://polygon-rpc.com', decimals: 18 },
  42161: { name: 'Arbitrum One', symbol: 'ETH', rpcUrl: 'https://arb1.arbitrum.io/rpc', decimals: 18 },
};

class MasterWalletService {
  constructor() {
    this.wallets = new Map();
    this.currentWalletId = null;
  }

  /**
   * Generate a new HD wallet
   */
  async generateWallet(password) {
    try {
      const response = await fetch(`${API_BASE}/wallets/generate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password }),
      });

      const data = await response.json();
      
      if (data.wallet_id) {
        this.wallets.set(data.wallet_id, {
          id: data.wallet_id,
          address: data.address,
          mnemonic: data.mnemonic,
        });
        this.currentWalletId = data.wallet_id;
        
        // Store in extension storage
        await this.storeWallet(data.wallet_id, data.address, data.mnemonic);
      }

      return data;
    } catch (error) {
      return { success: false, error: error.message };
    }
  }

  /**
   * Import existing wallet
   */
  async importWallet(mnemonic, password) {
    try {
      const response = await fetch(`${API_BASE}/wallets/import`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mnemonic, password }),
      });

      const data = await response.json();
      
      if (data.wallet_id) {
        this.wallets.set(data.wallet_id, {
          id: data.wallet_id,
          address: data.address,
          mnemonic: mnemonic,
        });
        this.currentWalletId = data.wallet_id;
        
        await this.storeWallet(data.wallet_id, data.address, mnemonic);
      }

      return data;
    } catch (error) {
      return { success: false, error: error.message };
    }
  }

  /**
   * Get balance
   */
  async getBalance(walletId, chainId = 1) {
    try {
      const response = await fetch(`${API_BASE}/wallets/${walletId}/balance?chain_id=${chainId}`);
      return await response.json();
    } catch (error) {
      return { success: false, error: error.message };
    }
  }

  /**
   * Send transaction
   */
  async sendTransaction(walletId, chainId, toAddress, amount) {
    try {
      const response = await fetch(`${API_BASE}/wallets/${walletId}/send`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          chain_id: chainId,
          to_address: toAddress,
          amount: amount,
        }),
      });

      return await response.json();
    } catch (error) {
      return { success: false, error: error.message };
    }
  }

  /**
   * Get supported chains
   */
  getSupportedChains() {
    return Object.entries(CHAIN_CONFIGS).map(([id, config]) => ({
      id: Number(id),
      ...config,
    }));
  }

  /**
   * Get current wallet address
   */
  getCurrentAddress() {
    if (!this.currentWalletId) return null;
    const wallet = this.wallets.get(this.currentWalletId);
    return wallet ? wallet.address : null;
  }

  /**
   * Store wallet in extension storage
   */
  async storeWallet(walletId, address, mnemonic) {
    return new Promise((resolve) => {
      chrome.storage.local.set({
        [`wallet_${walletId}`]: { address, mnemonic },
        currentWalletId: walletId,
      }, resolve);
    });
  }

  /**
   * Load wallet from storage
   */
  async loadWallets() {
    return new Promise((resolve) => {
      chrome.storage.local.get(null, (items) => {
        for (const [key, value] of Object.entries(items)) {
          if (key.startsWith('wallet_')) {
            const walletId = key.replace('wallet_', '');
            this.wallets.set(walletId, { id: walletId, ...value });
          }
        }
        this.currentWalletId = items.currentWalletId || null;
        resolve();
      });
    });
  }
}

// Export for use in extension
const masterWalletService = new MasterWalletService();

// Message handler for popup
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  switch (message.type) {
    case 'GENERATE_WALLET':
      masterWalletService.generateWallet(message.password).then(sendResponse);
      return true;

    case 'IMPORT_WALLET':
      masterWalletService.importWallet(message.mnemonic, message.password).then(sendResponse);
      return true;

    case 'GET_BALANCE':
      masterWalletService.getBalance(message.walletId, message.chainId).then(sendResponse);
      return true;

    case 'SEND_TRANSACTION':
      masterWalletService.sendTransaction(message.walletId, message.chainId, message.toAddress, message.amount).then(sendResponse);
      return true;

    case 'GET_CHAINS':
      sendResponse(masterWalletService.getSupportedChains());
      return false;

    case 'GET_ADDRESS':
      sendResponse(masterWalletService.getCurrentAddress());
      return false;

    case 'LOAD_WALLETS':
      masterWalletService.loadWallets().then(() => {
        sendResponse({ success: true, wallets: Array.from(masterWalletService.wallets.values()) });
      });
      return true;
  }
});
