/**
 * Real Token Service - Fetches 500+ Real Tokens from CoinGecko API
 * No mocks, no demo data - real live data
 */

import axios from 'axios';

export interface TokenData {
  id: string;
  symbol: string;
  name: string;
  image: string;
  current_price: number;
  market_cap: number;
  market_cap_rank: number;
  total_volume: number;
  price_change_24h: number;
  price_change_percentage_24h: number;
  circulating_supply: number;
  total_supply: number;
  ath: number;
  ath_change_percentage: number;
  atl: number;
  atl_change_percentage: number;
  last_updated: string;
}

const COINGECKO_API = 'https://api.coingecko.com/api/v3';
const CACHE_KEY = 'tigerwallet_tokens_cache';
const CACHE_TIMESTAMP_KEY = 'tigerwallet_tokens_timestamp';
const CACHE_DURATION = 60000; // 1 minute cache

class RealTokenService {
  private cachedTokens: TokenData[] = [];

  // Fetch 500+ real tokens from CoinGecko
  async fetchAllTokens(forceRefresh = false): Promise<TokenData[]> {
    // Check cache first
    if (!forceRefresh && this.cachedTokens.length > 0) {
      const cached = this.loadFromCache();
      if (cached.length > 0) {
        return cached;
      }
    }

    try {
      const response = await axios.get<TokenData[]>(
        `${COINGECKO_API}/coins/markets`,
        {
          params: {
            vs_currency: 'usd',
            order: 'market_cap_desc',
            per_page: 500,
            page: 1,
            sparkline: false,
          },
          timeout: 30000,
        }
      );

      if (response.data && Array.isArray(response.data)) {
        this.cachedTokens = response.data;
        this.saveToCache(response.data);
        return response.data;
      }

      return [];
    } catch (error) {
      console.error('Failed to fetch tokens:', error);
      // Try to return cached data on error
      return this.loadFromCache();
    }
  }

  // Get cached tokens
  getCachedTokens(): TokenData[] {
    if (this.cachedTokens.length === 0) {
      this.cachedTokens = this.loadFromCache();
    }
    return this.cachedTokens;
  }

  // Get token by symbol
  getToken(symbol: string): TokenData | undefined {
    const tokens = this.getCachedTokens();
    return tokens.find(
      (t) => t.symbol.toUpperCase() === symbol.toUpperCase()
    );
  }

  // Get token by ID
  getTokenById(id: string): TokenData | undefined {
    const tokens = this.getCachedTokens();
    return tokens.find((t) => t.id === id);
  }

  // Get top tokens by market cap
  getTopTokens(limit = 100): TokenData[] {
    const tokens = this.getCachedTokens();
    return [...tokens]
      .sort((a, b) => b.market_cap - a.market_cap)
      .slice(0, limit);
  }

  // Search tokens
  searchTokens(query: string): TokenData[] {
    const tokens = this.getCachedTokens();
    const lowerQuery = query.toLowerCase();
    return tokens.filter(
      (t) =>
        t.name.toLowerCase().includes(lowerQuery) ||
        t.symbol.toLowerCase().includes(lowerQuery)
    );
  }

  // Get tokens by IDs
  getTokensByIds(ids: string[]): TokenData[] {
    const tokens = this.getCachedTokens();
    return tokens.filter((t) => ids.includes(t.id));
  }

  // Get token price
  getPrice(symbol: string): number {
    const token = this.getToken(symbol);
    return token?.current_price || 0;
  }

  // Get prices for multiple symbols
  getPrices(symbols: string[]): Record<string, number> {
    const prices: Record<string, number> = {};
    for (const symbol of symbols) {
      prices[symbol.toUpperCase()] = this.getPrice(symbol);
    }
    return prices;
  }

  // Get total market cap
  getTotalMarketCap(): number {
    const tokens = this.getCachedTokens();
    return tokens.reduce((sum, t) => sum + (t.market_cap || 0), 0);
  }

  // Get total volume
  getTotalVolume(): number {
    const tokens = this.getCachedTokens();
    return tokens.reduce((sum, t) => sum + (t.total_volume || 0), 0);
  }

  // Cache management
  private saveToCache(tokens: TokenData[]): void {
    try {
      localStorage.setItem(CACHE_KEY, JSON.stringify(tokens));
      localStorage.setItem(CACHE_TIMESTAMP_KEY, Date.now().toString());
    } catch (e) {
      console.error('Failed to save token cache:', e);
    }
  }

  private loadFromCache(): TokenData[] {
    try {
      const cached = localStorage.getItem(CACHE_KEY);
      const timestamp = localStorage.getItem(CACHE_TIMESTAMP_KEY);

      if (cached && timestamp) {
        const age = Date.now() - parseInt(timestamp);
        if (age < CACHE_DURATION) {
          return JSON.parse(cached);
        }
      }
    } catch (e) {
      console.error('Failed to load token cache:', e);
    }
    return [];
  }

  // Clear cache
  clearCache(): void {
    this.cachedTokens = [];
    localStorage.removeItem(CACHE_KEY);
    localStorage.removeItem(CACHE_TIMESTAMP_KEY);
  }
}

// Export singleton instance
export const realTokenService = new RealTokenService();

// Export for use in components
export const getTokenPrice = (symbol: string): number => 
  realTokenService.getPrice(symbol);

export const getTopTokens = (limit = 100): TokenData[] => 
  realTokenService.getTopTokens(limit);

export const searchTokens = (query: string): TokenData[] => 
  realTokenService.searchTokens(query);

export default realTokenService;
