/**
 * Paymaster Service - Browser Extension
 */

class PaymasterService {
  static instance = null;

  static getInstance() {
    if (!PaymasterService.instance) {
      PaymasterService.instance = new PaymasterService();
    }
    return PaymasterService.instance;
  }

  constructor() {
    this.whitelistedDApps = new Map();
    this.gasToken = null;
  }

  async sponsorUserOp(userOp) {
    return {
      paymasterAndData: this.buildPaymasterData(userOp),
      preVerificationGas: '0x5208',
      verificationGasLimit: '0x186A0',
      callGasLimit: '0x5208',
    };
  }

  setPaymentToken(tokenAddress) {
    this.gasToken = tokenAddress;
    return true;
  }

  getPaymentToken() {
    return this.gasToken;
  }

  whitelistDApp(dAppAddress, limit, expiry) {
    this.whitelistedDApps.set(dAppAddress, {
      address: dAppAddress,
      sponsorLimit: limit,
      expiry,
      isActive: true,
    });
    return true;
  }

  getWhitelistStatus(address) {
    const entry = this.whitelistedDApps.get(address);
    if (!entry) return null;
    return {
      isWhitelisted: entry.isActive,
      limit: entry.sponsorLimit,
      expiry: entry.expiry,
      used: '0',
    };
  }

  getBalance() {
    return '1000000000000000000';
  }

  buildPaymasterData(userOp) {
    const hashResult = this.hash(`${userOp.sender}${userOp.nonce}${this.gasToken ?? ''}`);
    return `0xPaymasterAddress${'0'.repeat(64)}${hashResult.substring(0, 32)}`;
  }

  hash(input) {
    let hash = 0;
    for (let i = 0; i < input.length; i++) {
      hash = ((hash << 5) - hash + input.charCodeAt(i)) | 0;
    }
    return Math.abs(hash).toString(16).padStart(64, '0');
  }
}

export default PaymasterService.getInstance();
export { PaymasterService };
