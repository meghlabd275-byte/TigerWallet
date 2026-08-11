/**
 * Chrome Extension - Price Service
 * Real-time price feeds for browser extension
 */

// Price Service for Chrome Extension
class PriceService {
  constructor(apiKey) {
    this.apiKey = apiKey;
    this.subscribers = new Map();
    this.ws = null;
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
  }

  async getPrices(tokens = ['BTC', 'ETH', 'SOL', 'BNB', 'MATIC', 'AVAX']) {
    try {
      const response = await fetch('http://localhost:8443/api/v1/prices', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.apiKey}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ tokens })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to get prices:', error);
      return {};
    }
  }

  async getHistoricalPrices(token, timeframe = '24h') {
    try {
      const response = await fetch(`http://localhost:8443/api/v1/prices/${token}/history?timeframe=${timeframe}`, {
        headers: {
          'Authorization': `Bearer ${this.apiKey}`,
          'Content-Type': 'application/json'
        }
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to get historical prices:', error);
      return [];
    }
  }

  connectWebSocket() {
    // No live WebSocket price feed exists on the backend. Instead, poll the
    // real wallet_api REST endpoint (/api/v1/prices) on an interval and notify
    // subscribers. Honest: if the backend is unreachable, no prices are pushed
    // (no fabricated ticker data).
    if (this.pollInterval) return;
    const POLL_MS = 15000;
    const poll = async () => {
      try {
        const response = await fetch('http://localhost:8443/api/v1/prices', {
          headers: { 'Content-Type': 'application/json' }
        });
        if (!response.ok) return;
        const data = await response.json();
        // The backend returns a map of symbol -> {price, change_24h, ...}.
        const prices = data.prices || data;
        if (prices && typeof prices === 'object') {
          for (const [token, info] of Object.entries(prices)) {
            const price = typeof info === 'number' ? info : info.price;
            const change24h = typeof info === 'object' ? info.change_24h || info.change24h : 0;
            if (price !== undefined) {
              this.notifySubscribers({ token, price, change24h: change24h || 0 });
            }
          }
        }
        this.reconnectAttempts = 0;
      } catch (error) {
        console.error('Price poll failed:', error);
        this.reconnect();
      }
    };
    poll();
    this.pollInterval = setInterval(poll, POLL_MS);
    this.ws = { readyState: 1, close: () => this.disconnect() }; // stub handle for state checks
    console.log('Price polling started (REST /api/v1/prices, 15s interval)');
  }

  disconnect() {
    if (this.pollInterval) {
      clearInterval(this.pollInterval);
      this.pollInterval = null;
    }
    this.ws = null;
  }

  reconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      setTimeout(() => this.connectWebSocket(), 1000 * this.reconnectAttempts);
    }
  }

  subscribe(token, callback) {
    if (!this.subscribers.has(token)) {
      this.subscribers.set(token, new Set());
    }
    this.subscribers.get(token).add(callback);

    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      this.connectWebSocket();
    }
  }

  unsubscribe(token, callback) {
    if (this.subscribers.has(token)) {
      this.subscribers.get(token).delete(callback);
    }
  }

  notifySubscribers(data) {
    const { token, price, change24h } = data;

    if (this.subscribers.has(token)) {
      this.subscribers.get(token).forEach(callback => {
        try {
          callback({ token, price, change24h });
        } catch (error) {
          console.error('Error in price callback:', error);
        }
      });
    }

    // Notify all subscribers
    if (this.subscribers.has('*')) {
      this.subscribers.get('*').forEach(callback => {
        try {
          callback(data);
        } catch (error) {
          console.error('Error in global price callback:', error);
        }
      });
    }
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}

// Export
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { PriceService };
}
