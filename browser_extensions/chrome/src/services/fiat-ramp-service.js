/**
 * TigerWallet Browser Extension - Fiat Ramp Service
 */

class FiatRampService {
    constructor() {
        this.apiBase = 'http://localhost:8443/api/v1/fiat';
    }

    // Get available providers
    async getProviders() {
        const response = await fetch(`${this.apiBase}/providers`);
        if (!response.ok) throw new Error('Failed to fetch providers');
        return response.json();
    }

    // Get buy quotes
    async getBuyQuotes(fromCurrency, toCrypto, amount) {
        const response = await fetch(`${this.apiBase}/buy/quotes`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ fromCurrency, toCrypto, amount })
        });
        if (!response.ok) throw new Error('Failed to fetch quotes');
        return response.json();
    }

    // Get sell quotes
    async getSellQuotes(fromCrypto, toCurrency, amount) {
        const response = await fetch(`${this.apiBase}/sell/quotes`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ fromCrypto, toCurrency, amount })
        });
        if (!response.ok) throw new Error('Failed to fetch quotes');
        return response.json();
    }

    // Create buy order
    async createBuyOrder(providerId, fromCurrency, toCrypto, amount, paymentMethod) {
        const response = await fetch(`${this.apiBase}/buy/order`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ providerId, fromCurrency, toCrypto, amount, paymentMethod })
        });
        if (!response.ok) throw new Error('Failed to create order');
        return response.json();
    }

    // Create sell order
    async createSellOrder(providerId, fromCrypto, toCurrency, amount, bankAccount) {
        const response = await fetch(`${this.apiBase}/sell/order`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ providerId, fromCrypto, toCurrency, amount, bankAccount })
        });
        if (!response.ok) throw new Error('Failed to create order');
        return response.json();
    }

    // Get order status
    async getOrderStatus(orderId) {
        const response = await fetch(`${this.apiBase}/order/${orderId}`);
        if (!response.ok) throw new Error('Failed to fetch order status');
        return response.json();
    }

    // Get supported currencies
    async getSupportedCurrencies() {
        const response = await fetch(`${this.apiBase}/currencies`);
        if (!response.ok) throw new Error('Failed to fetch currencies');
        return response.json();
    }

    // Get payment methods
    async getPaymentMethods(providerId) {
        const response = await fetch(`${this.apiBase}/providers/${providerId}/payment-methods`);
        if (!response.ok) throw new Error('Failed to fetch payment methods');
        return response.json();
    }

    // Get user orders
    async getUserOrders(address) {
        const response = await fetch(`${this.apiBase}/orders/${address}`);
        if (!response.ok) throw new Error('Failed to fetch orders');
        return response.json();
    }

    // Get KYC status
    async getKYCStatus(address) {
        const response = await fetch(`${this.apiBase}/kyc/${address}`);
        if (!response.ok) throw new Error('Failed to fetch KYC status');
        return response.json();
    }

    // Submit KYC
    async submitKYC(address, documents) {
        const response = await fetch(`${this.apiBase}/kyc`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ address, documents })
        });
        if (!response.ok) throw new Error('Failed to submit KYC');
        return response.json();
    }
}

window.TigerWalletFiatRampService = new FiatRampService();
