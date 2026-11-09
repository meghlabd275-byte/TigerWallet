/**
 * TigerSwap TypeScript SDK
 * Production-ready SDK for interacting with TigerSwap DEX
 */

import { ethers, providers, Wallet, Contract, BigNumber } from 'ethers';

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
  /** The pool fee tier in hundredths of a basis point; defaults to 0.30%. */
  feeTier?: number;
  /** Optional price bound; zero lets the router use the whole range. */
  sqrtPriceLimitX96?: BigNumber;
}

export interface Quote {
  amountIn: string;
  amountOut: string;
  priceImpact: string;
  gasEstimate: string;
  /** The address the swap was priced against, for later execution. */
  router: string;
  /** The fee tier the pool was priced at, in hundredths of a basis point. */
  feeTier: number;
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
  /** The wrapped native token, which the router uses for native swaps. */
  wrappedNative: string;
  /** The pools the quoter is asked about, in the order they are tried. */
  feeTiers: number[];
}> = {
  1: { // Ethereum Mainnet - Uniswap V3
    factory: '0x1F98431c8aD98523631AE4a59f267346ea31F984',
    router: '0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45',
    positionManager: '0xC36442b4a4522E871399CD717aBDD847Ab11FE88',
    quoter: '0x61fFE014bA17989E743c5F6cB21bF9697530B21e',
    wrappedNative: '0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2',
    feeTiers: [100, 500, 3000, 10000],
  },
  10: { // Optimism - Uniswap V3
    factory: '0x1F98431c8aD98523631AE4a59f267346ea31F984',
    router: '0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45',
    positionManager: '0xC36442b4a4522E871399CD717aBDD847Ab11FE88',
    quoter: '0x61fFE014bA17989E743c5F6cB21bF9697530B21e',
    wrappedNative: '0x4200000000000000000000000000000000000006',
    feeTiers: [100, 500, 3000, 10000],
  },
  56: { // BSC - PancakeSwap V3
    factory: '0x0BFbCF9fa4f9C56B0F40a671Ad40E0805A091865',
    router: '0x13f4EA83D0bd40E75C8222255bc855a974568Dd4',
    positionManager: '0x46A15B0b27311cedF172AB29E4f4766fbE7F4364',
    quoter: '0xB048Bbc1Ee6b733FFfCFb9e9CeF7375518e25997',
    wrappedNative: '0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c',
    feeTiers: [100, 500, 2500, 10000],
  },
  137: { // Polygon - Uniswap V3
    factory: '0x1F98431c8aD98523631AE4a59f267346ea31F984',
    router: '0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45',
    positionManager: '0xC36442b4a4522E871399CD717aBDD847Ab11FE88',
    quoter: '0x61fFE014bA17989E743c5F6cB21bF9697530B21e',
    wrappedNative: '0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270',
    feeTiers: [100, 500, 3000, 10000],
  },
  8453: { // Base - Uniswap V3
    factory: '0x33128a8fC17869897dcE68Ed026d694621f6FDfD',
    router: '0x2626664c2603336E57B271c5C0b26F421741e481',
    positionManager: '0x03a520b32C04BF3bEEf7BEb72E919cf822Ed34f1',
    quoter: '0x3d4e44Eb1374240CE5F1B871ab261CD16335B76a',
    wrappedNative: '0x4200000000000000000000000000000000000006',
    feeTiers: [100, 500, 3000, 10000],
  },
  42161: { // Arbitrum - Uniswap V3
    factory: '0x1F98431c8aD98523631AE4a59f267346ea31F984',
    router: '0x68b3465833fb72A70ecDF485E0e4C7bD8665Fc45',
    positionManager: '0xC36442b4a4522E871399CD717aBDD847Ab11FE88',
    quoter: '0x61fFE014bA17989E743c5F6cB21bF9697530B21e',
    wrappedNative: '0x82aF49447D8a07e3bd95BD0d56f35241523fBab1',
    feeTiers: [100, 500, 3000, 10000],
  },
};

/** The zero price bound, which tells the quoter not to limit the price. */
const NO_PRICE_LIMIT = BigNumber.from(0);

