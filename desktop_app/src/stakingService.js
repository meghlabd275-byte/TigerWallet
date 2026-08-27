/**
 * TigerWallet Desktop - Staking Service
 * Complete staking functionality for desktop app
 */

class StakingService {
    constructor() {
        this.apiBaseUrl = 'http://localhost:8443/api/v1';
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
     * Fallback validators — returns an empty list (fail-closed) rather than
     * fabricating APR/TVL/risk market data. The canonical backend exposes
     * real validators via the staking quote/action endpoints and
     * /admin/chains/validators (admin-managed); when those are unreachable we
     * must not invent yield numbers or validator identities.
     */
    getDefaultValidators(chain) {
        console.warn('No real validator data available for', chain, '— returning empty (fail-closed)');
        return [];
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
