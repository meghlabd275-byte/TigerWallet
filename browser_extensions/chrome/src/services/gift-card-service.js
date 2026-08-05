/**
 * TigerWallet Browser Extension - Gift Card Service
 */

class GiftCardService {
    constructor() {
        this.apiBase = 'https://api.tigerwallet.com/v1/giftcards';
    }

    // Get available brands
    async getBrands() {
        const response = await fetch(`${this.apiBase}/brands`);
        if (!response.ok) throw new Error('Failed to fetch brands');
        return response.json();
    }

    // Get brand details
    async getBrandDetails(brandId) {
        const response = await fetch(`${this.apiBase}/brand/${brandId}`);
        if (!response.ok) throw new Error('Failed to fetch brand');
        return response.json();
    }

    // Purchase gift card
    async purchase(brandId, amount, currency, address) {
        const response = await fetch(`${this.apiBase}/purchase`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ brandId, amount, currency, address })
        });
        if (!response.ok) throw new Error('Failed to purchase');
        return response.json();
    }

    // Check balance
    async checkBalance(code) {
        const response = await fetch(`${this.apiBase}/balance/${code}`);
        if (!response.ok) throw new Error('Failed to check balance');
        return response.json();
    }

    // Redeem gift card
    async redeem(code, address) {
        const response = await fetch(`${this.apiBase}/redeem`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ code, address })
        });
        if (!response.ok) throw new Error('Failed to redeem');
        return response.json();
    }

    // Get user's gift cards
    async getUserCards(address) {
        const response = await fetch(`${this.apiBase}/user/${address}`);
        if (!response.ok) throw new Error('Failed to fetch cards');
        return response.json();
    }

    // Create custom gift card
    async createCustom(amount, currency, address) {
        const response = await fetch(`${this.apiBase}/create`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ amount, currency, address })
        });
        if (!response.ok) throw new Error('Failed to create');
        return response.json();
    }
}

window.TigerWalletGiftCardService = new GiftCardService();
