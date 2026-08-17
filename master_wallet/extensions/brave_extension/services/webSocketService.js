/**
 * MasterWallet extension - real-time WebSocket client for the canonical backend.
 *
 * Connects to ws://<apiBase>/ws?master_wallet_id=<id>&token=<JWT> and dispatches
 * live balance updates, transaction confirmations, and market-ticker events to
 * registered listeners. Uses the standard browser WebSocket API (available in
 * MV3 service workers and extension pages). Fail-closed: no fake events are
 * ever emitted; if the socket is closed or errored, listeners simply stop
 * receiving real-time data until reconnect.
 */
'use strict';

const { getConfig } = (typeof require === 'function')
  ? require('./config.js')
  : (globalThis.MW_CONFIG || {});

function storageGet(keys) {
  return new Promise((resolve) => {
    try {
      chrome.storage.local.get(keys, (res) => resolve(res || {}));
    } catch (e) {
      resolve({});
    }
  });
}

function toWsUrl(apiBase, wsPath, masterWalletId, token) {
  const base = (apiBase || '').replace(/\/+$/, '');
  const wsBase = base.replace(/^http/, 'ws');
  const params = new URLSearchParams();
  if (masterWalletId) params.set('master_wallet_id', masterWalletId);
  if (token) params.set('token', token);
  const qs = params.toString();
  return `${wsBase}${wsPath || '/ws'}${qs ? '?' + qs : ''}`;
}

class MasterWalletWebSocketService {
  constructor() {
    this.ws = null;
    this.listeners = new Map(); // event -> Set<callback>
    this.reconnectTimer = null;
    this.reconnectAttempts = 0;
    this.heartbeatTimer = null;
    this.activeMasterWalletId = null;
    this.manuallyClosed = false;
  }

  on(event, cb) {
    if (!this.listeners.has(event)) this.listeners.set(event, new Set());
    this.listeners.get(event).add(cb);
    return () => this.off(event, cb);
  }

  off(event, cb) {
    const set = this.listeners.get(event);
    if (set) set.delete(cb);
  }

  _emit(event, payload) {
    const set = this.listeners.get(event);
    if (set) for (const cb of set) {
      try { cb(payload); } catch (_) { /* listener error must not break others */ }
    }
  }

  /**
   * Connect to the canonical backend WebSocket. Reads the auth token + current
   * master wallet id from chrome.storage, builds ws://<base>/ws?... and opens.
   * Reconnects automatically with capped backoff on close/error.
   */
  async connect(masterWalletId) {
    if (masterWalletId) this.activeMasterWalletId = masterWalletId;
    const id = this.activeMasterWalletId;
    if (!id) {
      this._emit('error', new Error('No master wallet id selected for WebSocket'));
      return;
    }
    const ctx = await storageGet(['mw_auth_token', 'mw_current_wallet_id', 'MASTER_WALLET_API_URL']);
    const token = ctx.mw_auth_token;
    if (!token) {
      this._emit('error', new Error('Not authenticated for WebSocket'));
      return;
    }
    const cfg = await getConfig();
    const url = toWsUrl(ctx.MASTER_WALLET_API_URL || cfg.apiBase, cfg.wsPath, id, token);

    // Close any existing socket before opening a new one.
    this._cleanupSocket();

    try {
      this.ws = new WebSocket(url);
    } catch (e) {
      this._emit('error', e);
      this._scheduleReconnect();
      return;
    }

    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this._emit('open', { master_wallet_id: id });
      // Heartbeat: send a ping every 30s to keep the connection alive.
      this.heartbeatTimer = setInterval(() => {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
          try { this.ws.send(JSON.stringify({ type: 'ping' })); } catch (_) {}
        }
      }, 30000);
    };

    this.ws.onmessage = (event) => {
      let data;
      try { data = JSON.parse(event.data); } catch (_) { return; }
      // Dispatch by event type; also emit a wildcard 'message'.
      const type = data && data.type;
      if (type) this._emit(type, data);
      this._emit('message', data);
    };

    this.ws.onerror = (err) => {
      this._emit('error', err);
    };

    this.ws.onclose = () => {
      this._emit('close', { master_wallet_id: id });
      if (this.heartbeatTimer) { clearInterval(this.heartbeatTimer); this.heartbeatTimer = null; }
      if (!this.manuallyClosed) this._scheduleReconnect();
    };
  }

  _scheduleReconnect() {
    if (this.reconnectTimer) return;
    // Capped exponential backoff: 1s, 2s, 4s, ... max 30s.
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
    this.reconnectAttempts += 1;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, delay);
  }

  _cleanupSocket() {
    if (this.heartbeatTimer) { clearInterval(this.heartbeatTimer); this.heartbeatTimer = null; }
    if (this.ws) {
      try {
        this.ws.onopen = null;
        this.ws.onmessage = null;
        this.ws.onerror = null;
        this.ws.onclose = null;
        if (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING) {
          this.ws.close();
        }
      } catch (_) {}
      this.ws = null;
    }
  }

  /**
   * Send a JSON message to the backend over the open socket. Returns false if
   * the socket is not open (fail-closed — does not queue or fake delivery).
   */
  send(payload) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return false;
    try { this.ws.send(JSON.stringify(payload)); return true; } catch (_) { return false; }
  }

  /**
   * Disconnect and stop auto-reconnect. Use when the user logs out or switches
   * master wallets (call connect() again with the new id afterwards).
   */
  disconnect() {
    this.manuallyClosed = true;
    if (this.reconnectTimer) { clearTimeout(this.reconnectTimer); this.reconnectTimer = null; }
    this._cleanupSocket();
  }

  isConnected() {
    return !!this.ws && this.ws.readyState === WebSocket.OPEN;
  }
}

const webSocketService = new MasterWalletWebSocketService();

// UMD: CommonJS for node/tests, globalThis for MV3 service worker.
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { MasterWalletWebSocketService, webSocketService, toWsUrl };
}
if (typeof globalThis !== 'undefined') {
  globalThis.MW_WS = { webSocketService, MasterWalletWebSocketService };
}
