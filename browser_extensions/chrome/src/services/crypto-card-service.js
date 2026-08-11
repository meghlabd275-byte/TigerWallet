/**
 * TigerWallet Browser Extension - Crypto Card Service
 */

class CryptoCardService {
    constructor() {
        this.apiBase = 'http://localhost:8443/api/v1/card';
    }

    // Apply for card
    async apply(address, cardType = 'virtual') {
        const response = await fetch(`${this.apiBase}/apply`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ address, cardType })
        });
        if (!response.ok) throw new Error('Failed to apply');
        return response.json();
    }

    // Get card details
    async getCardDetails(cardId) {
        const response = await fetch(`${this.apiBase}/card/${cardId}`);
        if (!response.ok) throw new Error('Failed to get details');
        return response.json();
    }

    // Get balance
    async getBalance(cardId) {
        const response = await fetch(`${this.apiBase}/balance/${cardId}`);
        if (!response.ok) throw new Error('Failed to get balance');
        return response.json();
    }

    // Top up
    async topUp(cardId, amount, token = 'USDC') {
        const response = await fetch(`${this.apiBase}/topup`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ cardId, amount, token })
        });
        if (!response.ok) throw new Error('Failed to top up');
        return response.json();
    }

    // Withdraw
    async withdraw(cardId, amount) {
        const response = await fetch(`${this.apiBase}/withdraw`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ cardId, amount })
        });
        if (!response.ok) throw new Error('Failed to withdraw');
        return response.json();
    }

    // Freeze card
    async freeze(cardId) {
        const response = await fetch(`${this.apiBase}/freeze`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ cardId })
        });
        if (!response.ok) throw new Error('Failed to freeze');
        return response.json();
    }

    // Unfreeze card
    async unfreeze(cardId) {
        const response = await fetch(`${this.apiBase}/unfreeze`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ cardId })
        });
        if (!response.ok) throw new Error('Failed to unfreeze');
        return response.json();
    }

    // Get transaction history
    async getTransactions(cardId, limit = 50) {
        const response = await fetch(`${this.apiBase}/transactions/${cardId}?limit=${limit}`);
        if (!response.ok) throw new Error('Failed to get transactions');
        return response.json();
    }

    // Set spending limit
    async setLimit(cardId, dailyLimit, monthlyLimit) {
        const response = await fetch(`${this.apiBase}/limit`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ cardId, dailyLimit, monthlyLimit })
        });
        if (!response.ok) throw new Error('Failed to set limit');
        return response.json();
    }

    // Block card
    async block(cardId, reason) {
        const response = await fetch(`${this.apiBase}/block`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ cardId, reason })
        });
        if (!response.ok) throw new Error('Failed to block');
        return response.json();
    }

    // Get virtual card details
    async getVirtualCardDetails(cardId) {
        const response = await fetch(`${this.apiBase}/virtual/${cardId}`);
        if (!response.ok) throw new Error('Failed to get virtual card');
        return response.json();
    }

    // Get physical card status
    async getPhysicalCardStatus(cardId) {
        const response = await fetch(`${this.apiBase}/physical/${cardId}/status`);
        if (!response.ok) throw new Error('Failed to get status');
        return response.json();
    }
}

window.TigerWalletCryptoCardService = new CryptoCardService();
