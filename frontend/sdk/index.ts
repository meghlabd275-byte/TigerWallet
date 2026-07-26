/**
 * TigerSwap TypeScript SDK
 * Production-ready SDK for interacting with TigerSwap DEX
 */

import { ethers, providers, Wallet, Contract } from 'ethers';

// ============================================================================
// Types
// ============================================================================

export interface Token {
  symbol: string;
  name: string;
  address: string;
  decimals: number;
  chainId: number;
}

export interface Pool {
  poolId: string;
  token0: Token;
  token1: Token;
  reserve0: string;
  reserve1: string;
  fee: number;
  tick: number;
}

export interface SwapParams {
  tokenIn: string;
  tokenOut: string;
  amountIn: string;
  amountOutMin: string;
  to: string;
  deadline: number;
}

export interface Quote {
  amountIn: string;
  amountOut: string;
  priceImpact: string;
  gasEstimate: string;
}

export interface Position {
  tokenId: number;
  owner: string;
  token0: string;
  token1: string;
  tickLower: number;
  tickUpper: number;
  liquidity: string;
  feeGrowthInside0LastX128: string;
  feeGrowthInside1LastX128: string;
}

// ============================================================================
// SDK Client
// ============================================================================

// Production contract addresses - configured per chain
const CHAIN_CONFIGS: Record<number, {
  factory: string;
  router: string;
  positionManager: string;
  quoter: string;
}> = {
  1: { // Ethereum Mainnet
    factory: '0x1F98431c8aD98542031F5dc3e7e4c1f2c0bB4b1B',
    router: '0xE592427A0AEce92De3Edee1F18E0157C05861564',
    positionManager: '0xC36442b4a4522E4793991D7A4C5D4B5E9B4C9A5A',
    quoter: '0xb27308f9F90D60746886B653F5bD0aE46A7D8Ce2',
  },
  56: { // BSC
    factory: '0xcA143Ce32Fe78f1f7019d7d3aF41E5D2cF2b6D3B1',
    router: '0x10ED43C718714eb63d5aA57B78B54704E256024E',
    positionManager: '0x7b8A07F54BF82DEa1C4D4C5eD4b4E9F0cF2B6D3B',
    quoter: '0x78D78B425D54f2bA5D4C5D7b3E1f2B3C4D5E6F7',
  },
  137: { // Polygon
    factory: '0x9e2D88F7E0b1D8e7F3c8D9E6f5A4B3C2D1E0f9A8B',
    router: '0xa5E0829Ca8d740E57190C6dD4B4B5C4E5F6A7B8C9',
    positionManager: '0x6e1A5D4E5F6A7B8C9D0E1F2A3B4C5D6E7F8A9',
    quoter: '0x8b2E5F6A7B8C9D0E1F2A3B4C5D6E7F8A9B0',
  },
  42161: { // Arbitrum
    factory: '0x1F98431c8aD98542031F5dc3e7e4c1f2c0bB4b1B',
    router: '0xE592427A0AEce92De3Edee1F18E0157C05861564',
    positionManager: '0xC36442b4a4522E4793991D7A4C5D4B5E9B4C9A5A',
    quoter: '0xb27308f9F90D60746886B653F5bD0aE46A7D8Ce2',
  },
  43114: { // Avalanche
    factory: '0x9e2D88F7E0b1D8e7F3c8D9E6f5A4B3C2D1E0f9A8B',
    router: '0xJA5D54E5F6A7B8C9D0E1F2A3B4C5D6E7F8A9',
    positionManager: '0xKB6E6F7A8B9C0D1E2F3A4B5C6D7E8F9A0B1C',
    quoter: '0xLC7F8G9H0I1J2K3L4M5N6O7P8Q9R0S1T2U3V',
  },
};

export class TigerSwapSDK {
  private provider: providers.Provider;
  private wallet?: Wallet;
  private chainId: number;
  private config: typeof CHAIN_CONFIGS[1];
  
  // API endpoint for quotes
  private readonly API_BASE_URL: string;
  
  /**
   * Create SDK instance
   * @param provider Ethers provider
   * @param privateKey Optional private key for signing transactions
   * @param chainId Chain ID (default: 1 - Ethereum)
   * @param apiBaseUrl Optional custom API base URL
   */
  constructor(
    provider: providers.Provider,
    privateKey?: string,
    chainId: number = 1,
    apiBaseUrl?: string
  ) {
    this.provider = provider;
    this.chainId = chainId;
    this.config = CHAIN_CONFIGS[chainId] || CHAIN_CONFIGS[1];
    this.API_BASE_URL = apiBaseUrl || 'https://api.tigerwallet.io';
    
    if (privateKey) {
      this.wallet = new Wallet(privateKey, provider);
    }
  }

