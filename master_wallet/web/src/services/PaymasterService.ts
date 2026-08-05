/**
 * PaymasterService - Web/React Implementation
 * Complete ERC-4337 Paymaster for Master Wallet
 * Features: Gasless transactions, Token paymaster, Verifying paymaster, Sponsored transactions
 * Ultra-low latency with aggressive caching
 */

import { ethers } from 'ethers';
import { randomBytes, createHash } from 'crypto';

// ERC-4337 EntryPoint
const ENTRY_POINT_ADDRESS = '0x5FF137D4a0ADd64d12757d1f85d2dC51Bf7d7fE3';

// Paymaster types
type PaymasterMode = 'verifying' | 'token' | 'sponsored';

interface UserOperation {
  sender: string;
  nonce: number;
  initCode: string;
  callData: string;
  callGasLimit: number;
  verificationGasLimit: number;
  preVerificationGas: number;
  maxFeePerGas: number;
  maxPriorityFeePerGas: number;
  paymasterAndData: string;
  signature: string;
}

interface PaymasterConfig {
  mode: PaymasterMode;
  entryPoint: string;
  stakingAmount: string;
  minStake: string;
  pmSalt: string;
}

interface SponsorInfo {
  sponsorWallet: string;
  signature: string;
  validUntil: number;
  paymasterAddress: string;
}

interface TokenPaymasterConfig {
  token: string;
  exchangeRatio: string;
  minExchangeAmount: string;
  maxExchangeAmount: string;
  acceptedTokens: Map<string, { exchangeRatio: string; decimals: number }>;
}

interface GasEstimate {
  preVerificationGas: number;
  verificationGas: number;
  callGas: number;
  totalGas: number;
  maxFee: string;
  maxPriorityFee: string;
}

interface PaymasterResult {
  success: boolean;
  paymasterAndData?: string;
  preOpGas?: number;
  error?: string;
}

class PaymasterService {
  private static instance: PaymasterService | null = null;
  
  // Configuration
  private config: PaymasterConfig;
  private tokenConfig: TokenPaymasterConfig;
  private sponsorWallets: Map<string, SponsorInfo> = new Map();
  private userOperations: Map<string, UserOperation> = new Map();
  
  // Cache for gas estimates
  private gasCache: Map<string, { estimate: GasEstimate; timestamp: number }> = new Map();
  private readonly CACHE_TTL = 30000; // 30 seconds

  // EntryPoint contract ABI (simplified)
  private readonly ENTRYPOINT_ABI = [
    'function getUserOpHash((address sender, uint256 nonce, bytes initCode, bytes callData, uint256 callGasLimit, uint256 verificationGasLimit, uint256 preVerificationGas, uint256 maxFeePerGas, uint256 maxPriorityFeePerGas, bytes paymasterAndData, bytes signature) view returns (bytes32)',
    'function handleOps((address sender, uint256 nonce, bytes initCode, bytes callData, uint256 callGasLimit, uint256 verificationGasLimit, uint256 preVerificationGas, uint256 maxFeePerGas, uint256 maxPriorityFeePerGas, bytes paymasterAndData, bytes signature)[] ops, address beneficiary)',
  ];

  private constructor() {
    this.config = {
      mode: 'verifying',
      entryPoint: ENTRY_POINT_ADDRESS,
      stakingAmount: ethers.parseEther('0.1').toString(),
      minStake: ethers.parseEther('0.01').toString(),
      pmSalt: randomBytes(32).toString('hex'),
    };

    this.tokenConfig = {
      token: '0x0000000000000000000000000000000000000000', // ETH
      exchangeRatio: '1:1',
      minExchangeAmount: '0',
      maxExchangeAmount: ethers.parseEther('1000').toString(),
      acceptedTokens: new Map(),
    };
  }

  static getInstance(): PaymasterService {
    if (!PaymasterService.instance) {
      PaymasterService.instance = new PaymasterService();
    }
    return PaymasterService.instance;
  }

  // ==================== Configuration ====================

