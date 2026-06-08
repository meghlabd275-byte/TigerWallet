import { EventEmitter } from 'events';

export interface Ticker {
  symbol: string;
  price: string;
  priceChange: string;
  priceChangePercent: string;
  high24h: string;
  low24h: string;
  volume24h: string;
  turnover24h: string;
  openInterest: string;
  fundingRate: string;
  nextFundingTime: number;
}

export interface Depth {
  symbol: string;
  lastUpdateId: number;
  bids: [string, string][];
  asks: [string, string][];
}

export interface Trade {
  id: string;
  price: string;
  quantity: string;
  time: number;
  isBuyerMaker: boolean;
}

export interface Position {
  id: string;
  userId: string;
  symbol: string;
  side: 'LONG' | 'SHORT';
  quantity: string;
  entryPrice: string;
  markPrice: string;
  leverage: string;
  margin: string;
  unrealizedPnl: string;
  realizedPnl: string;
  liquidationPrice: string;
  marginType: 'CROSS' | 'ISOLATED';
  roe: string;
  marginRatio: string;
  updatedAt: number;
}

export interface Order {
  id: string;
  userId: string;
  symbol: string;
  side: 'BUY' | 'SELL';
  orderType: 'LIMIT' | 'MARKET' | 'STOP_MARKET' | 'STOP_LIMIT' | 'TAKE_PROFIT' | 'TRAILING_STOP';
  price: string;
  quantity: string;
  filledQuantity: string;
  remainingQty: string;
  avgFillPrice: string;
  status: 'PENDING' | 'OPEN' | 'PARTIALLY_FILLED' | 'FILLED' | 'CANCELLED' | 'REJECTED' | 'EXPIRED';
  reduceOnly: boolean;
  postOnly: boolean;
  timeInForce: 'GTC' | 'IOC' | 'FOK';
  stopPrice?: string;
  leverage: string;
  marginType: 'CROSS' | 'ISOLATED';
  positionSide?: 'LONG' | 'SHORT';
  createdAt: number;
  updatedAt: number;
  expiresAt?: number;
}

export interface OrderRequest {
  symbol: string;
  userId: string;
  side: 'BUY' | 'SELL';
  orderType: 'LIMIT' | 'MARKET' | 'STOP_MARKET' | 'STOP_LIMIT' | 'TAKE_PROFIT' | 'TRAILING_STOP';
  price?: string;
  quantity: string;
  reduceOnly?: boolean;
  postOnly?: boolean;
  timeInForce?: 'GTC' | 'IOC' | 'FOK';
  stopPrice?: string;
  leverage?: string;
  marginType?: 'CROSS' | 'ISOLATED';
  positionSide?: 'LONG' | 'SHORT';
}

export interface AccountInfo {
  userId: string;
  totalEquity: string;
  available: string;
  usedMargin: string;
  unrealizedPnl: string;
}

export interface FundingInfo {
  symbol: string;
  fundingRate: string;
  fundingRateReal: string;
  nextFundingTime: number;
}

export class PerpetualsService extends EventEmitter {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private baseUrl: string;
  private subscriptions: Set<string> = new Set();

  constructor(baseUrl?: string) {
    super();
    this.baseUrl = baseUrl || this.getWebSocketUrl();
  }

  private getWebSocketUrl(): string {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    return `${protocol}//${host}/api/v1/ws`;
  }

  async connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(this.baseUrl);

