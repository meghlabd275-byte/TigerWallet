// Trading Features Service - Desktop (Tauri/Electron) Implementation
// Supports Futures, Copy Trading, Options, Red Packet, Claim, Convert

// ============================================================================
// Futures Trading
// ============================================================================

const TW_API_BASE = 'http://localhost:8443/api/v1';

class FuturesService {
  constructor() {
    // Pair catalog is populated lazily from the live price feed (GET /price);
    // no fabricated prices/volumes. Native tokens come from the chain registry.
    this.pairs = [];
    this._loaded = false;
  }

  // Build the futures pair catalog from the live chain registry + price oracle.
  // Price/change come from GET /price?token= (CoinGecko-backed); volume/high/
  // low are NOT fabricated — they default to 0 until a market-data feed exists.
  async loadPairs() {
    if (this._loaded) return this.pairs;
    let bases = [];
    try {
      const res = await fetch(`${TW_API_BASE}/chains`);
      if (res.ok) {
        const data = await res.json();
        const arr = Array.isArray(data) ? data : (data.chains || data.evm || []);
        bases = arr.map(c => c.symbol || c.native_currency).filter(Boolean);
      }
    } catch (e) { /* registry unreachable */ }
    if (!bases.length) bases = ['ETH']; // never fabricate a fake catalog
    const quotes = ['USDT', 'USDC'];
    let id = 0;
    for (const base of bases) {
      for (const quote of quotes) {
        if (base === quote) continue;
        id++;
        let price = 0, change24h = 0;
        try {
          const pr = await fetch(`${TW_API_BASE}/price?token=${encodeURIComponent(base)}`);
          if (pr.ok) {
            const pj = await pr.json();
            price = pj.usd || 0;
            change24h = pj.usd_24h_change || 0;
          }
        } catch (e) { /* leave 0 */ }
        this.pairs.push({
          id: `pair-${id}`,
          base,
          quote,
          symbol: `${base}/${quote}`,
          price,            // live oracle price (0 if unavailable)
          change24h,        // live 24h change (0 if unavailable)
          volume24h: 0,     // real volume comes from a market-data feed (none yet)
          high24h: 0,       // not fabricated
          low24h: 0,        // not fabricated
          status: 'active',
          isPreInstalled: true,
          category: 'futures',
          minOrderSize: 0.001,
          maxOrderSize: 1000000,
          makerFee: 0.02,
          takerFee: 0.04,
        });
      }
    }
    this._loaded = true;
    return this.pairs;
  }

  getAllPairs() {
    return this.pairs;
  }

  getPreInstalledPairs() {
    return this.pairs.filter(p => p.isPreInstalled);
  }

  getPair(symbol) {
    return this.pairs.find(p => p.symbol === symbol);
  }

  // Real perpetual positions from the canonical backend (GET /perpetual/positions).
  // Never fabricates a position list.
  async getPositions() {
    try {
      const res = await fetch(`${TW_API_BASE}/perpetual/positions`);
      if (!res.ok) return [];
      const data = await res.json();
      return Array.isArray(data.data) ? data.data : (Array.isArray(data.positions) ? data.positions : []);
    } catch (e) {
      return [];
    }
  }

  // Open a perpetual position (POST /perpetual/positions). The backend records
  // + risk-checks the position; the client never fabricates PnL/liquidation.
  async openPosition({ walletId, password, symbol, side, size, leverage, chainId }) {
    const res = await fetch(`${TW_API_BASE}/perpetual/positions`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ wallet_id: walletId, password, symbol, side, size, leverage, chain_id: chainId })
    });
    return res.ok ? await res.json() : { error: await res.text() };
  }

  async closePosition(positionId) {
    const res = await fetch(`${TW_API_BASE}/perpetual/positions/${positionId}/close`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}'
    });
    return res.ok ? await res.json() : { error: await res.text() };
  }

  calculateRequiredMargin(orderValue, leverage) {
    return orderValue / leverage;
  }

  calculatePNL(entryPrice, currentPrice, size, side) {
    if (side === 'long') {
      return (currentPrice - entryPrice) * size;
    } else {
      return (entryPrice - currentPrice) * size;
    }
  }
}

