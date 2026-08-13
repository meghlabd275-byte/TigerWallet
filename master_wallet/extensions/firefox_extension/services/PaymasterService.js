/**
 * PaymasterService - ERC-4337 paymaster client for the extension.
 *
 * Gas sponsorship decisions, paymaster signing and balance accounting are
 * enforced by the backend (which owns the paymaster key and contract). This
 * client relays sponsorship requests and reads gas prices from the canonical
 * /gas route. It never fabricates a paymaster signature, a balance, or gas
 * prices. Unavailable operations throw (fail-closed).
 */

'use strict';

// UMD: CommonJS require under node/tests, globalThis under MV3 service worker.
const { keccak256 } = (typeof require === 'function')
  ? require('./keccak256.js')
  : ((globalThis.MW_KECCAK) || {});
const { authedFetch } = (typeof require === 'function')
  ? require('./apiClient.js')
  : ((globalThis.MW_API) || {});

class MasterPaymasterService {
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
    if (!this.isInitialized) throw new Error('Paymaster service not initialized');
    if (!this.masterWalletId) throw new Error('masterWalletId is required');
  }

  // Read real gas prices from the canonical /gas route.
  async fetchGasPrice(chainId) {
    if (!chainId) throw new Error('chainId is required');
    return authedFetch('/gas', { method: 'GET', auth: false, query: { chain_id: chainId } });
  }

  // Sponsorship eligibility is decided by the backend. The client never
  // fabricates a "canSponsor: true".
  async canSponsor({ sender, chainId, token, value }) {
    this._assert();
    const body = { sender, chain_id: chainId };
    if (token !== undefined) body.token = token;
    if (value !== undefined) body.value = value;
    return authedFetch('/master-wallet/' + this.masterWalletId + '/fees', {
      method: 'POST',
      body,
    });
  }

  // Request a paymaster signature for a userOp. Backend holds the key and
  // returns the signed paymasterAndData; this client does not sign.
  async requestPaymasterData(userOp, chainId = '1') {
    this._assert();
    return authedFetch('/master-wallet/' + this.masterWalletId + '/fees', {
      method: 'POST',
      body: { user_op: userOp, chain_id: chainId },
    });
  }

  // Real paymaster balance is read from the backend treasury route.
  async getPaymasterBalance(chainId = '1') {
    this._assert();
    const res = await authedFetch('/master-wallet/' + this.masterWalletId + '/treasury', {
      method: 'GET',
      query: { chain_id: chainId },
    });
    if (res && res.paymaster_balance !== undefined) return res.paymaster_balance;
    if (res && res.balance !== undefined) return res.balance;
    throw new Error('Paymaster balance not available from backend');
  }

  // Compute the deterministic paymaster-data hash (public fields only).
  hashPaymasterData(userOp, chainId, validUntil) {
    const enc = (v) => {
      const s = String(v);
      return s.startsWith('0x') ? s : '0x' + s;
    };
    const packed = [
      enc(validUntil || 0),
      enc(chainId),
      this.keccak256ish(userOp.sender || '0x'),
      enc(userOp.nonce || '0x0'),
      this.keccak256ish(userOp.callData || '0x'),
    ].join('');
    const bytes = new TextEncoder().encode(packed);
    return '0x' + keccak256(bytes);
  }

  // Wrapper returning 0x-prefixed digest for convenience.
  keccak256ish(data) {
    if (typeof data !== 'string') throw new Error('keccak256: expected hex or string input');
    let bytes;
    if (data.startsWith('0x') || data.startsWith('0X')) {
      const hex = data.slice(2);
      if (!/^[0-9a-fA-F]*$/.test(hex) || hex.length % 2 !== 0) throw new Error('keccak256: invalid hex input');
      bytes = new Uint8Array(hex.length / 2);
      for (let i = 0; i < bytes.length; i++) bytes[i] = parseInt(hex.substr(i * 2, 2), 16);
    } else {
      bytes = new TextEncoder().encode(data);
    }
    return '0x' + keccak256(bytes);
  }

  // Real fee calculation: gas limit * real gas price read from /gas.
  async calculateFee(userOp, chainId = '1') {
    this._assert();
    const gas = await this.fetchGasPrice(chainId);
    const gasLimit =
      (parseInt(userOp.callGasLimit || '0', 10) || 0) +
      (parseInt(userOp.verificationGasLimit || '0', 10) || 0) +
      (parseInt(userOp.preVerificationGas || '0', 10) || 0);
    const price = parseInt(gas.max_fee || gas.maxFeePerGas || gas.gas_price || '0', 10) || 0;
    const baseFee = (gasLimit * price).toString();
    return { baseFee, gasPrice: price, gasLimit };
  }
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = { MasterPaymasterService };
}
if (typeof globalThis !== 'undefined') {
  globalThis.MW_PAYMASTER = { MasterPaymasterService };
}
