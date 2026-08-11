/**
 * Chrome Extension - Trading Service
 * Order Book, Trading Charts, Positions
 */

// ============================================================================
// Trading Service
// ============================================================================

class TradingService {
  constructor() {
    this.baseURL = 'http://localhost:8443/api/v1/trading';
  }

  async getOrderBook(symbol, limit = 50) {
    try {
      const response = await fetch(`${this.baseURL}/orderbook?symbol=${symbol}&limit=${limit}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get order book:', error);
      return null;
    }
  }

  async getCandlesticks(symbol, interval = '1h', limit = 100) {
    try {
      const response = await fetch(`${this.baseURL}/klines?symbol=${symbol}&interval=${interval}&limit=${limit}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get candlesticks:', error);
      return [];
    }
  }

  async getPositions(walletAddress) {
    try {
      const response = await fetch(`${this.baseURL}/positions/${walletAddress}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get positions:', error);
      return [];
    }
  }

  async getOpenOrders(walletAddress) {
    try {
      const response = await fetch(`${this.baseURL}/orders/${walletAddress}?status=open`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get open orders:', error);
      return [];
    }
  }

  async placeMarketOrder(walletAddress, symbol, side, amount, leverage = 1) {
    try {
      const response = await fetch(`${this.baseURL}/orders`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          wallet_address: walletAddress,
          symbol,
          side,
          type: 'market',
          amount,
          leverage
        })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to place market order:', error);
      throw error;
    }
  }

  async placeLimitOrder(walletAddress, symbol, side, price, amount, leverage = 1) {
    try {
      const response = await fetch(`${this.baseURL}/orders`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          wallet_address: walletAddress,
          symbol,
          side,
          type: 'limit',
          price,
          amount,
          leverage
        })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to place limit order:', error);
      throw error;
    }
  }

  async cancelOrder(walletAddress, orderId) {
    try {
      const response = await fetch(`${this.baseURL}/orders/${orderId}`, {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ wallet_address: walletAddress })
      });
      return response.ok;
    } catch (error) {
      console.error('Failed to cancel order:', error);
      return false;
    }
  }

  async closePosition(walletAddress, positionId) {
    try {
      const response = await fetch(`${this.baseURL}/positions/${positionId}/close`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ wallet_address: walletAddress })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to close position:', error);
      throw error;
    }
  }
}

// ============================================================================
// MEV Protection Service
// ============================================================================

class MEVProtectionService {
  constructor() {
    this.baseURL = 'http://localhost:8443/api/v1/mev';
  }

  async detectSandwichAttack(txHash) {
    try {
      const response = await fetch(`${this.baseURL}/detect-sandwich?tx=${txHash}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to detect sandwich attack:', error);
      return { detected: false };
    }
  }

  async simulateTransaction(from, to, data, value) {
    try {
      const response = await fetch(`${this.baseURL}/simulate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ from, to, data, value })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to simulate transaction:', error);
      return { success: false, error: error.message };
    }
  }

  async submitWithProtection(signedTx, protectionLevel = 'medium') {
    try {
      const response = await fetch(`${this.baseURL}/submit`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ signed_tx: signedTx, protection_level: protectionLevel })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to submit with protection:', error);
      throw error;
    }
  }
}

// ============================================================================
// Session Keys Service
// ============================================================================

class SessionKeysService {
  constructor() {
    this.baseURL = 'http://localhost:8443/api/v1/session-keys';
  }

  async generate(walletAddress, dappUrl, permissions, expiresIn = 86400) {
    try {
      const response = await fetch(this.baseURL, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          wallet_address: walletAddress,
          dapp_url: dappUrl,
          permissions,
          expires_in: expiresIn
        })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to generate session key:', error);
      return null;
    }
  }

  async list(walletAddress) {
    try {
      const response = await fetch(`${this.baseURL}/${walletAddress}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to list session keys:', error);
      return [];
    }
  }

  async revoke(walletAddress, sessionKeyId) {
    try {
      const response = await fetch(`${this.baseURL}/${sessionKeyId}`, {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ wallet_address: walletAddress })
      });
      return response.ok;
    } catch (error) {
      console.error('Failed to revoke session key:', error);
      return false;
    }
  }
}

// ============================================================================
// Gas Optimization Service
// ============================================================================

class GasOptimizationService {
  constructor() {
    this.baseURL = 'http://localhost:8443/api/v1/gas';
  }

  async getPrices(chain = 'ethereum') {
    try {
      const response = await fetch(`${this.baseURL}/prices?chain=${chain}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get gas prices:', error);
      return null;
    }
  }

  async getSuggestions(from, to, data) {
    try {
      const response = await fetch(`${this.baseURL}/optimize`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ from, to, data })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to get optimization suggestions:', error);
      return [];
    }
  }

  async estimate(txData, chain = 'ethereum') {
    try {
      const response = await fetch(`${this.baseURL}/estimate?chain=${chain}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ data: txData })
      });
      return await response.json();
    } catch (error) {
      console.error('Failed to estimate gas:', error);
      return null;
    }
  }
}

// ============================================================================
// Widget SDK Service
// ============================================================================

class WidgetSDKService {
  constructor() {
    this.widgets = {};
  }

  createBalanceWidget(walletAddress) {
    return {
      type: 'balance',
      walletAddress,
      update: async () => {
        const response = await fetch(`http://localhost:8443/api/v1/wallet/${walletAddress}/balance`);
        return response.json();
      }
    };
  }

  createPriceWidget(token) {
    return {
      type: 'price',
      token,
      update: async () => {
        const response = await fetch(`http://localhost:8443/api/v1/prices/${token}`);
        return response.json();
      }
    };
  }

  createPortfolioWidget(walletAddress) {
    return {
      type: 'portfolio',
      walletAddress,
      update: async () => {
        const response = await fetch(`http://localhost:8443/api/v1/wallet/${walletAddress}/portfolio`);
        return response.json();
      }
    };
  }

  createQuickSendWidget() {
    return {
      type: 'quick_send',
      actions: ['send', 'swap', 'bridge']
    };
  }
}

// Export all services
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { 
    TradingService, 
    MEVProtectionService, 
    SessionKeysService, 
    GasOptimizationService,
    WidgetSDKService 
  };
}
