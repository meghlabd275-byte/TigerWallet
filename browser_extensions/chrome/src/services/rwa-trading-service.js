/**
 * TigerWallet Browser Extension - RWA Trading Service
 */

class RWATradingService {
    constructor() {
        this.apiBase = 'http://localhost:8443/api/v1/rwa';
    }

    // Get available RWA assets
    async getAssets(filters = {}) {
        const params = new URLSearchParams(filters).toString();
        const response = await fetch(`${this.apiBase}/assets?${params}`);
        if (!response.ok) throw new Error('Failed to fetch assets');
        return response.json();
    }

    // Get asset details
    async getAssetDetails(assetId) {
        const response = await fetch(`${this.apiBase}/asset/${assetId}`);
        if (!response.ok) throw new Error('Failed to fetch details');
        return response.json();
    }

    // Buy RWA
    async buy(assetId, amount, address) {
        const response = await fetch(`${this.apiBase}/buy`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ assetId, amount, address })
        });
        if (!response.ok) throw new Error('Failed to buy');
        return response.json();
    }

    // Sell RWA
    async sell(assetId, amount, address) {
        const response = await fetch(`${this.apiBase}/sell`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ assetId, amount, address })
        });
        if (!response.ok) throw new Error('Failed to sell');
        return response.json();
    }

    // Create sell order
    async createOrder(assetId, amount, price, address) {
        const response = await fetch(`${this.apiBase}/order`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ assetId, amount, price, address })
        });
        if (!response.ok) throw new Error('Failed to create order');
        return response.json();
    }

    // Cancel order
    async cancelOrder(orderId, address) {
        const response = await fetch(`${this.apiBase}/order/${orderId}`, {
            method: 'DELETE',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ address })
        });
        if (!response.ok) throw new Error('Failed to cancel');
        return response.json();
    }

    // Get user holdings
    async getHoldings(address) {
        const response = await fetch(`${this.apiBase}/holdings/${address}`);
        if (!response.ok) throw new Error('Failed to fetch holdings');
        return response.json();
    }

    // Get user orders
    async getOrders(address) {
        const response = await fetch(`${this.apiBase}/orders/${address}`);
        if (!response.ok) throw new Error('Failed to fetch orders');
        return response.json();
    }

    // Get transaction history
    async getTransactions(address) {
        const response = await fetch(`${this.apiBase}/transactions/${address}`);
        if (!response.ok) throw new Error('Failed to fetch transactions');
        return response.json();
    }

    // Get market data
    async getMarketData(assetId) {
        const response = await fetch(`${this.apiBase}/market/${assetId}`);
        if (!response.ok) throw new Error('Failed to fetch market data');
        return response.json();
    }
}

window.TigerWalletRWAService = new RWATradingService();
