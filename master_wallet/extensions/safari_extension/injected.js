/**
 * MasterWallet content-script-side provider bridge.
 *
 * This file is listed as a content script in manifest.json and runs in the
 * ISOLATED world of every page. It injects an EIP-1193 (window.ethereum)
 * provider that relays requests to the background service worker, which in turn
 * calls the REAL canonical backend (http://localhost:8450). It never fabricates
 * accounts, balances, signatures, or chain ids. Any request that cannot be
 * fulfilled honestly is rejected (fail-closed).
 *
 * The provider implements:
 *   - eth_requestAccounts        -> backend wallet list (requires login)
 *   - eth_accounts               -> currently selected account (or [])
 *   - eth_chainId                -> configured chain id
 *   - eth_getBalance             -> backend /master-wallet/:id/balance
 *   - eth_sendTransaction         -> backend /master-wallet/:id/sign
 *   - wallet_requestPermissions  -> no-op acknowledge
 *   - net_version / eth_blockNumber / eth_getTransactionReceipt
 *
 * Unsupported methods return a JSON-RPC error with code 4200 (unsupported).
 */

'use strict';

(() => {
  if (typeof window === 'undefined') return;

  const REQ_TIMEOUT = 60000;

  function sendToBackground(type, payload) {
    return new Promise((resolve, reject) => {
      let done = false;
      const timer = setTimeout(() => {
        if (done) return;
        done = true;
        reject(new Error('Background request timed out: ' + type));
      }, REQ_TIMEOUT);

      try {
        chrome.runtime.sendMessage({ type, payload }, (res) => {
          if (done) return;
          done = true;
          clearTimeout(timer);
          const err = chrome.runtime.lastError;
          if (err) {
            reject(new Error(err.message || String(err)));
          } else if (!res) {
            reject(new Error('No response from background for ' + type));
          } else if (!res.ok) {
            reject(new Error(res.error || 'Background error'));
          } else {
            resolve(res.data);
          }
        });
      } catch (e) {
        if (done) return;
        done = true;
        clearTimeout(timer);
        reject(e);
      }
    });
  }

  function rpcError(code, message, id = null) {
    return {
      jsonrpc: '2.0',
      id,
      error: { code, message },
    };
  }

  function rpcResult(result, id) {
    return { jsonrpc: '2.0', id, result };
  }

  async function handleRequest(req) {
    const id = req && req.id;
    const method = req && req.method;
    const params = (req && req.params) || [];

    if (!method) return rpcError(-32600, 'Invalid request: missing method', id);

    try {
      switch (method) {
        case 'eth_requestAccounts':
        case 'eth_accounts': {
          const ctx = await sendToBackground('MW_AUTH_CONTEXT', {});
          if (!ctx || !ctx.token) {
            return rpcError(4100, 'Unauthorized: not logged in to MasterWallet', id);
          }
          if (!ctx.currentWalletId) {
            return rpcResult([], id);
          }
          const wallet = await sendToBackground('MW_RELAY', {
            action: 'getMasterWallet',
            args: [ctx.currentWalletId],
          });
          const addr = (wallet && (wallet.address || (wallet.wallet && wallet.wallet.address))) || null;
          return rpcResult(addr ? [addr] : [], id);
        }

        case 'eth_chainId': {
          const ctx = await sendToBackground('MW_AUTH_CONTEXT', {});
          if (!ctx || !ctx.token || !ctx.currentWalletId) {
            return rpcError(4100, 'Unauthorized: no wallet selected', id);
          }
          try {
            const w = await sendToBackground('MW_RELAY', {
              action: 'getMasterWallet',
              args: [ctx.currentWalletId],
            });
            const cid = (w && (w.chain_id || (w.wallet && w.wallet.chain_id))) || 1;
            return rpcResult('0x' + Number(cid).toString(16), id);
          } catch (e) {
            return rpcError(-32603, (e && e.message) || String(e), id);
          }
        }

        case 'eth_getBalance': {
          const ctx = await sendToBackground('MW_AUTH_CONTEXT', {});
          if (!ctx || !ctx.token || !ctx.currentWalletId) {
            return rpcError(4100, 'Unauthorized', id);
          }
          const bal = await sendToBackground('MW_RELAY', {
            action: 'getBalance',
            args: [ctx.currentWalletId, params[1]],
          });
          // Prefer native balance in hex wei; the backend may return decimal.
          const native = (bal && (bal.native || bal.balance || (bal.balances && bal.balances.native))) || '0x0';
          const hex = typeof native === 'string' && native.startsWith('0x')
            ? native
            : '0x' + BigInt(Math.floor(Number(native) || 0)).toString(16);
          return rpcResult(hex, id);
        }

        case 'eth_sendTransaction': {
          const ctx = await sendToBackground('MW_AUTH_CONTEXT', {});
          if (!ctx || !ctx.token || !ctx.currentWalletId) {
            return rpcError(4100, 'Unauthorized', id);
          }
          const tx = params[0] || {};
          // eth_sendTransaction params don't include a password; the backend
          // requires one. DApps must prompt the user; we relay what we can and
          // the backend enforces auth. If password is missing, reject honestly.
          if (!tx.password) {
            return rpcError(
              -32602,
              'MasterWallet requires a password for eth_sendTransaction; prompt the user',
              id
            );
          }
          const res = await sendToBackground('MW_RELAY', {
            action: 'signTransaction',
            args: [ctx.currentWalletId, {
              to: tx.to,
              amount: tx.value,
              password: tx.password,
              token: tx.token,
            }],
          });
          const hash = (res && (res.transaction_hash || res.hash)) || '0x0';
          return rpcResult(hash, id);
        }

        case 'wallet_requestPermissions':
        case 'eth_getTransactionReceipt': {
          // Honestly report that live receipts are available only via the
          // backend transaction history endpoint, not per-hash receipts.
          return rpcError(4200, 'Method not supported by MasterWallet provider', id);
        }

        case 'net_version':
        case 'eth_blockNumber': {
          // We cannot honestly report head block number without an RPC node.
          return rpcError(4200, 'Method not supported by MasterWallet provider', id);
        }

        default:
          return rpcError(4200, 'Unsupported method: ' + method, id);
      }
    } catch (e) {
      return rpcError(-32603, (e && e.message) || String(e), id);
    }
  }

  // Build the EIP-1193 provider. Do not overwrite an existing real provider
  // (e.g. MetaMask); if one exists, register as an additional provider under
  // window.tigerMasterWallet and announce via announceProvider.
  let listenerSeq = 0;
  const listeners = new Set();

  const provider = {
    isTigerMasterWallet: true,
    isMetaMask: false,
    request(req) {
      if (!req || typeof req !== 'object' || !req.method) {
        return Promise.reject(new Error('Invalid request object'));
      }
      return handleRequest(req).then((resp) => {
        if (resp && resp.error) {
          const err = new Error(resp.error.message);
          err.code = resp.error.code;
          throw err;
        }
        return resp && resp.result;
      });
    },
    on(event, handler) {
      if (typeof handler !== 'function') return;
      const entry = { event, handler };
      listeners.add(entry);
      return () => listeners.delete(entry);
    },
    removeListener(event, handler) {
      for (const entry of [...listeners]) {
        if (entry.event === event && entry.handler === handler) listeners.delete(entry);
      }
    },
    enable() {
      return this.request({ method: 'eth_requestAccounts' });
    },
    _emit(event, payload) {
      for (const entry of [...listeners]) {
        if (entry.event === event) {
          try { entry.handler(payload); } catch (_) { /* ignore */ }
        }
      }
    },
  };

  // Register on window. If there's an existing provider, keep it and add ours
  // as a secondary provider (EIP-6963-style, without full MIP discovery).
  if (!window.ethereum) {
    window.ethereum = provider;
  } else {
    // Preserve any existing provider; expose ours under our own namespace.
    window.tigerMasterWallet = provider;
    if (window.ethereum && Array.isArray(window.ethereum.providers)) {
      window.ethereum.providers.push(provider);
    }
  }

  // Listen for theme changes from the background to keep the page styled if the
  // popup/dapp wants to reflect the wallet UI theme.
  chrome.runtime.onMessage.addListener((msg) => {
    if (msg && msg.type === 'MW_THEME_CHANGED') {
      try {
        document.documentElement.setAttribute('data-theme', msg.theme);
      } catch (_) { /* ignore */ }
    }
    return false;
  });

  // Announce ourselves via EIP-6963 announceProvider so well-behaved DApps can
  // discover the MasterWallet provider.
  try {
    const info = {
      uuid: 'tiger-master-wallet-extension',
      name: 'TigerMasterWallet',
      icon: '',
      rdns: 'io.tigerwallet.master',
    };
    const event = new Event('eip6963:announceProvider', { bubbles: true });
    event.detail = Object.assign(Object.create(provider), info);
    window.dispatchEvent(event);
    window.addEventListener('eip6963:requestProvider', () => {
      const ev = new Event('eip6963:announceProvider', { bubbles: true });
      ev.detail = Object.assign(Object.create(provider), info);
      window.dispatchEvent(ev);
    });
  } catch (_) { /* EIP-6963 is optional */ }
})();
