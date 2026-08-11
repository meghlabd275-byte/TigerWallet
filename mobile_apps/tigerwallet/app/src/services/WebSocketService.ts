/**
 * TigerWallet Mobile - WebSocket Service
 * Real-time WebSocket connection for mobile apps
 * Supports: Trading, Order Book, Tickers, Notifications, Wallet Updates
 */

import { EventEmitter } from 'events';
import { store } from '../store';
import { updateMarketData, updateOrderBook, updateTrade } from '../store/slices/marketSlice';
import { updateBalance } from '../store/slices/walletSlice';
import { addNotification } from '../store/slices/notificationSlice';

export interface WSMessage {
  type: string;
  channel: string;
  data: any;
  timestamp: number;
}

export interface Subscription {
  channel: string;
  params?: Record<string, any>;
}

export interface WSConfig {
  url: string;
  reconnectInterval: number;
  maxReconnectAttempts: number;
}

class WebSocketService extends EventEmitter {
  private ws: WebSocket | null = null;
  private config: WSConfig;
  private reconnectAttempts: number = 0;
  private subscriptions: Set<string> = new Set();
  private isConnected: boolean = false;
  private isConnecting: boolean = false;
  private messageQueue: WSMessage[] = [];
  private authToken: string | null = null;
  private heartbeat: NodeJS.Timeout | null = null;

  private static instance: WebSocketService;

  private constructor() {
    super();
    this.config = {
      // The wallet_api backend is REST-only (no live WebSocket endpoint).
      // Default to empty so connect() fails honestly rather than opening a
      // connection to a nonexistent host. Set REACT_APP_WS_URL to enable a
      // real WS feed when one exists.
      url: process.env.REACT_APP_WS_URL || '',
      reconnectInterval: 3000,
      maxReconnectAttempts: 10,
    };
  }

  public static getInstance(): WebSocketService {
    if (!WebSocketService.instance) {
      WebSocketService.instance = new WebSocketService();
    }
    return WebSocketService.instance;
  }

  public setAuthToken(token: string): void {
    this.authToken = token;
  }