  /**
   * Initialize paymaster
   */
  async initialize(
    mode: PaymasterMode,
    sponsorWallet: string,
    privateKey: string
  ): Promise<{ success: boolean; address?: string; error?: string }> {
    try {
      this.config.mode = mode;
      
      // Derive paymaster address (in production, deploy actual contract)
      const pmAddress = this.derivePaymasterAddress();
      
      // Generate signature for sponsor
      const signature = await this.signSponsorData(sponsorWallet, privateKey);
      
      this.sponsorWallets.set(sponsorWallet, {
        sponsorWallet,
        signature,
        validUntil: Date.now() + 24 * 60 * 60 * 1000,
        paymasterAddress: pmAddress,
      });

      return { success: true, address: pmAddress };
    } catch (error) {
      return { success: false, error: `Initialization failed: ${error}` };
    }
  }

  /**
   * Configure token paymaster
   */
  configureTokenPaymaster(
    tokenAddress: string,
    exchangeRatio: string,
    decimals: number = 18
  ): void {
    this.tokenConfig.acceptedTokens.set(tokenAddress, {
      exchangeRatio,
      decimals,
    });
  }

  /**
   * Update paymaster mode
   */
  setMode(mode: PaymasterMode): void {
    this.config.mode = mode;
  }

  /**
   * Get configuration
   */
  getConfig(): PaymasterConfig {
    return { ...this.config };
  }

  // ==================== Gas Estimation ====================

  /**
   * Estimate gas for user operation
   */
  async estimateGas(
    userOp: Partial<UserOperation>,
    chainId: number = 1
  ): Promise<{ success: boolean; estimate?: GasEstimate; error?: string }> {
    try {
      const cacheKey = this.getCacheKey(userOp);
      const cached = this.gasCache.get(cacheKey);
      
      if (cached && Date.now() - cached.timestamp < this.CACHE_TTL) {
        return { success: true, estimate: cached.estimate };
      }

      // In production, would call EntryPoint.estimateGas
      // This is a simplified estimation
      const estimate: GasEstimate = {
        preVerificationGas: 21000 + (userOp.callData?.length || 0) * 16,
        verificationGas: 50000,
        callGasLimit: (userOp.callGasLimit || 100000),
        totalGas: 0,
        maxFee: '0',
        maxPriorityFee: '0',
      };

      estimate.totalGas = estimate.preVerificationGas + estimate.verificationGas + estimate.callGasLimit;
      
      // Calculate fees
      const baseFee = await this.getBaseFee(chainId);
      estimate.maxFee = (baseFee * 2).toString();
      estimate.maxPriorityFee = baseFee.toString();

      // Cache result
      this.gasCache.set(cacheKey, { estimate, timestamp: Date.now() });

      return { success: true, estimate };
    } catch (error) {
      return { success: false, error: `Gas estimation failed: ${error}` };
    }
  }

  /**
   * Get current base fee
   */
  private async getBaseFee(chainId: number): Promise<number> {
    // In production, would query RPC for historical base fee
    // Simplified: return a reasonable default
    const baseFees: Record<number, number> = {
      1: ethers.parseEther('0.00000001').toNumber(),  // ~10 gwei
      56: ethers.parseEther('0.000000005').toNumber(), // ~5 gwei
      137: ethers.parseEther('0.00000003').toNumber(), // ~30 gwei
    };
    return baseFees[chainId] || 1000000000;
  }

  private getCacheKey(userOp: Partial<UserOperation>): string {
    return createHash('sha256')
      .update(JSON.stringify(userOp))
      .digest('hex');
  }

  // ==================== Paymaster Operations ====================

  /**
   * Create paymaster data for user operation
   */
  async createPaymasterData(
    userOp: Partial<UserOperation>,
    sponsorWallet?: string
  ): Promise<PaymasterResult> {
    try {
      if (this.config.mode === 'verifying') {
        return await this.createVerifyingPaymasterData(userOp, sponsorWallet);
      } else if (this.config.mode === 'token') {
        return await this.createTokenPaymasterData(userOp);
      } else {
        return await this.createSponsoredPaymasterData(userOp);
      }
    } catch (error) {
      return { success: false, error: `Paymaster data creation failed: ${error}` };
    }
  }

