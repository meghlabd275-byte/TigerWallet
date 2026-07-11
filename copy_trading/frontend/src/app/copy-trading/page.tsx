"use client";

import { useState, useCallback, useEffect } from "react";
import { 
  Wallet, 
  TrendingUp, 
  TrendingDown, 
  Users, 
  Copy,
  DollarSign,
  Activity,
  ArrowUpRight,
  ArrowDownRight,
  Star,
  Shield,
  BarChart3,
  Settings,
  RefreshCw,
  Play,
  Pause,
  X,
  CheckCircle,
  AlertCircle,
  Loader2,
  Search,
  Filter,
  Plus,
  Minus
} from "lucide-react";

// Types
interface Trader {
  id: string;
  name: string;
  avatar: string;
  winRate: number;
  totalTrades: number;
  profitShare: number;
  followers: number;
  totalPnL: number;
  monthlyPnL: number;
  riskScore: number;
  specialties: string[];
  isVerified: boolean;
  isPro: boolean;
}

interface CopiedPosition {
  id: string;
  traderId: string;
  traderName: string;
  pair: string;
  direction: "long" | "short";
  entryPrice: number;
  currentPrice: number;
  size: number;
  pnl: number;
  pnlPercent: number;
  openedAt: number;
  status: "open" | "closed";
}

interface Position {
  id: string;
  traderId: string;
  traderName: string;
  pair: string;
  direction: "long" | "short";
  entryPrice: number;
  amount: number;
  leverage: number;
  currentPrice: number;
  liquidationPrice: number;
  margin: number;
  pnl: number;
  pnlPercent: number;
  stopLoss?: number;
  takeProfit?: number;
  openedAt: number;
}

interface PerformanceData {
  date: string;
  pnl: number;
}

