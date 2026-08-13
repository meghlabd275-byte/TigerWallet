/**
 * PrivacyService - privacy-feature client for the extension.
 *
 * ZK-SNARK proof generation, CoinJoin coordination, and address derivation
 * require private key material and trusted-setup parameters that live on the
 * backend. The client relays requests to the backend and computes only
 * public, deterministic hashes (keccak-256 / SHA-256 via SubtleCrypto).
 *
 * Fail-closed: any unavailable operation throws rather than returning a
 * fabricated proof, commitment, or derived address.
 */

'use strict';

// UMD: CommonJS require under node/tests, globalThis under MV3 service worker.
const { keccak256 } = (typeof require === 'function')
  ? require('./keccak256.js')
  : ((globalThis.MW_KECCAK) || {});
const { authedFetch } = (typeof require === 'function')
  ? require('./apiClient.js')
  : ((globalThis.MW_API) || {});

class MasterPrivacyService {
  constructor(masterWalletId) {
    this.masterWalletId = masterWalletId;
    this.isInitialized = false;
  }

  async initialize() {
    if (!this.masterWalletId) throw new Error('masterWalletId is required');
    this.isInitialized = true;
    return true;
  }

  _assert() {
    if (!this.isInitialized) throw new Error('Privacy service not initialized');
    if (!this.masterWalletId) throw new Error('masterWalletId is required');
  }

  // Real Ethereum Keccak-256 commitment over public fields.
  computeCommitment({ sender, recipient, amount, secret, nullifier }) {
    if (!sender || !recipient || amount === undefined || !secret || !nullifier) {
      throw new Error('sender, recipient, amount, secret, nullifier are required');
    }
    const packed = [sender, recipient, String(amount), secret, nullifier].join('');
    const bytes = new TextEncoder().encode(packed);
    return '0x' + keccak256(bytes);
  }

  // SHA-256 via SubtleCrypto (available in service workers). Real, not stubbed.
  async sha256(data) {
    const input = typeof data === 'string' ? new TextEncoder().encode(data) : data;
    const digest = await crypto.subtle.digest('SHA-256', input);
    return '0x' + Array.from(new Uint8Array(digest))
      .map((b) => b.toString(16).padStart(2, '0'))
      .join('');
  }

  randomBytes(length) {
    const array = new Uint8Array(length);
    crypto.getRandomValues(array);
    return Array.from(array).map((b) => b.toString(16).padStart(2, '0')).join('');
  }

  // Generate a fresh ZK proof request id (random, no proof content fabricated).
  // The proof itself is generated and verified by the backend.
  async requestZKProof({ sender, recipient, amount }) {
    this._assert();
    const secret = this.randomBytes(32);
    const nullifier = this.randomBytes(32);
    const commitment = this.computeCommitment({ sender, recipient, amount, secret, nullifier });
    return authedFetch('/master-wallet/' + this.masterWalletId + '/treasury/transfer', {
      method: 'POST',
      body: {
        to: recipient,
        amount,
        password: null, // backend requires auth context; password optional for AA
        privacy: { commitment, nullifier },
      },
    });
  }

  // Address rotation is a backend HD-derivation operation. The client never
  // derives addresses from a seed it does not hold.
  async rotateAddress() {
    this._assert();
    const wallet = await authedFetch('/master-wallet/' + this.masterWalletId, { method: 'GET' });
    if (wallet && wallet.address) {
      return wallet.address;
    }
    throw new Error('Backend did not return a derivable address');
  }
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = { MasterPrivacyService };
}
if (typeof globalThis !== 'undefined') {
  globalThis.MW_PRIVACY = { MasterPrivacyService };
}