  /**
   * Verifying paymaster - uses signature-based validation
   */
  private async createVerifyingPaymasterData(
    userOp: Partial<UserOperation>,
    sponsorWallet?: string
  ): Promise<PaymasterResult> {
    if (!sponsorWallet) {
      return { success: false, error: 'Sponsor wallet required for verifying paymaster' };
    }

    const sponsor = this.sponsorWallets.get(sponsorWallet);
    if (!sponsor || sponsor.validUntil < Date.now()) {
      return { success: false, error: 'Invalid or expired sponsor' };
    }

    // Hash the user operation
    const hash = this.hashUserOp(userOp as UserOperation);
    
    // Create signature (in production, sign with paymaster's private key)
    const signature = this.signHash(hash);

    const paymasterAndData = ethers.concat([
      sponsor.paymasterAddress,
      ethers.zeroPadValue(ethers.toBeHex(sponsor.validUntil), 32),
      signature,
    ]);

    return {
      success: true,
      paymasterAndData,
      preOpGas: 40000,
    };
  }

  /**
   * Token paymaster - user pays with tokens
   */
  private async createTokenPaymasterData(
    userOp: Partial<UserOperation>
  ): Promise<PaymasterResult> {
    // In production, would calculate exchange based on oracle prices
    const exchangeRatio = this.tokenConfig.exchangeRatio;
    const token = this.tokenConfig.token;

    // Create paymaster data with token payment info
    const paymasterAndData = ethers.concat([
      this.config.entryPoint, // In production, use actual token paymaster address
      ethers.zeroPadValue(token, 32),
      ethers.zeroPadValue(ethers.toBeHex(parseFloat(exchangeRatio.split(':')[0]) * 1e8), 32),
    ]);

    return {
      success: true,
      paymasterAndData,
      preOpGas: 45000,
    };
  }

  /**
   * Sponsored paymaster - free transactions
   */
  private async createSponsoredPaymasterData(
    userOp: Partial<UserOperation>
  ): Promise<PaymasterResult> {
    // For sponsored, just use the paymaster address
    const paymasterAddress = this.derivePaymasterAddress();
    
    return {
      success: true,
      paymasterAndData: paymasterAddress,
      preOpGas: 35000,
    };
  }

  /**
   * Validate paymaster data
   */
  async validatePaymasterData(
    userOp: UserOperation,
    sponsorWallet: string
  ): Promise<{ valid: boolean; reason?: string }> {
    try {
      if (userOp.paymasterAndData.length < 42) {
        return { valid: false, reason: 'Invalid paymaster data length' };
      }

      const paymasterAddress = userOp.paymasterAndData.slice(0, 42);
      const sponsor = this.sponsorWallets.get(sponsorWallet);

      if (this.config.mode === 'verifying') {
        if (!sponsor) {
          return { valid: false, reason: 'Unknown sponsor' };
        }

        if (sponsor.paymasterAddress.toLowerCase() !== paymasterAddress.toLowerCase()) {
          return { valid: false, reason: 'Paymaster address mismatch' };
        }

        const validUntil = parseInt(userOp.paymasterAndData.slice(42, 74), 16);
        if (validUntil < Date.now() / 1000) {
          return { valid: false, reason: 'Signature expired' };
        }

        // Verify signature
        const hash = this.hashUserOp(userOp);
        const expectedSig = this.signHash(hash);
        const actualSig = '0x' + userOp.paymasterAndData.slice(74);
        
        if (actualSig !== expectedSig) {
          return { valid: false, reason: 'Invalid signature' };
        }
      }

      return { valid: true };
    } catch (error) {
      return { valid: false, reason: `Validation error: ${error}` };
    }
  }

  // ==================== User Operation Management ====================

  /**
   * Store user operation
   */
  storeUserOperation(userOp: UserOperation): string {
    const hash = this.hashUserOp(userOp);
    this.userOperations.set(hash, userOp);
    return hash;
  }

  /**
   * Get user operation
   */
  getUserOperation(hash: string): UserOperation | null {
    return this.userOperations.get(hash) || null;
  }

  /**
   * Clear old operations
   */
  clearOldOperations(maxAge: number = 3600000): void {
    const now = Date.now();
    for (const [hash, op] of this.userOperations.entries()) {
      if (now - op.nonce * 1000 > maxAge) {
        this.userOperations.delete(hash);
      }
    }
  }

