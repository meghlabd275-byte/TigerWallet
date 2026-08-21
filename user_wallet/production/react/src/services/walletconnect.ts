/**
 * UserWallet WalletConnect live-event socket (production react client).
 *
 * Connects to the canonical dapp_browser WalletConnect relay through the
 * wallet_api reverse proxy:  ws(s)://<host>/api/v1/dapp/ws/<topic>
 *
 * The wire protocol is JSON-RPC-style frames: { id, method, params }.
 * Server-pushed events arrive with a method field; client requests elicit
 * responses keyed by id. This helper only transports REAL frames — it never
 * fabricates events.
 */

const API_BASE_URL =
  import.meta.env.VITE_API_URL || 'http://localhost:8443/api/v1';

export function walletConnectWsBase(): string {
  const httpBase = API_BASE_URL.replace(/\/$/, '');
  const wsBase = httpBase.replace(/^http/, httpBase.startsWith('https') ? 'wss' : 'ws');
  return wsBase + '/dapp/ws';
}

export function walletConnectTopicUrl(topic: string): string {
  return walletConnectWsBase() + '/' + encodeURIComponent(topic);
}

export interface WCFrame {
  id?: number;
  method?: string;
  params?: unknown;
  result?: unknown;
  error?: { code: number; message: string };
}

export type WCMessageHandler = (frame: WCFrame) => void;

/**
 * Open a live WalletConnect socket for a pairing topic.
 * Returns the underlying WebSocket; callers close it when done.
 */
export function connectWalletConnect(
  topic: string,
  onMessage: WCMessageHandler,
  onClose?: (ev: Event | CloseEvent) => void,
  onError?: (ev: Event) => void,
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
