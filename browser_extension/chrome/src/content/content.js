// TigerWallet Content Script - Web3 Provider Injection
// Implements EIP-1193 Ethereum Provider for DApp interaction

(function() {
  'use strict';
  
  // ============================================================================
  // CONFIGURATION
  // ============================================================================
  
  const PROVIDER_INFO = {
    name: 'TigerWallet',
    iconURL: 'icons/icon128.png',
    uuid: 'tigerwallet-provider',
  };
  
  // ============================================================================
  // PROVIDER STATE
  // ============================================================================
  
  let providerState = {
    isConnected: false,
    chainId: null,
    accounts: [],
    isMetaMask: true, // For compatibility
    isTigerWallet: true,
    isUnlock: undefined,
  };
  
  let eventListeners = new Map();
  
  // ============================================================================
  // EIP-1193 PROVIDER IMPLEMENTATION
  // ============================================================================
  
  const ethereum = {
    // Provider identification
    isMetaMask: true,
    isTigerWallet: true,
    isConnected: () => providerState.isConnected,
    chainId: null, // Will be getter
    accounts: null, // Will be getter
    
    // Request - main entry point
    async request(args) {
      if (!args || !args.method) {
        throw new Error('args.method is required');
      }
      
      // Handle sync methods
      if (args.method === 'eth_chainId') {
        return await this._rpcRequest(args.method, []);
      }
      
      if (args.method === 'net_version') {
        const chainId = await this._rpcRequest('eth_chainId', []);
        return parseInt(chainId, 16).toString();
      }
      
      if (args.method === 'eth_accounts') {
        return await this._rpcRequest('eth_accounts', []);
      }
      
      if (args.method === 'eth_coinbase') {
        const accounts = await this._rpcRequest('eth_accounts', []);
        return accounts[0] || null;
      }
      
      if (args.method === 'eth_uninstallFilter') {
        return await this._rpcRequest(args.method, args.params || []);
      }
      
      if (args.method.startsWith('eth_') || args.method.startsWith('web3_')) {
        return await this._rpcRequest(args.method, args.params || []);
      }
      
      // Personal sign
      if (args.method === 'personal_sign') {
        const [message, address] = args.params;
        return await this._rpcRequest('personal_sign', [message, address]);
      }
      
      // Typed data sign
      if (args.method === 'eth_signTypedData_v4' || args.method === 'eth_signTypedData') {
        const [address, typedData] = args.params;
        return await this._rpcRequest('eth_signTypedData_v4', [address, typedData]);
      }
      
      // Wallet methods
      if (args.method === 'wallet_switchEthereumChain') {
        return await this._switchChain(args.params[0].chainId);
      }
      
      if (args.method === 'wallet_addEthereumChain') {
        return await this._addChain(args.params[0]);
      }
      
      if (args.method === 'wallet_requestPermissions') {
        return await this._requestPermissions(args.params[0]);
      }
      
      if (args.method === 'wallet_getPermissions') {
        return await this._getPermissions();
      }
      
      // Unknown method
      throw new Error(`Unknown method: ${args.method}`);
    },
    
    // Internal RPC request handler
    async _rpcRequest(method, params) {
      try {
        const response = await sendMessageToBackground({
          type: 'RPC_REQUEST',
          method,
          params,
        });
        
        if (!response.success) {
          throw new Error(response.error);
        }
        
        return response.data;
      } catch (error) {
        console.error('RPC request failed:', error);
        throw error;
      }
    },
    
    // Switch chain
    async _switchChain(chainId) {
      try {
        const response = await sendMessageToBackground({
          type: 'SWITCH_CHAIN',
          chainId,
        });
        
        if (!response.success) {
          throw new Error(response.error);
        }
        
        this._emit('chainChanged', chainId);
        return null;
      } catch (error) {
        throw error;
      }
    },
    
    // Add chain
    async _addChain(chainConfig) {
      try {
        const response = await sendMessageToBackground({
          type: 'ADD_CHAIN',
          chainConfig,
        });
        
        if (!response.success) {
          throw new Error(response.error);
        }
        
        this._emit('chainChanged', chainConfig.chainId);
        return null;
      } catch (error) {
        throw error;
      }
    },
    
    // Request permissions
    async _requestPermissions(permissions) {
      try {
        const response = await sendMessageToBackground({
          type: 'REQUEST_PERMISSIONS',
          origin: window.location.origin,
          permissions,
        });
        
        if (!response.success) {
          throw new Error(response.error);
        }
        
        return response.data;
      } catch (error) {
        throw error;
      }
    },
    
    // Get permissions
    async _getPermissions() {
      try {
        const response = await sendMessageToBackground({
          type: 'GET_PERMISSIONS',
          origin: window.location.origin,
        });
        
        return response.data || [];
      } catch (error) {
        return [];
      }
    },
    
    // Event emitter methods
    on(event, listener) {
      this._addListener(event, listener);
      return this;
    },
    
    once(event, listener) {
      const wrappedListener = (data) => {
        listener(data);
        this.removeListener(event, wrappedListener);
      };
      this._addListener(event, wrappedListener);
      return this;
    },
    
    removeListener(event, listener) {
      if (!eventListeners.has(event)) return;
      
      const listeners = eventListeners.get(event);
      listeners.delete(listener);
    },
    
    removeAllListeners(event) {
      if (event) {
        eventListeners.delete(event);
      } else {
        eventListeners.clear();
      }
    },
    
    _addListener(event, listener) {
      if (!eventListeners.has(event)) {
        eventListeners.set(event, new Set());
      }
      eventListeners.get(event).add(listener);
    },
    
    _emit(event, data) {
      if (!eventListeners.has(event)) return;
      
      eventListeners.get(event).forEach(listener => {
        try {
          listener(data);
        } catch (error) {
          console.error('Event listener error:', error);
        }
      });
    },
    
    // Aliases for compatibility
    enable: async function() {
      return await this.request({ method: 'eth_requestAccounts' });
    },
    
    send: async function(methodOrPayload, paramsOrCallback) {
      // Handle deprecated send method
      if (typeof methodOrPayload === 'string') {
        return await this.request({
          method: methodOrPayload,
          params: paramsOrCallback || [],
        });
      }
      
      // Handle payload object
      if (typeof methodOrPayload === 'object') {
        try {
          const result = await this.request(methodOrPayload);
          if (typeof paramsOrCallback === 'function') {
            paramsOrCallback(null, { result });
          }
          return { result };
        } catch (error) {
          if (typeof paramsOrCallback === 'function') {
            paramsOrCallback(error, null);
          }
          throw error;
        }
      }
    },
    
    sendAsync: function(payload, callback) {
      this.request(payload)
        .then(result => callback(null, { id: payload.id, jsonrpc: '2.0', result }))
        .catch(error => callback(error, null));
    },
    
    // Getter properties
    get chainId() {
      return providerState.chainId;
    },
    
    get accounts() {
      return providerState.accounts;
    },
    
    get networkVersion() {
      return parseInt(providerState.chainId, 16).toString();
    },
    
    get selectedAddress() {
      return providerState.accounts?.[0] || null;
    },
  };
  
  // ============================================================================
  // MESSAGE PASSING
  // ============================================================================
  
  function sendMessageToBackground(message) {
    return new Promise((resolve, reject) => {
      chrome.runtime.sendMessage(message, response => {
        if (chrome.runtime.lastError) {
          reject(new Error(chrome.runtime.lastError.message));
        } else {
          resolve(response);
        }
      });
    });
  }
  
  // ============================================================================
  // PROVIDER INJECTION
  // ============================================================================
  
  // Remove any existing providers
  const existingProvider = window.ethereum;
  const existingProviders = window.providers || [];
  
  // Define our provider
  Object.defineProperty(window, 'ethereum', {
    value: ethereum,
    writable: false,
    configurable: false,
  });
  
  // Also expose providers array for compatibility
  window.providers = [...existingProviders, ethereum];
  
  // Also expose as window.tigerwallet
  window.tigerwallet = {
    provider: ethereum,
    ...PROVIDER_INFO,
  };
  
  // ============================================================================
  // PROVIDER CHAIN DETECTION & SYNC
  // ============================================================================
  
  async function initializeProvider() {
    try {
      // Get initial chain and accounts
      const [chainId, accounts] = await Promise.all([
        ethereum.request({ method: 'eth_chainId' }),
        ethereum.request({ method: 'eth_accounts' }).catch(() => []),
      ]);
      
      providerState.chainId = chainId;
      providerState.accounts = accounts;
      providerState.isConnected = accounts.length > 0;
      
      // Emit connect event
      ethereum._emit('connect', { chainId });
      
      // Emit chainChanged if already connected to a chain
      if (chainId) {
        ethereum._emit('chainChanged', chainId);
      }
      
      // Emit accountsChanged if we have accounts
      if (accounts.length > 0) {
        ethereum._emit('accountsChanged', accounts);
      }
    } catch (error) {
      console.error('Failed to initialize provider:', error);
    }
  }
  
  // Listen for chain/account changes from background
  chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.type === 'CHAIN_CHANGED') {
      providerState.chainId = message.chainId;
      ethereum._emit('chainChanged', message.chainId);
    }
    
    if (message.type === 'ACCOUNTS_CHANGED') {
      providerState.accounts = message.accounts;
      providerState.isConnected = message.accounts.length > 0;
      ethereum._emit('accountsChanged', message.accounts);
    }
    
    if (message.type === 'DISCONNECT') {
      providerState.isConnected = false;
      ethereum._emit('disconnect', message.error);
    }
  });
  
  // Initialize
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initializeProvider);
  } else {
    initializeProvider();
  }
  
  // ============================================================================
  // PAGE RELOAD HANDLING
  // ============================================================================
  
  // Re-inject on page navigation (for SPAs)
  let lastUrl = location.href;
  const observer = new MutationObserver(() => {
    if (location.href !== lastUrl) {
      lastUrl = location.href;
      // Re-initialize on URL change
      initializeProvider();
    }
  });
  
  observer.observe(document.body, { childList: true, subtree: true });
  
  console.log('TigerWallet Web3 Provider Injected');
})();
