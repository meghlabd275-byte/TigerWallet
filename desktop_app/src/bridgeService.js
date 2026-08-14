/**
 * TigerWallet Desktop - Bridge Service
 * Complete cross-chain bridge functionality
 */

class BridgeService {
    constructor() {
        this.apiBaseUrl = 'http://localhost:8443/api/v1';
        this.supportedRoutes = this.getSupportedRoutes();
        this.bridgeProviders = ['stargate', 'layerzero', 'axelar', 'wormhole', 'allbridge'];
    }

    /**
     * Get supported bridge routes
     */
    getSupportedRoutes() {
        return [
            // Ethereum routes
            { from: 'ethereum', to: 'polygon', tokens: ['ETH', 'USDT', 'USDC', 'MATIC'], provider: 'stargate', time: '10-15m' },
            { from: 'ethereum', to: 'arbitrum', tokens: ['ETH', 'USDT', 'USDC'], provider: 'layerzero', time: '15-20m' },
            { from: 'ethereum', to: 'optimism', tokens: ['ETH', 'USDT', 'USDC'], provider: 'layerzero', time: '15-20m' },
            { from: 'ethereum', to: 'avalanche', tokens: ['ETH', 'USDT', 'USDC'], provider: 'axelar', time: '20-30m' },
            { from: 'ethereum', to: 'bsc', tokens: ['ETH', 'BNB', 'USDT'], provider: 'stargate', time: '5-10m' },
            { from: 'ethereum', to: 'base', tokens: ['ETH', 'USDC'], provider: 'native', time: '5m' },
            
            // Polygon routes
            { from: 'polygon', to: 'ethereum', tokens: ['MATIC', 'USDT', 'USDC'], provider: 'stargate', time: '10-15m' },
            { from: 'polygon', to: 'arbitrum', tokens: ['MATIC', 'USDC'], provider: 'layerzero', time: '15-20m' },
            
            // BSC routes
            { from: 'bsc', to: 'ethereum', tokens: ['BNB', 'ETH', 'USDT'], provider: 'stargate', time: '5-10m' },
            { from: 'bsc', to: 'polygon', tokens: ['BNB', 'USDT'], provider: 'axelar', time: '15-20m' },
            
            // Avalanche routes
            { from: 'avalanche', to: 'ethereum', tokens: ['AVAX', 'USDT', 'USDC'], provider: 'axelar', time: '20-30m' },
            
            // Solana routes
            { from: 'solana', to: 'ethereum', tokens: ['SOL', 'USDC'], provider: 'wormhole', time: '15-20m' },
            { from: 'solana', to: 'polygon', tokens: ['SOL', 'USDC'], provider: 'wormhole', time: '20-30m' },
            
            // Arbitrum routes
            { from: 'arbitrum', to: 'ethereum', tokens: ['ETH', 'USDC'], provider: 'layerzero', time: '15-20m' },
            
            // Optimism routes
            { from: 'optimism', to: 'ethereum', tokens: ['ETH', 'USDC'], provider: 'native', time: '15m' },
        ];
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
