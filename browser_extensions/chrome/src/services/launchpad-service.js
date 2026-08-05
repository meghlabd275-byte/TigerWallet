/**
 * TigerWallet Browser Extension - Launchpad Service
 */

class LaunchpadService {
    constructor() {
        this.apiBase = 'https://api.tigerwallet.com/v1/launchpad';
    }

    // Get upcoming IDOs
    async getUpcoming() {
        const response = await fetch(`${this.apiBase}/upcoming`);
        if (!response.ok) throw new Error('Failed to fetch upcoming');
        return response.json();
    }

    // Get active IDOs
    async getActive() {
        const response = await fetch(`${this.apiBase}/active`);
        if (!response.ok) throw new Error('Failed to fetch active');
        return response.json();
    }

    // Get ended IDOs
    async getEnded() {
        const response = await fetch(`${this.apiBase}/ended`);
        if (!response.ok) throw new Error('Failed to fetch ended');
        return response.json();
    }

    // Get IDO details
    async getDetails(idoId) {
        const response = await fetch(`${this.apiBase}/ido/${idoId}`);
        if (!response.ok) throw new Error('Failed to fetch details');
        return response.json();
    }

    // Participate in IDO
    async participate(idoId, amount, address) {
        const response = await fetch(`${this.apiBase}/participate`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ idoId, amount, address })
        });
        if (!response.ok) throw new Error('Failed to participate');
        return response.json();
    }

    // Claim tokens
    async claim(idoId, address) {
        const response = await fetch(`${this.apiBase}/claim`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ idoId, address })
        });
        if (!response.ok) throw new Error('Failed to claim');
        return response.json();
    }

    // Get user allocations
    async getAllocations(address) {
        const response = await fetch(`${this.apiBase}/allocations/${address}`);
        if (!response.ok) throw new Error('Failed to fetch allocations');
        return response.json();
    }

    // Get user participations
    async getUserParticipations(address) {
        const response = await fetch(`${this.apiBase}/participations/${address}`);
        if (!response.ok) throw new Error('Failed to fetch participations');
        return response.json();
    }

    // Get token info
    async getTokenInfo(idoId) {
        const response = await fetch(`${this.apiBase}/ido/${idoId}/token`);
        if (!response.ok) throw new Error('Failed to fetch token');
        return response.json();
    }
}

window.TigerWalletLaunchpadService = new LaunchpadService();