        this.ws.onopen = () => {
          console.log('WebSocket connected');
          this.reconnectAttempts = 0;
          this.resubscribe();
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const messages = event.data.split('\n');
            for (const msg of messages) {
              if (msg.trim()) {
                const data = JSON.parse(msg);
                this.handleMessage(data);
              }
            }
          } catch (error) {
            console.error('Failed to parse message:', error);
          }
        };

        this.ws.onclose = () => {
          console.log('WebSocket disconnected');
          this.attemptReconnect();
        };

        this.ws.onerror = (error) => {
          console.error('WebSocket error:', error);
          reject(error);
        };

        // Timeout for connection
        setTimeout(() => {
          if (this.ws?.readyState !== WebSocket.OPEN) {
            reject(new Error('Connection timeout'));
          }
        }, 5000);

      } catch (error) {
        reject(error);
      }
    });
  }

  private handleMessage(data: any) {
    const { type, channel, data: messageData } = data;
    
    if (type === 'ticker' && messageData) {
      this.emit('ticker', messageData);
    } else if (type === 'depth' && messageData) {
      this.emit('depth', messageData);
    } else if (type === 'trade' && messageData) {
      this.emit('trade', messageData);
    } else if (type === 'position' && messageData) {
      this.emit('positions', messageData);
    } else if (type === 'order' && messageData) {
      this.emit('order', messageData);
    } else if (type === 'funding' && messageData) {
      this.emit('funding', messageData);
    } else if (type === 'liquidation' && messageData) {
      this.emit('liquidation', messageData);
    } else if (type === 'pong') {
      // Keep-alive response
    }
  }

  private attemptReconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
      setTimeout(() => {
        this.connect().catch(console.error);
      }, delay);
    }
  }

  private resubscribe() {
    if (this.subscriptions.size > 0) {
      this.subscribe(Array.from(this.subscriptions));
    }
  }

  subscribe(channels: string[]) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      channels.forEach(ch => this.subscriptions.add(ch));
      this.ws.send(JSON.stringify({
        type: 'subscribe',
        channels
      }));
    }
  }

  unsubscribe(channels: string[]) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      channels.forEach(ch => this.subscriptions.delete(ch));
      this.ws.send(JSON.stringify({
        type: 'unsubscribe',
        channels
      }));
    }
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  // REST API methods

  async getTicker(symbol: string): Promise<Ticker> {
    const response = await fetch(`/api/v1/market/tickers?symbol=${symbol}`);
    if (!response.ok) throw new Error('Failed to fetch ticker');
    const data = await response.json();
    return data;
  }

  async getDepth(symbol: string, limit: number = 20): Promise<Depth> {
    const response = await fetch(`/api/v1/market/depth/${symbol}?limit=${limit}`);
    if (!response.ok) throw new Error('Failed to fetch depth');
    const data = await response.json();
    return data;
  }

  async getTrades(symbol: string, limit: number = 50): Promise<Trade[]> {
    const response = await fetch(`/api/v1/market/trades/${symbol}?limit=${limit}`);
    if (!response.ok) throw new Error('Failed to fetch trades');
    const data = await response.json();
    return data;
  }

  async getFundingRate(symbol: string): Promise<FundingInfo> {
    const response = await fetch(`/api/v1/market/funding/${symbol}`);
    if (!response.ok) throw new Error('Failed to fetch funding rate');
    const data = await response.json();
    return data;
  }

  async createOrder(order: OrderRequest): Promise<Order> {
    const response = await fetch('/api/v1/orders', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(order)
    });
    
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to create order');
    }
    
    const data = await response.json();
    return data;
  }

  async getOrder(orderId: string): Promise<Order> {
    const response = await fetch(`/api/v1/orders/${orderId}`);
    if (!response.ok) throw new Error('Failed to fetch order');
    const data = await response.json();
    return data;
  }

  async cancelOrder(orderId: string): Promise<void> {
    const response = await fetch(`/api/v1/orders/${orderId}`, {
      method: 'DELETE'
    });
    
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to cancel order');
    }
  }

  async getOrders(userId: string, symbol?: string): Promise<Order[]> {
    let url = `/api/v1/orders?userId=${userId}`;
    if (symbol) url += `&symbol=${symbol}`;
    
    const response = await fetch(url);
    if (!response.ok) throw new Error('Failed to fetch orders');
    const data = await response.json();
    return data;
  }

  async getPositions(userId: string): Promise<Position[]> {
    const response = await fetch(`/api/v1/positions?userId=${userId}`);
    if (!response.ok) throw new Error('Failed to fetch positions');
    const data = await response.json();
    return data;
  }

  async closePosition(request: { symbol: string; userId: string; quantity?: string }): Promise<void> {
    const response = await fetch('/api/v1/positions/close', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request)
    });
    
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to close position');
    }
  }

  async getAccountInfo(userId: string): Promise<AccountInfo> {
    const response = await fetch(`/api/v1/account/balance?userId=${userId}`);
    if (!response.ok) throw new Error('Failed to fetch account info');
    const data = await response.json();
    return data;
  }

  async getMarginInfo(userId: string): Promise<any> {
    const response = await fetch(`/api/v1/account/margin?userId=${userId}`);
    if (!response.ok) throw new Error('Failed to fetch margin info');
    const data = await response.json();
    return data;
  }
}