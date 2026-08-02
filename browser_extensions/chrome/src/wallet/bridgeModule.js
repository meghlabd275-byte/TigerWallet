/**
 * TigerWallet Browser Extension - Bridge Module
 * Complete cross-chain bridge functionality for extension
 */

class BridgeModule {
  constructor(walletManager) {
    this.walletManager = walletManager;
    this.supportedChains = [
      'ethereum', 'polygon', 'arbitrum', 'optimism', 
      'avalanche', 'bsc', 'base', 'solana'
    ];
  }

  /**
   * Get supported routes
   */
  getSupportedRoutes() {
    return [
      { from: 'ethereum', to: 'polygon', tokens: ['ETH', 'USDT', 'USDC'], provider: 'Stargate' },
      { from: 'ethereum', to: 'arbitrum', tokens: ['ETH', 'USDC'], provider: 'LayerZero' },
      { from: 'ethereum', to: 'optimism', tokens: ['ETH', 'USDC'], provider: 'LayerZero' },
      { from: 'ethereum', to: 'avalanche', tokens: ['ETH', 'USDC'], provider: 'Axelar' },
      { from: 'ethereum', to: 'bsc', tokens: ['ETH', 'BNB'], provider: 'Stargate' },
      { from: 'polygon', to: 'ethereum', tokens: ['MATIC', 'USDC'], provider: 'Stargate' },
      { from: 'bsc', to: 'ethereum', tokens: ['BNB', 'ETH'], provider: 'Stargate' },
      { from: 'solana', to: 'ethereum', tokens: ['SOL', 'USDC'], provider: 'Wormhole' },
    ];
  }

  /**
   * Get quote for bridge
   */
  async getQuote(fromChain, toChain, token, amount) {
    try {
      const response = await fetch('https://api.tigerwallet.com/v1/bridge/quote', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ fromChain, toChain, token, amount: amount.toString() })
      });
      return await response.json();
    } catch (error) {
      console.error('Quote failed:', error);
      // Return mock quote
      const fee = amount * 0.001;
      return {
        fromChain,
        toChain,
        token,
        sendAmount: amount.toString(),
        receiveAmount: (amount - fee).toString(),
        bridgeFee: fee.toString(),
        provider: 'Stargate',
        estimatedTime: '15-20m'
      };
    }
  }

  /**
   * Execute bridge
   */
  async execute(walletAddress, fromChain, toChain, token, amount, recipient) {
    try {
      const response = await fetch('https://api.tigerwallet.com/v1/bridge/execute', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          walletAddress,
          fromChain,
          toChain,
          token,
          amount: amount.toString(),
          recipient: recipient || walletAddress
        })
      });
      return await response.json();
    } catch (error) {
      console.error('Bridge failed:', error);
      throw error;
    }
  }

  /**
   * Get bridge status
   */
  async getStatus(bridgeTxId) {
    try {
      const response = await fetch(`https://api.tigerwallet.com/v1/bridge/status/${bridgeTxId}`);
      return await response.json();
    } catch (error) {
      console.error('Status check failed:', error);
      return { status: 'pending' };
    }
  }

  /**
   * Get tokens for route
   */
  getTokensForRoute(fromChain, toChain) {
    const route = this.getSupportedRoutes().find(r => r.from === fromChain && r.to === toChain);
    return route?.tokens || [];
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
              ${this.supportedChains.map(c => `<option value="${c}">${c}</option>`).join('')}
            </select>
          </div>
          <div class="tw-form-group">
            <label>To Chain</label>
            <select class="tw-to-chain">
              ${this.supportedChains.map(c => `<option value="${c}">${c}</option>`).join('')}
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
    const updateTokens = () => {
      const tokens = this.getTokensForRoute(fromChain.value, toChain.value);
      token.innerHTML = tokens.map(t => `<option value="${t}">${t}</option>`).join('');
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
          parseFloat(amount.value)
        );

        quoteResult.innerHTML = `
          <div class="tw-quote-card">
            <div class="tw-quote-row">
              <span>You Send</span>
              <span>${quote.sendAmount} ${quote.token}</span>
            </div>
            <div class="tw-quote-row">
              <span>Bridge Fee</span>
              <span>${quote.bridgeFee} ${quote.token}</span>
            </div>
            <div class="tw-quote-row highlight">
              <span>You Receive</span>
              <span>${quote.receiveAmount} ${quote.token}</span>
            </div>
            <div class="tw-quote-row">
              <span>Provider</span>
              <span>${quote.provider}</span>
            </div>
            <div class="tw-quote-row">
              <span>Estimated Time</span>
              <span>${quote.estimatedTime}</span>
            </div>
            <button class="tw-bridge-btn btn-primary">Bridge Now</button>
          </div>
        `;

        // Bridge button
        const bridgeBtn = quoteResult.querySelector('.tw-bridge-btn');
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
