/**
 * UserWallet WalletConnect live-event socket (browser extension).
 *
 * Connects to the canonical dapp_browser WalletConnect relay through the
 * wallet_api reverse proxy:  ws(s)://<host>/api/v1/dapp/ws/<topic>
 *
 * The wire protocol is JSON-RPC-style frames: { id, method, params }.
 * Server-pushed events arrive with a `method` field; client requests elicit
 * responses keyed by `id`. This helper only transports REAL frames — it never
 * fabricates events.
 *
 * Loaded before popup.js in popup.html. Exposes window.WalletConnectSocket.
 */
(function (global) {
  const DEFAULT_API_BASE = 'http://localhost:8443/api/v1';
  let apiBase = DEFAULT_API_BASE;

  // The popup reconfigures this from the user-configured backend
  // (chrome.storage.local tw_api_base) via setApiBase(); in a bare context we
  // also read storage directly so the socket never silently targets localhost.
  function setApiBase(httpBase) {
    if (typeof httpBase === 'string' && /^https?:\/\//.test(httpBase)) {
      apiBase = httpBase.replace(/\/+$/, '');
      if (!/\/api\/v1$/.test(apiBase)) apiBase += '/api/v1';
    }
  }
  if (typeof chrome !== 'undefined' && chrome.storage && chrome.storage.local) {
    chrome.storage.local.get('tw_api_base', (res) => {
      if (res && res.tw_api_base) setApiBase(res.tw_api_base);
    });
  }

  function wsBase() {
    const httpBase = apiBase.replace(/\/$/, '');
    const wsBase = httpBase.replace(/^http/, httpBase.startsWith('https') ? 'wss' : 'ws');
    return wsBase + '/dapp/ws';
  }

  function topicUrl(topic) {
    return wsBase() + '/' + encodeURIComponent(topic);
  }

  /**
   * Open a live WalletConnect socket for a pairing topic.
   * Returns the underlying WebSocket; callers close it when done.
   */
  function connect(topic, onMessage, onClose, onError) {
    if (typeof WebSocket === 'undefined') {
      throw new Error('WebSocket is not available in this environment');
    }
    const ws = new WebSocket(topicUrl(topic));
    ws.onmessage = (ev) => {
      try {
        onMessage(JSON.parse(ev.data));
      } catch (e) {
        // Non-JSON frames are ignored (server never sends them today).
      }
    };
    if (onClose) ws.onclose = onClose;
    if (onError) ws.onerror = onError;
    return ws;
  }

  let nextId = 1;

  /** Send a JSON-RPC request frame over an open WalletConnect socket. */
  function sendRequest(ws, method, params) {
    const id = nextId++;
    ws.send(JSON.stringify({ id, method, ...(params !== undefined ? { params } : {}) }));
    return id;
  }

  global.WalletConnectSocket = { wsBase, topicUrl, connect, sendRequest, setApiBase };
})(typeof window !== 'undefined' ? window : this);