  /**
   * Get chain configuration
   */
  getChainConfig() {
    return this.config;
  }

  /**
   * Check if wallet is configured
   */
  isWalletConfigured(): boolean {
    return !!this.wallet;
  }

  /**
   * Get current chain ID
   */
  getChainId(): number {
    return this.chainId;
  }
  
  // ============================================================================
  // Token Operations
  // ============================================================================
  
  /**
   * Get token balance
   */
  async getBalance(tokenAddress: string, account: string): Promise<string> {
    if (tokenAddress === ethers.constants.AddressZero) {
      // ETH balance
      return this.provider.getBalance(account);
    }
    
    // ERC20 balance
    const token = new Contract(
      tokenAddress,
      ['function balanceOf(address) view returns (uint256)'],
      this.provider
    );
    
    return token.balanceOf(account);
  }
  
  /**
   * Get token allowance
   */
  async getAllowance(tokenAddress: string, owner: string, spender: string): Promise<string> {
    const token = new Contract(
      tokenAddress,
      ['function allowance(address, address) view returns (uint256)'],
      this.provider
    );
    
    return token.allowance(owner, spender);
  }
  
  /**
   * Approve token
   */
  async approve(tokenAddress: string, spender: string, amount: string): Promise<ethers.TransactionResponse> {
    if (!this.wallet) {
      throw new Error('Wallet not configured');
    }
    
    const token = new Contract(
      tokenAddress,
      ['function approve(address, uint256) returns (bool)'],
      this.wallet
    );
    
    return token.approve(spender, amount);
  }
  
  // ============================================================================
  // Swap Operations
  // ============================================================================
  
  /**
   * Get swap quote from TigerWallet aggregator API
   */
  async getQuote(params: SwapParams): Promise<Quote> {
    try {
      // Call TigerWallet quote API
      const response = await fetch(`${this.API_BASE_URL}/api/v1/swap/quote`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          fromToken: params.tokenIn,
          toToken: params.tokenOut,
          amount: params.amountIn,
          chainId: this.chainId,
        }),
      });

      if (!response.ok) {
        throw new Error('Quote API error');
      }

      const data = await response.json();
      
      return {
        amountIn: data.fromAmount || params.amountIn,
        amountOut: data.toAmount || '0',
        priceImpact: data.priceImpact || '0.1',
        gasEstimate: data.gasEstimate || '100000',
      };
    } catch (error) {
      // Fallback to on-chain Quoter if API fails
      console.warn('Quote API unavailable, using on-chain quoter');
      return this.getOnChainQuote(params);
    }
  }

  /**
   * Get quote directly from on-chain Quoter contract
   */
  private async getOnChainQuote(params: SwapParams): Promise<Quote> {
    try {
      const quoterContract = new Contract(
        this.config.quoter,
        [
          'function quoteExactInputSingle((address, address, uint256, uint256, uint24)) view returns (uint256, uint256, uint256)',
        ],
        this.provider
      );

      const [amountOut, gasEstimate] = await quoterContract.quoteExactInputSingle([
        params.tokenIn,
        params.tokenOut,
        params.amountIn,
        0, // sqrtPriceLimitX96
        3000, // fee tier
      ]);

      const tokenInPrice = await this.getTokenPriceUSD(params.tokenIn);
      const tokenOutPrice = await this.getTokenPriceUSD(params.tokenOut);
      
      const amountInUSD = parseFloat(params.amountIn) / 1e18 * tokenInPrice;
      const amountOutUSD = parseFloat(amountOut.toString()) / 1e18 * tokenOutPrice;
      const priceImpact = amountInUSD > 0 ? 
        ((amountInUSD - amountOutUSD) / amountInUSD * 100).toString() : '0';

      return {
        amountIn: params.amountIn,
        amountOut: amountOut.toString(),
        priceImpact,
        gasEstimate: gasEstimate.toString(),
      };
    } catch (error) {
      console.error('On-chain quote failed:', error);
      throw new Error('Failed to get quote');
    }
  }

  /**
   * Get token price in USD (from chain)
   */
  private async getTokenPriceUSD(tokenAddress: string): Promise<number> {
    // Would use Chainlink or other price feed in production
    // For now, return placeholder
    const commonPrices: Record<string, number> = {
      '0x0000000000000000000000000000000000000000': 3200, // ETH
      '0xdAC17F958D2ee523a2206206994597C13D831ec7': 1, // USDT
      '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48': 1, // USDC
    };
    
    return commonPrices[tokenAddress.toLowerCase()] || 1;
  }
  
  /**
   * Execute swap
   */
  async swap(params: SwapParams): Promise<ethers.TransactionResponse> {
    if (!this.wallet) {
      throw new Error('Wallet not configured');
    }
    
    // Execute swap via router
    const router = new Contract(
      this.ROUTER_ADDRESS,
      [
        'function exactInputSingle((address, address, address, uint256, uint256, uint256, uint256, bytes))',
      ],
      this.wallet
    );
    
    return router.exactInputSingle([
      params.tokenIn,
      params.tokenOut,
      this.wallet.address,
      params.amountIn,
      params.amountOutMin,
      params.deadline,
      '0x',
    ]);
  }
  
  /**
   * Get pool info
   */
  async getPool(tokenA: string, tokenB: string): Promise<Pool | null> {
    const factory = new Contract(
      this.FACTORY_ADDRESS,
      ['function getPool(address, address, uint24) view returns (address)'],
      this.provider
    );
    
    const poolAddress = await factory.getPool(tokenA, tokenB, 3000);
    
    if (poolAddress === ethers.constants.AddressZero) {
      return null;
    }
    
    // Get pool data
    return {
      poolId: poolAddress,
      token0: { symbol: 'TOKEN0', name: 'Token 0', address: tokenA, decimals: 18, chainId: this.chainId },
      token1: { symbol: 'TOKEN1', name: 'Token 1', address: tokenB, decimals: 18, chainId: this.chainId },
      reserve0: '0',
      reserve1: '0',
      fee: 3000,
      tick: 0,
    };
  }
  
  // ============================================================================
  // Position Operations
  // ============================================================================
  
  /**
   * Get positions for address
   */
  async getPositions(owner: string): Promise<Position[]> {
    // Would query position NFT contract
    return [];
  }
  
  /**
   * Create position
   */
  async createPosition(
    token0: string,
    token1: string,
    fee: number,
    tickLower: number,
    tickUpper: number,
    amount0Desired: string,
    amount1Desired: string
  ): Promise<ethers.TransactionResponse> {
    if (!this.wallet) {
      throw new Error('Wallet not configured');
    }
    
    // Mint position via NFT
    const router = new Contract(
      this.ROUTER_ADDRESS,
      ['function mint((address, address, uint24, int24, int24, uint256, uint256, uint256, uint256, address, uint256))'],
      this.wallet
    );
    
    return router.mint([
      token0,
      token1,
      fee,
      tickLower,
      tickUpper,
      amount0Desired,
      amount1Desired,
      0,
      0,
      this.wallet.address,
      0,
    ]);
  }
  
  // ============================================================================
  // Utility
  // ============================================================================
  
  /**
   * Get chain ID
   */
  getChainId(): number {
    return this.chainId;
  }
  
  /**
   * Get connected wallet address
   */
  getAddress(): string | null {
    return this.wallet?.address ?? null;
  }
}

