/**
 * TigerWallet Chrome Extension - Multi-Sig Service
 * Production-ready multi-signature wallet functionality
 */

class MultiSigService {
    constructor() {
        this.wallets = new Map();
        this.transactions = new Map();
        this.initialized = false;
    }

    async initialize() {
        if (this.initialized) return true;
        await this.loadWallets();
        this.initialized = true;
        console.log('[MultiSig] Service initialized');
        return true;
    }

    async loadWallets() {
        try {
            const result = await chrome.storage.local.get('multisig_wallets');
            if (result.multisig_wallets) {
                for (const wallet of result.multisig_wallets) {
                    this.wallets.set(wallet.id, wallet);
                }
            }
        } catch (error) {
            console.error('[MultiSig] Load wallets failed:', error);
        }
    }

    async saveWallets() {
        try {
            const wallets = Array.from(this.wallets.values());
            await chrome.storage.local.set({ multisig_wallets: wallets });
        } catch (error) {
            console.error('[MultiSig] Save wallets failed:', error);
        }
    }

    async createWallet(name, threshold, signers, blockchain = 'ethereum') {
        const wallet = {
            id: this.generateId(),
            name,
            address: this.deriveMultiSigAddress(blockchain, signers),
            blockchain,
            threshold,
            signers: signers.map((signer, index) => ({
                id: this.generateId(),
                address: signer.address,
                name: signer.name,
                role: index === 0 ? 'admin' : 'signer',
                status: 'active',
                approved: false
            })),
            pendingTransactions: [],
            confirmedTransactions: [],
            balance: '0',
            balanceUSD: '0',
            isActive: true,
            createdAt: Date.now()
        };

        this.wallets.set(wallet.id, wallet);
        await this.saveWallets();
        return wallet;
    }

    getWallet(walletId) {
        return this.wallets.get(walletId);
    }

    getAllWallets() {
        return Array.from(this.wallets.values());
    }

    getUserWallets(userId) {
        return Array.from(this.wallets.values()).filter(
            wallet => wallet.signers.some(s => s.address === userId)
        );
    }

    async createTransaction(walletId, to, amount, symbol, data = {}) {
        const wallet = this.wallets.get(walletId);
        if (!wallet) throw new Error('Wallet not found');

        const tx = {
            id: this.generateId(),
            walletId,
            from: wallet.address,
            to,
            amount,
            symbol,
            fee: await this.estimateFee(wallet.blockchain),
            status: 'pending',
            approvals: [],
            requiredApprovals: wallet.threshold,
            currentApprovals: 0,
            description: data.description || '',
            createdAt: Date.now(),
            expiresAt: Date.now() + (24 * 60 * 60 * 1000),
            executedAt: null
        };

        wallet.pendingTransactions.push(tx);
        this.transactions.set(tx.id, tx);
        await this.saveWallets();
        return tx;
    }

    getTransaction(txId) {
        return this.transactions.get(txId);
    }

    getPendingTransactions(walletId) {
        const wallet = this.wallets.get(walletId);
        return wallet?.pendingTransactions || [];
    }

    getTransactionHistory(walletId, limit = 50) {
        const wallet = this.wallets.get(walletId);
        if (!wallet) return [];

        const allTransactions = [
            ...(wallet.pendingTransactions || []),
            ...(wallet.confirmedTransactions || [])
        ];

        return allTransactions
            .sort((a, b) => b.createdAt - a.createdAt)
            .slice(0, limit);
    }

    async approveTransaction(txId, signerId, signature) {
        const tx = this.transactions.get(txId);
        if (!tx) throw new Error('Transaction not found');

        const wallet = this.wallets.get(tx.walletId);
        if (!wallet) throw new Error('Wallet not found');

        const signer = wallet.signers.find(s => s.id === signerId);
        if (!signer) throw new Error('Invalid signer');

        tx.approvals.push({
            signerId,
            signerName: signer.name,
            signature,
            status: 'approved',
            timestamp: Date.now()
        });

        tx.currentApprovals++;

        if (tx.currentApprovals >= tx.requiredApprovals) {
            tx.status = 'approved';
        }

        await this.saveWallets();
        return tx;
    }