// ============================================================================
// Options Trading
// ============================================================================

class OptionsService {
  constructor() {
    this.pairs = [];
    this._loaded = false;
    this.expiries = [
      { value: '1h', label: '1 Hour' },
      { value: '4h', label: '4 Hours' },
      { value: '1d', label: '1 Day' },
      { value: '1w', label: '1 Week' },
      { value: '2w', label: '2 Weeks' },
      { value: '1m', label: '1 Month' },
      { value: '3m', label: '3 Months' },
    ];
  }

  // Build the options pair catalog from the live chain registry + price oracle.
  // No fabricated prices; currentPrice comes from GET /price (0 if unavailable).
  async loadPairs() {
    if (this._loaded) return this.pairs;
    let bases = [];
    try {
      const res = await fetch(`${TW_API_BASE}/chains`);
      if (res.ok) {
        const data = await res.json();
        const arr = Array.isArray(data) ? data : (data.chains || data.evm || []);
        bases = arr.map(c => c.symbol || c.native_currency).filter(Boolean);
      }
    } catch (e) { /* registry unreachable */ }
    if (!bases.length) bases = ['ETH'];
    let id = 0;
    for (const base of bases) {
      id++;
      let price = 0;
      try {
        const pr = await fetch(`${TW_API_BASE}/price?token=${encodeURIComponent(base)}`);
        if (pr.ok) { const pj = await pr.json(); price = pj.usd || 0; }
      } catch (e) { /* leave 0 */ }
      this.pairs.push({
        id: `pair-${id}`,
        symbol: `${base}/USDT`,
        base,
        quote: 'USDT',
        currentPrice: price, // live oracle price (0 if unavailable)
        isPreInstalled: true,
      });
    }
    this._loaded = true;
    return this.pairs;
  }

  getPairs() {
    return this.pairs;
  }

  getPreInstalledPairs() {
    return this.pairs.filter(p => p.isPreInstalled);
  }

  getExpiries() {
    return this.expiries;
  }

  // The options chain is NOT available until a real options market-data feed
  // (premiums/IV/greeks) is wired to the backend. We refuse to fabricate
  // bid/ask/premium/greeks — return an empty chain when no feed exists.
  getOptionChain(currentPrice, expiry) {
    return [];
  }
}

// ============================================================================
// Copy Trading
// ============================================================================

class CopyTradingService {
  constructor() {
    this.traders = [];
    this._loaded = false;
  }

  // Fetch the REAL trader leaderboard from the canonical copy_trading backend
  // (proxied via wallet_api at GET /copytrading/traders, DB-backed). Never
  // fabricates a trader roster, win-rate, PnL, or follower counts.
  async loadTraders() {
    if (this._loaded) return this.traders;
    try {
      const res = await fetch(`${TW_API_BASE}/copytrading/traders`);
      if (res.ok) {
        const data = await res.json();
        const arr = Array.isArray(data) ? data : (data.traders || data.data || []);
        this.traders = arr.map((t, i) => ({
          id: t.id || `trader-${i + 1}`,
          username: t.username || t.name || 'Anonymous',
          avatar: t.avatar || '👤',
          winRate: t.win_rate ?? t.winRate ?? 0,
          totalPnL: t.total_pnl ?? t.pnl ?? 0,
          pnlPercent: t.pnl_percent ?? 0,
          followers: t.followers ?? 0,
          copyCount: t.copy_count ?? 0,
          tradingPair: t.trading_pair || t.pair || '',
          monthlyPnL: t.monthly_pnl ?? 0,
          weeklyPnL: t.weekly_pnl ?? 0,
          dailyPnL: t.daily_pnl ?? 0,
          maxDrawdown: t.max_drawdown ?? 0,
          riskLevel: t.risk_level || t.risk || 'medium',
          isFollowing: !!t.is_following,
          isPreInstalled: !!t.is_pre_installed,
        }));
      }
    } catch (e) { /* backend unreachable: leave empty */ }
    this._loaded = true;
    return this.traders;
  }

