"use client";

import { useState, useCallback, useEffect } from "react";
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
  Zap
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

export default function PerpetualsPage() {
  const [selectedPair, setSelectedPair] = useState<PerpetualPair | null>(null);
  const [pairs, setPairs] = useState<PerpetualPair[]>([]);
  const [positions, setPositions] = useState<Position[]>([]);
  const [orders, setOrders] = useState<Order[]>([]);
  const [candles, setCandles] = useState<CandleData[]>([]);
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

  // Load mock data
  useEffect(() => {
    setIsLoading(true);
    setTimeout(() => {
      setPairs([
        {
          symbol: "BTC-PERP",
          name: "Bitcoin Perpetual",
          price: 64500,
          change24h: 2500,
          changePercent24h: 4.03,
          high24h: 65200,
          low24h: 61800,
          volume24h: 1250000000,
          openInterest: 2500000000,
          fundingRate: 0.01,
          nextFundingTime: Date.now() + 28800000,
          maxLeverage: 100,
          liquidationThreshold: 0.5,
        },
        {
          symbol: "ETH-PERP",
          name: "Ethereum Perpetual",
          price: 3520,
          change24h: 120,
          changePercent24h: 3.53,
          high24h: 3580,
          low24h: 3380,
          volume24h: 850000000,
          openInterest: 1200000000,
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
      ]);

      setSelectedPair({
        symbol: "BTC-PERP",
        name: "Bitcoin Perpetual",
        price: 64500,
        change24h: 2500,
        changePercent24h: 4.03,
        high24h: 65200,
        low24h: 61800,
        volume24h: 1250000000,
        openInterest: 2500000000,
        fundingRate: 0.01,
        nextFundingTime: Date.now() + 28800000,
        maxLeverage: 100,
        liquidationThreshold: 0.5,
      });

      setPositions([
        {
          id: "p1",
          pair: "BTC-PERP",
          direction: "long",
          size: 0.5,
          entryPrice: 63000,
          currentPrice: 64500,
          leverage: 5,
          margin: 6300,
          unrealizedPnL: 750,
          unrealizedPnLPercent: 11.9,
          liquidationPrice: 57600,
          openedAt: Date.now() - 86400000 * 2,
        },
      ]);

      setOrders([
        {
          id: "o1",
          pair: "BTC-PERP",
          type: "limit",
          direction: "long",
          size: 0.1,
          price: 62000,
          status: "pending",
          createdAt: Date.now() - 3600000,
        },
      ]);

      // Generate mock candles
      const mockCandles: CandleData[] = [];
      let price = 62000;
      for (let i = 0; i < 100; i++) {
        const change = (Math.random() - 0.5) * 1000;
        price += change;
        mockCandles.push({
          time: Date.now() - (100 - i) * 3600000,
          open: price - change / 2,
          high: price + Math.random() * 500,
          low: price - Math.random() * 500,
          close: price,
          volume: Math.random() * 1000000,
        });
      }
      setCandles(mockCandles);

      setIsLoading(false);
    }, 1000);
  }, []);

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