/** The native token is not a contract; it is wrapped before it is quoted. */
const NATIVE_TOKEN = '0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE';

/** The Quoter reverts rather than returning zero when a pool does not exist. */
const QUOTER_ABI = [
  'function quoteExactInputSingle((address tokenIn, address tokenOut, uint256 amountIn, uint24 fee, uint160 sqrtPriceLimitX96)) returns (uint256 amountOut, uint160 sqrtPriceX96After, uint32 initializedTicksCrossed, uint256 gasEstimate)',
];

/** The fee tier used when a caller does not name one: 0.30%. */
const DEFAULT_FEE_TIER = 3000;

/** SwapRouter02's single hop swap, whose parameters are a struct. */
const ROUTER_ABI = [
  'function exactInputSingle((address tokenIn, address tokenOut, uint24 fee, address recipient, uint256 amountIn, uint256 amountOutMinimum, uint160 sqrtPriceLimitX96)) payable returns (uint256 amountOut)',
];

/** The factory reports the pool address for a pair and fee. */
const FACTORY_ABI = ['function getPool(address, address, uint24) view returns (address)'];

/** A Uniswap V3 pool exposes its tokens, fee, tick and liquidity. */
const POOL_ABI = [
  'function token0() view returns (address)',
  'function token1() view returns (address)',
  'function fee() view returns (uint24)',
  'function slot0() view returns (uint160 sqrtPriceX96, int24 tick, uint16 observationIndex, uint16 observationCardinality, uint16 observationCardinalityNext, uint8 feeProtocol, bool unlocked)',
  'function liquidity() view returns (uint128)',
];

/** The non fungible position manager owns every position. */
const POSITION_MANAGER_ABI = [
  'function balanceOf(address owner) view returns (uint256)',
  'function tokenOfOwnerByIndex(address owner, uint256 index) view returns (uint256)',
  'function positions(uint256 tokenId) view returns (uint96 nonce, address operator, address token0, address token1, uint24 fee, int24 tickLower, int24 tickUpper, uint128 liquidity, uint256 feeGrowthInside0LastX128, uint256 feeGrowthInside1LastX128, uint128 tokensOwed0, uint128 tokensOwed1)',
  'function mint((address token0, address token1, uint24 fee, int24 tickLower, int24 tickUpper, uint256 amount0Desired, uint256 amount1Desired, uint256 amount0Min, uint256 amount1Min, address recipient, uint256 deadline)) payable returns (uint256 tokenId, uint128 liquidity, uint256 amount0, uint256 amount1)',
];

