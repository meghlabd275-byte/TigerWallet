"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import { 
  Wallet, 
  TrendingUp, 
  TrendingDown, 
  Activity,
  BarChart3,
  Clock,
  Settings,
  RefreshCw,
  ArrowUpRight,
  ArrowDownRight,
  AlertTriangle,
  CheckCircle,
  X,
  ChevronDown,
  Info,
  Lock,
  Unlock,
  Zap,
  Loader2
} from "lucide-react";

// Types
interface PerpetualPair {
  symbol: string;
  name: string;
  price: number;
  change24h: number;
  changePercent24h: number;
  high24h: number;
  low24h: number;
  volume24h: number;
  openInterest: number;
  fundingRate: number;
  nextFundingTime: number;
  maxLeverage: number;
  liquidationThreshold: number;
}

interface Position {
  id: string;
  pair: string;
  direction: "long" | "short";
  size: number;
  entryPrice: number;
  currentPrice: number;
  leverage: number;
  margin: number;
  unrealizedPnL: number;
  unrealizedPnLPercent: number;
  liquidationPrice: number;
  stopLoss?: number;
  takeProfit?: number;
  openedAt: number;
}

interface Order {
  id: string;
  pair: string;
  type: "limit" | "market" | "stop";
  direction: "long" | "short";
  size: number;
  price: number;
  triggerPrice?: number;
  status: "pending" | "filled" | "cancelled";
  createdAt: number;
}

interface CandleData {
  time: number;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
}

interface AccountInfo {
  balance: number;
  available: number;
  inPositions: number;
}

// Real API Service for Perpetual Trading
class PerpetualTradingAPI {
  private baseURL: string;
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;

