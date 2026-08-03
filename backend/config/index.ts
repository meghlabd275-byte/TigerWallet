// TigerWallet Backend - Main Configuration
// Production-ready backend configuration

export const config = {
  // Server
  port: process.env.PORT || 3000,
  nodeEnv: process.env.NODE_ENV || 'development',
  
  // Database
  database: {
    host: process.env.DB_HOST || 'localhost',
    port: parseInt(process.env.DB_PORT || '5432'),
    name: process.env.DB_NAME || 'tigerwallet',
    user: process.env.DB_USER || 'postgres',
    password: process.env.DB_PASSWORD || '',
    ssl: process.env.DB_SSL === 'true',
  },
  
  // Redis (for caching & sessions)
  redis: {
    host: process.env.REDIS_HOST || 'localhost',
    port: parseInt(process.env.REDIS_PORT || '6379'),
    password: process.env.REDIS_PASSWORD || '',
  },
  
  // JWT
  jwt: {
    secret: process.env.JWT_SECRET || '',
    expiresIn: '7d',
    refreshExpiresIn: '30d',
  },
  
  // External APIs
  apis: {
    // Blockchain nodes
    ethereum: {
      rpcUrl: process.env.ETHEREUM_RPC_URL || '',
      chainId: 1,
    },
    bsc: {
      rpcUrl: process.env.BSC_RPC_URL || '',
      chainId: 56,
    },
    polygon: {
      rpcUrl: process.env.POLYGON_RPC_URL || '',
      chainId: 137,
    },
    
    // Price feeds
    coingecko: process.env.COINGECKO_API_KEY || '',
    coinmarketcap: process.env.COINMARKETCAP_API_KEY || '',
    
    // Payment providers
    stripe: process.env.STRIPE_SECRET_KEY || '',
    paypal: process.env.PAYPAL_CLIENT_ID || '',
    
    // KYC
    sumsub: process.env.SUMSUB_SECRET_KEY || '',
  },
  
  // Security
  security: {
    rateLimit: {
      windowMs: 15 * 60 * 1000, // 15 minutes
      maxRequests: 100,
    },
    corsOrigins: process.env.CORS_ORIGINS?.split(',') || ['https://tigerwallet.com'],
    helmet: true,
    hpp: true,
  },
  
  // Feature flags
  features: {
    p2pTrading: true,
    marginTrading: process.env.ENABLE_MARGIN === 'true',
    futures: process.env.ENABLE_FUTURES === 'true',
    fiatRamp: process.env.ENABLE_FIAT_RAMP === 'true',
    merchantSystem: process.env.ENABLE_MERCHANT === 'true',
  },
  
  // Merchant collateral requirements (in USD)
  merchant: {
    levels: {
      NEWBIE: { collateral: 100, feeDiscount: 0 },
      BRONZE: { collateral: 250, feeDiscount: 5 },
      SILVER: { collateral: 500, feeDiscount: 10 },
      GOLD: { collateral: 1000, feeDiscount: 15 },
      PLATINUM: { collateral: 2500, feeDiscount: 20 },
      DIAMOND: { collateral: 5000, feeDiscount: 30 },
    },
    securityDepositPercent: {
      buyer: 2, // 2% of trade value
      seller: 3, // 3% of trade value
    },
    protectionFund: 5000000, // $5M
  },
  
  // Fee structure
  fees: {
    p2p: {
      maker: 0.001, // 0.1%
      taker: 0.001, // 0.1%
    },
    margin: {
      maker: 0.0002, // 0.02%
      taker: 0.0004, // 0.04%
    },
    futures: {
      maker: 0.0002,
      taker: 0.0004,
    },
    convert: 0.001, // 0.1%
  },
};

export default config;
