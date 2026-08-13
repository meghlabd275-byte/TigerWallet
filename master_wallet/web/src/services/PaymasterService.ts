/**
 * PaymasterService - Web (React/TypeScript)
 *
 * NOTE: ERC-4337 paymaster sponsorship is NOT part of the canonical
 * MasterWallet backend contract (port 8450). This module exposes the public
 * typing surface only; operations that previously returned fabricated gas
 * sponsorship estimates now return a descriptive error instead of fake data.
 */

export interface UserOperation {
  sender: string;
  nonce: string;
  initCode: string;
  callData: string;
  callGasLimit: number;
  verificationGasLimit: number;
  preVerificationGas: number;
  maxFeePerGas: string;
  maxPriorityFeePerGas: string;
  paymasterAndData: string;
  signature: string;
}

export interface PaymasterConfig {
  mode: 'verifying' | 'token' | 'sponsored';
  paymasterAddress: string;
  tokenAddress?: string;
}

export interface SponsorInfo {
  sponsorAddress: string;
  maxGasPerTransaction: number;
  dailyBudget: string;
  usedToday: string;
}

export interface GasEstimate {
  callGasLimit: number;
  verificationGasLimit: number;
  preVerificationGas: number;
  totalGas: number;
}

export interface PaymasterResult {
  success: boolean;
  paymasterAndData?: string;
  error?: string;
}

class PaymasterServiceClass {
  private static instance: PaymasterServiceClass | null = null;
  private constructor() {}
  static getInstance(): PaymasterServiceClass {
    if (!PaymasterServiceClass.instance) PaymasterServiceClass.instance = new PaymasterServiceClass();
    return PaymasterServiceClass.instance;
  }

  estimateGas(_userOp: Partial<UserOperation>): { success: false; error: string } {
    return { success: false, error: 'ERC-4337 paymaster gas estimation is not supported by the canonical MasterWallet backend' };
  }

  sponsor(_userOp: Partial<UserOperation>, _config: PaymasterConfig): PaymasterResult {
    return { success: false, error: 'Paymaster sponsorship is not supported by the canonical MasterWallet backend' };
  }

  getSponsorInfo(): { success: false; error: string } {
    return { success: false, error: 'Sponsor info is not supported by the canonical MasterWallet backend' };
  }
}

export const PaymasterService = PaymasterServiceClass;
export default PaymasterServiceClass.getInstance();
