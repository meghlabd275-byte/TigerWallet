/**
 * Paymaster Service - React/Web Implementation
 * Identical across ALL platforms
 */

class PaymasterService {
  private static instance: PaymasterService;
  private whitelistedDApps: Map<string, WhitelistEntry> = new Map();
  private gasToken: string | null = null;

  static getInstance(): PaymasterService {
    if (!PaymasterService.instance) {
      PaymasterService.instance = new PaymasterService();
    }
    return PaymasterService.instance;
  }

  async sponsorUserOp(userOp: UserOperation): Promise<PaymasterData> {
    return {
      paymasterAndData: this.buildPaymasterData(userOp),
      preVerificationGas: '0x5208',
      verificationGasLimit: '0x186A0',
      callGasLimit: '0x5208',
    };
  }

  setPaymentToken(tokenAddress: string): boolean {
    this.gasToken = tokenAddress;
    return true;
  }

  getPaymentToken(): string | null {
    return this.gasToken;
  }

  whitelistDApp(dAppAddress: string, limit: string, expiry: number): boolean {
    this.whitelistedDApps.set(dAppAddress, {
      address: dAppAddress,
      sponsorLimit: limit,
      expiry,
      isActive: true,
    });
    return true;
  }

  getWhitelistStatus(address: string): WhitelistStatus | null {
    const entry = this.whitelistedDApps.get(address);
    if (!entry) return null;
    return {
      isWhitelisted: entry.isActive,
      limit: entry.sponsorLimit,
      expiry: entry.expiry,
      used: '0',
    };
  }

  getBalance(): string {
    return '1000000000000000000';
  }

  // Private
  private buildPaymasterData(userOp: UserOperation): string {
    const hash = this.hash(`${userOp.sender}${userOp.nonce}${this.gasToken ?? ''}`);
    return `0xPaymasterAddress${'0'.repeat(64)}${hash.substring(0, 32)}`;
  }

  private hash(input: string): string {
    let hash = 0;
    for (let i = 0; i < input.length; i++) {
      hash = ((hash << 5) - hash + input.charCodeAt(i)) | 0;
    }
    return Math.abs(hash).toString(16).padStart(64, '0');
  }
}

interface WhitelistEntry {
  address: string;
  sponsorLimit: string;
  expiry: number;
  isActive: boolean;
}

export interface WhitelistStatus {
  isWhitelisted: boolean;
  limit: string;
  expiry: number;
  used: string;
}

export interface PaymasterData {
  paymasterAndData: string;
  preVerificationGas: string;
  verificationGasLimit: string;
  callGasLimit: string;
}

export interface UserOperation {
  sender: string;
  nonce: string;
  initCode: string;
  callData: string;
  callGasLimit: string;
  verificationGasLimit: string;
  preVerificationGas: string;
  maxFeePerGas: string;
  maxPriorityFeePerGas: string;
  paymasterAndData: string;
  signature: string;
}

export default PaymasterService.getInstance();