  // Follow / stop-following a trader (POST /copytrading/follow).
  async follow(traderId, walletId) {
    const res = await fetch(`${TW_API_BASE}/copytrading/follow`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ trader_id: traderId, wallet_id: walletId })
    });
    return res.ok ? await res.json() : { error: await res.text() };
  }

  getAllTraders() {
    return this.traders;
  }

  getTopTraders() {
    return this.traders.filter(t => t.isPreInstalled);
  }

  filterByRisk(risk) {
    if (risk === 'all') return this.traders;
    return this.traders.filter(t => t.riskLevel === risk);
  }
}

// ============================================================================
// Convert Service
// ============================================================================

class ConvertService {
  constructor() {
    // No hardcoded rates or balances; everything comes from the live backend
    // (GET /swap/quote for cross-rates, GET /chains for the token catalog).
    this.pairs = {};
    this.balances = {};
  }

  // Fetch the live conversion rate from the backend swap engine
  // (GET /swap/quote, CoinGecko-backed). Returns { rate, fee } or null.
  async getRate(from, to) {
    if (from === to) return { rate: 1, fee: 0 };
    try {
      const res = await fetch(`${TW_API_BASE}/swap/quote?from_token=${encodeURIComponent(from)}&to_token=${encodeURIComponent(to)}&from_amount=1`);
      if (!res.ok) return null;
      const q = await res.json();
      if (!q.rate) return null;
      const rate = parseFloat(q.rate);
      const fee = q.fee ? parseFloat(q.fee) : 0;
      return { rate, fee };
    } catch (e) {
      return null;
    }
  }

  // Live indicative quote for a real conversion amount (GET /swap/quote).
  // Never fabricates a "completed" tx; the caller broadcasts via the backend.
  async getQuote(from, to, amount) {
    try {
      const res = await fetch(`${TW_API_BASE}/swap/quote?from_token=${encodeURIComponent(from)}&to_token=${encodeURIComponent(to)}&from_amount=${encodeURIComponent(amount)}`);
      if (!res.ok) return null;
      const q = await res.json();
      return {
        fromToken: q.from_token || from,
        toToken: q.to_token || to,
        fromAmount: parseFloat(q.from_amount || amount),
        toAmount: parseFloat(q.to_amount || 0),
        rate: parseFloat(q.rate || 0),
        minReceived: parseFloat(q.min_received || 0),
        fee: q.fee ? parseFloat(q.fee) : 0,
      };
    } catch (e) {
      return null;
    }
  }

  // Available convert tokens come from the live chain registry (no fake balances).
  async getAvailableTokens() {
    try {
      const res = await fetch(`${TW_API_BASE}/chains`);
      if (!res.ok) return [];
      const data = await res.json();
      const arr = Array.isArray(data) ? data : (data.chains || data.evm || []);
      return arr.map(c => ({
        symbol: c.symbol || c.native_currency || 'ETH',
        name: c.name || '',
        balance: 0, // real balances come from the wallet view, never fabricated
        icon: '🪙',
      }));
    } catch (e) {
      return [];
    }
  }

  // Convert is executed on-chain via the backend swap/amm endpoints; this client
  // helper only returns the live quote. Callers must broadcast through /send or
  // /amm/swap. Never returns a fabricated "completed" status.
  async convert(userId, from, to, amount) {
    const quote = await this.getQuote(from, to, amount);
    if (!quote) return null;
    return { id: `convert-${Date.now()}`, ...quote, status: 'quoted' };
  }
}

// ============================================================================
// Export
// ============================================================================

if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    FuturesService,
    OptionsService,
    CopyTradingService,
    ConvertService,
  };
}