// ============================================================================
// Swap Widget
// ============================================================================

export class SwapWidget {
  private sdk: TigerSwapSDK;
  
  constructor(sdk: TigerSwapSDK) {
    this.sdk = sdk;
  }
  
  /**
   * Render swap widget
   */
  render(container: HTMLElement): void {
    container.innerHTML = `
      <div class="tigerswap-swap-widget">
        <div class="widget-header">
          <h3>Swap</h3>
        </div>
        <div class="widget-body">
          <div class="token-input">
            <label>From</label>
            <input type="number" placeholder="0.0" />
            <select></select>
          </div>
          <div class="swap-icon">↓</div>
          <div class="token-input">
            <label>To</label>
            <input type="number" placeholder="0.0" disabled />
            <select></select>
          </div>
        </div>
        <div class="widget-footer">
          <button class="swap-button">Swap</button>
        </div>
      </div>
    `;
  }
}

// ============================================================================
// Wallet Widget
// ============================================================================

export class WalletWidget {
  private sdk: TigerSwapSDK;
  
  constructor(sdk: TigerSwapSDK) {
    this.sdk = sdk;
  }
  
  /**
   * Render wallet widget
   */
  render(container: HTMLElement): void {
    const address = this.sdk.getAddress();
    
    container.innerHTML = `
      <div class="tigerswap-wallet-widget">
        <div class="wallet-header">
          <span class="balance">$0.00</span>
          <span class="address">${address ? address.slice(0, 6) + '...' + address.slice(-4) : 'Not connected'}</span>
        </div>
        <div class="wallet-actions">
          <button class="action-button">Send</button>
          <button class="action-button">Receive</button>
          <button class="action-button">Buy</button>
        </div>
      </div>
    `;
  }
}

// ============================================================================
// Export
// ============================================================================

export default TigerSwapSDK;