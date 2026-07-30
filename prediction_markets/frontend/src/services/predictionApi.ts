/**
 * TigerWallet Prediction Markets - Frontend React Application
 * Complete UI for trading prediction markets
 */

import React, { useState, useEffect, createContext, useContext, useCallback } from 'react';

// ============================================================================
// Types
// ============================================================================

export interface Outcome {
  outcome_id: number;
  name: string;
  price: number;
  volume: number;
  probability: number;
}

export interface Market {
  market_id: number;
  question: string;
  description: string;
  category: string;
  outcome_type: 'binary' | 'categorical' | 'scalar';
  outcomes: Outcome[];
  status: 'active' | 'paused' | 'resolving' | 'resolved' | 'cancelled';
  resolution_time: number;
  resolved_outcome?: number;
  volume_24h: number;
  total_volume: number;
  featured: boolean;
  image_url?: string;
  created_at: number;
  updated_at: number;
}

export interface Order {
  order_id: number;
  market_id: number;
  outcome_id: number;
  user_id: number;
  order_type: 'market' | 'limit' | 'stop_loss' | 'take_profit';
  side: 'buy' | 'sell';
  price: number;
  amount: number;
  filled_amount: number;
  status: 'pending' | 'partially_filled' | 'filled' | 'cancelled' | 'expired';
  timestamp: number;
  expires_at: number;
}

export interface Position {
  market_id: number;
  outcome_id: number;
  user_id: number;
  quantity: number;
  avg_price: number;
  invested: number;
  current_value: number;
  profit_loss: number;
}

export interface BetSlip {
  market_id: number;
  outcome_id: number;
  side: 'buy' | 'sell';
  price: number;
  amount: number;
  potential_winnings: number;
  fees: number;
  total_cost: number;
}

export interface Trade {
  trade_id: number;
  order_id: number;
  market_id: number;
  outcome_id: number;
  side: 'buy' | 'sell';
  price: number;
  amount: number;
  fees: number;
  timestamp: number;
  user_id: number;
  tx_hash?: string;
}

export interface MarketStats {
  total_markets: number;
  active_markets: number;
  total_volume: number;
  volume_24h: number;
  total_users: number;
  total_trades: number;
  avg_trade_size: number;
}

export interface APIResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
  };
}

// ============================================================================
// API Service
// ============================================================================

const API_BASE_URL = process.env.NEXT_PUBLIC_PREDICTION_API_URL || 'http://localhost:8443/api/v1';

class PredictionAPIService {
  private baseUrl: string;

