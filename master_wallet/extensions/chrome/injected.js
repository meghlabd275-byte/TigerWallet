// TigerMasterWallet - Injected Script
// This script runs on every page to detect DApps and enable wallet connection

(function() {
  'use strict';
  
  // MasterWallet provider
  const masterWalletProvider = {
    isTigerMasterWallet: true,
    isMetaMask: false,
    
    // Request account access
    request: function(params) {
      return new Promise((resolve, reject) => {
        if (!params || !params.method) {
          reject({ code: -32600, message: 'Invalid request' });
          return;
        }
        
        switch (params.method) {
          case 'eth_requestAccounts':
          case 'eth_accounts':
            chrome.runtime.sendMessage(
              { action: 'getMasterWalletInfo' },
              (response) => {
                if (response && response.masterAddress) {
                  resolve({ result: [response.masterAddress] });
                } else {
                  resolve({ result: [] });
                }
              }
            );
            break;
            
          case 'eth_chainId':
            resolve({ result: '0x1' }); // Ethereum mainnet
            break;
            
          case 'net_version':
            resolve({ result: '1' });
            break;
            
          case 'eth_blockNumber':
            chrome.runtime.sendMessage(
              { action: 'getLatestBlock' },
              (response) => {
                resolve({ result: response.blockNumber || '0x0' });
              }
            );
            break;
            
          case 'eth_sendTransaction':
            const txParams = params.params[0];
            chrome.runtime.sendMessage(
              { 
                action: 'queueTransaction',
                tx: txParams
              },
              (response) => {
                if (response && response.success) {
                  resolve({ result: response.txHash });
                } else {
                  reject({ code: -32000, message: 'Transaction rejected' });
                }
              }
            );
            break;
            
          default:
            // Forward to blockchain
            chrome.runtime.sendMessage(
              { 
                action: 'ethCall',
                method: params.method,
                params: params.params || []
              },
              (response) => {
                if (response && response.result !== undefined) {
                  resolve({ result: response.result });
                } else {
                  reject({ code: -32601, message: 'Method not found' });
                }
              }
            );
        }
      });
    },
    
    // Event emitters
    on: function(event, callback) {
      if (event === 'accountsChanged') {
        this._accountsChangedCallback = callback;
      } else if (event === 'chainChanged') {
        this._chainChangedCallback = callback;
      }
    },
    
    emit: function(event, data) {
      if (event === 'accountsChanged' && this._accountsChangedCallback) {
        this._accountsChangedCallback(data);
      } else if (event === 'chainChanged' && this._chainChangedCallback) {
        this._chainChangedCallback(data);
      }
    },
    
    // Remove listener
    removeListener: function(event, callback) {
      if (event === 'accountsChanged') {
        this._accountsChangedCallback = null;
      } else if (event === 'chainChanged') {
        this._chainChangedCallback = null;
      }
    },
    
    // Network properties
    networkVersion: '1',
    chainId: '0x1',
    
    // Selected address
    _selectedAddress: null,
    get selectedAddress() {
      return this._selectedAddress;
    }
  };
  
  // Inject provider
  function injectProvider() {
    // Override window.ethereum
    Object.defineProperty(window, 'ethereum', {
      value: masterWalletProvider,
      writable: false,
      configurable: false
    });
    
    // Also expose as window.tigerMasterWallet
    window.tigerMasterWallet = masterWalletProvider;
    
    console.log('TigerMasterWallet injected');
  }
  
  // Listen for messages from background
  chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.action === 'walletStatusChanged') {
      masterWalletProvider.emit('accountsChanged', message.accounts);
    }
  });
  
  // Initialize
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', injectProvider);
  } else {
    injectProvider();
  }
})();
