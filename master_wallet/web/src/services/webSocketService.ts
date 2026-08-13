/**
 * WebSocketService - Web (React/TypeScript)
 * Real-time connection to the canonical backend WebSocket at ws://localhost:8450/ws
 * (derived from MASTER_WALLET_API_URL). Sends master_wallet_id + token query params
 * per the canonical contract; no loopback fakes.
 */

import { masterWalletAPI, getAuthToken } from '../api';

type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'reconnecting' | 'error';

interface BalanceUpdate {
  chainId: number;
  address: string;
  balance: string;
  token: string;
  timestamp: number;
}

interface TransactionUpdate {
  txHash: string;
  from: string;
  to: string;
  amount: string;
  status: string;
  timestamp: number;
}

class WebSocketService {
  private ws: WebSocket | null = null;
  private walletId: string | null = null;
  private authToken: string | null = null;
  private reconnectAttempts = 0;
  private heartbeatInterval: ReturnType<typeof setInterval> | null = null;
  private reconnectTimeout: ReturnType<typeof setTimeout> | null = null;

  private static readonly MAX_RECONNECT_ATTEMPTS = 10;
  private static readonly RECONNECT_DELAY = 5000;

  private _connectionState: ConnectionState = 'disconnected';
  get connectionState(): ConnectionState { return this._connectionState; }

  private stateListeners: Set<(state: ConnectionState) => void> = new Set();
  private messageListeners: Set<(message: string) => void> = new Set();
  private balanceListeners: Set<(update: BalanceUpdate) => void> = new Set();
  private transactionListeners: Set<(update: TransactionUpdate) => void> = new Set();

  private wsUrl(): string {
    return masterWalletAPI.wsUrl; // e.g. ws://localhost:8450/ws
  }

  /**
   * Connect to the canonical backend WebSocket. master_wallet_id + token are
   * passed as query params (the backend scopes broadcasts by master_wallet_id).
   */
  connect(walletId: string, token?: string): void {
    this.walletId = walletId;
    this.authToken = token ?? getAuthToken();
    this._connect();
  }

  private _connect(): void {
    this._connectionState = 'connecting';
    this._notifyStateChange();

    let url = this.wsUrl();
    const params = new URLSearchParams();
    if (this.walletId) params.set('master_wallet_id', this.walletId);
    if (this.authToken) params.set('token', this.authToken);
    const qs = params.toString();
    if (qs) url += `?${qs}`;

    try {
      this.ws = new WebSocket(url);

      this.ws.onopen = () => {
        this._connectionState = 'connected';
        this._notifyStateChange();
        this.reconnectAttempts = 0;
        this._startHeartbeat();
      };

      this.ws.onmessage = (event) => {
        this._handleMessage(event.data);
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        this._connectionState = 'error';
        this._notifyStateChange();
        this._stopHeartbeat();
        this._scheduleReconnect();
      };

      this.ws.onclose = () => {
        this._connectionState = 'disconnected';
        this._notifyStateChange();
        this._stopHeartbeat();
        this._scheduleReconnect();
      };
    } catch (error) {
      console.error('WebSocket connect failed:', error);
      this._connectionState = 'error';
      this._notifyStateChange();
      this._scheduleReconnect();
    }
  }

  /**
   * Disconnect from server
   */
  disconnect(): void {
    this._stopHeartbeat();
    this._cancelReconnect();

    if (this.ws) {
      this.ws.close(1000, 'Client disconnected');
      this.ws = null;
    }

    this._connectionState = 'disconnected';
    this._notifyStateChange();
  }

  /**
   * Subscribe to balance updates
   */
  subscribeToBalance(chainId: number): void {
    this._sendMessage('subscribe', 'balance', { chainId });
  }

  /**
   * Unsubscribe from balance updates
   */
  unsubscribeFromBalance(chainId: number): void {
    this._sendMessage('unsubscribe', 'balance', { chainId });
  }

  /**
   * Subscribe to transaction updates
   */
  subscribeToTransactions(address: string): void {
    this._sendMessage('subscribe', 'transactions', { address });
  }

