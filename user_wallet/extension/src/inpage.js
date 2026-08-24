// In-page EIP-1193 provider injected into every dApp page (MAIN world).
// Implements window.ethereum so dApps can detect and talk to UserWallet.
// All privileged work is delegated to the content script via window.postMessage;
// this file holds no keys and performs no network I/O itself.
(function () {
  'use strict';
  if (window.ethereum && window.ethereum.__isTigerWallet) return;

  const listeners = {};
  let chainId = '0x1';
  let accounts = [];
  let connected = false;

  function emit(event, payload) {
    (listeners[event] || []).forEach((fn) => {
      try { fn(payload); } catch (e) { console.error(e); }
    });
  }

  // Pending RPC calls keyed by request id; resolved by content-script replies.
  const pending = new Map();
  let nextId = 1;

  window.addEventListener('message', (ev) => {
    if (ev.source !== window) return;
    const msg = ev.data;
    if (!msg || msg.__tigerwallet !== 'inpage-response') return;
    if (msg.type === 'event') {
      // Push events from the content script (accountsChanged / chainChanged / connect).
      if (msg.event === 'chainChanged') chainId = msg.payload;
      if (msg.event === 'accountsChanged') accounts = msg.payload || [];
      emit(msg.event, msg.payload);
      return;
    }
    const entry = pending.get(msg.id);
    if (!entry) return;
    pending.delete(msg.id);
    if (msg.error) entry.reject(new Error(msg.error.message || 'User rejected request'));
    else entry.resolve(msg.result);
  });

  function sendToBridge(method, params) {
    return new Promise((resolve, reject) => {
      const id = nextId++;
      pending.set(id, { resolve, reject });
      window.postMessage({ __tigerwallet: 'inpage-request', id, method, params }, '*');
      setTimeout(() => {
        if (pending.delete(id)) reject(new Error('Request timed out'));
      }, 60000);
    });
  }

  const provider = {
    __isTigerWallet: true,
    isTigerWallet: true,
    isMetaMask: false,
    chainId: () => chainId,
    selectedAddress: () => accounts[0] || null,
    isConnected: () => connected,

    // EIP-1193 request interface.
    async request({ method, params }) {
      switch (method) {
        case 'eth_chainId':
          return sendToBridge(method, params);
        case 'net_version':
          return sendToBridge(method, params);
        case 'eth_accounts':
          return accounts;
        case 'eth_requestAccounts': {
          const accs = await sendToBridge(method, params);
          accounts = accs;
          connected = accs && accs.length > 0;
          emit('connect', { chainId });
          emit('accountsChanged', accounts);
          return accs;
        }
        case 'wallet_switchEthereumChain': {
          const result = await sendToBridge(method, params);
          if (params && params[0] && params[0].chainId) {
            chainId = params[0].chainId;
            emit('chainChanged', chainId);
          }
          return result;
        }
        case 'personal_sign':
        case 'eth_sign':
        case 'eth_signTypedData':
        case 'eth_signTypedData_v4':
        case 'eth_sendTransaction':
        case 'eth_call':
        case 'eth_estimateGas':
        case 'eth_getBalance':
        case 'eth_blockNumber':
        case 'eth_getTransactionReceipt':
        case 'eth_getTransactionByHash':
        case 'eth_gasPrice':
          return sendToBridge(method, params);
        default:
          // Pass-through for read-only RPC methods the node can answer directly.
          return sendToBridge(method, params);
      }
    },

    // Legacy convenience APIs used by older dApps.
    send(method, params) {
      if (typeof method === 'string') return this.request({ method, params });
      return this.request(method);
    },
    sendAsync(payload, callback) {
      this.request(payload)
        .then((result) => callback(null, { id: payload.id, jsonrpc: '2.0', result }))
        .catch((error) => callback(error, { id: payload.id, jsonrpc: '2.0', error: { code: 4001, message: error.message } }));
    },
    on(event, fn) {
      (listeners[event] = listeners[event] || []).push(fn);
      return provider;
    },
    removeListener(event, fn) {
      const l = listeners[event] || [];
      const i = l.indexOf(fn);
      if (i >= 0) l.splice(i, 1);
      return provider;
    },
  };

  Object.defineProperty(window, 'ethereum', {
    value: provider,
    configurable: false,
    writable: false,
    enumerable: true,
  });
  window.dispatchEvent(new Event('ethereum#initialized'));
})();
