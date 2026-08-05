/**
 * TigerWallet Browser Extension - MPC Wallet Service
 */

class MPCWalletService {
    constructor() {
        this.apiBase = 'https://api.tigerwallet.com/v1/mpc';
        this.sessionKey = null;
    }

    // Create MPC wallet
    async createWallet(userId) {
        const response = await fetch(`${this.apiBase}/wallet`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ userId })
        });
        if (!response.ok) throw new Error('Failed to create wallet');
        return response.json();
    }

    // Import wallet
    async importWallet(mnemonic, userId) {
        const response = await fetch(`${this.apiBase}/wallet/import`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ mnemonic, userId })
        });
        if (!response.ok) throw new Error('Failed to import wallet');
        return response.json();
    }

    // Get public key
    async getPublicKey(address) {
        const response = await fetch(`${this.apiBase}/publickey/${address}`);
        if (!response.ok) throw new Error('Failed to get public key');
        return response.json();
    }

    // Sign transaction (MPC)
    async signTransaction(txData, address) {
        const response = await fetch(`${this.apiBase}/sign`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ txData, address })
        });
        if (!response.ok) throw new Error('Failed to sign');
        return response.json();
    }

    // Sign message
    async signMessage(message, address) {
        const response = await fetch(`${this.apiBase}/sign-message`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ message, address })
        });
        if (!response.ok) throw new Error('Failed to sign message');
        return response.json();
    }

    // Get session key
    async getSessionKey(address) {
        const response = await fetch(`${this.apiBase}/session/${address}`);
        if (!response.ok) throw new Error('Failed to get session');
        return response.json();
    }

    // Rotate key share
    async rotateKey(address) {
        const response = await fetch(`${this.apiBase}/rotate`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ address })
        });
        if (!response.ok) throw new Error('Failed to rotate key');
        return response.json();
    }

    // Get wallet info
    async getWalletInfo(address) {
        const response = await fetch(`${this.apiBase}/wallet/${address}`);
        if (!response.ok) throw new Error('Failed to get wallet info');
        return response.json();
    }

    // Export key share (requires auth)
    async exportKeyShare(address, authToken) {
        const response = await fetch(`${this.apiBase}/export`, {
            method: 'POST',
            headers: { 
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${authToken}`
            },
            body: JSON.stringify({ address })
        });
        if (!response.ok) throw new Error('Failed to export key');
        return response.json();
    }
}

window.TigerWalletMPCService = new MPCWalletService();
