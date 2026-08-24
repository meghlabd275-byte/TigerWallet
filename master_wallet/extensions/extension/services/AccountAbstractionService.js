/**
 * AccountAbstractionService - ERC-4337 smart-wallet client for the extension.
 *
 * Heavy operations (deployment, bundler submission, signature creation) MUST
 * be performed by the canonical backend, which holds the keys and the deployed
 * factory/paymaster contracts. The client never fabricates an address, a
 * signature, or a hash. When an operation is not available it throws
 * (fail-closed) rather than returning placeholder data.
 *
 * Keccak-256 uses the vendored implementation in keccak256.js (Ethereum
 * padding 0x01), NOT the placeholder hash that used to live here.
 */

'use strict';

// UMD: CommonJS require under node/tests, globalThis under MV3 service worker.
const { keccak256 } = (typeof require === 'function')
  ? require('./keccak256.js')
  : ((globalThis.MW_KECCAK) || {});
const { authedFetch, getAuthContext } = (typeof require === 'function')
  ? require('./apiClient.js')
  : ((globalThis.MW_API) || {});

// Known mainnet EntryPoint address (ERC-4337 v0.6). Constant, not a stub.
const ENTRYPOINT_ADDRESS = '0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3';

class MasterAccountAbstractionService {
  constructor(masterWalletId) {
    this.masterWalletId = masterWalletId;
    this.isInitialized = false;
  }

  async initialize() {
    if (!this.masterWalletId) {
      throw new Error('masterWalletId is required');
    }
    const ctx = await getAuthContext();
    if (!ctx.token) {
      throw new Error('Not authenticated: cannot initialize AA service');
    }
    this.isInitialized = true;
    return true;
  }

  _requireInit() {
    if (!this.isInitialized) {
      throw new Error('AccountAbstractionService not initialized');
    }
  }

  _assertId() {
    if (!this.masterWalletId) throw new Error('masterWalletId is required');
  }

  // Real Ethereum Keccak-256. Accepts hex (0x-prefixed) or a UTF-8 string and
  // returns a 0x-prefixed 32-byte hex digest.
  keccak256(data) {
    if (typeof data !== 'string') {
      throw new Error('keccak256: expected hex or string input');
    }
    let bytes;
    if (data.startsWith('0x') || data.startsWith('0X')) {
      const hex = data.slice(2);
      if (!/^[0-9a-fA-F]*$/.test(hex) || hex.length % 2 !== 0) {
        throw new Error('keccak256: invalid hex input');
      }
      bytes = new Uint8Array(hex.length / 2);
      for (let i = 0; i < bytes.length; i++) {
        bytes[i] = parseInt(hex.substr(i * 2, 2), 16);
      }
    } else {
      bytes = new TextEncoder().encode(data);
    }
    return '0x' + keccak256(bytes);
  }

  get entryPoint() {
    return ENTRYPOINT_ADDRESS;
  }

  // ---- Smart wallet CRUD (backend-owned) ----

  async listSmartWallets() {
    this._assertId();
    const res = await authedFetch('/master-wallet/' + this.masterWalletId + '/multisig/wallets', { method: 'GET' });
    return res.wallets || res || [];
  }

  async createSmartWallet({ name, owners, threshold }) {
    this._assertId();
    if (!owners || !Array.isArray(owners) || owners.length === 0) {
      throw new Error('At least one owner address is required');
    }
    if (!threshold || threshold < 1) {
      throw new Error('A valid threshold is required');
    }
    return authedFetch('/master-wallet/' + this.masterWalletId + '/multisig/wallets', {
      method: 'POST',
      body: { name, owners, threshold },
    });
  }

  async getSmartWalletTransactions(walletId) {
    this._assertId();
    const res = await authedFetch(
      '/master-wallet/' + this.masterWalletId + '/multisig/wallets/' + walletId + '/transactions',
      { method: 'GET' }
    );
    return res.transactions || res || [];
  }

  async submitMultisigTransaction(walletId, body) {
    this._assertId();
    return authedFetch(
      '/master-wallet/' + this.masterWalletId + '/multisig/wallets/' + walletId + '/transactions',
      { method: 'POST', body }
    );
  }

  async signMultisigTransaction(transactionId) {
    this._assertId();
    return authedFetch(
      '/master-wallet/' + this.masterWalletId + '/multisig/transactions/' + transactionId + '/sign',
      { method: 'POST' }
    );
  }

  async executeMultisigTransaction(transactionId) {
    this._assertId();
    return authedFetch(
      '/master-wallet/' + this.masterWalletId + '/multisig/transactions/' + transactionId + '/execute',
      { method: 'POST' }
    );
  }

  // ---- Pure client-side crypto helpers (no key material) ----

  /**
   * Compute the ERC-4337 userOpHash (packed encoding) for a user operation.
   * This is a deterministic hash of public fields only; it is safe to compute
   * client-side. Signing must be done by the backend (signMultisigTransaction).
   */
  computeUserOpHash(userOp) {
    if (!userOp) throw new Error('userOp required');
    const packed = [
      userOp.sender || '0x',
      userOp.nonce || '0x0',
      this.keccak256(userOp.initCode || '0x'),
      this.keccak256(userOp.callData || '0x'),
      userOp.callGasLimit || '0x0',
      userOp.verificationGasLimit || '0x0',
      userOp.preVerificationGas || '0x0',
      userOp.maxFeePerGas || '0x0',
      userOp.maxPriorityFeePerGas || '0x0',
      this.keccak256(userOp.paymasterAndData || '0x'),
    ].join('');
    return this.keccak256(packed);
  }

  /**
   * Compute a CREATE2 counterfactual address from the factory, salt and
   * init code hash. Pure public-data crypto; no deployment or signing here.
   * address = keccak256(0xff ++ factory ++ salt ++ initCodeHash)[12:]
   */
  computeCreate2Address(factoryAddress, salt, initCodeHash) {
    if (!factoryAddress || !salt || !initCodeHash) {
      throw new Error('factoryAddress, salt and initCodeHash are required');
    }
    const ff = 'ff';
    const factory = factoryAddress.toLowerCase().replace(/^0x/, '').padStart(40, '0');
    const s = salt.toLowerCase().replace(/^0x/, '').padStart(64, '0');
    const ich = initCodeHash.toLowerCase().replace(/^0x/, '').padStart(64, '0');
    const digest = this.keccak256('0x' + ff + factory + s + ich);
    return '0x' + digest.slice(2 + 24); // last 20 bytes
  }

  randomBytes(length) {
    const array = new Uint8Array(length);
    crypto.getRandomValues(array);
    return Array.from(array).map((b) => b.toString(16).padStart(2, '0')).join('');
  }
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = { MasterAccountAbstractionService, ENTRYPOINT_ADDRESS };
}
if (typeof globalThis !== 'undefined') {
  globalThis.MW_AA = { MasterAccountAbstractionService, ENTRYPOINT_ADDRESS };
}
