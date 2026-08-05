/**
 * TigerWallet Browser Extension - Staking Service
 */

class StakingService {
    constructor() {
        this.apiBase = 'https://api.tigerwallet.com/v1/staking';
    }

    // Get staking chains
    async getChains() {
        const response = await fetch(`${this.apiBase}/chains`);
        if (!response.ok) throw new Error('Failed to fetch chains');
        return response.json();
    }

    // Get validators for a chain
    async getValidators(chain) {
        const response = await fetch(`${this.apiBase}/${chain}/validators`);
        if (!response.ok) throw new Error('Failed to fetch validators');
        return response.json();
    }

    // Get validator details
    async getValidator(chain, validatorAddress) {
        const response = await fetch(`${this.apiBase}/${chain}/validator/${validatorAddress}`);
        if (!response.ok) throw new Error('Failed to fetch validator');
        return response.json();
    }

    // Stake
    async stake(chain, validatorAddress, amount, address) {
        const response = await fetch(`${this.apiBase}/${chain}/stake`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ validatorAddress, amount, address })
        });
        if (!response.ok) throw new Error('Failed to stake');
        return response.json();
    }

    // Unstake
    async unstake(chain, validatorAddress, amount, address) {
        const response = await fetch(`${this.apiBase}/${chain}/unstake`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ validatorAddress, amount, address })
        });
        if (!response.ok) throw new Error('Failed to unstake');
        return response.json();
    }

    // Claim rewards
    async claimRewards(chain, validatorAddress, address) {
        const response = await fetch(`${this.apiBase}/${chain}/claim`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ validatorAddress, address })
        });
        if (!response.ok) throw new Error('Failed to claim rewards');
        return response.json();
    }

    // Get staking positions
    async getPositions(address, chain) {
        const response = await fetch(`${this.apiBase}/${chain}/positions/${address}`);
        if (!response.ok) throw new Error('Failed to fetch positions');
        return response.json();
    }

    // Get pending unstaking
    async getPendingUnstakes(address, chain) {
        const response = await fetch(`${this.apiBase}/${chain}/pending/${address}`);
        if (!response.ok) throw new Error('Failed to fetch pending');
        return response.json();
    }

    // Get rewards history
    async getRewardsHistory(address, chain) {
        const response = await fetch(`${this.apiBase}/${chain}/rewards/${address}`);
        if (!response.ok) throw new Error('Failed to fetch rewards');
        return response.json();
    }

    // Calculate APY
    calculateAPY(rewardRate, compoundingPeriods = 365) {
        return Math.pow(1 + rewardRate, compoundingPeriods) - 1;
    }
}

window.TigerWalletStakingService = new StakingService();