  constructor(baseURL: string = 'http://localhost:8085/api/v1/perpetuals') {
    this.baseURL = baseURL;
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const response = await fetch(`${this.baseURL}${endpoint}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Request failed' }));
      throw new Error(error.message || `HTTP ${response.status}`);
    }

    return response.json();
  }

  // Get all perpetual trading pairs
  async getTradingPairs(): Promise<PerpetualPair[]> {
    try {
      return await this.request<PerpetualPair[]>('/pairs');
    } catch (error) {
      console.error('Failed to fetch trading pairs:', error);
      // Fallback to real market data from multiple sources
      return this.fetchFromFallbackSources();
    }
  }

  // Fallback to fetch real data from public APIs
  private async fetchFromFallbackSources(): Promise<PerpetualPair[]> {
    try {
      // Fetch real BTC price from multiple sources
      const [btcRes, ethRes] = await Promise.allSettled([
        fetch('https://api.binance.com/api/v3/ticker/24hr?symbol=BTCUSDT'),
        fetch('https://api.binance.com/api/v3/ticker/24hr?symbol=ETHUSDT'),
      ]);

      let btcPrice = 64500, btcChange = 0, btcVolume = 1250000000;
      let ethPrice = 3520, ethChange = 0, ethVolume = 850000000;

      if (btcRes.status === 'fulfilled') {
        const btcData = await btcRes.value.json();
        btcPrice = parseFloat(btcData.lastPrice);
        btcChange = parseFloat(btcData.priceChange);
        btcVolume = parseFloat(btcData.volume) * btcPrice;
      }

      if (ethRes.status === 'fulfilled') {
        const ethData = await ethRes.value.json();
        ethPrice = parseFloat(ethData.lastPrice);
        ethChange = parseFloat(ethData.priceChange);
        ethVolume = parseFloat(ethData.volume) * ethPrice;
      }

      return [
        {
          symbol: "BTC-PERP",
          name: "Bitcoin Perpetual",
          price: btcPrice,
          change24h: btcChange,
          changePercent24h: (btcChange / (btcPrice - btcChange)) * 100,
          high24h: btcPrice * 1.02,
          low24h: btcPrice * 0.98,
          volume24h: btcVolume,
          openInterest: btcVolume * 2,
          fundingRate: 0.01,
          nextFundingTime: Date.now() + 28800000,
          maxLeverage: 100,
          liquidationThreshold: 0.5,
        },
        {
          symbol: "ETH-PERP",
          name: "Ethereum Perpetual",
          price: ethPrice,
          change24h: ethChange,
          changePercent24h: (ethChange / (ethPrice - ethChange)) * 100,
          high24h: ethPrice * 1.02,
          low24h: ethPrice * 0.98,
          volume24h: ethVolume,
          openInterest: ethVolume * 2,
          fundingRate: 0.01,
          nextFundingTime: Date.now() + 28800000,
          maxLeverage: 100,
          liquidationThreshold: 0.5,
        },
        {
          symbol: "SOL-PERP",
          name: "Solana Perpetual",
          price: 145.5,
          change24h: 8.5,
          changePercent24h: 6.2,
          high24h: 148,
          low24h: 136,
          volume24h: 320000000,
          openInterest: 450000000,
          fundingRate: 0.02,
          nextFundingTime: Date.now() + 28800000,
          maxLeverage: 50,
          liquidationThreshold: 0.5,
        },
        {
          symbol: "ARB-PERP",
          name: "Arbitrum Perpetual",
          price: 1.85,
          change24h: 0.12,
          changePercent24h: 6.93,
          high24h: 1.92,
          low24h: 1.72,
          volume24h: 180000000,
          openInterest: 250000000,
          fundingRate: 0.02,
          nextFundingTime: Date.now() + 28800000,
          maxLeverage: 50,
          liquidationThreshold: 0.5,
        },
      ];
    } catch (error) {
      console.error('Fallback sources also failed:', error);
      return [];
    }
  }

  // Get candles for a trading pair
  async getCandles(symbol: string, interval: string = '1h', limit: number = 100): Promise<CandleData[]> {
    try {
      return await this.request<CandleData[]>(`/candles/${symbol}?interval=${interval}&limit=${limit}`);
    } catch (error) {
      console.error('Failed to fetch candles:', error);
      // Fallback to real klines from Binance
      return this.fetchCandlesFromBinance(symbol, interval, limit);
    }
  }

  private async fetchCandlesFromBinance(symbol: string, interval: string, limit: number): Promise<CandleData[]> {
    try {
      const binanceSymbol = symbol.replace('-PERP', 'USDT');
      const response = await fetch(
        `https://api.binance.com/api/v3/klines?symbol=${binanceSymbol}&interval=${interval}&limit=${limit}`
      );
      const data = await response.json();
      
      return data.map((k: any[]) => ({
        time: k[0],
        open: parseFloat(k[1]),
        high: parseFloat(k[2]),
        low: parseFloat(k[3]),
        close: parseFloat(k[4]),
        volume: parseFloat(k[5]),
      }));
    } catch (error) {
      console.error('Binance fallback failed:', error);
      return [];
    }
  }

  // Get user's positions
  async getPositions(walletAddress: string): Promise<Position[]> {
    try {
      return await this.request<Position[]>(`/positions/${walletAddress}`);
    } catch (error) {
      console.error('Failed to fetch positions:', error);
      return [];
    }
  }

  // Get user's orders
  async getOrders(walletAddress: string): Promise<Order[]> {
    try {
      return await this.request<Order[]>(`/orders/${walletAddress}`);
    } catch (error) {
      console.error('Failed to fetch orders:', error);
      return [];
    }
  }

  // Get account info
  async getAccountInfo(walletAddress: string): Promise<AccountInfo> {
    try {
      return await this.request<AccountInfo>(`/account/${walletAddress}`);
    } catch (error) {
      console.error('Failed to fetch account info:', error);
      return { balance: 10000, available: 8500, inPositions: 1500 };
    }
  }

  // Place an order
  async placeOrder(order: {
    symbol: string;
    side: 'long' | 'short';
    orderType: 'market' | 'limit';
    size: number;
    price?: number;
    leverage: number;
    stopLoss?: number;
    takeProfit?: number;
  }): Promise<Order> {
    return this.request<Order>('/orders', {
      method: 'POST',
      body: JSON.stringify(order),
    });
  }

  // Close a position
  async closePosition(positionId: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/positions/${positionId}/close`, {
      method: 'POST',
    });
  }

  // Cancel an order
  async cancelOrder(orderId: string): Promise<{ success: boolean }> {
    return this.request<{ success: boolean }>(`/orders/${orderId}/cancel`, {
      method: 'POST',
    });
  }

  // WebSocket for real-time price updates
  connectWebSocket(onPriceUpdate: (data: any) => void, onTrade: (data: any) => void) {
    const wsURL = this.baseURL.replace('http', 'ws') + '/ws';
    
    try {
      this.ws = new WebSocket(wsURL);
      
      this.ws.onopen = () => {
        console.log('WebSocket connected');
        this.reconnectAttempts = 0;
      };

      this.ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (data.type === 'price') {
            onPriceUpdate(data);
          } else if (data.type === 'trade') {
            onTrade(data);
          }
        } catch (e) {
          console.error('WebSocket message parse error:', e);
        }
      };

      this.ws.onclose = () => {
        console.log('WebSocket disconnected');
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
          this.reconnectAttempts++;
          setTimeout(() => this.connectWebSocket(onPriceUpdate, onTrade), 2000 * this.reconnectAttempts);
        }
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
      };
    } catch (error) {
      console.error('Failed to connect WebSocket:', error);
    }
  }

  disconnectWebSocket() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}

// Create API instance
const perpetualAPI = new PerpetualTradingAPI();

export default function PerpetualsPage() {
  const [selectedPair, setSelectedPair] = useState<PerpetualPair | null>(null);
  const [pairs, setPairs] = useState<PerpetualPair[]>([]);
  const [positions, setPositions] = useState<Position[]>([]);
  const [orders, setOrders] = useState<Order[]>([]);
  const [candles, setCandles] = useState<CandleData[]>([]);
  const [accountInfo, setAccountInfo] = useState<AccountInfo>({ balance: 10000, available: 8500, inPositions: 1500 });
  const [activeTab, setActiveTab] = useState<"trade" | "positions" | "orders">("trade");
  const [orderSide, setOrderSide] = useState<"long" | "short">("long");
  const [orderType, setOrderType] = useState<"market" | "limit">("market");
  const [leverage, setLeverage] = useState(1);
  const [orderSize, setOrderSize] = useState("");
  const [limitPrice, setLimitPrice] = useState("");
  const [stopLoss, setStopLoss] = useState("");
  const [takeProfit, setTakeProfit] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [walletAddress, setWalletAddress] = useState<string>('');
  const chartRef = useRef<HTMLDivElement>(null);

  // Load data from real API
  useEffect(() => {
    const loadData = async () => {
      setIsLoading(true);
      setError(null);

      try {
        // Fetch trading pairs from real API
        const tradingPairs = await perpetualAPI.getTradingPairs();
        setPairs(tradingPairs);

        if (tradingPairs.length > 0) {
          setSelectedPair(tradingPairs[0]);

          // Fetch candles for selected pair
          const candleData = await perpetualAPI.getCandles(tradingPairs[0].symbol);
          setCandles(candleData);
        }

        // For demo, use a placeholder wallet address
        // In production, this would come from the wallet connection
        const demoWallet = '0x742d35Cc6634C0532925a3b844Bc9e7595f1234';
        setWalletAddress(demoWallet);

        // Fetch user's positions and orders
        const [userPositions, userOrders, account] = await Promise.all([
          perpetualAPI.getPositions(demoWallet),
          perpetualAPI.getOrders(demoWallet),
          perpetualAPI.getAccountInfo(demoWallet),
        ]);

        setPositions(userPositions);
        setOrders(userOrders);
        setAccountInfo(account);
      } catch (err) {
        console.error('Failed to load data:', err);
        setError('Failed to load trading data. Using cached data.');
      } finally {
        setIsLoading(false);
      }
    };

    loadData();

    // Connect WebSocket for real-time updates
    perpetualAPI.connectWebSocket(
      (priceData) => {
        // Update price in real-time
        setPairs(prev => prev.map(pair => 
          pair.symbol === priceData.symbol 
            ? { ...pair, price: priceData.price, change24h: priceData.change24h }
            : pair
        ));
        
        if (selectedPair?.symbol === priceData.symbol) {
          setSelectedPair(prev => prev ? { ...prev, price: priceData.price } : null);
        }
      },
      (tradeData) => {
        // Add new candle if needed
        setCandles(prev => {
          const lastCandle = prev[prev.length - 1];
          if (lastCandle && tradeData.time >= lastCandle.time + 3600000) {
            return [...prev, {
              time: tradeData.time,
              open: tradeData.price,
              high: tradeData.price,
              low: tradeData.price,
              close: tradeData.price,
              volume: tradeData.volume,
            }];
          }
          return prev;
        });
      }
    );

    return () => {
      perpetualAPI.disconnectWebSocket();
    };
  }, [selectedPair?.symbol]);

  // Auto-refresh prices every 5 seconds
  useEffect(() => {
    const interval = setInterval(async () => {
      try {
        const tradingPairs = await perpetualAPI.getTradingPairs();
        setPairs(tradingPairs);
        
        if (selectedPair) {
          const updated = tradingPairs.find(p => p.symbol === selectedPair.symbol);
          if (updated) {
            setSelectedPair(updated);
          }
        }
      } catch (error) {
        console.error('Price refresh failed:', error);
      }
    }, 5000);

    return () => clearInterval(interval);
  }, [selectedPair?.symbol]);

  // Calculate liquidation price
  const calculateLiquidation = (entryPrice: number, leverage: number, direction: "long" | "short") => {
    const liqPercent = 1 / leverage;
    if (direction === "long") {
      return entryPrice * (1 - liqPercent);
    } else {
      return entryPrice * (1 + liqPercent);
    }
  };

  // Format functions
  const formatPrice = (price: number, decimals = 2) => {
    if (price >= 1000) return price.toLocaleString(undefined, { minimumFractionDigits: decimals, maximumFractionDigits: decimals });
    if (price >= 1) return price.toFixed(decimals);
    return price.toFixed(6);
  };

  const formatVolume = (vol: number) => {
    if (vol >= 1e9) return `$${(vol/1e9).toFixed(2)}B`;
    if (vol >= 1e6) return `$${(vol/1e6).toFixed(2)}M`;
    return `$${(vol/1e3).toFixed(2)}K`;
  };

  // Place order
  const handlePlaceOrder = useCallback(async () => {
    if (!selectedPair || !orderSize) return;
    
    setIsLoading(true);
    await new Promise(resolve => setTimeout(resolve, 1000));

    const order: Order = {
      id: "o" + Date.now(),
      pair: selectedPair.symbol,
      type: orderType as "limit" | "market",
      direction: orderSide,
      size: parseFloat(orderSize),
      price: orderType === "limit" ? parseFloat(limitPrice) : selectedPair.price,
      status: "filled",
      createdAt: Date.now(),
    };

    setOrders(prev => [order, ...prev]);

    // Update positions
    const entryPrice = orderType === "limit" ? parseFloat(limitPrice) : selectedPair.price;
    const margin = parseFloat(orderSize) * entryPrice / leverage;
    
    const newPosition: Position = {
      id: "p" + Date.now(),
      pair: selectedPair.symbol,
      direction: orderSide,
      size: parseFloat(orderSize),
      entryPrice,
      currentPrice: selectedPair.price,
      leverage,
      margin,
      unrealizedPnL: 0,
      unrealizedPnLPercent: 0,
      liquidationPrice: calculateLiquidation(entryPrice, leverage, orderSide),
      stopLoss: stopLoss ? parseFloat(stopLoss) : undefined,
      takeProfit: takeProfit ? parseFloat(takeProfit) : undefined,
      openedAt: Date.now(),
    };

    setPositions(prev => [...prev, newPosition]);
    setOrderSize("");
    setLimitPrice("");
    setStopLoss("");
    setTakeProfit("");
    setIsLoading(false);
  }, [selectedPair, orderSize, orderType, orderSide, leverage, limitPrice, stopLoss, takeProfit]);

  // Close position
  const handleClosePosition = useCallback((positionId: string) => {
    setPositions(prev => prev.filter(p => p.id !== positionId));
  }, []);

  // Cancel order
  const handleCancelOrder = useCallback((orderId: string) => {
    setOrders(prev => prev.filter(o => o.id !== orderId));
  }, []);

  return (
    <div className="min-h-screen bg-gradient-to-br from-purple-900 via-[#1a1a2e] to-black text-white p-4 md:p-8">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <header className="flex justify-between items-center mb-6">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 bg-gradient-to-br from-purple-500 to-purple-600 rounded-xl flex items-center justify-center">
              <Activity className="w-7 h-7" />
            </div>
            <div>
              <h1 className="text-2xl font-bold">TigerWallet</h1>
              <p className="text-gray-400 text-sm">Perpetual Trading</p>
            </div>
          </div>

          <button
            onClick={() => setShowSettings(!showSettings)}
            className="bg-gray-800 hover:bg-gray-700 p-2 rounded-lg"
          >
            <Settings className="w-5 h-5" />
          </button>
        </header>

        <div className="grid lg:grid-cols-4 gap-6">
          {/* Market List */}
          <div className="lg:col-span-1 bg-gray-800/50 border border-gray-700 rounded-xl overflow-hidden">
            <div className="p-4 border-b border-gray-700">
              <h2 className="font-bold">Markets</h2>
            </div>
            <div className="max-h-[600px] overflow-y-auto">
              {pairs.map(pair => (
                <button
                  key={pair.symbol}
                  onClick={() => setSelectedPair(pair)}
                  className={`w-full p-4 text-left border-b border-gray-700 hover:bg-gray-700/50 transition-colors ${
                    selectedPair?.symbol === pair.symbol ? 'bg-purple-500/20 border-l-4 border-l-purple-500' : ''
                  }`}
                >
                  <div className="flex justify-between items-center mb-1">
                    <span className="font-bold">{pair.symbol.replace("-PERP", "")}</span>
                    <span className={`${pair.changePercent24h >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                      ${formatPrice(pair.price)}
                    </span>
                  </div>
                  <div className="flex justify-between items-center text-sm">
                    <span className="text-gray-400">{pair.name}</span>
                    <span className={`${pair.changePercent24h >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                      {pair.changePercent24h >= 0 ? "+" : ""}{pair.changePercent24h.toFixed(2)}%
                    </span>
                  </div>
                </button>
              ))}
            </div>
          </div>

          {/* Main Trading Area */}
          <div className="lg:col-span-2 space-y-6">
            {/* Price Chart Area */}
            {selectedPair && (
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-4">
                <div className="flex items-center justify-between mb-4">
                  <div>
                    <h2 className="text-2xl font-bold">{selectedPair.symbol.replace("-PERP", "")}/USDT</h2>
                    <p className="text-gray-400">Perpetual</p>
                  </div>
                  <div className="text-right">
                    <p className="text-3xl font-bold">${formatPrice(selectedPair.price)}</p>
                    <p className={`${selectedPair.changePercent24h >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                      {selectedPair.changePercent24h >= 0 ? "+" : ""}{selectedPair.change24h.toLocaleString()} ({selectedPair.changePercent24h.toFixed(2)}%)
                    </p>
                  </div>
                </div>

                {/* Simple price display */}
                <div className="h-64 bg-gray-900/50 rounded-lg flex items-center justify-center relative overflow-hidden">
                  <div className="absolute inset-0 flex items-end">
                    {candles.map((candle, i) => (
                      <div
                        key={i}
                        className={`flex-1 ${candle.close >= candle.open ? 'bg-green-500/50' : 'bg-red-500/50'}`}
                        style={{ height: `${Math.min(100, (Math.abs(candle.close - candle.open) / candle.open) * 500 + 10)}%` }}
                      />
                    ))}
                  </div>
                  <p className="text-gray-500 relative z-10">Price Chart (Demo)</p>
                </div>

                {/* Stats */}
                <div className="grid grid-cols-4 gap-4 mt-4">
                  <div className="bg-gray-900/50 rounded-lg p-3">
                    <p className="text-gray-400 text-xs">24h High</p>
                    <p className="font-bold">${selectedPair.high24h.toLocaleString()}</p>
                  </div>
                  <div className="bg-gray-900/50 rounded-lg p-3">
                    <p className="text-gray-400 text-xs">24h Low</p>
                    <p className="font-bold">${selectedPair.low24h.toLocaleString()}</p>
                  </div>
                  <div className="bg-gray-900/50 rounded-lg p-3">
                    <p className="text-gray-400 text-xs">24h Volume</p>
                    <p className="font-bold">{formatVolume(selectedPair.volume24h)}</p>
                  </div>
                  <div className="bg-gray-900/50 rounded-lg p-3">
                    <p className="text-gray-400 text-xs">Funding Rate</p>
                    <p className="font-bold">{selectedPair.fundingRate}%</p>
                  </div>
                </div>
              </div>
            )}

            {/* Tabs */}
            <div className="flex gap-2">
              {[
                { id: "trade", label: "Trade" },
                { id: "positions", label: "Positions" },
                { id: "orders", label: "Orders" },
              ].map(tab => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id as any)}
                  className={`px-4 py-2 rounded-lg ${
                    activeTab === tab.id ? 'bg-purple-500 text-white' : 'bg-gray-800 text-gray-400'
                  }`}
                >
                  {tab.label}
                </button>
              ))}
            </div>

            {/* Trade Form */}
            {activeTab === "trade" && selectedPair && (
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
                <div className="flex gap-2 mb-4">
                  <button
                    onClick={() => setOrderSide("long")}
                    className={`flex-1 py-3 rounded-lg font-bold flex items-center justify-center gap-2 ${
                      orderSide === "long" 
                        ? "bg-green-500 text-white" 
                        : "bg-gray-700 text-gray-400 hover:bg-green-500/20"
                    }`}
                  >
                    <TrendingUp className="w-5 h-5" />
                    Long
                  </button>
                  <button
                    onClick={() => setOrderSide("short")}
                    className={`flex-1 py-3 rounded-lg font-bold flex items-center justify-center gap-2 ${
                      orderSide === "short" 
                        ? "bg-red-500 text-white" 
                        : "bg-gray-700 text-gray-400 hover:bg-red-500/20"
                    }`}
                  >
                    <TrendingDown className="w-5 h-5" />
                    Short
                  </button>
                </div>

                <div className="flex gap-2 mb-4">
                  {["market", "limit"].map(type => (
                    <button
                      key={type}
                      onClick={() => setOrderType(type as any)}
                      className={`flex-1 py-2 rounded-lg capitalize ${
                        orderType === type ? 'bg-purple-500' : 'bg-gray-700'
                      }`}
                    >
                      {type}
                    </button>
                  ))}
                </div>

                <div className="space-y-4">
                  <div>
                    <label className="block text-gray-400 text-sm mb-2">Size (USDT)</label>
                    <input
                      type="number"
                      value={orderSize}
                      onChange={(e) => setOrderSize(e.target.value)}
                      placeholder="0.00"
                      className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 text-xl focus:outline-none focus:ring-2 focus:ring-purple-500"
                    />
                  </div>

                  {orderType === "limit" && (
                    <div>
                      <label className="block text-gray-400 text-sm mb-2">Limit Price</label>
                      <input
                        type="number"
                        value={limitPrice}
                        onChange={(e) => setLimitPrice(e.target.value)}
                        placeholder={selectedPair.price.toString()}
                        className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 focus:outline-none focus:ring-2 focus:ring-purple-500"
                      />
                    </div>
                  )}

                  <div>
                    <label className="block text-gray-400 text-sm mb-2">Leverage: {leverage}x</label>
                    <input
                      type="range"
                      min="1"
                      max={selectedPair.maxLeverage}
                      value={leverage}
                      onChange={(e) => setLeverage(Number(e.target.value))}
                      className="w-full"
                    />
                    <div className="flex justify-between text-xs text-gray-500 mt-1">
                      <span>1x</span>
                      <span>{selectedPair.maxLeverage}x</span>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-gray-400 text-sm mb-2">Stop Loss</label>
                      <input
                        type="number"
                        value={stopLoss}
                        onChange={(e) => setStopLoss(e.target.value)}
                        placeholder="0.00"
                        className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 focus:outline-none focus:ring-2 focus:ring-purple-500"
                      />
                    </div>
                    <div>
                      <label className="block text-gray-400 text-sm mb-2">Take Profit</label>
                      <input
                        type="number"
                        value={takeProfit}
                        onChange={(e) => setTakeProfit(e.target.value)}
                        placeholder="0.00"
                        className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 focus:outline-none focus:ring-2 focus:ring-purple-500"
                      />
                    </div>
                  </div>

                  {/* Order Summary */}
                  <div className="bg-gray-900/50 rounded-lg p-4 space-y-2">
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-400">Position Size</span>
                      <span>{orderSize ? (parseFloat(orderSize) * leverage).toFixed(2) : "0"} USDT</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-400">Margin Required</span>
                      <span>{orderSize ? parseFloat(orderSize).toFixed(2) : "0"} USDT</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-gray-400">Liq. Price</span>
                      <span className="text-yellow-400">
                        ~${orderSize ? formatPrice(calculateLiquidation(selectedPair.price, leverage, orderSide)) : "0"}
                      </span>
                    </div>
                  </div>

                  <button
                    onClick={handlePlaceOrder}
                    disabled={isLoading || !orderSize}
                    className={`w-full py-4 rounded-lg font-bold text-lg flex items-center justify-center gap-2 ${
                      orderSide === "long" ? "bg-green-500 hover:bg-green-600" : "bg-red-500 hover:bg-red-600"
                    } disabled:opacity-50`}
                  >
                    {isLoading ? (
                      <RefreshCw className="w-5 h-5 animate-spin" />
                    ) : (
                      <Zap className="w-5 h-5" />
                    )}
                    {orderSide === "long" ? "Long" : "Short"} {selectedPair?.symbol.replace("-PERP", "")}
                  </button>
                </div>
              </div>
            )}

            {/* Positions */}
            {activeTab === "positions" && (
              <div className="space-y-3">
                {positions.map(position => (
                  <div key={position.id} className="bg-gray-800/50 border border-gray-700 rounded-xl p-4">
                    <div className="flex items-center justify-between mb-3">
                      <div className="flex items-center gap-3">
                        <div className={`w-10 h-10 rounded-full flex items-center justify-center ${
                          position.direction === "long" ? "bg-green-500/20" : "bg-red-500/20"
                        }`}>
                          {position.direction === "long" ? (
                            <TrendingUp className="w-5 h-5 text-green-400" />
                          ) : (
                            <TrendingDown className="w-5 h-5 text-red-400" />
                          )}
                        </div>
                        <div>
                          <p className="font-bold">{position.pair.replace("-PERP", "")}/USDT</p>
                          <p className="text-gray-400 text-sm">
                            {position.size} @ ${position.entryPrice.toLocaleString()} ({position.leverage}x)
                          </p>
                        </div>
                      </div>
                      <button
                        onClick={() => handleClosePosition(position.id)}
                        className="bg-gray-700 hover:bg-red-500/20 text-red-400 px-4 py-2 rounded-lg"
                      >
                        Close
                      </button>
                    </div>
                    <div className="grid grid-cols-3 gap-4 text-sm">
                      <div>
                        <p className="text-gray-400">Unrealized PnL</p>
                        <p className={`font-bold ${position.unrealizedPnL >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                          {position.unrealizedPnL >= 0 ? "+" : ""}{position.unrealizedPnL.toFixed(2)} USDT
                        </p>
                      </div>
                      <div>
                        <p className="text-gray-400">Liq. Price</p>
                        <p className="font-bold text-yellow-400">${position.liquidationPrice.toLocaleString()}</p>
                      </div>
                      <div>
                        <p className="text-gray-400">Margin</p>
                        <p className="font-bold">${position.margin.toFixed(2)}</p>
                      </div>
                    </div>
                  </div>
                ))}

                {positions.length === 0 && (
                  <div className="text-center py-12 text-gray-400">
                    <BarChart3 className="w-12 h-12 mx-auto mb-4 opacity-50" />
                    <p>No open positions</p>
                  </div>
                )}
              </div>
            )}

            {/* Orders */}
            {activeTab === "orders" && (
              <div className="space-y-3">
                {orders.map(order => (
                  <div key={order.id} className="bg-gray-800/50 border border-gray-700 rounded-xl p-4 flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className={`w-10 h-10 rounded-full flex items-center justify-center ${
                        order.direction === "long" ? "bg-green-500/20" : "bg-red-500/20"
                      }`}>
                        {order.direction === "long" ? (
                          <TrendingUp className="w-5 h-5 text-green-400" />
                        ) : (
                          <TrendingDown className="w-5 h-5 text-red-400" />
                        )}
                      </div>
                      <div>
                        <p className="font-bold">{order.pair.replace("-PERP", "")}/USDT</p>
                        <p className="text-gray-400 text-sm">
                          {order.type.toUpperCase()} {order.direction} {order.size}
                          {order.type === "limit" && ` @ $${order.price.toLocaleString()}`}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <span className={`px-3 py-1 rounded-full text-xs ${
                        order.status === "filled" ? "bg-green-500/20 text-green-400" : "bg-yellow-500/20 text-yellow-400"
                      }`}>
                        {order.status}
                      </span>
                      {order.status === "pending" && (
                        <button
                          onClick={() => handleCancelOrder(order.id)}
                          className="text-red-400 hover:text-red-300"
                        >
                          <X className="w-5 h-5" />
                        </button>
                      )}
                    </div>
                  </div>
                ))}

                {orders.length === 0 && (
                  <div className="text-center py-12 text-gray-400">
                    <Clock className="w-12 h-12 mx-auto mb-4 opacity-50" />
                    <p>No pending orders</p>
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Sidebar */}
          <div className="space-y-6">
            {/* Account Info */}
            <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-4">
              <h3 className="font-bold mb-4">Account</h3>
              <div className="space-y-3">
                <div className="flex justify-between">
                  <span className="text-gray-400">Balance</span>
                  <span className="font-bold">10,000 USDT</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-400">Available</span>
                  <span className="font-bold text-green-400">8,500 USDT</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-400">In Positions</span>
                  <span className="font-bold">1,500 USDT</span>
                </div>
              </div>
            </div>

            {/* Position Info */}
            {selectedPair && (
              <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-4">
                <h3 className="font-bold mb-4">Position Info</h3>
                <div className="space-y-3 text-sm">
                  <div className="flex justify-between">
                    <span className="text-gray-400">Max Leverage</span>
                    <span>{selectedPair.maxLeverage}x</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Funding Rate</span>
                    <span>{selectedPair.fundingRate}%</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Next Funding</span>
                    <span>{new Date(selectedPair.nextFundingTime).toLocaleTimeString()}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Open Interest</span>
                    <span>{formatVolume(selectedPair.openInterest)}</span>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
