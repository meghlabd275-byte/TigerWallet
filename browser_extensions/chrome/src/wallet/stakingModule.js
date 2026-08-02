/**
 * TigerWallet Browser Extension - Staking Module
 * Complete staking functionality for browser extension
 */

class StakingModule {
  constructor(walletManager) {
    this.walletManager = walletManager;
    this.stakingChains = ['ethereum', 'polygon', 'solana'];
  }

  /**
   * Get staking positions
   */
  async getPositions(walletAddress) {
    try {
      const response = await fetch(`https://api.tigerwallet.com/v1/staking/positions?address=${walletAddress}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get positions:', error);
      return [];
    }
  }

  /**
   * Get validators for chain
   */
  async getValidators(chain) {
    try {
      const response = await fetch(`https://api.tigerwallet.com/v1/staking/validators?chain=${chain}`);
      return await response.json();
    } catch (error) {
      console.error('Failed to get validators:', error);
      return this.getDefaultValidators(chain);
    }
  }

  getDefaultValidators(chain) {
    const defaults = {
      ethereum: [
        { id: 'lido', name: 'Lido', apr: 4.2, minStake: '0.01' },
        { id: 'rocketpool', name: 'Rocket Pool', apr: 3.8, minStake: '0.01' },
        { id: 'coinbase', name: 'Coinbase Staking', apr: 3.5, minStake: '0.01' },
      ],
      polygon: [
        { id: 'ankr', name: 'Ankr', apr: 5.2, minStake: '10' },
        { id: 'stader', name: 'Stader', apr: 4.8, minStake: '10' },
      ],
      solana: [
        { id: 'marinade', name: 'Marinade Finance', apr: 6.5, minStake: '1' },
        { id: 'jpool', name: 'JPool', apr: 6.2, minStake: '1' },
      ]
    };
    return defaults[chain] || [];
  }

  /**
   * Stake tokens
   */
  async stake(walletAddress, chain, validatorId, amount) {
    try {
      const response = await fetch('https://api.tigerwallet.com/v1/staking/stake', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          walletAddress,
          chain,
          validatorId,
          amount: amount.toString()
        })
      });
      return await response.json();
    } catch (error) {
      console.error('Stake failed:', error);
      throw error;
    }
  }

  /**
   * Unstake tokens
   */
  async unstake(positionId) {
    try {
      const response = await fetch('https://api.tigerwallet.com/v1/staking/unstake', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ positionId })
      });
      return await response.json();
    } catch (error) {
      console.error('Unstake failed:', error);
      throw error;
    }
  }

  /**
   * Claim rewards
   */
  async claimRewards(positionId) {
    try {
      const response = await fetch('https://api.tigerwallet.com/v1/staking/claim', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ positionId })
      });
      return await response.json();
    } catch (error) {
      console.error('Claim failed:', error);
      throw error;
    }
  }

  /**
   * Create staking popup UI
   */
  createStakingUI() {
    const container = document.createElement('div');
    container.className = 'tigerwallet-staking-popup';
    container.innerHTML = `
      <div class="tw-staking-header">
        <h2>📈 Staking</h2>
        <button class="tw-close-btn">&times;</button>
      </div>
      <div class="tw-staking-content">
        <div class="tw-chain-selector">
          <button class="tw-chain-btn active" data-chain="ethereum">ETH</button>
          <button class="tw-chain-btn" data-chain="polygon">MATIC</button>
          <button class="tw-chain-btn" data-chain="solana">SOL</button>
        </div>
        <div class="tw-validators-list">
          <div class="tw-loading">Loading validators...</div>
        </div>
      </div>
    `;
    return container;
  }

  /**
   * Render staking interface
   */
  async renderStakingInterface(container, walletAddress) {
    const validators = await this.getValidators('ethereum');
    
    const validatorsList = container.querySelector('.tw-validators-list');
    validatorsList.innerHTML = validators.map(v => `
      <div class="tw-validator-card" data-validator-id="${v.id}">
        <div class="tw-validator-name">${v.name}</div>
        <div class="tw-validator-apr">${v.apr}% APR</div>
        <div class="tw-validator-min">Min: ${v.minStake}</div>
      </div>
    `).join('');

    // Chain selector
    container.querySelectorAll('.tw-chain-btn').forEach(btn => {
      btn.addEventListener('click', async (e) => {
        container.querySelectorAll('.tw-chain-btn').forEach(b => b.classList.remove('active'));
        e.target.classList.add('active');
        
        const chain = e.target.dataset.chain;
        const validators = await this.getValidators(chain);
        
        validatorsList.innerHTML = validators.map(v => `
          <div class="tw-validator-card" data-validator-id="${v.id}">
            <div class="tw-validator-name">${v.name}</div>
            <div class="tw-validator-apr">${v.apr}% APR</div>
            <div class="tw-validator-min">Min: ${v.minStake}</div>
          </div>
        `).join('');
      });
    });
  }
}

// Export for extension
if (typeof module !== 'undefined' && module.exports) {
  module.exports = StakingModule;
}
