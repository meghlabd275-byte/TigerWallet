/**
 * TigerWallet Browser Extension - Content Script
 * Injected into DApp pages to provide Web3 functionality
 */

(function() {
  'use strict';

  // ============================================================================
  // Configuration
  // ============================================================================

  const CHAIN_IDS: Record<string, string> = {
    ethereum: '0x1',
    polygon: '0x89',
    bsc: '0x38',
    arbitrum: '0xa4b1',
    optimism: '0xa',
    avalanche: '0xa86a',
    base: '0x2105',
    zksync: '0x144',
    linea: '0xe708',
    scroll: '0x82750'
  };

  // ============================================================================
  // State
  // ============================================================================

  let walletState = {
    isConnected: false,
    chainId: CHAIN_IDS.ethereum,
    accounts: [] as string[],
    selectedAddress: ''
  };

  let eventListeners: Map<string, Set<Function>> = new Map();

  // ============================================================================
  // Utility Functions
  // ============================================================================

  function generateId(): string {
    if (!globalThis.crypto || typeof globalThis.crypto.randomUUID !== 'function') {
      throw new Error('Secure request ID generation is unavailable');
    }
    return `req_${globalThis.crypto.randomUUID()}`;
  }

  function notifyBackground(message: any): Promise<any> {
    return new Promise((resolve, reject) => {
      chrome.runtime.sendMessage(message, (response) => {
        if (response?.success) {
          resolve(response.data);
        } else {
          reject(new Error(response?.error || 'Request failed'));
        }
      });
    });
  }

  function emit(event: string, data: any) {
    const listeners = eventListeners.get(event);
    if (listeners) {
      listeners.forEach(listener => {
        try {
          listener(data);
        } catch (error) {
          console.error('Event listener error:', error);
        }
      });
    }
  }

  // ============================================================================
  // Ethereum Provider API
  // ============================================================================

  class TigerWalletProvider {
    public isTigerWallet = true;
    public isMetaMask = true; // For DApp compatibility
    public isConnected = () => walletState.isConnected;
    public chainId = () => walletState.chainId;
    public selectedAddress = () => walletState.selectedAddress;
    
    private _events: Map<string, Set<Function>> = new Map();

    async request(args: { method: string; params?: any[] }): Promise<any> {
      const id = generateId();

      try {
        const response = await notifyBackground({
          type: 'INTERNAL_REQUEST',
          payload: {
            id,
            method: args.method,
            params: args.params || [],
            origin: window.location.origin,
            chainId: walletState.chainId
          }
        });

        return response;
      } catch (error) {
        console.error('Provider request error:', error);
        throw error;
      }
    }

    on(event: string, listener: Function): void {
      if (!this._events.has(event)) {
        this._events.set(event, new Set());
      }
      this._events.get(event)!.add(listener);
    }

    removeListener(event: string, listener: Function): void {
      this._events.get(event)?.delete(listener);
    }

    emit(event: string, ...args: any[]): void {
      this._events.get(event)?.forEach(listener => {
        try {
          listener(...args);
        } catch (error) {
          console.error('Event emission error:', error);
        }
      });
    }

    // Compatibility methods
    enable(): Promise<string[]> {
      return this.request({ method: 'eth_requestAccounts' });
    }

    async send(method: string, params?: any[]): Promise<any> {
      return this.request({ method, params });
    }

    async sendAsync(payload: any, callback: (err: any, result: any) => void): Promise<void> {
      try {
        const result = await this.request(payload);
        callback(null, { id: payload.id, jsonrpc: '2.0', result });
      } catch (error) {
        callback(error, null);
      }
    }

    // Event emitter compatibility
    addListener(event: string, listener: Function): void {
      this.on(event, listener);
    }

    removeAllListeners(event?: string): void {
      if (event) {
        this._events.delete(event);
      } else {
        this._events.clear();
      }
    }

    listenerCount(event?: string): number {
      if (event) {
        return this._events.get(event)?.size || 0;
      }
      let count = 0;
      this._events.forEach(set => count += set.size);
      return count;
    }
  }

  // Create provider instance
  const provider = new TigerWalletProvider();

  // ============================================================================
  // Message Handling from Background
  // ============================================================================

  function handleBackgroundMessage(message: any) {
    switch (message.type) {
      case 'CONNECTION_CHANGED':
        walletState.isConnected = message.payload.connected;
        walletState.accounts = message.payload.accounts || [];
        walletState.selectedAddress = walletState.accounts[0] || '';
        
        provider.emit('connect', { chainId: walletState.chainId });
        provider.emit('accountsChanged', walletState.accounts);
        emit('accountsChanged', walletState.accounts);
        break;

      case 'CHAIN_CHANGED':
        walletState.chainId = message.payload.chainId;
        
        provider.emit('chainChanged', message.payload.chainId);
        emit('chainChanged', message.payload.chainId);
        break;

      case 'TRANSACTION_RESULT':
        provider.emit('transactionResult', message.payload);
        emit('transactionResult', message.payload);
        break;
    }
  }

  // Listen for messages from background
  chrome.runtime?.onMessage?.addListener((message, sender, sendResponse) => {
    handleBackgroundMessage(message);
    sendResponse({ received: true });
  });

  // ============================================================================
  // Initialize Provider
  // ============================================================================

  function initializeProvider() {
    // Check if provider already exists
    const existingProvider = (window as any).ethereum;
    
    if (existingProvider) {
      // Already has a provider, add TigerWallet as secondary
      (window as any).tigerwallet = provider;
    } else {
      // Set as primary provider
      (window as any).ethereum = provider;
      (window as any).web3 = {
        currentProvider: provider
      };
    }

    // Also expose on window for direct access
    (window as any).tigerwallet = provider;

    console.log('TigerWallet provider initialized');
  }

  // ============================================================================
  // Auto-connect if previously connected
  // ============================================================================

  async function checkConnection() {
    try {
      const connections = await notifyBackground({
        type: 'GET_CONNECTIONS'
      });

      const currentConnection = connections?.find(
        (c: any) => c.origin === window.location.origin
      );

      if (currentConnection?.connected) {
        walletState.isConnected = true;
        walletState.accounts = currentConnection.accounts || [];
        walletState.selectedAddress = walletState.accounts[0] || '';
        walletState.chainId = currentConnection.chainId || CHAIN_IDS.ethereum;

        // Emit connect event
        provider.emit('connect', { chainId: walletState.chainId });
      }
    } catch (error) {
      console.error('Connection check failed:', error);
    }
  }

  // ============================================================================
  // Initialize
  // ============================================================================

  function init() {
    initializeProvider();
    checkConnection();
  }

  // Run on DOM ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  // Also run on frame load
  if (window.frameElement) {
    init();
  }

})();
