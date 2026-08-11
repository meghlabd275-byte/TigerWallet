/**
 * TigerWallet Browser Extension - Prediction Markets Service
 */

class PredictionMarketsService {
    constructor() {
        this.apiBase = 'http://localhost:8443/api/v1/predictions';
    }

    // Get markets
    async getMarkets(status = 'active') {
        const response = await fetch(`${this.apiBase}/markets?status=${status}`);
        if (!response.ok) throw new Error('Failed to fetch markets');
        return response.json();
    }

    // Get market details
    async getMarketDetails(marketId) {
        const response = await fetch(`${this.apiBase}/market/${marketId}`);
        if (!response.ok) throw new Error('Failed to fetch market');
        return response.json();
    }

    // Place bet
    async placeBet(marketId, outcome, amount, address) {
        const response = await fetch(`${this.apiBase}/bet`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ marketId, outcome, amount, address })
        });
        if (!response.ok) throw new Error('Failed to place bet');
        return response.json();
    }

    // Resolve market (admin)
    async resolveMarket(marketId, outcome) {
        const response = await fetch(`${this.apiBase}/market/${marketId}/resolve`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ outcome })
        });
        if (!response.ok) throw new Error('Failed to resolve');
        return response.json();
    }

    // Get user bets
    async getUserBets(address) {
        const response = await fetch(`${this.apiBase}/bets/${address}`);
        if (!response.ok) throw new Error('Failed to fetch bets');
        return response.json();
    }

    // Get market history
    async getMarketHistory(marketId) {
        const response = await fetch(`${this.apiBase}/market/${marketId}/history`);
        if (!response.ok) throw new Error('Failed to fetch history');
        return response.json();
    }

    // Get odds
    async getOdds(marketId) {
        const response = await fetch(`${this.apiBase}/market/${marketId}/odds`);
        if (!response.ok) throw new Error('Failed to fetch odds');
        return response.json();
    }
}

window.TigerWalletPredictionService = new PredictionMarketsService();
