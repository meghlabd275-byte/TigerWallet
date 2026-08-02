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
      const response = await fetch('https://api.tigerwallet.com/v1/prices', {
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
      const response = await fetch(`https://api.tigerwallet.com/v1/prices/${token}/history?timeframe=${timeframe}`, {
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
    try {
      this.ws = new WebSocket('wss://api.tigerwallet.com/ws/prices');

      this.ws.onopen = () => {
        console.log('Price WebSocket connected');
        this.reconnectAttempts = 0;
      };

      this.ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          this.notifySubscribers(data);
        } catch (error) {
          console.error('Failed to parse price update:', error);
        }
      };

      this.ws.onclose = () => {
        console.log('Price WebSocket disconnected');
        this.reconnect();
      };

      this.ws.onerror = (error) => {
        console.error('Price WebSocket error:', error);
      };
    } catch (error) {
      console.error('Failed to connect price WebSocket:', error);
    }
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
