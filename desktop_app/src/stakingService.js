/**
 * TigerWallet Desktop - Staking Service
 * Complete staking functionality for desktop app
 */

class StakingService {
    constructor() {
        // Canonical UserWallet staking backend (wallet_api :8443). Mirrors the
        // user_wallet/web client contract: GET /staking/quote + POST
        // /staking/{stake,unstake,claim} with wallet_id + password + asset +
        // amount + chain_id. No fabricated APY/positions/validators.
        this.apiBaseUrl = 'http://localhost:8443/api/v1';
        this.stakingChains = ['ethereum', 'polygon', 'solana', 'cosmos', 'polkadot', 'near'];
        this.validators = {};
    }

    /**
     * Get the supported native staking assets (real quote; APY is 0 until a
     * live staking contract/oracle is configured — never fabricated).
     */
    async getStakingQuote() {
        try {
            const response = await fetch(`${this.apiBaseUrl}/staking/quote`);
            return await response.json();
        } catch (error) {
            console.error('Failed to get staking quote:', error);
            return { success: false, assets: [], apy: 0, min_stake: 0, lock_period: 0 };
        }
    }

    /**
     * Stake tokens (on-chain transaction, submitted via /api/v1/send).
     */
    async stake(walletId, password, asset, amount, chainId = 1, validator = '') {
        try {
            const response = await fetch(`${this.apiBaseUrl}/staking/stake`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    wallet_id: walletId,
                    password,
                    asset,
                    amount: amount.toString(),
                    chain_id: chainId,
                    validator,
                }),
            });
            return await response.json();
        } catch (error) {
            console.error('Failed to stake:', error);
            throw error;
        }
    }

    /**
     * Unstake tokens (on-chain transaction, submitted via /api/v1/send).
     */
    async unstake(walletId, password, asset, amount, chainId = 1) {
        try {
            const response = await fetch(`${this.apiBaseUrl}/staking/unstake`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    wallet_id: walletId,
                    password,
                    asset,
                    amount: amount.toString(),
                    chain_id: chainId,
                }),
            });
            return await response.json();
        } catch (error) {
            console.error('Failed to unstake:', error);
            throw error;
        }
    }

    /**
     * Claim staking rewards (on-chain transaction, submitted via /api/v1/send).
     */
    async claimRewards(walletId, password, asset, chainId = 1) {
        try {
            const response = await fetch(`${this.apiBaseUrl}/staking/claim`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    wallet_id: walletId,
                    password,
                    asset,
                    chain_id: chainId,
                }),
            });
            return await response.json();
        } catch (error) {
            console.error('Failed to claim rewards:', error);
            throw error;
        }
    }

    /**
     * Calculate staking rewards (pure arithmetic — no fabricated yield).
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