  /**
   * Subscribe to ticker updates
   */
  subscribeToTicker(pair: string): void {
    this._sendMessage('subscribe', 'ticker', { pair });
  }

  /**
   * Subscribe to order book
   */
  subscribeToOrderBook(pair: string): void {
    this._sendMessage('subscribe', 'orderbook', { pair });
  }

  /**
   * Add state listener
   */
  onStateChange(listener: (state: ConnectionState) => void): void {
    this.stateListeners.add(listener);
  }

  /**
   * Remove state listener
   */
  offStateChange(listener: (state: ConnectionState) => void): void {
    this.stateListeners.delete(listener);
  }

  /**
   * Add message listener
   */
  onMessage(listener: (message: string) => void): void {
    this.messageListeners.add(listener);
  }

  /**
   * Add balance listener
   */
  onBalanceUpdate(listener: (update: BalanceUpdate) => void): void {
    this.balanceListeners.add(listener);
  }

  /**
   * Add transaction listener
   */
  onTransactionUpdate(listener: (update: TransactionUpdate) => void): void {
    this.transactionListeners.add(listener);
  }

  private _sendMessage(type: string, channel: string, data: Record<string, unknown>): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return;
    }

    const message = {
      type,
      channel,
      data,
      timestamp: Date.now(),
    };

    this.ws.send(JSON.stringify(message));
  }

  private _handleMessage(data: string): void {
    try {
      const json = JSON.parse(data) as Record<string, unknown>;
      // The backend broadcasts messages with a `type` field (e.g. "balance",
      // "tx_confirmation", "ticker"). Route by `type`, falling back to
      // `channel` for client-originated subscriptions.
      const channel = (json.channel as string) ?? (json.type as string) ?? '';
      const payload = (json.data as Record<string, unknown>) ?? json;

      this.messageListeners.forEach((listener) => listener(data));

      switch (channel) {
        case 'balance':
          this._handleBalanceUpdate(payload);
          break;
        case 'tx_confirmation':
        case 'transactions':
          this._handleTransactionUpdate(payload);
          break;
        default:
          break;
      }
    } catch (error) {
      console.error('Error parsing message:', error);
    }
  }

  private _handleBalanceUpdate(data: Record<string, unknown>): void {
    const update: BalanceUpdate = {
      chainId: data.chain_id as number,
      address: data.address as string,
      balance: data.balance as string,
      token: (data.symbol as string) ?? (data.token as string) ?? '',
      timestamp: data.timestamp ? Number(data.timestamp) : Date.now(),
    };

    this.balanceListeners.forEach((listener) => listener(update));
  }

  private _handleTransactionUpdate(data: Record<string, unknown>): void {
    const update: TransactionUpdate = {
      txHash: (data.tx_hash as string) ?? (data.transaction_hash as string) ?? '',
      from: (data.from as string) ?? (data.from_address as string) ?? '',
      to: (data.to as string) ?? (data.to_address as string) ?? '',
      amount: data.amount as string,
      status: data.status as string,
      timestamp: data.timestamp ? Number(data.timestamp) : Date.now(),
    };

    this.transactionListeners.forEach((listener) => listener(update));
  }

  private _notifyStateChange(): void {
    this.stateListeners.forEach((listener) => listener(this._connectionState));
  }

  private _scheduleReconnect(): void {
    if (this.reconnectAttempts >= WebSocketService.MAX_RECONNECT_ATTEMPTS) {
      this._connectionState = 'error';
      this._notifyStateChange();
      return;
    }

    this.reconnectAttempts++;
    this._connectionState = 'reconnecting';
    this._notifyStateChange();

    this.reconnectTimeout = setTimeout(
      () => this._connect(),
      WebSocketService.RECONNECT_DELAY * this.reconnectAttempts
    );
  }

  private _cancelReconnect(): void {
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }
  }

  private _startHeartbeat(): void {
    this.heartbeatInterval = setInterval(() => {
      this._sendMessage('ping', 'heartbeat', {});
    }, 15000);
  }

  private _stopHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }
}

export const webSocketService = new WebSocketService();
export default webSocketService;
