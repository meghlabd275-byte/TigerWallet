/**
 * TigerWallet Browser Extension - P2P Trading Service
 */

class P2PTradingService {
    constructor() {
        this.apiBase = 'https://api.tigerwallet.com/v1/p2p';
    }

    // Get available orders
    async getOrders(filters = {}) {
        const params = new URLSearchParams(filters).toString();
        const response = await fetch(`${this.apiBase}/orders?${params}`);
        if (!response.ok) throw new Error('Failed to fetch orders');
        return response.json();
    }

    // Get order by ID
    async getOrder(orderId) {
        const response = await fetch(`${this.apiBase}/order/${orderId}`);
        if (!response.ok) throw new Error('Failed to fetch order');
        return response.json();
    }

    // Create buy order
    async createBuyOrder(toToken, fromToken, amount, price, paymentMethods) {
        const response = await fetch(`${this.apiBase}/orders`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                side: 'BUY',
                toToken,
                fromToken,
                amount,
                price,
                paymentMethods
            })
        });
        if (!response.ok) throw new Error('Failed to create order');
        return response.json();
    }

    // Create sell order
    async createSellOrder(fromToken, toToken, amount, price, paymentMethods) {
        const response = await fetch(`${this.apiBase}/orders`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                side: 'SELL',
                fromToken,
                toToken,
                amount,
                price,
                paymentMethods
            })
        });
        if (!response.ok) throw new Error('Failed to create order');
        return response.json();
    }

    // Cancel order
    async cancelOrder(orderId) {
        const response = await fetch(`${this.apiBase}/order/${orderId}`, {
            method: 'DELETE'
        });
        if (!response.ok) throw new Error('Failed to cancel order');
        return response.json();
    }

    // Accept order (take order)
    async acceptOrder(orderId, address) {
        const response = await fetch(`${this.apiBase}/order/${orderId}/accept`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ address })
        });
        if (!response.ok) throw new Error('Failed to accept order');
        return response.json();
    }

    // Release crypto (after payment)
    async releaseCrypto(tradeId) {
        const response = await fetch(`${this.apiBase}/trade/${tradeId}/release`, {
            method: 'POST'
        });
        if (!response.ok) throw new Error('Failed to release crypto');
        return response.json();
    }

    // Dispute trade
    async disputeTrade(tradeId, reason) {
        const response = await fetch(`${this.apiBase}/trade/${tradeId}/dispute`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ reason })
        });
        if (!response.ok) throw new Error('Failed to open dispute');
        return response.json();
    }

    // Get user's orders
    async getMyOrders(address) {
        const response = await fetch(`${this.apiBase}/my-orders/${address}`);
        if (!response.ok) throw new Error('Failed to fetch orders');
        return response.json();
    }

    // Get user's trades
    async getMyTrades(address) {
        const response = await fetch(`${this.apiBase}/my-trades/${address}`);
        if (!response.ok) throw new Error('Failed to fetch trades');
        return response.json();
    }

    // Get payment methods
    async getPaymentMethods() {
        return [
            { id: 'bank_transfer', name: 'Bank Transfer' },
            { id: 'upi', name: 'UPI' },
            { id: 'paytm', name: 'PayTM' },
            { id: 'google_pay', name: 'Google Pay' },
            { id: 'phonepe', name: 'PhonePe' },
            { id: 'cash', name: 'Cash Deposit' }
        ];
    }

    // Get merchant status
    async getMerchantStatus(address) {
        const response = await fetch(`${this.apiBase}/merchant/${address}`);
        if (!response.ok) return { isMerchant: false };
        return response.json();
    }

    // Become merchant
    async becomeMerchant(details) {
        const response = await fetch(`${this.apiBase}/merchant`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(details)
        });
        if (!response.ok) throw new Error('Failed to become merchant');
        return response.json();
    }
}

window.TigerWalletP2PService = new P2PTradingService();
