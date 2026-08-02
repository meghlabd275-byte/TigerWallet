/**
 * TigerWallet Desktop - Staking Service
 * Complete staking functionality for desktop app
 */

class StakingService {
    constructor() {
        this.apiBaseUrl = 'https://api.tigerwallet.com/v1';
        this.stakingChains = ['ethereum', 'polygon', 'solana', 'cosmos', 'polkadot', 'near'];
        this.validators = {};
    }

    /**
     * Get staking positions for wallet
     */
    async getPositions(walletAddress) {
        try {
            const response = await fetch(`${this.apiBaseUrl}/staking/positions?address=${walletAddress}`);
            return await response.json();
        } catch (error) {
            console.error('Failed to get staking positions:', error);
            return [];
        }
    }

    /**
     * Get validators for chain
     */
    async getValidators(chain) {
        if (this.validators[chain]) {
            return this.validators[chain];
        }

        try {
            const response = await fetch(`${this.apiBaseUrl}/staking/validators?chain=${chain}`);
            const validators = await response.json();
            this.validators[chain] = validators;
            return validators;
        } catch (error) {
            console.error('Failed to get validators:', error);
            return this.getDefaultValidators(chain);
        }
    }

    /**
     * Get default validators
     */
    getDefaultValidators(chain) {
        const defaults = {
            ethereum: [
                { id: 'lido', name: 'Lido', apr: 4.2, tvl: '15000000000', risk: 'low' },
                { id: 'rocketpool', name: 'Rocket Pool', apr: 3.8, tvl: '1500000000', risk: 'low' },
                { id: 'coinbase', name: 'Coinbase Staking', apr: 3.5, tvl: '8000000000', risk: 'low' },
                { id: 'kraken', name: 'Kraken Staking', apr: 4.0, tvl: '5000000000', risk: 'medium' },
            ],
            polygon: [
                { id: 'ankr', name: 'Ankr', apr: 5.2, tvl: '500000000', risk: 'medium' },
                { id: 'stader', name: 'Stader', apr: 4.8, tvl: '300000000', risk: 'medium' },
            ],
            solana: [
                { id: 'marinade', name: 'Marinade Finance', apr: 6.5, tvl: '400000000', risk: 'low' },
                { id: 'jpool', name: 'JPool', apr: 6.2, tvl: '200000000', risk: 'medium' },
                { id: 'lido-sol', name: 'Lido (SOL)', apr: 5.8, tvl: '300000000', risk: 'low' },
            ]
        };
        return defaults[chain] || [];
    }

    /**
     * Stake tokens
     */
    async stake(walletAddress, chain, validatorId, amount) {
        try {
            const response = await fetch(`${this.apiBaseUrl}/staking/stake`, {
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
            console.error('Failed to stake:', error);
            throw error;
        }
    }

    /**
     * Unstake tokens
     */
    async unstake(positionId) {
        try {
            const response = await fetch(`${this.apiBaseUrl}/staking/unstake`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ positionId })
            });
            return await response.json();
        } catch (error) {
            console.error('Failed to unstake:', error);
            throw error;
        }
    }

    /**
     * Claim staking rewards
     */
    async claimRewards(positionId) {
        try {
            const response = await fetch(`${this.apiBaseUrl}/staking/claim`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ positionId })
            });
            return await response.json();
        } catch (error) {
            console.error('Failed to claim rewards:', error);
            throw error;
        }
    }

    /**
     * Get staking rewards
     */
    async getRewards(walletAddress) {
        try {
            const response = await fetch(`${this.apiBaseUrl}/staking/rewards?address=${walletAddress}`);
            return await response.json();
        } catch (error) {
            console.error('Failed to get rewards:', error);
            return [];
        }
    }

    /**
     * Calculate staking rewards
     */
    calculateRewards(amount, apr, days) {
        const dailyRate = apr / 100 / 365;
        return amount * dailyRate * days;
    }
}

// Export
if (typeof module !== 'undefined' && module.exports) {
    module.exports = StakingService;
}
