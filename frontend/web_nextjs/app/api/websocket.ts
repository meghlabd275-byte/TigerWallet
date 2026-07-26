/**
 * TigerWallet WebSocket Service
 * Real-time updates for transactions, prices, and notifications
 */

import { api } from './service';

type EventCallback = (data: any) => void;

class WebSocketService {
  private socket: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 10;
  private reconnectDelay = 1000;
  private listeners: Map<string, Set<EventCallback>> = new Map();
  private messageQueue: any[] = [];
  private isConnected = false;

  constructor() {
    if (typeof window !== 'undefined') {
      this.connect();
    }
  }

  private connect() {
    if (this.socket?.readyState === WebSocket.OPEN) {
      return;
    }

    const wsURL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8443/ws';

    try {
      this.socket = new WebSocket(wsURL);

      this.socket.onopen = () => {
        console.log('WebSocket connected');
        this.isConnected = true;
        this.reconnectAttempts = 0;
        
        // Send queued messages
        while (this.messageQueue.length > 0) {
          const message = this.messageQueue.shift();
          this.socket?.send(JSON.stringify(message));
        }

        // Re-subscribe to events
        this.listeners.forEach((_, event) => {
          this.subscribe(event, () => {});
        });
      };

      this.socket.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          const { type, payload } = data;

          const eventListeners = this.listeners.get(type);
          if (eventListeners) {
            eventListeners.forEach((callback) => {
              try {
                callback(payload);
              } catch (e) {
                console.error(`Error in ${type} listener:`, e);
              }
            });
          }

          // Also notify wildcard listeners
          const wildcardListeners = this.listeners.get('*');
          if (wildcardListeners) {
            wildcardListeners.forEach((callback) => {
              try {
                callback(data);
              } catch (e) {
                console.error('Error in wildcard listener:', e);
              }
            });
          }
        } catch (e) {
          console.error('WebSocket message parse error:', e);
        }
      };

      this.socket.onclose = (event) => {
        console.log('WebSocket closed:', event.code, event.reason);
        this.isConnected = false;
        this.attemptReconnect();
      };

      this.socket.onerror = (error) => {
        console.error('WebSocket error:', error);
      };
    } catch (error) {
      console.error('Failed to create WebSocket:', error);
      this.attemptReconnect();
    }
  }

  private attemptReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

    console.log(`Attempting to reconnect in ${delay}ms (attempt ${this.reconnectAttempts})`);

    setTimeout(() => {
      this.connect();
    }, delay);
  }

  subscribe(event: string, callback: EventCallback): () => void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }

    this.listeners.get(event)!.add(callback);

    // Send subscription to server
    this.send({
      type: 'subscribe',
      event,
      timestamp: Date.now(),
    });

    // Return unsubscribe function
    return () => {
      this.unsubscribe(event, callback);
    };
  }

  unsubscribe(event: string, callback: EventCallback) {
    const eventListeners = this.listeners.get(event);
    if (eventListeners) {
      eventListeners.delete(callback);

      if (eventListeners.size === 0) {
        this.listeners.delete(event);

        // Send unsubscription to server
        this.send({
          type: 'unsubscribe',
          event,
          timestamp: Date.now(),
        });
      }
    }
  }

  private send(data: any) {
    const message = JSON.stringify(data);

    if (this.socket?.readyState === WebSocket.OPEN) {
      this.socket.send(message);
    } else {
      // Queue message for when connection is restored
      this.messageQueue.push(data);
    }
  }

  // ==========================================================================
  // Event Emitters
  // ==========================================================================

  // Transaction events
  onTransactionUpdate(callback: (tx: any) => void) {
    return this.subscribe('transaction:update', callback);
  }

  onTransactionConfirmed(callback: (tx: any) => void) {
    return this.subscribe('transaction:confirmed', callback);
  }

  onTransactionFailed(callback: (tx: any) => void) {
    return this.subscribe('transaction:failed', callback);
  }

  // Wallet events
  onBalanceUpdate(callback: (data: { address: string; balance: string; chainId: number }) => void) {
    return this.subscribe('wallet:balance', callback);
  }

  onNewBlock(callback: (block: { number: number; hash: string }) => void) {
    return this.subscribe('block:new', callback);
  }

  // Price events
  onPriceUpdate(callback: (data: { symbol: string; price: number; change24h: number }) => void) {
    return this.subscribe('price:update', callback);
  }

  // Notification events
  onNotification(callback: (notification: any) => void) {
    return this.subscribe('notification:new', callback);
  }

  // Market events
  onMarketUpdate(callback: (data: any) => void) {
    return this.subscribe('market:update', callback);
  }

  // Order events (for trading)
  onOrderUpdate(callback: (order: any) => void) {
    return this.subscribe('order:update', callback);
  }

  // Chat/Support events
  onMessage(callback: (message: any) => void) {
    return this.subscribe('chat:message', callback);
  }

  // Generic
  onAny(callback: (data: any) => void) {
    return this.subscribe('*', callback);
  }

  // ==========================================================================
  // Send Events
  // ==========================================================================

  sendTransactionRequest(tx: any) {
    this.send({
      type: 'transaction:request',
      payload: tx,
      timestamp: Date.now(),
    });
  }

  sendMessage(message: string) {
    this.send({
      type: 'chat:message',
      payload: { message },
      timestamp: Date.now(),
    });
  }

  // ==========================================================================
  // Utility
  // ==========================================================================

  disconnect() {
    if (this.socket) {
      this.socket.close();
      this.socket = null;
    }
    this.listeners.clear();
  }

  getConnectionStatus() {
    return {
      isConnected: this.isConnected,
      reconnectAttempts: this.reconnectAttempts,
      queueLength: this.messageQueue.length,
    };
  }
}

// Singleton instance
export const wsService = typeof window !== 'undefined' ? new WebSocketService() : null;

// ============================================================================
// React Hook
// ============================================================================

export function useWebSocket(event: string, callback: EventCallback) {
  if (typeof window === 'undefined') {
    return () => {};
  }

  // This would be used with React in a real implementation
  // For now, just return the subscribe function
  return wsService?.subscribe(event, callback) || (() => {});
}

export default wsService;
