/**
 * TigerWallet Chrome Extension - Content Script
 * 
 * PRODUCTION-READY - Injects Web3 provider and handles DApp communication
 */

(function() {
  'use strict';

  // Prevent multiple injections
  if (window.tigerWalletInjected) return;
  window.tigerWalletInjected = true;

  // Configuration
  const PROVIDER_NAME = 'TigerWallet';
  const PROVIDER_VERSION = '1.0.0';

  // Message handling
  const messageListeners = new Map();
  let messageId = 0;

  // Send message to background script
  function sendMessage(type, data = {}) {
    return new Promise((resolve, reject) => {
      const id = ++messageId;
      const listener = (response) => {
        if (response.id === id) {
          messageListeners.delete(id);
          clearTimeout(timeout);
          if (response.error) {
            reject(new Error(response.error));
          } else {
            resolve(response);
          }
        }
      };
      messageListeners.set(id, listener);

      const timeout = setTimeout(() => {
        messageListeners.delete(id);
        reject(new Error('Message timeout'));
      }, 30000);

      chrome.runtime.sendMessage({ type, data, id });
    });
  }

  // Listen for messages from background
  chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (messageListeners.has(message.id)) {
      messageListeners.get(message.id)(message);
    }
    
    // Handle DApp events
    if (message.type === 'DAPP_CONNECTED' || message.type === 'DAPP_DISCONNECTED') {
      window.dispatchEvent(new Event('tigerWallet_update'));
    }
  });

  // =========================================================================
  // EIP-1193 Provider Implementation
  // =========================================================================

  class TigerWalletProvider {
    constructor() {
      this.isTigerWallet = true;
      this.isMetaMask = true; // For compatibility
      this.isConnected = false;
      this.chainId = '0x1';
      this.selectedAddress = null;
      this._events = {};
    }

    // Connection status
    async isConnected() {
      try {
        const accounts = await this.request({ method: 'eth_accounts' });
        return accounts && accounts.length > 0;
      } catch (e) {
        return false;
      }
    }

    // Request handler (EIP-1193)
    async request(args) {
      const { method, params = [] } = args;

      try {
        // Wallet methods
        if (method === 'eth_requestAccounts' || method === 'eth_accounts') {
          const response = await sendMessage('GET_ACCOUNTS');
          this.selectedAddress = response[0] || null;
          this.isConnected = !!this.selectedAddress;
          return response;
        }

        if (method === 'eth_chainId') {
          const state = await sendMessage('GET_STATE');
          return state.network || '0x1';
        }

        if (method === 'net_version') {
          const state = await sendMessage('GET_STATE');
          const network = state.network || 'ethereum';
          const networkIds = {
            ethereum: '1',
            sepolia: '11155111',
            bsc: '56',
            polygon: '137',
            arbitrum: '42161',
            optimism: '10',
            base: '8453',
            avalanche: '43114'
          };
          return networkIds[network] || '1';
        }

        // Balance
        if (method === 'eth_getBalance') {
          const response = await sendMessage('GET_BALANCE', {
            network: await this._getNetworkName()
          });
          return response;
        }

        // Transaction
        if (method === 'eth_sendTransaction') {
          const [tx] = params;
          const response = await sendMessage('SEND_TRANSACTION', {
            to: tx.to,
            value: tx.value || '0',
            data: tx.data || '0x',
            network: await this._getNetworkName()
          });
          return response.hash;
        }

        // Signing
        if (method === 'personal_sign') {
          const [message, address] = params;
          const response = await sendMessage('SIGN_MESSAGE', { message });
          return response;
        }

        if (method === 'eth_signTypedData_v4' || method === 'eth_signTypedData') {
          const [address, data] = params;
          const response = await sendMessage('SIGN_TYPED_DATA', { data: JSON.parse(data) });
          return response;
        }

        // Chain switching
        if (method === 'wallet_switchEthereumChain' || method === 'wallet_addEthereumChain') {
          const [chainConfig] = params;
          const chainId = chainConfig.chainId;
          
          const networkMap = {
            '0x1': 'ethereum',
            '0xaa36a7': 'sepolia',
            '0x38': 'bsc',
            '0x89': 'polygon',
            '0xa4b1': 'arbitrum',
            '0xa': 'optimism',
            '0x2105': 'base',
            '0xa86a': 'avalanche'
          };
          
          const network = networkMap[chainId];
          if (network) {
            await sendMessage('SWITCH_NETWORK', { network });
            this.chainId = chainId;
            this._emit('chainChanged', chainId);
            return null;
          }
        }

        // Default: forward to background
        const response = await sendMessage('RPC_REQUEST', { method, params });
        return response;

      } catch (error) {
        console.error('TigerWallet request error:', error);
        throw error;
      }
    }

    async _getNetworkName() {
      const networkMap = {
        '0x1': 'ethereum',
        '0xaa36a7': 'sepolia',
        '0x38': 'bsc',
        '0x89': 'polygon',
        '0xa4b1': 'arbitrum',
        '0xa': 'optimism',
        '0x2105': 'base',
        '0xa86a': 'avalanche'
      };
      return networkMap[this.chainId] || 'ethereum';
    }

    // Event emitters (EIP-1193)
    on(event, listener) {
      if (!this._events[event]) {
        this._events[event] = [];
      }
      this._events[event].push(listener);
    }

    removeListener(event, listener) {
      if (!this._events[event]) return;
      this._events[event] = this._events[event].filter(l => l !== listener);
    }

    emit(event, data) {
      if (!this._events[event]) return;
      this._events[event].forEach(listener => {
        try {
          listener(data);
        } catch (e) {
          console.error('Event listener error:', e);
        }
      });
    }

    // Legacy events
    addListener(event, listener) {
      this.on(event, listener);
    }

    // EIP-1193 events
    _emit(event, data) {
      this.emit(event, data);
    }

    // Initialize
    async _init() {
      try {
        // Check connection
        const accounts = await this.request({ method: 'eth_accounts' });
        if (accounts && accounts.length > 0) {
          this.selectedAddress = accounts[0];
          this.isConnected = true;
          this._emit('connect', { chainId: this.chainId });
        }

        // Listen for changes
        window.addEventListener('tigerWallet_update', async () => {
          const accounts = await this.request({ method: 'eth_accounts' });
          if (accounts && accounts.length > 0) {
            this.selectedAddress = accounts[0];
            this.isConnected = true;
            this._emit('accountsChanged', accounts);
          } else {
            this.selectedAddress = null;
            this.isConnected = false;
            this._emit('accountsChanged', []);
            this._emit('disconnect', { code: 1000, message: 'Disconnected' });
          }
        });

      } catch (e) {
        console.error('TigerWallet init error:', e);
      }
    }
  }

  // Create and inject provider
  const provider = new TigerWalletProvider();
  provider._init();

  // Inject provider
  Object.defineProperty(window, 'ethereum', {
    get: () => provider,
    configurable: true
  });

  // Legacy compatibility
  window.tigerWallet = provider;
  window.web3 = {
    currentProvider: provider
  };

  console.log('TigerWallet injected successfully');

})();