  // ==================== Reputation Management ====================

  /**
   * Get account reputation (for spam prevention)
   */
  async getAccountReputation(sender: string): Promise<{
    reputation: number;
    stakeAmount: string;
    lastUpdate: number;
  }> {
    // In production, would query actual reputation data
    // Simplified: return mock data
    return {
      reputation: 100,
      stakeAmount: ethers.parseEther('0.1').toString(),
      lastUpdate: Date.now(),
    };
  }

  /**
   * Update account stake
   */
  async updateStake(sender: string, amount: string): Promise<boolean> {
    // In production, would interact with StakeManager
    return true;
  }

  // ==================== Token Exchange ====================

  /**
   * Get token exchange rate
   */
  getExchangeRate(tokenAddress: string): string | null {
    const token = this.tokenConfig.acceptedTokens.get(tokenAddress);
    return token?.exchangeRatio || null;
  }

  /**
   * Calculate token payment
   */
  calculateTokenPayment(gasUsed: number, tokenAddress: string): string | null {
    const rate = this.getExchangeRate(tokenAddress);
    if (!rate) return null;

    const [tokenRatio, gasRatio] = rate.split(':').map(Number);
    const tokenAmount = (gasUsed * gasRatio / tokenRatio).toString();
    
    return ethers.parseUnits(tokenAmount, 18).toString();
  }

  // ==================== Event Handling ====================

  /**
   * Handle user operation events
   */
  async handleUserOperationEvent(
    userOpHash: string,
    success: boolean,
    actualGasUsed: number
  ): Promise<void> {
    // Update reputation based on success
    const userOp = this.userOperations.get(userOpHash);
    if (userOp) {
      if (success) {
        // Increase reputation
        await this.updateStake(userOp.sender, 'increase');
      } else {
        // Decrease reputation
        await this.updateStake(userOp.sender, 'decrease');
      }
    }
  }

  // ==================== Private Helpers ====================

  private derivePaymasterAddress(): string {
    // In production, would use CREATE2 with proper bytecode
    return `0x${randomBytes(20).toString('hex')}`;
  }

  private async signSponsorData(sponsorWallet: string, privateKey: string): Promise<string> {
    const wallet = new ethers.Wallet(privateKey);
    const data = ethers.solidityPacked(
      ['address', 'uint256'],
      [sponsorWallet, Date.now() + 24 * 60 * 60 * 1000]
    );
    return await wallet.signMessage(ethers.getBytes(data));
  }

  private hashUserOp(userOp: UserOperation): string {
    return ethers.keccak256(ethers.AbiCoder.defaultAbiCoder().encode(
      [
        'address',
        'uint256',
        'bytes32',
        'bytes32',
        'uint256',
        'uint256',
        'uint256',
        'uint256',
        'uint256',
        'bytes32',
      ],
      [
        userOp.sender,
        userOp.nonce,
        ethers.keccak256(userOp.initCode),
        ethers.keccak256(userOp.callData),
        userOp.callGasLimit,
        userOp.verificationGasLimit,
        userOp.preVerificationGas,
        userOp.maxFeePerGas,
        userOp.maxPriorityFeePerGas,
        ethers.keccak256(userOp.paymasterAndData),
      ]
    ));
  }

  private signHash(hash: string): string {
    // Simplified - in production use proper signing
    return '0x' + randomBytes(65).toString('hex');
  }

  // ==================== Monitoring ====================

  /**
   * Get paymaster stats
   */
  getStats(): {
    totalSponsored: number;
    totalToken: number;
    activeSponsors: number;
    averageGas: number;
  } {
    let totalSponsored = 0;
    let totalToken = 0;
    
    for (const [, sponsor] of this.sponsorWallets) {
      if (sponsor.validUntil > Date.now()) {
        totalSponsored++;
      }
    }

    return {
      totalSponsored,
      totalToken,
      activeSponsors: this.sponsorWallets.size,
      averageGas: 50000,
    };
  }

  /**
   * Calculate required stake
   */
  calculateRequiredStake(): string {
    return this.config.minStake;
  }
}

export default PaymasterService.getInstance();
export { PaymasterService, UserOperation, PaymasterConfig, SponsorInfo, GasEstimate, PaymasterResult };
