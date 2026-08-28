/**
 * TigerWallet Desktop - Bridge Service
 * Complete cross-chain bridge functionality
 */

class BridgeService {
    constructor() {
        this.apiBaseUrl = 'http://localhost:8443/api/v1';
        this.supportedRoutes = [];
        this._routesLoaded = false;
        this.bridgeProviders = ['stargate', 'layerzero', 'axelar', 'wormhole', 'allbridge'];
    }

    /**
     * Get supported bridge routes from the canonical bridge_service
     * (proxied via wallet_api at GET /bridge/routes). No hardcoded route
     * catalog; falls back to an empty list when the backend is unreachable.
     */
    async getSupportedRoutes() {
        if (this._routesLoaded) return this.supportedRoutes;
        try {
            const res = await fetch(`${this.apiBaseUrl}/bridge/routes`);
            if (res.ok) {
                const data = await res.json();
                const arr = Array.isArray(data) ? data : (data.routes || data.data || []);
                this.supportedRoutes = arr;
            }
        } catch (e) { /* backend unreachable: leave empty */ }
        this._routesLoaded = true;
        return this.supportedRoutes;
    }

    /**
     * Get bridge quote
     */
    async getQuote(fromChain, toChain, token, amount) {
        try {
            const response = await fetch(`${this.apiBaseUrl}/bridge/quote`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    fromChain,
                    toChain,
                    token,
                    amount: amount.toString()
                })
            });
            return await response.json();
        } catch (error) {
            // Fail-closed: never fabricate a bridge quote. Surface the real error
            // so the caller can retry against the canonical bridge service.
            console.error('Failed to get bridge quote:', error);
            throw new Error('Bridge quote unavailable: ' + (error?.message || 'backend unreachable'));
        }
    }

    /**
     * Execute bridge
     */
    async execute(walletAddress, fromChain, toChain, token, amount, recipientAddress) {
        try {
            const response = await fetch(`${this.apiBaseUrl}/bridge/execute`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    walletAddress,
                    fromChain,
                    toChain,
                    token,
                    amount: amount.toString(),
                    recipient: recipientAddress || walletAddress
                })
            });
            return await response.json();
        } catch (error) {
            console.error('Failed to execute bridge:', error);
            throw error;
        }
    }

    /**
     * Get bridge status
     */
    async getStatus(bridgeTxId) {
        try {
            const response = await fetch(`${this.apiBaseUrl}/bridge/status/${bridgeTxId}`);
            return await response.json();
        } catch (error) {
            console.error('Failed to get bridge status:', error);
            return null;
        }
    }

    /**
     * Get bridge history
     */
    async getHistory(walletAddress) {
        try {
            const response = await fetch(`${this.apiBaseUrl}/bridge/history?address=${walletAddress}`);
            return await response.json();
        } catch (error) {
            console.error('Failed to get bridge history:', error);
            return [];
        }
    }

    /**
     * Get supported tokens for route
     */
    getTokensForRoute(fromChain, toChain) {
        const route = this.supportedRoutes.find(
            r => r.from === fromChain && r.to === toChain
        );
        return route?.tokens || [];
    }

    /**
     * Get all available chains
     */
    getAvailableChains() {
        const chains = new Set();
        this.supportedRoutes.forEach(route => {
            chains.add(route.from);
            chains.add(route.to);
        });
        return Array.from(chains);
    }
}

// Export
if (typeof module !== 'undefined' && module.exports) {
    module.exports = BridgeService;
}