  constructor(baseUrl: string = API_BASE_URL) {
    this.baseUrl = baseUrl;
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    const response = await fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
    });

    const data: APIResponse<T> = await response.json();

    if (!data.success || !data.data) {
      throw new Error(data.error?.message || 'API request failed');
    }

    return data.data;
  }

  async getMarkets(params: {
    status?: string;
    category?: string;
    featured?: boolean;
    offset?: number;
    limit?: number;
  } = {}): Promise<Market[]> {
    const queryParams = new URLSearchParams();
    if (params.status) queryParams.set('status', params.status);
    if (params.category) queryParams.set('category', params.category);
    if (params.featured !== undefined) queryParams.set('featured', String(params.featured));
    if (params.offset !== undefined) queryParams.set('offset', String(params.offset));
    if (params.limit !== undefined) queryParams.set('limit', String(params.limit));

    const query = queryParams.toString();
    return this.request<Market[]>(`/markets${query ? `?${query}` : ''}`);
  }

  async getMarket(marketId: number): Promise<Market> {
    return this.request<Market>(`/markets/${marketId}`);
  }

  async getFeaturedMarkets(): Promise<Market[]> {
    return this.request<Market[]>('/markets/featured');
  }

  async createMarket(market: {
    question: string;
    description: string;
    outcome_type: string;
    outcome_names: string[];
    resolution_time: number;
    category: string;
  }): Promise<Market> {
    return this.request<Market>('/markets', {
      method: 'POST',
      body: JSON.stringify(market),
    });
  }

  async placeOrder(order: {
    market_id: number;
    outcome_id: number;
    order_type: string;
    side: string;
    price: number;
    amount: number;
    expires_at?: number;
  }): Promise<Order> {
    return this.request<Order>('/orders', {
      method: 'POST',
      body: JSON.stringify(order),
    });
  }

  async cancelOrder(orderId: number): Promise<void> {
    await this.request<void>(`/orders/${orderId}`, {
      method: 'DELETE',
    });
  }

  async getPositions(): Promise<Position[]> {
    return this.request<Position[]>('/positions');
  }

  async getTrades(params: { page?: number; page_size?: number } = {}): Promise<{
    trades: Trade[];
    total: number;
    page: number;
    page_size: number;
  }> {
    const queryParams = new URLSearchParams();
    if (params.page !== undefined) queryParams.set('page', String(params.page));
    if (params.page_size !== undefined) queryParams.set('page_size', String(params.page_size));

    const query = queryParams.toString();
    return this.request<{
      trades: Trade[];
      total: number;
      page: number;
      page_size: number;
    }>(`/trades${query ? `?${query}` : ''}`);
  }

  async getBalance(): Promise<{ balance: number }> {
    return this.request<{ balance: number }>('/balance');
  }

  async addFunds(amount: number): Promise<void> {
    await this.request<void>('/balance/add', {
      method: 'POST',
      body: JSON.stringify({ amount }),
    });
  }

  async calculateBetSlip(params: {
    market_id: number;
    outcome_id: number;
    side: string;
    amount: number;
  }): Promise<BetSlip> {
    return this.request<BetSlip>('/slip', {
      method: 'POST',
      body: JSON.stringify(params),
    });
  }

  async getStats(): Promise<MarketStats> {
    return this.request<MarketStats>('/stats');
  }
}

export const apiService = new PredictionAPIService();

// ============================================================================
// Context
// ============================================================================

interface PredictionContextType {
  markets: Market[];
  featuredMarkets: Market[];
  selectedMarket: Market | null;
  positions: Position[];
  trades: Trade[];
  balance: number;
  stats: MarketStats | null;
  loading: boolean;
  error: string | null;
  theme: 'light' | 'dark';
  setTheme: (theme: 'light' | 'dark') => void;
  selectMarket: (market: Market | null) => void;
  refreshMarkets: () => Promise<void>;
  refreshPositions: () => Promise<void>;
  refreshTrades: () => Promise<void>;
  refreshBalance: () => Promise<void>;
  placeOrder: (order: {
    market_id: number;
    outcome_id: number;
    order_type: string;
    side: string;
    price: number;
    amount: number;
  }) => Promise<Order>;
  cancelOrder: (orderId: number) => Promise<void>;
  addFunds: (amount: number) => Promise<void>;
}

const PredictionContext = createContext<PredictionContextType | null>(null);

export function usePrediction() {
  const context = useContext(PredictionContext);
  if (!context) {
    throw new Error('usePrediction must be used within PredictionProvider');
  }
  return context;
}

// ============================================================================
// Provider
// ============================================================================

interface PredictionProviderProps {
  children: React.ReactNode;
}

