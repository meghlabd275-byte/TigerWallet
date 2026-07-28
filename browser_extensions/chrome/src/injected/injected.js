/**
 * TigerWallet - Injected Provider (EIP-1193)
 * Real Ethereum Provider Implementation
 */

(function() {
  // Prevent multiple injections
  if (window.tigerWalletProvider) {
    return;
  }

  // ========================================
  // Provider State
  // ========================================
  
  let chainId = '0x1'; // Ethereum mainnet
  let selectedAddress = null;
  let isConnected = false;
  let isUnlocked = false;
  const listeners = new Map();

  // ========================================
  // Message Passing
  // ========================================

  function sendMessage(message) {
    return new Promise((resolve, reject) => {
      const id = Date.now() + Math.random();
      
      // Store callbacks
      const responseCallback = (event) => {
        if (event.data.id === id && event.data.target === 'injected') {
          window.removeEventListener('message', responseCallback);
          if (event.data.error) {
            reject(new Error(event.data.error));
          } else {
            resolve(event.data.result);
          }
        }
      };
      
      window.addEventListener('message', responseCallback);
      
      // Send to background or popup
      window.postMessage({
        id,
        target: 'tiger-wallet',
        ...message
      }, '*');
      
      // Timeout after 30 seconds
      setTimeout(() => {
        window.removeEventListener('message', responseCallback);
        reject(new Error('Request timeout'));
      }, 30000);
    });
  }

  // ========================================
  // Provider Methods (EIP-1193)
  // ========================================

  async function request(args) {
    const { method, params = [] } = args;
    
    switch (method) {
      case 'eth_requestAccounts':
      case 'eth_accounts':
        return await getAccounts();
        
      case 'eth_chainId':
        return chainId;
        
      case 'net_version':
        return parseInt(chainId, 16).toString();
        
      case 'eth_blockNumber':
        return await sendMessage({ method: 'eth_blockNumber', params });
        
      case 'eth_getBalance':
        return await sendMessage({ method: 'eth_getBalance', params });
        
      case 'eth_getTransactionCount':
        return await sendMessage({ method: 'eth_getTransactionCount', params });
        
      case 'eth_call':
        return await sendMessage({ method: 'eth_call', params });
        
      case 'eth_sendTransaction':
        return await sendMessage({ method: 'eth_sendTransaction', params });
        
      case 'eth_sendRawTransaction':
        return await sendMessage({ method: 'eth_sendRawTransaction', params });
        
      case 'eth_getTransactionReceipt':
        return await sendMessage({ method: 'eth_getTransactionReceipt', params });
        
      case 'eth_estimateGas':
        return await sendMessage({ method: 'eth_estimateGas', params });
        
      case 'eth_gasPrice':
        return await sendMessage({ method: 'eth_gasPrice', params });
        
      case 'eth_getCode':
        return await sendMessage({ method: 'eth_getCode', params });
        
      case 'eth_getStorageAt':
        return await sendMessage({ method: 'eth_getStorageAt', params });
        
      case 'eth_getLogs':
        return await sendMessage({ method: 'eth_getLogs', params });
        
      case 'eth_getTransactionByHash':
        return await sendMessage({ method: 'eth_getTransactionByHash', params });
        
      // ERC-20 Methods
      case 'eth_call - erc20':
        return await sendMessage({ method: 'eth_call_erc20', params });
        
      // Wallet Methods
      case 'wallet_switchEthereumChain':
        return await switchChain(params[0]);
        
      case 'wallet_addEthereumChain':
        return await addChain(params[0]);
        
      case 'wallet_requestPermissions':
        return await requestPermissions(params[0]);
        
      // Personal Sign
      case 'personal_sign':
        return await personalSign(params[0], params[1]);
        
      case 'personal_ecRecover':
        return await personalRecover(params[0], params[1]);
        
      // Typed Data
      case 'eth_signTypedData_v4':
      case 'eth_signTypedData':
        return await signTypedData(params[0], params[1]);
        
      // Web3
      case 'web3_clientVersion':
        return 'TigerWallet/1.0.0';
        
      default:
        throw new Error(`Unknown method: ${method}`);
    }
  }

  async function getAccounts() {
    try {
      const accounts = await sendMessage({ method: 'eth_accounts' });
      selectedAddress = accounts && accounts.length > 0 ? accounts[0] : null;
      isConnected = !!selectedAddress;
      isUnlocked = !!selectedAddress;
      return accounts || [];
    } catch (error) {
      return [];
    }
  }

  async function switchChain(chainParams) {
    const chainIdHex = chainParams.chainId;
    chainId = chainIdHex;
    emit('chainChanged', chainId);
    return null;
  }

  async function addChain(chainConfig) {
    // Would save to local storage
    return null;
  }

  async function requestPermissions(permissions) {
    const result = await sendMessage({ 
      method: 'wallet_requestPermissions', 
      params: [permissions] 
    });
    return result;
  }

  async function personalSign(message, address) {
    return await sendMessage({ 
      method: 'personal_sign', 
      params: [message, address] 
    });
  }

  async function personalRecover(message, signature) {
    return await sendMessage({ 
      method: 'personal_ecRecover', 
      params: [message, signature] 
    });
  }

  async function signTypedData(domain, message) {
    return await sendMessage({ 
      method: 'eth_signTypedData_v4', 
      params: [selectedAddress, message] 
    });
  }

  // ========================================
  // Event Listeners (EIP-1193)
  // ========================================

  function on(event, listener) {
    if (!listeners.has(event)) {
      listeners.set(event, new Set());
    }
    listeners.get(event).add(listener);
    
    return () => {
      listeners.get(event)?.delete(listener);
    };
  }

  function emit(event, data) {
    if (listeners.has(event)) {
      listeners.get(event).forEach(listener => {
        try {
          listener(data);
        } catch (error) {
          console.error('Listener error:', error);
        }
      });
    }
  }

  // Remove listener (legacy)
  function removeListener(event, listener) {
    listeners.get(event)?.delete(listener);
  }

  // Remove all listeners (legacy)
  function removeAllListeners(event) {
    if (event) {
      listeners.delete(event);
    } else {
      listeners.clear();
    }
  }

  // ========================================
  // Provider Properties
  // ========================================

  const isMetaMask = true;
  const isTigerWallet = true;
  const isConnected = () => isConnected;
  const chainId = () => chainId;
  const selectedAddress = () => selectedAddress;

  // ========================================
  // Initialize Provider
  // ========================================

  const provider = {
    // EIP-1193
    request,
    isMetaMask,
    isTigerWallet,
    isConnected,
    chainId,
    selectedAddress,
    
    // Events
    on,
    emit,
    removeListener,
    removeAllListeners,
    
    // Legacy
    enable: () => request({ method: 'eth_requestAccounts' }),
    send: (methodOrPayload, paramsOrCallback) => {
      if (typeof methodOrPayload === 'string') {
        return request({ method: methodOrPayload, params: paramsOrCallback });
      }
      return request(methodOrPayload);
    },
    sendAsync: (payload, callback) => {
      request(payload)
        .then(result => callback(null, { id: payload.id, jsonrpc: '2.0', result }))
        .catch(error => callback(error, { id: payload.id, jsonrpc: '2.0', error: error.message }));
    },
    
    // Subscription (not fully implemented)
    subscribe: (type, filter, callback) => {
      console.warn('Subscriptions not yet implemented');
      return null;
    },
    unsubscribe: (subscriptionId, callback) => {
      console.warn('Subscriptions not yet implemented');
      return true;
    },
    
    // Chain & network
    chain: null,
    networkVersion: null,
    
    // Debug
    _TigerWallet: true,
  };

  // Make provider available globally
  window.ethereum = provider;
  window.tigerWalletProvider = provider;

  // Inject Banner
  injectBanner();
  
  function injectBanner() {
    const banner = document.createElement('div');
    banner.id = 'tiger-wallet-banner';
    banner.innerHTML = `
      <style>
        #tiger-wallet-banner {
          position: fixed;
          bottom: 20px;
          right: 20px;
          background: linear-gradient(135deg, #FF6B35 0%, #FF8C5A 100%);
          color: white;
          padding: 16px 24px;
          border-radius: 12px;
          font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
          font-size: 14px;
          box-shadow: 0 4px 20px rgba(255, 107, 53, 0.4);
          z-index: 999999;
          display: flex;
          align-items: center;
          gap: 12px;
          cursor: pointer;
          animation: slideIn 0.3s ease-out;
        }
        @keyframes slideIn {
          from { transform: translateY(100px); opacity: 0; }
          to { transform: translateY(0); opacity: 1; }
        }
        #tiger-wallet-banner .icon {
          font-size: 24px;
        }
        #tiger-wallet-banner .text {
          font-weight: 600;
        }
        #tiger-wallet-banner .close {
          background: rgba(255,255,255,0.2);
          border: none;
          color: white;
          width: 24px;
          height: 24px;
          border-radius: 50%;
          cursor: pointer;
          display: flex;
          align-items: center;
          justify-content: center;
          font-size: 14px;
        }
      </style>
      <span class="icon">🐯</span>
      <span class="text">TigerWallet Connected</span>
      <button class="close" onclick="this.parentElement.remove()">×</button>
    `;
    
    // Don't show banner automatically, wait for connection
    // document.body.appendChild(banner);
  }

  // Auto-connect if previously connected
  getAccounts().catch(console.error);

  console.log('TigerWallet Provider Injected');
})();
