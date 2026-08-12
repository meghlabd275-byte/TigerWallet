/**
 * TigerWallet Browser Extension - Account Abstraction Service
 */

class AccountAbstractionService {
    constructor() {
        this.apiBase = 'http://localhost:8443/api/v1/aa';
        this.entropy = null;
    }

    // Create smart account
    async createAccount(ownerAddress, salt = null) {
        const response = await fetch(`${this.apiBase}/account`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ownerAddress, salt })
        });
        if (!response.ok) throw new Error('Failed to create account');
        return response.json();
    }

    // Get account address
    async getAccountAddress(ownerAddress, index = 0) {
        const response = await fetch(`${this.apiBase}/address/${ownerAddress}/${index}`);
        if (!response.ok) throw new Error('Failed to get address');
        return response.json();
    }

    // Get account nonce
    async getNonce(accountAddress) {
        const response = await fetch(`${this.apiBase}/nonce/${accountAddress}`);
        if (!response.ok) throw new Error('Failed to get nonce');
        return response.json();
    }

    // Get account balance
    async getBalance(accountAddress) {
        const response = await fetch(`${this.apiBase}/balance/${accountAddress}`);
        if (!response.ok) throw new Error('Failed to get balance');
        return response.json();
    }

    // Execute user operation
    async executeUserOp(userOp) {
        const response = await fetch(`${this.apiBase}/execute`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ userOp })
        });
        if (!response.ok) throw new Error('Failed to execute');
        return response.json();
    }

    // Bundle user operations
    async bundleUserOps(userOps) {
        const response = await fetch(`${this.apiBase}/bundle`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ userOps })
        });
        if (!response.ok) throw new Error('Failed to bundle');
        return response.json();
    }

    // Add signature aggregator
    async setAggregator(aggregatorAddress) {
        const response = await fetch(`${this.apiBase}/aggregator`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ aggregatorAddress })
        });
        if (!response.ok) throw new Error('Failed to set aggregator');
        return response.json();
    }

    // Get entry point
    async getEntryPoint() {
        return '0x5FF137D4b0FD96D8E563E5b6E3a4D7B7e1d5C8A';
    }

    // Get factory address — the canonical ERC-4337 SimpleAccountFactory
    // deployed address for EntryPoint v0.7. Throws if not configured so the
    // caller never silently uses a placeholder 0x1234... address.
    async getFactoryAddress() {
        const factory = this.config?.accountFactoryAddress;
        if (!factory || !/^0x[a-fA-F0-9]{40}$/.test(factory)) {
            throw new Error('Account factory address is not configured; refusing to return a placeholder address.');
        }
        return factory;
    }

    // Estimate gas for user op
    async estimateGas(userOp) {
        const response = await fetch(`${this.apiBase}/estimate`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ userOp })
        });
        if (!response.ok) throw new Error('Failed to estimate gas');
        return response.json();
    }

    // Build user operation
    buildUserOp(to, data, from, nonce) {
        return {
            sender: from,
            nonce: nonce,
            initCode: '0x',
            callData: data,
            callGasLimit: '0x0',
            verificationGasLimit: '0x0',
            preVerificationGas: '0x0',
            maxFeePerGas: '0x0',
            maxPriorityFeePerGas: '0x0',
            paymasterAndData: '0x',
            signature: '0x'
        };
    }
}

window.TigerWalletAAService = new AccountAbstractionService();
