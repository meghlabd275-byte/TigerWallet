/**
 * TigerWallet Browser Extension - Liquid Staking Service
 */

class LiquidStakingService {
    constructor() {
        this.apiBase = 'http://localhost:8443/api/v1/liquid-staking';
    }

    // Get available liquid staking protocols
    async getProtocols(chain) {
        const response = await fetch(`${this.apiBase}/${chain}/protocols`);
        if (!response.ok) throw new Error('Failed to fetch protocols');
        return response.json();
    }

    // Get protocol details
    async getProtocolDetails(protocolId, chain) {
        const response = await fetch(`${this.apiBase}/${chain}/protocol/${protocolId}`);
        if (!response.ok) throw new Error('Failed to fetch protocol');
        return response.json();
    }

    // Stake and get liquid token
    async stake(protocolId, amount, address, chain) {
        const response = await fetch(`${this.apiBase}/${chain}/stake`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ protocolId, amount, address })
        });
        if (!response.ok) throw new Error('Failed to stake');
        return response.json();
    }

    // Unstake (burn liquid token)
    async unstake(protocolId, amount, address, chain) {
        const response = await fetch(`${this.apiBase}/${chain}/unstake`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ protocolId, amount, address })
        });
        if (!response.ok) throw new Error('Failed to unstake');
        return response.json();
    }

    // Get staked balance
    async getStakedBalance(address, chain) {
        const response = await fetch(`${this.apiBase}/${chain}/balance/${address}`);
        if (!response.ok) throw new Error('Failed to fetch balance');
        return response.json();
    }

    // Get rewards
    async getRewards(address, chain) {
        const response = await fetch(`${this.apiBase}/${chain}/rewards/${address}`);
        if (!response.ok) throw new Error('Failed to fetch rewards');
        return response.json();
    }

    // Claim rewards
    async claimRewards(address, chain) {
        const response = await fetch(`${this.apiBase}/${chain}/claim`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ address })
        });
        if (!response.ok) throw new Error('Failed to claim rewards');
        return response.json();
    }

    // Get liquid token price
    async getLiquidTokenPrice(protocolId, chain) {
        const response = await fetch(`${this.apiBase}/${chain}/price/${protocolId}`);
        if (!response.ok) throw new Error('Failed to fetch price');
        return response.json();
    }

    // Get staking history
    async getHistory(address, chain) {
        const response = await fetch(`${this.apiBase}/${chain}/history/${address}`);
        if (!response.ok) throw new Error('Failed to fetch history');
        return response.json();
    }
}

window.TigerWalletLiquidStakingService = new LiquidStakingService();
