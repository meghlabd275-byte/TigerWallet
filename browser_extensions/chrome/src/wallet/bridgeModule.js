
// Hardened HTML escaper. Any value rendered into markup (token/chain/NFT names,
// order fields, backend or RPC error text) is attacker-influenceable and must
// pass through esc() before it reaches innerHTML.
function esc(v) {
    if (v === null || v === undefined) return "";
    if (typeof v === "object") { try { v = JSON.stringify(v); } catch (_) { v = String(v); } }
    return String(v).replace(/[&<>"'`=\\/]/g, (c) => ({
        "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;",
        "'": "&#39;", "`": "&#96;", "=": "&#61;", "/": "&#47;"
    }[c]));
}

/**
 * TigerWallet Browser Extension - Bridge Module
 * Complete cross-chain bridge functionality for extension
 */

class BridgeModule {
  constructor(walletManager) {
    this.walletManager = walletManager;
    // Routes come from the backend. A hardcoded table here would advertise
    // provider names and token sets the relayer does not actually support.
    this.routes = [];
    this.routesLoaded = false;
  }

  /**
   * The TigerBridge backend base URL.
   *
   * Resolved from extension storage so a user pointing at their own
   * deployment is not silently sent to localhost.
   */
  async apiBase() {
    const stored = await chrome.storage.local.get('tw_api_base');
    const base = stored.tw_api_base || 'http://localhost:8443';
    return String(base).replace(/\/+$/, '');
  }

  /**
   * Load supported routes from the backend.
   */
  async loadRoutes() {
    const base = await this.apiBase();
    const response = await fetch(`${base}/api/v1/bridge/routes`);
    if (!response.ok) {
      throw new Error(`bridge routes unavailable (HTTP ${response.status})`);
    }
    const payload = await response.json();
    const routes = Array.isArray(payload) ? payload : payload.routes;
    if (!Array.isArray(routes)) {
      throw new Error('bridge routes response was not a list');
    }
    this.routes = routes;
    this.routesLoaded = true;
    return routes;
  }

  /**
   * Get supported routes, loading them on first use.
   */
  async getSupportedRoutes() {
    if (!this.routesLoaded) {
      await this.loadRoutes();
    }
    return this.routes;
  }