export default function CopyTradingPage() {
  const [activeTab, setActiveTab] = useState<"discover" | "my-copies" | "leaderboard">("discover");
  const [traders, setTraders] = useState<Trader[]>([]);
  const [copiedPositions, setCopiedPositions] = useState<CopiedPosition[]>([]);
  const [myPositions, setMyPositions] = useState<Position[]>([]);
  const [searchQuery, setSearchQuery] = useState("");
  const [filterRisk, setFilterRisk] = useState<"all" | "low" | "medium" | "high">("all");
  const [sortBy, setSortBy] = useState<"pnl" | "winrate" | "followers">("pnl");
  const [isLoading, setIsLoading] = useState(false);
  const [selectedTrader, setSelectedTrader] = useState<Trader | null>(null);
  const [copyModal, setCopyModal] = useState<{open: boolean; trader: Trader | null}>({open: false, trader: null});
  const [copySettings, setCopySettings] = useState({
    amount: 100,
    leverage: 1,
    stopLoss: 0,
    takeProfit: 0,
    autoCopy: true
  });

  // Load mock data
  useEffect(() => {
    setIsLoading(true);
    setTimeout(() => {
      setTraders([
        {
          id: "1",
          name: "CryptoWhale",
          avatar: "🐋",
          winRate: 78.5,
          totalTrades: 1250,
          profitShare: 10,
          followers: 15420,
          totalPnL: 2450000,
          monthlyPnL: 125000,
          riskScore: 3.2,
          specialties: ["BTC", "ETH", "SOL"],
          isVerified: true,
          isPro: true
        },
        {
          id: "2",
          name: "DeFiMaster",
          avatar: "🎓",
          winRate: 72.3,
          totalTrades: 890,
          profitShare: 12,
          followers: 8920,
          totalPnL: 1890000,
          monthlyPnL: 98000,
          riskScore: 4.1,
          specialties: ["ARB", "OP", "MATIC"],
          isVerified: true,
          isPro: true
        },
        {
          id: "3",
          name: "AltSeason",
          avatar: "🚀",
          winRate: 65.8,
          totalTrades: 2100,
          profitShare: 15,
          followers: 22500,
          totalPnL: 3200000,
          monthlyPnL: 210000,
          riskScore: 6.5,
          specialties: ["PEPE", "WIF", "BONK"],
          isVerified: false,
          isPro: true
        },
        {
          id: "4",
          name: "StableTrader",
          avatar: "📊",
          winRate: 85.2,
          totalTrades: 3400,
          profitShare: 8,
          followers: 5600,
          totalPnL: 980000,
          monthlyPnL: 45000,
          riskScore: 1.8,
          specialties: ["USDC", "USDT", "DAI"],
          isVerified: true,
          isPro: false
        },
        {
          id: "5",
          name: "MomentumKing",
          avatar: "👑",
          winRate: 69.4,
          totalTrades: 1560,
          profitShare: 12,
          followers: 12300,
          totalPnL: 2100000,
          monthlyPnL: 156000,
          riskScore: 5.2,
          specialties: ["BTC", "ETH", "SOL"],
          isVerified: true,
          isPro: true
        },
      ]);

      setCopiedPositions([
        {
          id: "p1",
          traderId: "1",
          traderName: "CryptoWhale",
          pair: "BTC/USDT",
          direction: "long",
          entryPrice: 62000,
          currentPrice: 64500,
          size: 0.5,
          pnl: 1250,
          pnlPercent: 4.03,
          openedAt: Date.now() - 86400000 * 3,
          status: "open"
        },
        {
          id: "p2",
          traderId: "2",
          traderName: "DeFiMaster",
          pair: "ETH/USDT",
          direction: "long",
          entryPrice: 3400,
          currentPrice: 3520,
          size: 2,
          pnl: 240,
          pnlPercent: 3.53,
          openedAt: Date.now() - 86400000,
          status: "open"
        }
      ]);

      setMyPositions([
        {
          id: "mp1",
          traderId: "self",
          traderName: "My Position",
          pair: "BTC/USDT",
          direction: "long",
          entryPrice: 63000,
          amount: 0.1,
          leverage: 5,
          currentPrice: 64500,
          liquidationPrice: 57600,
          margin: 630,
          pnl: 150,
          pnlPercent: 2.38,
          stopLoss: 60000,
          takeProfit: 70000,
          openedAt: Date.now() - 86400000 * 2
        }
      ]);

      setIsLoading(false);
    }, 1000);
  }, []);

  // Copy trader
  const handleCopy = useCallback(async () => {
    if (!copyModal.trader) return;
    setIsLoading(true);
    
    // Simulate API call
    await new Promise(resolve => setTimeout(resolve, 1500));
    
    const newPosition: CopiedPosition = {
      id: "p" + Date.now(),
      traderId: copyModal.trader.id,
      traderName: copyModal.trader.name,
      pair: "BTC/USDT",
      direction: "long",
      entryPrice: 64500,
      size: copySettings.amount / 64500,
      pnl: 0,
      pnlPercent: 0,
      openedAt: Date.now(),
      status: "open"
    };

    setCopiedPositions(prev => [newPosition, ...prev]);
    setCopyModal({open: false, trader: null});
    setIsLoading(false);
  }, [copyModal.trader, copySettings]);

  // Stop copying
  const handleStopCopy = useCallback((traderId: string) => {
    setCopiedPositions(prev => prev.map(p => 
      p.traderId === traderId ? {...p, status: "closed"} : p
    ));
  }, []);

  // Format numbers
  const formatCurrency = (num: number) => {
    if (num >= 1000000) return `$${(num/1000000).toFixed(2)}M`;
    if (num >= 1000) return `$${(num/1000).toFixed(1)}K`;
    return `$${num.toFixed(2)}`;
  };

  const formatPercent = (num: number) => {
    return `${num >= 0 ? "+" : ""}${num.toFixed(2)}%`;
  };

  // Filter traders
  const filteredTraders = traders
    .filter(t => t.name.toLowerCase().includes(searchQuery.toLowerCase()))
    .filter(t => filterRisk === "all" || 
      (filterRisk === "low" && t.riskScore < 3) ||
      (filterRisk === "medium" && t.riskScore >= 3 && t.riskScore < 5) ||
      (filterRisk === "high" && t.riskScore >= 5))
    .sort((a, b) => {
      if (sortBy === "pnl") return b.totalPnL - a.totalPnL;
      if (sortBy === "winrate") return b.winRate - a.winRate;
      return b.followers - a.followers;
    });

  // Stats
  const totalPnL = copiedPositions.reduce((sum, p) => sum + p.pnl, 0);
  const totalInvested = copiedPositions.reduce((sum, p) => p.size * p.entryPrice, 0);

  return (
    <div className="min-h-screen bg-gradient-to-br from-tiger-dark via-[#1a1a2e] to-black text-white p-4 md:p-8">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <header className="flex flex-col md:flex-row justify-between items-center mb-8 gap-4">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 bg-gradient-to-br from-green-500 to-green-600 rounded-xl flex items-center justify-center">
              <Copy className="w-7 h-7" />
            </div>
            <div>
              <h1 className="text-2xl font-bold">TigerWallet</h1>
              <p className="text-gray-400 text-sm">Copy Trading</p>
            </div>
          </div>

          <div className="flex items-center gap-4">
            <div className="bg-gray-800/50 border border-gray-700 rounded-lg px-4 py-2">
              <p className="text-gray-400 text-xs">Total PnL</p>
              <p className={`font-bold ${totalPnL >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                {formatCurrency(totalPnL)}
              </p>
            </div>
            <div className="bg-gray-800/50 border border-gray-700 rounded-lg px-4 py-2">
              <p className="text-gray-400 text-xs">Active Copies</p>
              <p className="font-bold">{copiedPositions.filter(p => p.status === "open").length}</p>
            </div>
          </div>
        </header>

        {/* Tabs */}
        <div className="flex gap-2 mb-6 overflow-x-auto">
          {[
            { id: "discover", label: "Discover Traders", icon: Search },
            { id: "my-copies", label: "My Copies", icon: Copy },
            { id: "leaderboard", label: "Leaderboard", icon: BarChart3 },
          ].map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as any)}
              className={`flex items-center gap-2 px-4 py-2 rounded-lg whitespace-nowrap ${
                activeTab === tab.id 
                  ? "bg-green-500 text-white" 
                  : "bg-gray-800 text-gray-400 hover:text-white"
              }`}
            >
              <tab.icon className="w-4 h-4" />
              {tab.label}
            </button>
          ))}
        </div>

        {/* Discover Tab */}
        {activeTab === "discover" && (
          <div className="space-y-6">
            {/* Filters */}
            <div className="flex flex-wrap gap-4">
              <div className="flex-1 min-w-[200px]">
                <div className="relative">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                  <input
                    type="text"
                    placeholder="Search traders..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="w-full bg-gray-800 border border-gray-700 rounded-lg pl-10 pr-4 py-2 focus:outline-none focus:ring-2 focus:ring-green-500"
                  />
                </div>
              </div>
              <select
                value={filterRisk}
                onChange={(e) => setFilterRisk(e.target.value as any)}
                className="bg-gray-800 border border-gray-700 rounded-lg px-4 py-2"
              >
                <option value="all">All Risk Levels</option>
                <option value="low">Low Risk</option>
                <option value="medium">Medium Risk</option>
                <option value="high">High Risk</option>
              </select>
              <select
                value={sortBy}
                onChange={(e) => setSortBy(e.target.value as any)}
                className="bg-gray-800 border border-gray-700 rounded-lg px-4 py-2"
              >
                <option value="pnl">Sort by PnL</option>
                <option value="winrate">Sort by Win Rate</option>
                <option value="followers">Sort by Followers</option>
              </select>
            </div>

            {/* Trader Cards */}
            <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-4">
              {filteredTraders.map(trader => (
                <div 
                  key={trader.id} 
                  className="bg-gray-800/50 border border-gray-700 rounded-xl p-5 hover:border-gray-600 transition-colors"
                >
                  <div className="flex items-start justify-between mb-4">
                    <div className="flex items-center gap-3">
                      <div className="w-12 h-12 bg-gray-700 rounded-full flex items-center justify-center text-2xl">
                        {trader.avatar}
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <h3 className="font-bold">{trader.name}</h3>
                          {trader.isVerified && <CheckCircle className="w-4 h-4 text-green-500" />}
                          {trader.isPro && <Star className="w-4 h-4 text-yellow-500" />}
                        </div>
                        <p className="text-gray-400 text-sm">{trader.specialties.join(", ")}</p>
                      </div>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-3 mb-4">
                    <div className="bg-gray-900/50 rounded-lg p-3">
                      <p className="text-gray-400 text-xs">Win Rate</p>
                      <p className="font-bold text-green-400">{trader.winRate}%</p>
                    </div>
                    <div className="bg-gray-900/50 rounded-lg p-3">
                      <p className="text-gray-400 text-xs">Risk Score</p>
                      <p className={`font-bold ${trader.riskScore < 3 ? 'text-green-400' : trader.riskScore < 5 ? 'text-yellow-400' : 'text-red-400'}`}>
                        {trader.riskScore}/10
                      </p>
                    </div>
                    <div className="bg-gray-900/50 rounded-lg p-3">
                      <p className="text-gray-400 text-xs">Total PnL</p>
                      <p className="font-bold">{formatCurrency(trader.totalPnL)}</p>
                    </div>
                    <div className="bg-gray-900/50 rounded-lg p-3">
                      <p className="text-gray-400 text-xs">Followers</p>
                      <p className="font-bold">{trader.followers.toLocaleString()}</p>
                    </div>
                  </div>

                  <button
                    onClick={() => setCopyModal({open: true, trader})}
                    className="w-full bg-green-500 hover:bg-green-600 py-2 rounded-lg font-medium flex items-center justify-center gap-2"
                  >
                    <Copy className="w-4 h-4" />
                    Copy Trade
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* My Copies Tab */}
        {activeTab === "my-copies" && (
          <div className="space-y-6">
            {/* Active Positions */}
            <div>
              <h2 className="text-xl font-bold mb-4">Active Positions</h2>
              <div className="space-y-3">
                {copiedPositions.filter(p => p.status === "open").map(position => (
                  <div 
                    key={position.id}
                    className="bg-gray-800/50 border border-gray-700 rounded-xl p-4"
                  >
                    <div className="flex items-center justify-between">
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
                          <p className="font-bold">{position.pair}</p>
                          <p className="text-gray-400 text-sm">Copied from {position.traderName}</p>
                        </div>
                      </div>
                      <div className="text-right">
                        <p className={`font-bold ${position.pnl >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                          {formatCurrency(position.pnl)}
                        </p>
                        <p className={`text-sm ${position.pnlPercent >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                          {formatPercent(position.pnlPercent)}
                        </p>
                      </div>
                      <button
                        onClick={() => handleStopCopy(position.traderId)}
                        className="bg-red-500/20 hover:bg-red-500/30 text-red-400 px-4 py-2 rounded-lg"
                      >
                        Stop
                      </button>
                    </div>
                  </div>
                ))}

                {copiedPositions.filter(p => p.status === "open").length === 0 && (
                  <div className="text-center py-12 text-gray-400">
                    <Copy className="w-12 h-12 mx-auto mb-4 opacity-50" />
                    <p>No active copied positions</p>
                    <p className="text-sm">Start copying traders from the Discover tab</p>
                  </div>
                )}
              </div>
            </div>

            {/* My Own Positions */}
            <div>
              <h2 className="text-xl font-bold mb-4">My Positions</h2>
              <div className="space-y-3">
                {myPositions.map(position => (
                  <div 
                    key={position.id}
                    className="bg-gray-800/50 border border-gray-700 rounded-xl p-4"
                  >
                    <div className="flex items-center justify-between flex-wrap gap-4">
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
                          <div className="flex items-center gap-2">
                            <p className="font-bold">{position.pair}</p>
                            <span className="bg-gray-700 px-2 py-0.5 rounded text-xs">{position.leverage}x</span>
                          </div>
                          <p className="text-gray-400 text-sm">
                            Entry: ${position.entryPrice.toLocaleString()} | Current: ${position.currentPrice.toLocaleString()}
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center gap-6">
                        <div className="text-right">
                          <p className="text-gray-400 text-xs">PnL</p>
                          <p className={`font-bold ${position.pnl >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                            {formatCurrency(position.pnl)}
                          </p>
                        </div>
                        <div className="text-right">
                          <p className="text-gray-400 text-xs">Liq. Price</p>
                          <p className="font-bold text-yellow-400">${position.liquidationPrice.toLocaleString()}</p>
                        </div>
                        <button className="bg-gray-700 hover:bg-gray-600 px-4 py-2 rounded-lg">
                          Close
                        </button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}

        {/* Leaderboard Tab */}
        {activeTab === "leaderboard" && (
          <div className="space-y-6">
            <div className="bg-gray-800/50 border border-gray-700 rounded-xl overflow-hidden">
              <table className="w-full">
                <thead className="bg-gray-900/50">
                  <tr>
                    <th className="px-4 py-3 text-left text-gray-400">Rank</th>
                    <th className="px-4 py-3 text-left text-gray-400">Trader</th>
                    <th className="px-4 py-3 text-right text-gray-400">Win Rate</th>
                    <th className="px-4 py-3 text-right text-gray-400">Total Trades</th>
                    <th className="px-4 py-3 text-right text-gray-400">Monthly PnL</th>
                    <th className="px-4 py-3 text-right text-gray-400">Total PnL</th>
                    <th className="px-4 py-3 text-right text-gray-400">Followers</th>
                    <th className="px-4 py-3 text-right text-gray-400">Action</th>
                  </tr>
                </thead>
                <tbody>
                  {traders.sort((a, b) => b.totalPnL - a.totalPnL).map((trader, index) => (
                    <tr key={trader.id} className="border-t border-gray-700 hover:bg-gray-800/50">
                      <td className="px-4 py-3">
                        <span className={`font-bold ${index === 0 ? 'text-yellow-400' : index === 1 ? 'text-gray-400' : index === 2 ? 'text-orange-400' : ''}`}>
                          #{index + 1}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          <span className="text-xl">{trader.avatar}</span>
                          <span className="font-bold">{trader.name}</span>
                          {trader.isVerified && <CheckCircle className="w-4 h-4 text-green-500" />}
                        </div>
                      </td>
                      <td className="px-4 py-3 text-right text-green-400">{trader.winRate}%</td>
                      <td className="px-4 py-3 text-right">{trader.totalTrades.toLocaleString()}</td>
                      <td className="px-4 py-3 text-right text-green-400">{formatCurrency(trader.monthlyPnL)}</td>
                      <td className="px-4 py-3 text-right font-bold">{formatCurrency(trader.totalPnL)}</td>
                      <td className="px-4 py-3 text-right">{trader.followers.toLocaleString()}</td>
                      <td className="px-4 py-3 text-right">
                        <button
                          onClick={() => setCopyModal({open: true, trader})}
                          className="bg-green-500 hover:bg-green-600 px-3 py-1 rounded text-sm"
                        >
                          Copy
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Copy Modal */}
        {copyModal.open && copyModal.trader && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
            <div className="bg-gray-800 border border-gray-700 rounded-xl p-6 max-w-md w-full">
              <div className="flex items-center justify-between mb-6">
                <h3 className="text-xl font-bold">Copy {copyModal.trader.name}</h3>
                <button onClick={() => setCopyModal({open: false, trader: null})} className="text-gray-400 hover:text-white">
                  <X className="w-5 h-5" />
                </button>
              </div>

              <div className="space-y-4">
                <div>
                  <label className="block text-gray-400 text-sm mb-2">Copy Amount (USDT)</label>
                  <input
                    type="number"
                    value={copySettings.amount}
                    onChange={(e) => setCopySettings({...copySettings, amount: Number(e.target.value)})}
                    className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 focus:outline-none focus:ring-2 focus:ring-green-500"
                  />
                </div>

                <div>
                  <label className="block text-gray-400 text-sm mb-2">Leverage</label>
                  <div className="flex gap-2">
                    {[1, 2, 5, 10].map(lev => (
                      <button
                        key={lev}
                        onClick={() => setCopySettings({...copySettings, leverage: lev})}
                        className={`flex-1 py-2 rounded-lg ${copySettings.leverage === lev ? 'bg-green-500' : 'bg-gray-700'}`}
                      >
                        {lev}x
                      </button>
                    ))}
                  </div>
                </div>

                <div className="flex gap-4">
                  <div className="flex-1">
                    <label className="block text-gray-400 text-sm mb-2">Stop Loss (%)</label>
                    <input
                      type="number"
                      value={copySettings.stopLoss}
                      onChange={(e) => setCopySettings({...copySettings, stopLoss: Number(e.target.value)})}
                      className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 focus:outline-none focus:ring-2 focus:ring-green-500"
                    />
                  </div>
                  <div className="flex-1">
                    <label className="block text-gray-400 text-sm mb-2">Take Profit (%)</label>
                    <input
                      type="number"
                      value={copySettings.takeProfit}
                      onChange={(e) => setCopySettings({...copySettings, takeProfit: Number(e.target.value)})}
                      className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-3 focus:outline-none focus:ring-2 focus:ring-green-500"
                    />
                  </div>
                </div>

                <label className="flex items-center gap-3 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={copySettings.autoCopy}
                    onChange={(e) => setCopySettings({...copySettings, autoCopy: e.target.checked})}
                    className="w-5 h-5 rounded bg-gray-700 border-gray-600"
                  />
                  <span>Auto-copy new trades</span>
                </label>

                <div className="bg-gray-900/50 rounded-lg p-4">
                  <div className="flex justify-between text-sm mb-2">
                    <span className="text-gray-400">Trader's Profit Share</span>
                    <span>{copyModal.trader.profitShare}%</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-gray-400">Est. Position Size</span>
                    <span>{(copySettings.amount * copySettings.leverage).toLocaleString()} USDT</span>
                  </div>
                </div>

                <button
                  onClick={handleCopy}
                  disabled={isLoading}
                  className="w-full bg-green-500 hover:bg-green-600 py-3 rounded-lg font-bold flex items-center justify-center gap-2"
                >
                  {isLoading ? <Loader2 className="w-5 h-5 animate-spin" /> : <Copy className="w-5 h-5" />}
                  Start Copying
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
