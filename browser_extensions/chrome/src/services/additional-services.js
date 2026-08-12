// Swap/DEX Service - Browser Extension
const API_BASE = 'http://localhost:8443/api/v1';

class SwapService {
  constructor(token) {
    this.token = token;
  }

  async getHeaders() {
    return {
      'Content-Type': 'application/json',
      'Authorization': this.token ? `Bearer ${this.token}` : ''
    };
  }

  async getTokens() {
    try {
      const response = await fetch(`${API_BASE}/swap/tokens`, {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        return data.data || [];
      }
    } catch (error) {
      console.error('Failed to get tokens:', error);
    }
    return [];
  }

  async getQuote(fromToken, toToken, amount) {
    try {
      const response = await fetch(
        `${API_BASE}/swap/quote?from=${fromToken}&to=${toToken}&amount=${amount}`,
        { headers: await this.getHeaders() }
      );
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to get quote:', error);
    }
    return null;
  }

  async executeSwap(fromToken, toToken, amount, slippage) {
    try {
      const response = await fetch(`${API_BASE}/swap/execute`, {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ fromToken, toToken, amount, slippage })
      });
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to execute swap:', error);
    }
    return null;
  }
}

class StakingService {
  constructor(token) {
    this.token = token;
  }

  async getHeaders() {
    return {
      'Content-Type': 'application/json',
      'Authorization': this.token ? `Bearer ${this.token}` : ''
    };
  }

  async getPools() {
    try {
      const response = await fetch(`${API_BASE}/staking/pools`, {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        return data.data || [];
      }
    } catch (error) {
      console.error('Failed to get pools:', error);
    }
    return [];
  }

  async stake(poolId, amount) {
    try {
      const response = await fetch(`${API_BASE}/staking/stake`, {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ poolId, amount })
      });
      return response.ok;
    } catch (error) {
      console.error('Failed to stake:', error);
    }
    return false;
  }

  async unstake(poolId, amount) {
    try {
      const response = await fetch(`${API_BASE}/staking/unstake`, {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ poolId, amount })
      });
      return response.ok;
    } catch (error) {
      console.error('Failed to unstake:', error);
    }
    return false;
  }
}

class BridgeService {
  constructor(token) {
    this.token = token;
  }

  async getHeaders() {
    return {
      'Content-Type': 'application/json',
      'Authorization': this.token ? `Bearer ${this.token}` : ''
    };
  }

  async getChains() {
    // Canonical chain registry from go/wallet_api (120 EVM + 66 non-EVM
    // mainnet chains). The /bridge/chains sub-endpoint was a non-canonical
    // stub and is replaced here.
    try {
      const response = await fetch(`${API_BASE}/chains`, {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        // Backend envelope: { chains: [...], count, evm_count, ... }
        return data.chains || [];
      }
    } catch (error) {
      console.error('Failed to get chains:', error);
    }
    return [];
  }

  async getQuote(fromChain, toChain, token, amount) {
    try {
      const response = await fetch(
        `${API_BASE}/bridge/quote?from=${fromChain}&to=${toChain}&token=${token}&amount=${amount}`,
        { headers: await this.getHeaders() }
      );
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to get quote:', error);
    }
    return null;
  }

  async initiateBridge(fromChain, toChain, token, amount, toAddress) {
    try {
      const response = await fetch(`${API_BASE}/bridge/initiate`, {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ fromChain, toChain, token, amount, toAddress })
      });
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to initiate bridge:', error);
    }
    return null;
  }
}

class LendingService {
  constructor(token) {
    this.token = token;
  }

  async getHeaders() {
    return {
      'Content-Type': 'application/json',
      'Authorization': this.token ? `Bearer ${this.token}` : ''
    };
  }

  async getPools() {
    try {
      const response = await fetch(`${API_BASE}/lending/pools`, {
        headers: await this.getHeaders()
      });
      if (response.ok) {
        const data = await response.json();
        return data.data || [];
      }
    } catch (error) {
      console.error('Failed to get pools:', error);
    }
    return [];
  }

  async supply(poolId, amount) {
    try {
      const response = await fetch(`${API_BASE}/lending/supply`, {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ poolId, amount })
      });
      return response.ok;
    } catch (error) {
      console.error('Failed to supply:', error);
    }
    return false;
  }

  async borrow(poolId, amount) {
    try {
      const response = await fetch(`${API_BASE}/lending/borrow`, {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ poolId, amount })
      });
      return response.ok;
    } catch (error) {
      console.error('Failed to borrow:', error);
    }
    return false;
  }
}

class GasTrackerService {
  async getGasPrice(chain) {
    try {
      const response = await fetch(`${API_BASE}/gas/price?chain=${chain}`);
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to get gas price:', error);
    }
    return null;
  }
}

class TWAPService {
  constructor(token) {
    this.token = token;
  }

  async getHeaders() {
    return {
      'Content-Type': 'application/json',
      'Authorization': this.token ? `Bearer ${this.token}` : ''
    };
  }

  async createTWAP(symbol, totalAmount, intervals, side) {
    try {
      const response = await fetch(`${API_BASE}/twap/create`, {
        method: 'POST',
        headers: await this.getHeaders(),
        body: JSON.stringify({ symbol, totalAmount, intervals, side })
      });
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to create TWAP:', error);
    }
    return null;
  }

  async cancelTWAP(orderId) {
    try {
      const response = await fetch(`${API_BASE}/twap/${orderId}/cancel`, {
        method: 'POST',
        headers: await this.getHeaders()
      });
      return response.ok;
    } catch (error) {
      console.error('Failed to cancel TWAP:', error);
    }
    return false;
  }
}

class IntentRoutingService {
  async findBestRoute(intent) {
    try {
      const response = await fetch(`${API_BASE}/intent/route`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ intent })
      });
      if (response.ok) {
        const data = await response.json();
        return data.data;
      }
    } catch (error) {
      console.error('Failed to find route:', error);
    }
    return null;
  }
}

module.exports = {
  SwapService,
  StakingService,
  BridgeService,
  LendingService,
  GasTrackerService,
  TWAPService,
  IntentRoutingService
};
