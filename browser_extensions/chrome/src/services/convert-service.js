/**
 * TigerWallet Browser Extension - Convert Service
 * Instant token conversion
 */

class ConvertService {
    constructor() {
        this.apiBase = 'http://localhost:8443/api/v1/convert';
        this.cache = new Map();
    }

    async getQuote(fromToken, toToken, amount) {
        const cacheKey = `convert_${fromToken}_${toToken}_${amount}`;
        const cached = this.cache.get(cacheKey);
        if (cached && Date.now() - cached.timestamp < 10000) {
            return cached.data;
        }

        const response = await fetch(`${this.apiBase}/quote`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ fromToken, toToken, amount })
        });
        
        if (!response.ok) throw new Error('Failed to get quote');
        const data = await response.json();
        this.cache.set(cacheKey, { data, timestamp: Date.now() });
        return data;
    }

    async executeConversion(quoteId, fromToken, toToken, amount, fromAddress) {
        const response = await fetch(`${this.apiBase}/execute`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                quoteId,
                fromToken,
                toToken,
                amount,
                fromAddress
            })
        });
        
        if (!response.ok) throw new Error('Failed to execute conversion');
        return response.json();
    }

    async getSupportedTokens() {
        const response = await fetch(`${this.apiBase}/tokens`);
        if (!response.ok) throw new Error('Failed to fetch tokens');
        return response.json();
    }

    async getConversionHistory(address) {
        const response = await fetch(`${this.apiBase}/history/${address}`);
        if (!response.ok) throw new Error('Failed to fetch history');
        return response.json();
    }
}

window.TigerWalletConvertService = new ConvertService();
