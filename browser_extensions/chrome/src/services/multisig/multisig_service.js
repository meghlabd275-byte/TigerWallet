/**
 * TigerWallet Chrome Extension - Multi-Sig Service
 *
 * SEPARATION: the extension talks ONLY to the UserWallet backend
 * (go/wallet_api, http://localhost:8443). The wallet_api multisig proxy
 * delegates threshold signature assembly and on-chain broadcast to the
 * MasterWallet backend server-side — the client NEVER calls the
 * MasterWallet backend (:8450) directly. Honest results only: never
 * fabricates a wallet address, tx id, or transaction hash.
 */

const MULTISIG_API_BASE = 'http://localhost:8443/api/v1/wallet/multisig';

class MultiSigService {
    constructor() {
        this.initialized = false;
    }

    async initialize() {
        if (this.initialized) return true;
        this.initialized = true;
        console.log('[MultiSig] Service initialized (backend:', MULTISIG_API_BASE + ')');
        return true;
    }

    async _fetch(path, opts = {}) {
        const resp = await fetch(`${MULTISIG_API_BASE}${path}`, {
            headers: { 'Content-Type': 'application/json' },
            ...opts
        });
        if (!resp.ok) {
            throw new Error(`multisig backend ${opts.method || 'GET'} ${path} -> ${resp.status}`);
        }
        return resp.json();
    }

    async createWallet(name, threshold, signers, blockchain = 'ethereum') {
        const owners = signers.map(s => s.address);
        const data = await this._fetch('/api/v1/multisig/wallets', {
            method: 'POST',
            body: JSON.stringify({ name, threshold, owners, blockchain })
        });
        return {
            id: data.id || data.wallet_id || '',
            name,
            address: data.address || '',
            blockchain,
            threshold,
            signers: owners.map((addr, i) => ({
                id: addr,
                address: addr,
                name: signers[i].name || '',
                role: i === 0 ? 'admin' : 'signer',
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
    }

    async getWallet(walletId) {
        try {
            return await this._fetch('/api/v1/multisig/wallets/' + encodeURIComponent(walletId));
        } catch (e) {
            return null;
        }
    }

    async getAllWallets() {
        try {
            return await this._fetch('/api/v1/multisig/wallets');
        } catch (e) {
            return [];
        }
    }

    getUserWallets(userId) {
        // Defer to getAllWallets() and filter on the caller's side; the backend
        // returns all wallets accessible to the authenticated user.
        return this.getAllWallets();
    }

    async createTransaction(walletId, to, amount, symbol, data = {}) {
        const resp = await this._fetch('/api/v1/multisig/transactions', {
            method: 'POST',
            body: JSON.stringify({
                wallet_id: walletId,
                to,
                value: String(amount),
                data: typeof data === 'string' ? data : JSON.stringify(data)
            })
        });
        return {
            id: resp.id || resp.tx_id || '',
            walletId,
            to,
            amount,
            symbol,
            data,
            status: 'pending',
            confirmations: 0,
            threshold: resp.threshold || 1,
            txHash: '' // populated after executeTransaction
        };
    }

    getTransaction(txId) {
        return this._fetch('/api/v1/multisig/transactions/' + encodeURIComponent(txId));
    }

    async approveTransaction(txId, signerId, signature) {
        await this._fetch('/api/v1/multisig/transactions/' + encodeURIComponent(txId) + '/sign', {
            method: 'POST',
            body: JSON.stringify({ signer: signerId, signature })
        });
        return true;
    }

    async rejectTransaction(txId, signerId, reason) {
        await this._fetch('/api/v1/multisig/transactions/' + encodeURIComponent(txId) + '/revoke', {
            method: 'POST',
            body: JSON.stringify({ signer: signerId, reason })
        });
        return true;
    }

    async executeTransaction(txId) {
        // The backend collects the threshold owner signatures off-chain,
        // assembles the on-chain MultisigWallet.executeTransaction call,
        // signs + broadcasts it via eth_sendRawTransaction, and returns the
        // REAL transaction hash. Empty string on failure — never fabricated.
        try {
            const resp = await this._fetch('/api/v1/multisig/transactions/' + encodeURIComponent(txId) + '/execute', {
                method: 'POST',
                body: JSON.stringify({})
            });
            return { txHash: resp.tx_hash || resp.txHash || '', executed: resp.executed !== false };
        } catch (e) {
            console.error('[MultiSig] executeTransaction failed:', e);
            return { txHash: '', executed: false };
        }
    }

    async cancelTransaction(txId) {
        await this._fetch('/api/v1/multisig/transactions/' + encodeURIComponent(txId) + '/revoke', {
            method: 'POST',
            body: JSON.stringify({})
        });
        return true;
    }

    async addSigner(walletId, signer) {
        await this._fetch('/api/v1/multisig/wallets/' + encodeURIComponent(walletId) + '/owners', {
            method: 'POST',
            body: JSON.stringify({ owner: signer.address })
        });
        return true;
    }

    async removeSigner(walletId, signerId) {
        // Backend owner-removal is routed through the wallet's own execute path
        // (on-chain governance). We surface a clear error rather than silently
        // mutating local state.
        throw new Error('Owner removal must be submitted as a multisig transaction on-chain');
    }

    async estimateFee(blockchain) {
        // No fabricated fee. Returns null; the real gas estimate comes from
        // the wallet_api /api/v1/gas endpoint.
        return null;
    }

    async deleteWallet(walletId) {
        throw new Error('Multisig wallets are on-chain contracts and cannot be deleted from the extension');
    }
}

window.MultiSigService = new MultiSigService();