export function PredictionProvider({ children }: PredictionProviderProps) {
  const [markets, setMarkets] = useState<Market[]>([]);
  const [featuredMarkets, setFeaturedMarkets] = useState<Market[]>([]);
  const [selectedMarket, setSelectedMarket] = useState<Market | null>(null);
  const [positions, setPositions] = useState<Position[]>([]);
  const [trades, setTrades] = useState<Trade[]>([]);
  const [balance, setBalance] = useState<number>(0);
  const [stats, setStats] = useState<MarketStats | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [theme, setTheme] = useState<'light' | 'dark'>('dark');

  const refreshMarkets = useCallback(async () => {
    try {
      setLoading(true);
      const [allMarkets, featured] = await Promise.all([
        apiService.getMarkets({ limit: 50 }),
        apiService.getFeaturedMarkets(),
      ]);
      setMarkets(allMarkets);
      setFeaturedMarkets(featured);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load markets');
    } finally {
      setLoading(false);
    }
  }, []);

  const refreshPositions = useCallback(async () => {
    try {
      const userPositions = await apiService.getPositions();
      setPositions(userPositions);
    } catch (err) {
      console.error('Failed to load positions:', err);
    }
  }, []);

  const refreshTrades = useCallback(async () => {
    try {
      const tradesData = await apiService.getTrades({ page: 0, page_size: 50 });
      setTrades(tradesData.trades);
    } catch (err) {
      console.error('Failed to load trades:', err);
    }
  }, []);

  const refreshBalance = useCallback(async () => {
    try {
      const balanceData = await apiService.getBalance();
      setBalance(balanceData.balance);
    } catch (err) {
      console.error('Failed to load balance:', err);
    }
  }, []);

  const selectMarket = useCallback((market: Market | null) => {
    setSelectedMarket(market);
  }, []);

  const placeOrder = useCallback(async (order: {
    market_id: number;
    outcome_id: number;
    order_type: string;
    side: string;
    price: number;
    amount: number;
  }) => {
    const newOrder = await apiService.placeOrder(order);
    await Promise.all([refreshPositions(), refreshBalance(), refreshTrades()]);
    return newOrder;
  }, [refreshPositions, refreshBalance, refreshTrades]);

  const cancelOrder = useCallback(async (orderId: number) => {
    await apiService.cancelOrder(orderId);
    await Promise.all([refreshPositions(), refreshBalance()]);
  }, [refreshPositions, refreshBalance]);

  const addFunds = useCallback(async (amount: number) => {
    await apiService.addFunds(amount);
    await refreshBalance();
  }, [refreshBalance]);

  useEffect(() => {
    const init = async () => {
      await Promise.all([
        refreshMarkets(),
        refreshPositions(),
        refreshTrades(),
        refreshBalance(),
      ]);
      try {
        const statsData = await apiService.getStats();
        setStats(statsData);
      } catch (err) {
        console.error('Failed to load stats:', err);
      }
    };
    init();
  }, [refreshMarkets, refreshPositions, refreshTrades, refreshBalance]);

  const value: PredictionContextType = {
    markets,
    featuredMarkets,
    selectedMarket,
    positions,
    trades,
    balance,
    stats,
    loading,
    error,
    theme,
    setTheme,
    selectMarket,
    refreshMarkets,
    refreshPositions,
    refreshTrades,
    refreshBalance,
    placeOrder,
    cancelOrder,
    addFunds,
  };

  return (
    <PredictionContext.Provider value={value}>
      <div className={`prediction-app ${theme}`} data-theme={theme}>
        {children}
      </div>
    </PredictionContext.Provider>
  );
}

// ============================================================================
// Utility Functions
// ============================================================================

export function formatPrice(price: number): string {
  return (price / 1000000).toFixed(2);
}

export function formatVolume(volume: number): string {
  if (volume >= 1000000000) {
    return `$${(volume / 1000000000).toFixed(2)}B`;
  }
  if (volume >= 1000000) {
    return `$${(volume / 1000000).toFixed(2)}M`;
  }
  if (volume >= 1000) {
    return `$${(volume / 1000).toFixed(2)}K`;
  }
  return `$${volume.toFixed(2)}`;
}

export function formatTimestamp(timestamp: number): string {
  return new Date(timestamp).toLocaleString();
}

export function formatPercentage(value: number): string {
  return `${(value / 10000).toFixed(2)}%`;
}

export function getOutcomeColor(price: number): string {
  const percentage = price / 1000000;
  if (percentage >= 0.7) return 'text-green-500';
  if (percentage >= 0.4) return 'text-yellow-500';
  return 'text-red-500';
}

// ============================================================================
// Components
// ============================================================================

export { PredictionMarkets } from './PredictionMarkets';
export { MarketCard } from './MarketCard';
export { MarketDetail } from './MarketDetail';
export { OrderPanel } from './OrderPanel';
export { PositionsList } from './PositionsList';
export { TradeHistory } from './TradeHistory';
export { StatsPanel } from './StatsPanel';
export { BetSlip } from './BetSlip';