/** Enough of an ERC-20 to describe a token. */
const ERC20_ABI = [
  'function decimals() view returns (uint8)',
  'function symbol() view returns (string)',
  'function name() view returns (string)',
  'function balanceOf(address) view returns (uint256)',
];

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
      // Native balance.
      return (await this.provider.getBalance(account)).toString();
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
  async approve(tokenAddress: string, spender: string, amount: string): Promise<ethers.ContractTransaction> {
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

      // The API reports its own price impact; when it omits one it is left
      // blank rather than filled with a made up figure.
      return {
        amountIn: data.fromAmount || params.amountIn,
        amountOut: data.toAmount || '0',
        priceImpact: data.priceImpact ?? '',
        gasEstimate: data.gasEstimate ?? '',
        router: data.router || this.config.router,
        feeTier: Number(data.fee ?? data.feeTier ?? 0),
      };
    } catch (error) {
      // Fallback to on-chain Quoter if API fails
      console.warn('Quote API unavailable, using on-chain quoter');
      return this.getOnChainQuote(params);
    }
  }

  /** Replace the native token marker with its wrapped address. */
  private wrapNative(token: string): string {
    return token.toLowerCase() === NATIVE_TOKEN.toLowerCase()
      ? this.config.wrappedNative
      : token;
  }

  /** Read the token's decimals, so amounts can be converted without guessing. */
  private async readDecimals(token: string): Promise<number> {
    const contract = new Contract(
      token,
      ['function decimals() view returns (uint8)'],
      this.provider
    );
    return (await contract.decimals()).toNumber();
  }

  /** Describe a token from its own contract. */
  private async describeToken(address: string): Promise<Token> {
    const contract = new Contract(address, ERC20_ABI, this.provider);

    const [symbol, name, decimals] = await Promise.all([
      contract.symbol(),
      contract.name(),
      contract.decimals(),
    ]);

    return {
      symbol,
      name,
      address,
      decimals: decimals.toNumber(),
      chainId: this.chainId,
    };
  }

  /** How many base units one whole token is worth. */
  private unitFor(decimals: number): BigNumber {
    return BigNumber.from(10).pow(decimals);
  }

  /** Convert an on-chain amount into whole tokens, for ratio arithmetic. */
  private toWhole(amount: BigNumber, decimals: number): number {
    return Number(amount.toString()) / 10 ** decimals;
  }

  /** Price one token in the other, straight from the pool, with no fee taken. */
  private async spotPrice(
    tokenIn: string,
    tokenOut: string,
    feeTier: number,
    decimalsIn: number,
    decimalsOut: number
  ): Promise<number> {
    const quoter = new Contract(this.config.quoter, QUOTER_ABI, this.provider);
    // One whole unit of the input token, so the result is a per-token price.
    const probe = this.unitFor(decimalsIn);

    const result = await quoter.callStatic.quoteExactInputSingle({
      tokenIn: this.wrapNative(tokenIn),
      tokenOut: this.wrapNative(tokenOut),
      amountIn: probe,
      fee: feeTier,
      sqrtPriceLimitX96: NO_PRICE_LIMIT,
    });

    if (result.amountOut.isZero()) {
      return 0;
    }

    return this.toWhole(result.amountOut, decimalsOut);
  }

  /**
   * Price a swap against the on-chain Quoter.
   *
   * Every pool fee tier is tried and the best output wins. Price impact is the
   * difference between the pool's spot price and the price this trade actually
   * gets, which is measured on-chain rather than inferred from a USD figure.
   */
  private async getOnChainQuote(params: SwapParams): Promise<Quote> {
    const quoter = new Contract(this.config.quoter, QUOTER_ABI, this.provider);
    const amountIn = BigNumber.from(params.amountIn);

    if (amountIn.lte(0)) {
      throw new Error('The amount to swap must be greater than zero');
    }

    let best: { amountOut: BigNumber; feeTier: number; gas: BigNumber } | null = null;

    for (const feeTier of this.config.feeTiers) {
      try {
        const result = await quoter.callStatic.quoteExactInputSingle({
          tokenIn: this.wrapNative(params.tokenIn),
          tokenOut: this.wrapNative(params.tokenOut),
          amountIn,
          fee: feeTier,
          sqrtPriceLimitX96: NO_PRICE_LIMIT,
        });

        const out: BigNumber = result.amountOut;
        if (out.gt(0) && (best === null || out.gt(best.amountOut))) {
          best = { amountOut: out, feeTier, gas: result.gasEstimate };
        }
      } catch {
        // No pool at this fee tier; the next one is tried.
      }
    }

    if (best === null) {
      throw new Error('No pool exists for this pair on this chain');
    }

    const [decimalsIn, decimalsOut] = await Promise.all([
      this.readDecimals(this.wrapNative(params.tokenIn)),
      this.readDecimals(this.wrapNative(params.tokenOut)),
    ]);

    let priceImpact = '';
    try {
      const spot = await this.spotPrice(
        params.tokenIn,
        params.tokenOut,
        best.feeTier,
        decimalsIn,
        decimalsOut
      );
      // The price this trade actually pays, per whole input token.
      const executed =
        this.toWhole(best.amountOut, decimalsOut) /
        this.toWhole(amountIn, decimalsIn);

      if (spot > 0) {
        const impact = ((spot - executed) / spot) * 100;
        priceImpact = (impact > 0 ? impact : 0).toFixed(4);
      }
    } catch {
      // The pool answered for the trade but not for the spot probe; the impact
      // is left blank rather than invented.
      priceImpact = '';
    }

    return {
      amountIn: params.amountIn,
      amountOut: best.amountOut.toString(),
      priceImpact,
      gasEstimate: best.gas.toString(),
      router: this.config.router,
      feeTier: best.feeTier,
    };
  }
  /**
   * Execute swap
   */
  async swap(params: SwapParams): Promise<ethers.ContractTransaction> {
    if (!this.wallet) {
      throw new Error('Wallet not configured');
    }

    const router = new Contract(this.config.router, ROUTER_ABI, this.wallet);

    // The native token has no contract, so it is wrapped before the swap.
    return router.exactInputSingle({
      tokenIn: this.wrapNative(params.tokenIn),
      tokenOut: this.wrapNative(params.tokenOut),
      fee: params.feeTier ?? DEFAULT_FEE_TIER,
      recipient: params.to || this.wallet.address,
      amountIn: params.amountIn,
      amountOutMinimum: params.amountOutMin,
      sqrtPriceLimitX96: params.sqrtPriceLimitX96 ?? NO_PRICE_LIMIT,
    });
  }

  /**
   * Get pool info
   */
  async getPool(tokenA: string, tokenB: string): Promise<Pool | null> {
    const factory = new Contract(this.config.factory, FACTORY_ABI, this.provider);
    const poolAddress = await factory.getPool(
      this.wrapNative(tokenA),
      this.wrapNative(tokenB),
      DEFAULT_FEE_TIER
    );

    if (poolAddress === ethers.constants.AddressZero) {
      return null;
    }

    const pool = new Contract(poolAddress, POOL_ABI, this.provider);
    const [rawToken0, rawToken1, fee, slot0] = await Promise.all([
      pool.token0(),
      pool.token1(),
      pool.fee(),
      pool.slot0(),
    ]);

    const [token0, token1] = await Promise.all([
      this.describeToken(rawToken0),
      this.describeToken(rawToken1),
    ]);

    // A V3 pool holds no reserves of its own; what it holds is the balances of
    // the two tokens that liquidity providers have deposited.
    const [balance0, balance1] = await Promise.all([
      new Contract(rawToken0, ERC20_ABI, this.provider).balanceOf(poolAddress),
      new Contract(rawToken1, ERC20_ABI, this.provider).balanceOf(poolAddress),
    ]);

    return {
      poolId: poolAddress,
      token0,
      token1,
      reserve0: balance0.toString(),
      reserve1: balance1.toString(),
      fee: fee.toNumber(),
      tick: slot0.tick,
    };
  }

  // ============================================================================
  // Position Operations
  // ============================================================================

  /**
   * Get positions for address
   */
  async getPositions(owner: string): Promise<Position[]> {
    const manager = new Contract(this.config.positionManager, POSITION_MANAGER_ABI, this.provider);
    const count: BigNumber = await manager.balanceOf(owner);

    const positions: Position[] = [];
    for (let index = 0; index < count.toNumber(); index += 1) {
      const tokenId: BigNumber = await manager.tokenOfOwnerByIndex(owner, index);
      const raw = await manager.positions(tokenId);

      positions.push({
        tokenId: tokenId.toNumber(),
        owner,
        token0: raw.token0,
        token1: raw.token1,
        tickLower: raw.tickLower,
        tickUpper: raw.tickUpper,
        liquidity: raw.liquidity.toString(),
        feeGrowthInside0LastX128: raw.feeGrowthInside0LastX128.toString(),
        feeGrowthInside1LastX128: raw.feeGrowthInside1LastX128.toString(),
      });
    }

    return positions;
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
  ): Promise<ethers.ContractTransaction> {
    if (!this.wallet) {
      throw new Error('Wallet not configured');
    }

    const manager = new Contract(this.config.positionManager, POSITION_MANAGER_ABI, this.wallet);

    // The caller must have approved the manager for both tokens first.
    return manager.mint({
      token0,
      token1,
      fee,
      tickLower,
      tickUpper,
      amount0Desired,
      amount1Desired,
      amount0Min: 0,
      amount1Min: 0,
      recipient: this.wallet.address,
      deadline: Math.floor(Date.now() / 1000) + 1200,
    });
  }

  // ============================================================================
  // Utility
  // ============================================================================



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