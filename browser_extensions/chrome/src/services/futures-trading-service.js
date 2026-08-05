/**
 * TigerWallet Browser Extension - Futures Trading Service
 */

class FuturesTradingService {
    constructor() {
        this.apiBase = 'https://api.tigerwallet.com/v1/futures';
    }

    // Get available futures contracts
    async getContracts(chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/contracts`);
        if (!response.ok) throw new Error('Failed to fetch contracts');
        return response.json();
    }

    // Get contract details
    async getContract(symbol, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/contract/${symbol}`);
        if (!response.ok) throw new Error('Failed to fetch contract');
        return response.json();
    }

    // Get mark price and funding rate
    async getMarkPrice(symbol, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/mark-price/${symbol}`);
        if (!response.ok) throw new Error('Failed to fetch mark price');
        return response.json();
    }

    // Open long position
    async openLong(symbol, amount, leverage, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/position`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ symbol, side: 'LONG', amount, leverage })
        });
        if (!response.ok) throw new Error('Failed to open position');
        return response.json();
    }

    // Open short position
    async openShort(symbol, amount, leverage, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/position`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ symbol, side: 'SHORT', amount, leverage })
        });
        if (!response.ok) throw new Error('Failed to open position');
        return response.json();
    }

    // Close position
    async closePosition(positionId, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/position/${positionId}`, {
            method: 'DELETE'
        });
        if (!response.ok) throw new Error('Failed to close position');
        return response.json();
    }

    // Get user's positions
    async getPositions(address, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/positions/${address}`);
        if (!response.ok) throw new Error('Failed to fetch positions');
        return response.json();
    }

    // Get position history
    async getPositionHistory(address, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/history/${address}`);
        if (!response.ok) throw new Error('Failed to fetch history');
        return response.json();
    }

    // Get funding rate history
    async getFundingHistory(symbol, chain = 'ethereum', limit = 100) {
        const response = await fetch(`${this.apiBase}/${chain}/funding/${symbol}?limit=${limit}`);
        if (!response.ok) throw new Error('Failed to fetch funding history');
        return response.json();
    }

    // Get open interest
    async getOpenInterest(symbol, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/open-interest/${symbol}`);
        if (!response.ok) throw new Error('Failed to fetch open interest');
        return response.json();
    }

    // Get liquidation price
    calculateLiquidationPrice(entryPrice, leverage, side) {
        const liquidationPercent = 1 / leverage;
        if (side === 'LONG') {
            return entryPrice * (1 - liquidationPercent);
        } else {
            return entryPrice * (1 + liquidationPercent);
        }
    }

    // Get estimated PnL
    calculatePnL(entryPrice, currentPrice, amount, leverage, side) {
        const priceChange = side === 'LONG' 
            ? (currentPrice - entryPrice) / entryPrice
            : (entryPrice - currentPrice) / entryPrice;
        return amount * leverage * priceChange;
    }
}

window.TigerWalletFuturesService = new FuturesTradingService();
