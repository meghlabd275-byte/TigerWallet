/**
 * UserWallet WalletConnect live-event socket.
 *
 * Connects to the canonical dapp_browser WalletConnect relay through the
 * wallet_api reverse proxy:  ws(s)://<host>/api/v1/dapp/ws/<topic>
 *
 * The wire protocol is JSON-RPC-style frames:
 *   { id: number, method: string, params?: unknown }
 * Server-pushed events arrive as messages with a `method` field (e.g.
 * session_request, eth_sendTransaction); client requests elicit responses
 * keyed by `id`. This helper only transports REAL frames — it never
 * fabricates events.
 */

// CRA (react-scripts) exposes REACT_APP_* env vars — same convention as api.ts.
const apiBase = process.env.REACT_APP_API_URL || 'http://localhost:8443/api/v1';

/** Derive the WebSocket base URL for the dapp relay from the HTTP API base. */
export function walletConnectWsBase(): string {
  const httpBase = apiBase.replace(/\/$/, '');
  const wsBase = httpBase.replace(/^http/, httpBase.startsWith('https') ? 'wss' : 'ws');
  return `${wsBase}/dapp/ws`;
}

export function walletConnectTopicUrl(topic: string): string {
  return `${walletConnectWsBase()}/${encodeURIComponent(topic)}`;
}

export interface WCFrame {
  id?: number;
  method?: string;
  params?: unknown;
  result?: unknown;
  error?: { code: number; message: string };
}

export type WCMessageHandler = (frame: WCFrame) => void;
export type WCStateHandler = (event: Event | CloseEvent) => void;

/**
 * Open a live WalletConnect socket for a pairing topic.
 * Returns the underlying WebSocket; callers close it when done.
 * Throws synchronously if WebSocket is unavailable in this environment.
 */
export function connectWalletConnect(
  topic: string,
  onMessage: WCMessageHandler,
  onClose?: WCStateHandler,
  onError?: WCStateHandler,
): WebSocket {
  if (typeof WebSocket === 'undefined') {
    throw new Error('WebSocket is not available in this environment');
  }
  const ws = new WebSocket(walletConnectTopicUrl(topic));
  ws.onmessage = (ev: MessageEvent) => {
    try {
      onMessage(JSON.parse(ev.data as string) as WCFrame);
    } catch {
      // Non-JSON frames are ignored (server never sends them today).
    }
  };
  if (onClose) ws.onclose = onClose;
  if (onError) ws.onerror = onError;
  return ws;
}

let nextId = 1;

/** Send a JSON-RPC request frame over an open WalletConnect socket. */
export function sendWCRequest(
  ws: WebSocket,
  method: string,
  params?: unknown,
): number {
  const id = nextId++;
  ws.send(JSON.stringify({ id, method, ...(params !== undefined ? { params } : {}) }));
  return id;
}