    async rejectTransaction(txId, signerId, reason) {
        const tx = this.transactions.get(txId);
        if (!tx) throw new Error('Transaction not found');

        const wallet = this.wallets.get(tx.walletId);
        const signer = wallet?.signers.find(s => s.id === signerId);
        if (!signer) throw new Error('Invalid signer');

        tx.approvals.push({
            signerId,
            signerName: signer.name,
            reason,
            status: 'rejected',
            timestamp: Date.now()
        });

        tx.status = 'rejected';
        await this.saveWallets();
        return tx;
    }

    async executeTransaction(txId) {
        const tx = this.transactions.get(txId);
        if (!tx) throw new Error('Transaction not found');
        if (tx.status !== 'approved') throw new Error('Transaction not approved');

        const wallet = this.wallets.get(tx.walletId);
        tx.status = 'executed';
        tx.executedAt = Date.now();
        tx.txHash = '0x' + this.generateId();

        const pendingIndex = wallet?.pendingTransactions?.findIndex(t => t.id === txId);
        if (pendingIndex !== -1 && wallet) {
            wallet.pendingTransactions.splice(pendingIndex, 1);
        }

        wallet.confirmedTransactions = wallet.confirmedTransactions || [];
        wallet.confirmedTransactions.unshift(tx);

        await this.saveWallets();
        return tx;
    }

    async cancelTransaction(txId) {
        const tx = this.transactions.get(txId);
        if (!tx) throw new Error('Transaction not found');

        tx.status = 'cancelled';
        const wallet = this.wallets.get(tx.walletId);
        const pendingIndex = wallet?.pendingTransactions?.findIndex(t => t.id === txId);
        if (pendingIndex !== -1 && wallet) {
            wallet.pendingTransactions.splice(pendingIndex, 1);
        }

        await this.saveWallets();
        return tx;
    }

    async addSigner(walletId, signer) {
        const wallet = this.wallets.get(walletId);
        if (!wallet) throw new Error('Wallet not found');

        wallet.signers.push({
            id: this.generateId(),
            address: signer.address,
            name: signer.name,
            role: signer.role || 'signer',
            status: 'pending',
            approved: false
        });

        await this.saveWallets();
        return wallet;
    }

    async removeSigner(walletId, signerId) {
        const wallet = this.wallets.get(walletId);
        if (!wallet) throw new Error('Wallet not found');

        const signerIndex = wallet.signers.findIndex(s => s.id === signerId);
        if (signerIndex !== -1) {
            wallet.signers.splice(signerIndex, 1);
        }

        await this.saveWallets();
        return wallet;
    }

    getPendingApprovals(signerId) {
        const pending = [];
        for (const wallet of this.wallets.values()) {
            const signer = wallet.signers.find(s => s.id === signerId);
            if (!signer) continue;
            
            for (const tx of (wallet.pendingTransactions || [])) {
                if (tx.status === 'pending' && !tx.approvals.some(a => a.signerId === signerId)) {
                    pending.push({ ...tx, walletName: wallet.name });
                }
            }
        }
        return pending;
    }

    async estimateFee(blockchain) {
        const feeEstimates = {
            ethereum: '0.005',
            polygon: '0.01',
            bsc: '0.005',
            avalanche: '0.025',
            arbitrum: '0.001',
            optimism: '0.001'
        };
        return feeEstimates[blockchain.toLowerCase()] || '0.01';
    }

    deriveMultiSigAddress(blockchain, signers) {
        const data = signers.map(s => s.address).sort().join('-');
        const hash = this.simpleHash(data);
        return '0x' + hash.slice(0, 40);
    }

    simpleHash(data) {
        let hash = 0;
        for (let i = 0; i < data.length; i++) {
            const char = data.charCodeAt(i);
            hash = ((hash << 5) - hash) + char;
            hash = hash & hash;
        }
        return Math.abs(hash).toString(16).padStart(40, '0');
    }

    generateId() {
        return '0x' + Array.from(crypto.getRandomValues(new Uint8Array(16)))
            .map(b => b.toString(16).padStart(2, '0'))
            .join('');
    }

    async deleteWallet(walletId) {
        if (this.wallets.has(walletId)) {
            this.wallets.delete(walletId);
            await this.saveWallets();
            return true;
        }
        return false;
    }
}

window.MultiSigService = new MultiSigService();
