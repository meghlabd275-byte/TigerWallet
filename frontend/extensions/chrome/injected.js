// TigerWallet Injected Web3 Provider
// Provides Ethereum-compatible Web3 interface to DApps

(function() {
  // Prevent multiple injections
  if (window.tigerwalletInjected) return;
  window.tigerwalletInjected = true;

  // TigerWallet Provider API
  class TigerWalletProvider {
    constructor() {
      this.isTigerWallet = true;
      this.isMetaMask = false; // Compatibility
      this.isConnected = false;
      this.chainId = null;
      this.networkVersion = null;
      this.selectedAddress = null;
      
      this._events = {};
      this._requestId = 0;
      this._callbacks = new Map();
    }

    // Event emitter methods
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

    emit(event, ...args) {
      if (!this._events[event]) return;
      this._events[event].forEach(listener => listener(...args));
    }

    // Request handler - communicates with background script
    async request(args) {
      const id = ++this._requestId;
      
      return new Promise((resolve, reject) => {
        this._callbacks.set(id, { resolve, reject });
        
        chrome.runtime.sendMessage({
          id,
          type: args.method,
          payload: args.params
        }, (response) => {
          if (chrome.runtime.lastError) {
            reject(new Error(chrome.runtime.lastError.message));
            return;
          }
          
          if (response && response.error) {
            reject(new Error(response.error));
          } else {
            resolve(response);
          }
          this._callbacks.delete(id);
        });

        // Timeout
        setTimeout(() => {
          if (this._callbacks.has(id)) {
            this._callbacks.delete(id);
            reject(new Error('Request timeout'));
          }
        }, 30000);
      });
    }

    // Legacy methods for compatibility
    async enable() {
      return this.request({ method: 'eth_requestAccounts' });
    }

    async send(methodOrPayload, paramsOrCallback) {
      if (typeof methodOrPayload === 'string') {
        return this.request({ method: methodOrPayload, params: paramsOrCallback || [] });
      }
      return this.request(methodOrPayload);
    }

    sendAsync(payload, callback) {
      this.request(payload)
        .then(result => callback(null, { result }))
        .catch(error => callback(error, null));
    }

    // EIP-1193 events
    onaccountsChanged(accounts) {
      this.selectedAddress = accounts[0] || null;
      this.emit('accountsChanged', accounts);
    }

    onchainChanged(chainId) {
      this.chainId = chainId;
      this.emit('chainChanged', chainId);
    }

    onconnect(info) {
      this.isConnected = true;
      this.emit('connect', info);
    }

    ondisconnect(error) {
      this.isConnected = false;
      this.emit('disconnect', error);
    }

    // Message handler from background
    _handleMessage(message) {
      switch (message.type) {
        case 'WALLET_STATE_CHANGED':
          if (message.state.address) {
            this.selectedAddress = message.state.address;
            this.isConnected = true;
            this.emit('accountsChanged', [message.state.address]);
            this.emit('connect', { chainId: message.state.network.chainId });
          } else {
            this.selectedAddress = null;
            this.isConnected = false;
            this.emit('accountsChanged', []);
            this.emit('disconnect', { code: 1000, message: 'Wallet locked' });
          }
          break;
          
        case 'NETWORK_CHANGED':
          this.chainId = message.network.chainId;
          this.emit('chainChanged', message.network.chainId);
          break;
      }
    }
  }

  // Create and inject provider
  const provider = new TigerWalletProvider();

  // Listen for messages from background
  chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    provider._handleMessage(message);
  });

  // Expose provider
  window.ethereum = provider;
  window.tigerwallet = provider;

  // Inject for backwards compatibility
  Object.defineProperty(window, 'web3', {
    get: () => provider,
    configurable: true
  });

  // Request account access on first DApp connection
  provider.request({ method: 'eth_requestAccounts' })
    .catch(() => {});

  console.log('TigerWallet injected');
})();
