/**
 * TigerWallet Browser Extension - MPC Wallet Service
 *
 * Thin client over the canonical TSS service (go/mpc). The service performs the
 * threshold cryptography; this file only speaks its HTTP contract:
 *
 *   POST /api/v1/mpc/create          {threshold,totalShards}  -> wallet
 *   POST /api/v1/mpc/sign            {keyId,messageHash}      -> signature
 *   GET  /api/v1/mpc/wallet/<keyId>                           -> wallet
 *   POST /api/v1/mpc/keygen/export   {keyId,passphrase}       -> encrypted shares
 *   POST /api/v1/mpc/keygen/recover  {passphrase,shares}      -> wallet
 *   GET  /api/v1/health                                       -> liveness
 *
 * Every route except health requires a bearer token; the service binds a wallet
 * to the subject that created it and rejects other subjects.
 */

class MPCWalletService {
    constructor(apiBase) {
        this.apiBase = (apiBase || 'http://localhost:9099') + '/api/v1/mpc';
    }

    async authHeaders(extra = {}) {
        const headers = { 'Content-Type': 'application/json', ...extra };
        const stored = await chrome.storage.local.get(['tigerwallet_jwt']);
        if (stored.tigerwallet_jwt) {
            headers['Authorization'] = `Bearer ${stored.tigerwallet_jwt}`;
        }
        return headers;
    }

    async request(path, { method = 'GET', body } = {}) {
        const response = await fetch(`${this.apiBase}${path}`, {
            method,
            headers: await this.authHeaders(),
            body: body === undefined ? undefined : JSON.stringify(body)
        });
        const payload = await response.json().catch(() => ({}));
        if (!response.ok) {
            throw new Error(payload.error || `MPC request ${path} failed (${response.status})`);
        }
        return payload;
    }

    /** Create a threshold wallet. Returns {keyId,address,publicKey,threshold,totalShards}. */
    async createWallet(threshold = 2, totalShards = 3) {
        return this.request('/create', { method: 'POST', body: { threshold, totalShards } });
    }

    /** Sign a 32-byte digest. `messageHash` is hex, with or without 0x. */
    async sign(keyId, messageHash) {
        return this.request('/sign', { method: 'POST', body: { keyId, messageHash } });
    }

    /** Fetch wallet metadata for a key id the caller owns. */
    async getWallet(keyId) {
        return this.request(`/wallet/${encodeURIComponent(keyId)}`);
    }

    /** Export the encrypted share blobs for offline backup. */
    async exportShares(keyId, passphrase) {
        return this.request('/keygen/export', {
            method: 'POST',
            body: { keyId, passphrase }
        });
    }

    /** Restore a wallet from a threshold of encrypted share blobs (operator only). */
    async recoverWallet(passphrase, shares) {
        return this.request('/keygen/recover', { method: 'POST', body: { passphrase, shares } });
    }

    async health() {
        const response = await fetch(`${this.apiBase.replace('/mpc', '')}/health`);
        return response.ok;
    }
}