  public connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.isConnected) {
        resolve();
        return;
      }

      if (this.isConnecting) {
        const checkConnection = setInterval(() => {
          if (this.isConnected) {
            clearInterval(checkConnection);
            resolve();
          }
        }, 100);
        return;
      }

      this.isConnecting = true;

      try {
        this.ws = new WebSocket(this.config.url);

        this.ws.onopen = () => {
          console.log('[WS] Connected to WebSocket server');
          this.isConnected = true;
          this.isConnecting = false;
          this.reconnectAttempts = 0;
          this.startHeartbeat();
          this.resubscribe();
          this.flushMessageQueue();
          this.emit('connected');
          resolve();
        };

        this.ws.onmessage = (event) => {
          this.handleMessage(event.data);
        };

        this.ws.onerror = (error) => {
          console.error('[WS] Error:', error);
          this.emit('error', error);
        };

        this.ws.onclose = (event) => {
          console.log('[WS] Connection closed:', event.code, event.reason);
          this.isConnected = false;
          this.isConnecting = false;
          this.stopHeartbeat();
          this.emit('disconnected', event);
          this.handleReconnect();
        };

      } catch (error) {
        this.isConnecting = false;
        reject(error);
      }
    });
  }

  public disconnect(): void {
    if (this.ws) {
      this.ws.close(1000, 'Client disconnected');
      this.ws = null;
    }
    this.isConnected = false;
    this.stopHeartbeat();
  }

  public subscribe(channel: string, params?: Record<string, any>): void {
    const subscriptionKey = params ? `${channel}:${JSON.stringify(params)}` : channel;
    
    if (this.subscriptions.has(subscriptionKey)) {
      return;
    }

    this.subscriptions.add(subscriptionKey);

    if (this.isConnected) {
      this.send({
        type: 'subscribe',
        channel: channel,
        data: params || {},
        timestamp: Date.now(),
      });
    }
  }

  public unsubscribe(channel: string, params?: Record<string, any>): void {
    const subscriptionKey = params ? `${channel}:${JSON.stringify(params)}` : channel;
    
    this.subscriptions.delete(subscriptionKey);

    if (this.isConnected) {
      this.send({
        type: 'unsubscribe',
        channel: channel,
        data: params || {},
        timestamp: Date.now(),
      });
    }
  }

  public send(message: WSMessage): void {
    if (!this.isConnected || !this.ws) {
      this.messageQueue.push(message);
      return;
    }

    try {
      this.ws.send(JSON.stringify(message));
    } catch (error) {
      console.error('[WS] Send error:', error);
    }
  }

  private handleMessage(data: string): void {
    try {
      const message: WSMessage = JSON.parse(data);
      
      switch (message.channel) {
        case 'ticker':
          this.handleTickerUpdate(message.data);
          break;
        case 'orderbook':
          this.handleOrderBookUpdate(message.data);
          break;
        case 'trade':
          this.handleTradeUpdate(message.data);
          break;
        case 'wallet':
          this.handleWalletUpdate(message.data);
          break;
        case 'notification':
          this.handleNotification(message.data);
          break;
        case 'auth':
          this.handleAuthResponse(message.data);
          break;
        case 'pong':
          // Heartbeat response
          break;
        default:
          this.emit('message', message);
      }

      this.emit(message.channel, message.data);
      
    } catch (error) {
      console.error('[WS] Parse error:', error);
    }
  }

  private handleTickerUpdate(data: any): void {
    if (data && data.pair) {
      store.dispatch(updateMarketData({
        pair: data.pair,
        price: data.price,
        change24h: data.change24h,
        volume24h: data.volume24h,
        high24h: data.high24h,
        low24h: data.low24h,
      }));
    }
  }

  private handleOrderBookUpdate(data: any): void {
    if (data && data.pair) {
      store.dispatch(updateOrderBook({
        pair: data.pair,
        bids: data.bids || [],
        asks: data.asks || [],
      }));
    }
  }

  private handleTradeUpdate(data: any): void {
    if (data) {
      store.dispatch(updateTrade(data));
    }
  }

  private handleWalletUpdate(data: any): void {
    if (data && data.balances) {
      store.dispatch(updateBalance(data.balances));
    }
  }

  private handleNotification(data: any): void {
    if (data) {
      store.dispatch(addNotification({
        id: Date.now().toString(),
        type: data.type || 'info',
        title: data.title || 'Notification',
        message: data.message || '',
        timestamp: Date.now(),
        read: false,
      }));
      
      this.emit('notification', data);
    }
  }

  private handleAuthResponse(data: any): void {
    if (data.success) {
      console.log('[WS] Authenticated successfully');
      this.emit('authenticated');
    } else {
      console.error('[WS] Auth failed:', data.error);
      this.emit('authError', data.error);
    }
  }

  private startHeartbeat(): void {
    this.heartbeat = setInterval(() => {
      this.send({
        type: 'ping',
        channel: 'heartbeat',
        data: {},
        timestamp: Date.now(),
      });
    }, 30000);
  }

  private stopHeartbeat(): void {
    if (this.heartbeat) {
      clearInterval(this.heartbeat);
      this.heartbeat = null;
    }
  }

  private handleReconnect(): void {
    if (this.reconnectAttempts >= this.config.maxReconnectAttempts) {
      console.error('[WS] Max reconnection attempts reached');
      this.emit('maxReconnectAttemptsReached');
      return;
    }

    this.reconnectAttempts++;
    console.log(`[WS] Reconnecting... Attempt ${this.reconnectAttempts}`);

    setTimeout(() => {
      this.connect().catch((error) => {
        console.error('[WS] Reconnection failed:', error);
      });
    }, this.config.reconnectInterval * this.reconnectAttempts);
  }

  private resubscribe(): void {
    this.subscriptions.forEach((subscription) => {
      const [channel, paramsStr] = subscription.split(':');
      const params = paramsStr ? JSON.parse(paramsStr) : undefined;
      
      this.send({
        type: 'subscribe',
        channel: channel,
        data: params || {},
        timestamp: Date.now(),
      });
    });
  }

  private flushMessageQueue(): void {
    while (this.messageQueue.length > 0) {
      const message = this.messageQueue.shift();
      if (message) {
        this.send(message);
      }
    }
  }

  // Convenience methods for common subscriptions
  public subscribeToTicker(pair: string): void {
    this.subscribe('ticker', { pair });
  }

  public subscribeToOrderBook(pair: string): void {
    this.subscribe('orderbook', { pair });
  }

  public subscribeToTrades(pair: string): void {
    this.subscribe('trade', { pair });
  }

  public subscribeToWallet(): void {
    this.subscribe('wallet');
  }

  public subscribeToNotifications(): void {
    this.subscribe('notification');
  }

  public authenticate(): void {
    if (this.authToken) {
      this.send({
        type: 'auth',
        channel: 'auth',
        data: { token: this.authToken },
        timestamp: Date.now(),
      });
    }
  }

  public getConnectionStatus(): boolean {
    return this.isConnected;
  }
}

export const wsService = WebSocketService.getInstance();
export default wsService;
