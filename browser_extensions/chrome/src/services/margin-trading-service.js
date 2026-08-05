/**
 * TigerWallet Browser Extension - Margin Trading Service
 */

class MarginTradingService {
    constructor() {
        this.apiBase = 'https://api.tigerwallet.com/v1/margin';
    }

    // Get available pools
    async getPools(chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/pools`);
        if (!response.ok) throw new Error('Failed to fetch pools');
        return response.json();
    }

    // Get pool details
    async getPoolDetails(poolAddress, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/pool/${poolAddress}`);
        if (!response.ok) throw new Error('Failed to fetch pool details');
        return response.json();
    }

    // Supply collateral
    async supply(poolAddress, amount, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/supply`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ poolAddress, amount })
        });
        if (!response.ok) throw new Error('Failed to supply');
        return response.json();
    }

    // Withdraw collateral
    async withdraw(poolAddress, amount, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/withdraw`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ poolAddress, amount })
        });
        if (!response.ok) throw new Error('Failed to withdraw');
        return response.json();
    }

    // Borrow
    async borrow(poolAddress, amount, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/borrow`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ poolAddress, amount })
        });
        if (!response.ok) throw new Error('Failed to borrow');
        return response.json();
    }

    // Repay
    async repay(poolAddress, amount, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/repay`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ poolAddress, amount })
        });
        if (!response.ok) throw new Error('Failed to repay');
        return response.json();
    }

    // Get user's positions
    async getPositions(address, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/positions/${address}`);
        if (!response.ok) throw new Error('Failed to fetch positions');
        return response.json();
    }

    // Get account health
    async getHealthFactor(address, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/health/${address}`);
        if (!response.ok) throw new Error('Failed to fetch health factor');
        return response.json();
    }

    // Calculate health factor
    calculateHealthFactor(collateralValue, borrowedValue, liquidationThreshold) {
        if (borrowedValue === 0) return Infinity;
        return (collateralValue * liquidationThreshold) / borrowedValue;
    }

    // Get liquidation price
    getLiquidationPrice(collateral, debt, liquidationThreshold) {
        return (debt / liquidationThreshold) / collateral;
    }
}

window.TigerWalletMarginService = new MarginTradingService();