  /**
   * Get quote for bridge
   */
  async getQuote(fromChain, toChain, token, amount) {
    const base = await this.apiBase();
    const response = await fetch(`${base}/api/v1/bridge/quote`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        from_chain: fromChain,
        to_chain: toChain,
        token,
        amount: amount.toString()
      })
    });
    if (!response.ok) {
      // Do not fabricate a bridge quote. Signal failure to the caller.
      return null;
    }
    return response.json();
  }

  /**
   * Execute bridge
   */
  async execute(walletAddress, fromChain, toChain, token, amount, recipient) {
    const base = await this.apiBase();
    const response = await fetch(`${base}/api/v1/bridge/transfer`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        wallet_address: walletAddress,
        from_chain: fromChain,
        to_chain: toChain,
        token,
        amount: amount.toString(),
        recipient: recipient || walletAddress
      })
    });
    if (!response.ok) {
      const detail = await response.text();
      throw new Error(`bridge transfer failed (HTTP ${response.status}): ${detail.slice(0, 200)}`);
    }
    return response.json();
  }

  /**
   * Get bridge status.
   *
   * Returns null rather than a placeholder when the status is unknown, so a
   * caller cannot mistake an unreachable backend for a pending transfer.
   */
  async getStatus(bridgeTxId) {
    const base = await this.apiBase();
    const response = await fetch(
      `${base}/api/v1/bridge/status/${encodeURIComponent(bridgeTxId)}`
    );
    if (!response.ok) {
      return null;
    }
    return response.json();
  }

  /**
   * Get tokens for route
   */
  async getTokensForRoute(fromChain, toChain) {
    const routes = await this.getSupportedRoutes();
    const route = routes.find(r => r.from === fromChain && r.to === toChain);
    return route && Array.isArray(route.tokens) ? route.tokens : [];
  }

  /**
   * Create bridge UI
   */
  createBridgeUI() {
    const container = document.createElement('div');
    container.className = 'tigerwallet-bridge-popup';
    container.innerHTML = `
      <div class="tw-bridge-header">
        <h2>🌉 Bridge</h2>
        <button class="tw-close-btn">&times;</button>
      </div>
      <div class="tw-bridge-content">
        <div class="tw-bridge-form">
          <div class="tw-form-group">
            <label>From Chain</label>
            <select class="tw-from-chain">
              ${this.supportedChains.map(c => `<option value="${esc(c)}">${esc(c)}</option>`).join('')}
            </select>
          </div>
          <div class="tw-form-group">
            <label>To Chain</label>
            <select class="tw-to-chain">
              ${this.supportedChains.map(c => `<option value="${esc(c)}">${esc(c)}</option>`).join('')}
            </select>
          </div>
          <div class="tw-form-group">
            <label>Token</label>
            <select class="tw-token">
              <option value="ETH">ETH</option>
              <option value="USDC">USDC</option>
              <option value="USDT">USDT</option>
            </select>
          </div>
          <div class="tw-form-group">
            <label>Amount</label>
            <input type="number" class="tw-amount" placeholder="0.00" />
          </div>
          <button class="tw-get-quote-btn btn-primary">Get Quote</button>
        </div>
        <div class="tw-quote-result"></div>
      </div>
    `;
    return container;
  }

  /**
   * Setup bridge event listeners
   */
  setupBridgeListeners(container, walletAddress) {
    const fromChain = container.querySelector('.tw-from-chain');
    const toChain = container.querySelector('.tw-to-chain');
    const token = container.querySelector('.tw-token');
    const amount = container.querySelector('.tw-amount');
    const quoteBtn = container.querySelector('.tw-get-quote-btn');
    const quoteResult = container.querySelector('.tw-quote-result');

    // Update tokens when chains change
    const updateTokens = async () => {
      let tokens = [];
      try {
        tokens = await this.getTokensForRoute(fromChain.value, toChain.value);
      } catch (error) {
        tokens = [];
      }
      token.textContent = '';
      for (const symbol of tokens) {
        const option = document.createElement('option');
        option.value = String(symbol);
        option.textContent = String(symbol);
        token.appendChild(option);
      }
    };

    fromChain.addEventListener('change', updateTokens);
    toChain.addEventListener('change', updateTokens);

    // Get quote
    quoteBtn.addEventListener('click', async () => {
      if (!amount.value) return;
      
      quoteResult.innerHTML = '<div class="tw-loading">Getting quote...</div>';
      
      try {
        const quote = await this.getQuote(
          fromChain.value,
          toChain.value,
          token.value,
          amount.value
        );

        if (!quote) {
          quoteResult.innerHTML = '<div class="tw-error">No bridge quote is available for this route.</div>';
          return;
        }

        // Backend values are inserted as text, never as markup.
        quoteResult.textContent = '';
        const card = document.createElement('div');
        card.className = 'tw-quote-card';

        const rows = [
          ['You Send', `${quote.sendAmount} ${quote.token}`],
          ['Bridge Fee', `${quote.bridgeFee} ${quote.token}`],
          ['You Receive', `${quote.receiveAmount} ${quote.token}`],
          ['Provider', quote.provider],
          ['Estimated Time', quote.estimatedTime]
        ];
        for (const [label, value] of rows) {
          const row = document.createElement('div');
          row.className = 'tw-quote-row';
          const labelEl = document.createElement('span');
          labelEl.textContent = label;
          const valueEl = document.createElement('span');
          valueEl.textContent = value === undefined || value === null ? '-' : String(value);
          row.appendChild(labelEl);
          row.appendChild(valueEl);
          card.appendChild(row);
        }

        const bridgeBtn = document.createElement('button');
        bridgeBtn.className = 'tw-bridge-btn btn-primary';
        bridgeBtn.textContent = 'Bridge Now';
        card.appendChild(bridgeBtn);
        quoteResult.appendChild(card);

        bridgeBtn.addEventListener('click', async () => {
          bridgeBtn.textContent = 'Bridging...';
          try {
            await this.execute(walletAddress, fromChain.value, toChain.value, token.value, parseFloat(amount.value));
            alert('Bridge initiated successfully!');
            quoteResult.innerHTML = '';
          } catch (error) {
            alert('Bridge failed: ' + error.message);
          }
        });
      } catch (error) {
        quoteResult.innerHTML = `<div class="tw-error">Failed to get quote</div>`;
      }
    });
  }
}

// Export for extension
if (typeof module !== 'undefined' && module.exports) {
  module.exports = BridgeModule;
}
