/**
 * Account Abstraction Service - Browser Extension
 * ERC-4337 Implementation
 */

const ENTRY_POINT_ADDRESS = '0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3';

class AccountAbstractionService {
  static instance = null;

  static getInstance() {
    if (!AccountAbstractionService.instance) {
      AccountAbstractionService.instance = new AccountAbstractionService();
    }
    return AccountAbstractionService.instance;
  }

  constructor() {
    this.smartAccount = null;
    this.sessionKeys = new Map();
    this.isInitialized = false;
  }

  initialize(ownerAddress) {
    this.smartAccount = {
      address: this.deriveSmartAccountAddress(ownerAddress),
      owner: ownerAddress,
      nonce: 0,
      isDeployed: false,
      entryPoint: ENTRY_POINT_ADDRESS,
    };
    this.isInitialized = true;
    return this.smartAccount;
  }

  getAccountAddress() {
    return this.smartAccount?.address ?? '';
  }

  async sendUserOp(to, value, data, paymaster = true) {
    const userOp = this.createUserOperation(to, value, data, paymaster);
    const hashResult = this.hashUserOperation(userOp);
    return `0x${hashResult}${Date.now()}`;
  }

  createSessionKey(config) {
    const key = {
      keyAddress: this.generateKeyAddress(),
      dAppAddress: config.dAppAddress,
      validUntil: config.validUntil,
      allowedContracts: config.allowedContracts,
      allowedSelectors: config.allowedSelectors,
      spendingLimit: config.spendingLimit,
      spentAmount: '0',
      isRevoked: false,
    };
    this.sessionKeys.set(key.keyAddress, key);
    return key;
  }

  revokeSessionKey(keyAddress) {
    const key = this.sessionKeys.get(keyAddress);
    if (key) {
      key.isRevoked = true;
      return true;
    }
    return false;
  }

  getActiveSessionKeys() {
    const now = Date.now();
    return Array.from(this.sessionKeys.values()).filter(
      (k) => !k.isRevoked && k.validUntil > now
    );
  }

  async executeWithSessionKey(keyAddress, to, data) {
    const key = this.sessionKeys.get(keyAddress);
    if (!key) throw new Error('Session key not found');
    if (key.isRevoked) throw new Error('Session key revoked');
    if (Date.now() > key.validUntil) throw new Error('Session key expired');

    const dataStr = Array.from(data).join('');
    return `0x${this.hash(`${to}${dataStr}`)}`;
  }

  // Private helpers
  deriveSmartAccountAddress(owner) {
    const hashResult = this.hash(`${owner}_smart_account`);
    return `0x${hashResult.substring(0, 40)}`;
  }

  generateKeyAddress() {
    const bytes = Array.from({ length: 32 }, () => Math.floor(Math.random() * 256));
    const hashResult = this.hash(bytes.join(''));
    return `0x${hashResult.substring(0, 40)}`;
  }

  createUserOperation(to, value, data, paymaster) {
    return {
      sender: this.smartAccount?.address ?? '',
      nonce: this.smartAccount?.nonce?.toString() ?? '0',
      initCode: this.smartAccount?.isDeployed ? '0x' : '0x',
      callData: this.encodeCallData(to, value, data),
      callGasLimit: '0x5208',
      verificationGasLimit: '0x186A0',
      preVerificationGas: '0x5208',
      maxFeePerGas: '0x3B9ACA00',
      maxPriorityFeePerGas: '0x3B9ACA00',
      paymasterAndData: paymaster ? '0xPaymasterAddress' : '0x',
      signature: '0x',
    };
  }

  encodeCallData(to, value, data) {
    const toClean = to.replace('0x', '');
    const dataStr = Array.from(data).map((b) => b.toString(16).padStart(2, '0')).join('');
    return `0x${toClean.padStart(64, '0')}${value.padStart(64, '0')}${data.length.toString(16).padStart(64, '0')}${dataStr}`;
  }

  hashUserOperation(userOp) {
    return this.hash(`${userOp.sender}${userOp.nonce}${userOp.callData}`);
  }

  hash(input) {
    let hash = 0;
    for (let i = 0; i < input.length; i++) {
      hash = ((hash << 5) - hash + input.charCodeAt(i)) | 0;
    }
    return Math.abs(hash).toString(16).padStart(64, '0');
  }
}

export default AccountAbstractionService.getInstance();
export { AccountAbstractionService };
