/**
 * AccountAbstractionService - Web (React/TypeScript)
 *
 * NOTE: ERC-4337 account abstraction (smart accounts, session keys, social
 * recovery, batched calls) is NOT part of the canonical MasterWallet backend
 * contract (port 8450). This module exposes the public typing surface only;
 * operations that previously returned fabricated smart-account state now return
 * a descriptive error instead of fake data.
 */

export interface SmartAccount {
  address: string;
  owner: string;
  salt: string;
  deployed: boolean;
}

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

export interface SessionKey {
  address: string;
  privateKey: string;
  permissions: SessionKeyPermission[];
  expiresAt: number;
}

export interface SessionKeyPermission {
  targetContract: string;
  functionSelector: string;
  maxAmount: string;
}

export interface SocialRecoveryConfig {
  guardians: string[];
  threshold: number;
  delaySeconds: number;
}

export interface BatchedCall {
  to: string;
  value: string;
  data: string;
}

export interface DeployResult {
  success: boolean;
  address?: string;
  error?: string;
}

export interface ExecuteResult {
  success: boolean;
  transactionHash?: string;
  error?: string;
}

class AccountAbstractionServiceClass {
  private static instance: AccountAbstractionServiceClass | null = null;
  private constructor() {}
  static getInstance(): AccountAbstractionServiceClass {
    if (!AccountAbstractionServiceClass.instance) AccountAbstractionServiceClass.instance = new AccountAbstractionServiceClass();
    return AccountAbstractionServiceClass.instance;
  }

  createAccount(_owner: string, _salt?: string): DeployResult {
    return { success: false, error: 'ERC-4337 smart account creation is not supported by the canonical MasterWallet backend' };
  }

  deployAccount(_account: SmartAccount): DeployResult {
    return { success: false, error: 'Smart account deployment is not supported by the canonical MasterWallet backend' };
  }

  execute(_account: SmartAccount, _calls: BatchedCall[]): ExecuteResult {
    return { success: false, error: 'Batched execution is not supported by the canonical MasterWallet backend' };
  }

  createSessionKey(_account: SmartAccount, _permissions: SessionKeyPermission[], _ttlSeconds: number): { success: false; error: string } {
    return { success: false, error: 'Session keys are not supported by the canonical MasterWallet backend' };
  }

  configureRecovery(_account: SmartAccount, _config: SocialRecoveryConfig): { success: false; error: string } {
    return { success: false, error: 'Social recovery is not supported by the canonical MasterWallet backend' };
  }
}

export const AccountAbstractionService = AccountAbstractionServiceClass;
export default AccountAbstractionServiceClass.getInstance();
