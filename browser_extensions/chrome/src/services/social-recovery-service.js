/**
 * TigerWallet Browser Extension - Social Recovery Service
 */

class SocialRecoveryService {
    constructor() {
        this.apiBase = 'http://localhost:8443/api/v1/social-recovery';
    }

    // Setup social recovery
    async setupRecovery(walletAddress, guardians) {
        const response = await fetch(`${this.apiBase}/setup`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ walletAddress, guardians })
        });
        if (!response.ok) throw new Error('Failed to setup recovery');
        return response.json();
    }

    // Add guardian
    async addGuardian(walletAddress, guardianAddress, guardianType) {
        const response = await fetch(`${this.apiBase}/guardian`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ walletAddress, guardianAddress, guardianType })
        });
        if (!response.ok) throw new Error('Failed to add guardian');
        return response.json();
    }

    // Remove guardian
    async removeGuardian(walletAddress, guardianAddress) {
        const response = await fetch(`${this.apiBase}/guardian`, {
            method: 'DELETE',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ walletAddress, guardianAddress })
        });
        if (!response.ok) throw new Error('Failed to remove guardian');
        return response.json();
    }

    // Get guardians
    async getGuardians(walletAddress) {
        const response = await fetch(`${this.apiBase}/guardians/${walletAddress}`);
        if (!response.ok) throw new Error('Failed to get guardians');
        return response.json();
    }

    // Initiate recovery
    async initiateRecovery(walletAddress, newOwnerAddress) {
        const response = await fetch(`${this.apiBase}/initiate`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ walletAddress, newOwnerAddress })
        });
        if (!response.ok) throw new Error('Failed to initiate recovery');
        return response.json();
    }

    // Confirm recovery (guardian)
    async confirmRecovery(recoveryId, guardianAddress) {
        const response = await fetch(`${this.apiBase}/confirm`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ recoveryId, guardianAddress })
        });
        if (!response.ok) throw new Error('Failed to confirm');
        return response.json();
    }

    // Execute recovery
    async executeRecovery(recoveryId, newOwnerAddress) {
        const response = await fetch(`${this.apiBase}/execute`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ recoveryId, newOwnerAddress })
        });
        if (!response.ok) throw new Error('Failed to execute recovery');
        return response.json();
    }

    // Cancel recovery
    async cancelRecovery(walletAddress) {
        const response = await fetch(`${this.apiBase}/cancel`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ walletAddress })
        });
        if (!response.ok) throw new Error('Failed to cancel');
        return response.json();
    }

    // Get pending recoveries
    async getPendingRecoveries(address) {
        const response = await fetch(`${this.apiBase}/pending/${address}`);
        if (!response.ok) throw new Error('Failed to get pending');
        return response.json();
    }

    // Change threshold
    async changeThreshold(walletAddress, newThreshold) {
        const response = await fetch(`${this.apiBase}/threshold`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ walletAddress, threshold: newThreshold })
        });
        if (!response.ok) throw new Error('Failed to change threshold');
        return response.json();
    }
}

window.TigerWalletSocialRecoveryService = new SocialRecoveryService();
