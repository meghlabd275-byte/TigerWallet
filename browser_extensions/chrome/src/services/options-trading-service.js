/**
 * TigerWallet Browser Extension - Options Trading Service
 * Complete implementation with real-time data
 */

class OptionsTradingService {
    constructor() {
        this.apiBase = 'https://api.tigerwallet.com/v1/options';
        this.cache = new Map();
        this.cacheTimeout = 5000;
    }

    async fetchWithCache(key, fetcher) {
        const cached = this.cache.get(key);
        if (cached && Date.now() - cached.timestamp < this.cacheTimeout) {
            return cached.data;
        }
        const data = await fetcher();
        this.cache.set(key, { data, timestamp: Date.now() });
        return data;
    }

    // Get available options chains
    async getChains() {
        return this.fetchWithCache('chains', async () => {
            return [
                { id: 'ethereum', name: 'Ethereum', underlying: 'ETH' },
                { id: 'bitcoin', name: 'Bitcoin', underlying: 'BTC' },
                { id: 'polygon', name: 'Polygon', underlying: 'MATIC' }
            ];
        });
    }

    // Get options for an underlying
    async getOptions(underlying, chain = 'ethereum') {
        return this.fetchWithCache(`options_${underlying}_${chain}`, async () => {
            const response = await fetch(`${this.apiBase}/${chain}/${underlying}/options`);
            if (!response.ok) throw new Error('Failed to fetch options');
            return response.json();
        });
    }

    // Get option details
    async getOptionDetails(optionAddress, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/option/${optionAddress}`);
        if (!response.ok) throw new Error('Failed to fetch option details');
        return response.json();
    }

    // Get Greeks (Delta, Gamma, Theta, Vega, Rho)
    async getGreeks(optionAddress, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/option/${optionAddress}/greeks`);
        if (!response.ok) throw new Error('Failed to fetch Greeks');
        return response.json();
    }

    // Get implied volatility surface
    async getIVSurface(underlying, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/${underlying}/iv-surface`);
        if (!response.ok) throw new Error('Failed to fetch IV surface');
        return response.json();
    }

    // Place buy order
    async buyOption(optionAddress, amount, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/order`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                optionAddress,
                side: 'BUY',
                amount,
                chain
            })
        });
        if (!response.ok) throw new Error('Failed to place buy order');
        return response.json();
    }

    // Place sell order
    async sellOption(optionAddress, amount, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/order`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                optionAddress,
                side: 'SELL',
                amount,
                chain
            })
        });
        if (!response.ok) throw new Error('Failed to place sell order');
        return response.json();
    }

    // Get user's positions
    async getPositions(userAddress, chain = 'ethereum') {
        return this.fetchWithCache(`positions_${userAddress}_${chain}`, async () => {
            const response = await fetch(`${this.apiBase}/${chain}/positions/${userAddress}`);
            if (!response.ok) throw new Error('Failed to fetch positions');
            return response.json();
        });
    }

    // Get order history
    async getOrderHistory(userAddress, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/orders/${userAddress}`);
        if (!response.ok) throw new Error('Failed to fetch order history');
        return response.json();
    }

    // Calculate option price (Black-Scholes)
    calculateBlackScholes(S, K, T, r, sigma, optionType = 'call') {
        const d1 = (Math.log(S / K) + (r + sigma * sigma / 2) * T) / (sigma * Math.sqrt(T));
        const d2 = d1 - sigma * Math.sqrt(T);
        
        const N = (x) => {
            const a1 = 0.254829592;
            const a2 = -0.284496736;
            const a3 = 1.421413741;
            const a4 = -1.453152027;
            const a5 = 1.061405429;
            const p = 0.3275911;
            const sign = x < 0 ? -1 : 1;
            x = Math.abs(x) / Math.sqrt(2);
            const t = 1.0 / (1.0 + p * x);
            const y = 1.0 - (((((a5 * t + a4) * t) + a3) * t + a2) * t + a1) * t * Math.exp(-x * x);
            return 0.5 * (1.0 + sign * y);
        };
        
        if (optionType === 'call') {
            return S * N(d1) - K * Math.exp(-r * T) * N(d2);
        } else {
            return K * Math.exp(-r * T) * N(-d2) - S * N(-d1);
        }
    }

    // Estimate gas for option trade
    async estimateGas(optionAddress, amount, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/estimate-gas`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ optionAddress, amount })
        });
        if (!response.ok) throw new Error('Failed to estimate gas');
        return response.json();
    }

    // Get open interest
    async getOpenInterest(underlying, chain = 'ethereum') {
        const response = await fetch(`${this.apiBase}/${chain}/${underlying}/open-interest`);
        if (!response.ok) throw new Error('Failed to fetch open interest');
        return response.json();
    }

    // Get volume
    async getVolume(underlying, chain = 'ethereum', period = '24h') {
        const response = await fetch(`${this.apiBase}/${chain}/${underlying}/volume?period=${period}`);
        if (!response.ok) throw new Error('Failed to fetch volume');
        return response.json();
    }
}

window.TigerWalletOptionsService = new OptionsTradingService();
