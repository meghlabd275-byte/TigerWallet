/**
 * TigerWallet - Copy Trading Component
 * Follow top traders and copy their strategies
 */

import React, { useState, useCallback, useEffect } from 'react';

type TraderStatus = 'active' | 'paused' | 'closed';
type TradeType = 'spot' | 'futures';

interface Trader {
  id: string;
  address: string;
  name: string;
  totalTrades: number;
  winRate: number;
  profitShare: number;
  aum: number;
  followers: number;
  status: TraderStatus;
  performance: { monthly: number };
}

interface CopyPosition {
  id: string;
  traderId: string;
  traderName: string;
  pair: string;
  side: 'long' | 'short';
  amount: number;
  entryPrice: number;
  currentPrice: number;
  pnl: number;
  pnlPercent: number;
  openedAt: string;
}

interface CopySettings {
  copyAmount: number;
  copyAmountType: 'fixed' | 'percentage';
  maxPositionSize: number;
  stopLoss?: number;
  takeProfit?: number;
  autoCopy: boolean;
  closeOnPause: boolean;
}

// Copy Trading Dashboard Component
export function CopyTradingDashboard() {
  const [activeTab, setActiveTab] = useState<'traders' | 'positions' | 'settings'>('traders');
  const [traders, setTraders] = useState<Trader[]>([]);
  const [positions, setPositions] = useState<CopyPosition[]>([]);
  const [settings, setSettings] = useState<CopySettings>({
    copyAmount: 100,
    copyAmountType: 'fixed',
    maxPositionSize: 1000,
    autoCopy: true,
    closeOnPause: true,
  });

  // Load traders
  useEffect(() => {
    // Mock data
    setTraders([
      {
        id: '1',
        address: '0x1234567890abcdef1234567890abcdef12345678',
        name: 'CryptoMaster',
        totalTrades: 1250,
        winRate: 72,
        profitShare: 10,
        aum: 2500000,
        followers: 5420,
        status: 'active',
        performance: { monthly: 15.5 },
      },
      {
        id: '2',
        address: '0xabcdef1234567890abcdef1234567890abcdef12',
        name: 'DeFiWhale',
        totalTrades: 890,
        winRate: 65,
        profitShare: 8,
        aum: 1800000,
        followers: 3200,
        status: 'active',
        performance: { monthly: 8.2 },
      },
    ]);
  }, []);

  const totalPnl = positions.reduce((sum, p) => sum + p.pnl, 0);

  return (
    <div className="max-w-6xl mx-auto p-6">
      <h1 className="text-3xl font-bold mb-6">Copy Trading</h1>

      {/* Stats */}
      <div className="grid grid-cols-4 gap-4 mb-6">
        <div className="bg-white rounded-lg shadow-md p-4">
          <p className="text-sm text-gray-500">Active Positions</p>
          <p className="text-2xl font-bold">{positions.length}</p>
        </div>
        <div className="bg-white rounded-lg shadow-md p-4">
          <p className="text-sm text-gray-500">Total P&L</p>
          <p className={`text-2xl font-bold ${totalPnl >= 0 ? 'text-green-600' : 'text-red-600'}`}>
            ${totalPnl.toLocaleString()}
          </p>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-4 mb-6 border-b">
        <button
          onClick={() => setActiveTab('traders')}
          className={`pb-3 px-4 font-medium ${
            activeTab === 'traders' 
              ? 'border-b-2 border-blue-600 text-blue-600' 
              : 'text-gray-500'
          }`}
        >
          Top Traders
        </button>
        <button
          onClick={() => setActiveTab('positions')}
          className={`pb-3 px-4 font-medium ${
            activeTab === 'positions' 
              ? 'border-b-2 border-blue-600 text-blue-600' 
              : 'text-gray-500'
          }`}
        >
          My Positions ({positions.length})
        </button>
        <button
          onClick={() => setActiveTab('settings')}
          className={`pb-3 px-4 font-medium ${
            activeTab === 'settings' 
              ? 'border-b-2 border-blue-600 text-blue-600' 
              : 'text-gray-500'
          }`}
        >
          Settings
        </button>
      </div>

      {/* Content */}
      {activeTab === 'traders' && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {traders.map(trader => (
            <div key={trader.id} className="bg-white rounded-lg shadow-md p-6">
              <div className="flex items-center gap-3 mb-4">
                <div className="w-12 h-12 rounded-full bg-gradient-to-br from-blue-500 to-purple-500 flex items-center justify-center text-white font-bold">
                  {trader.name.charAt(0)}
                </div>
                <div>
                  <h3 className="font-semibold">{trader.name}</h3>
                  <p className="text-sm text-gray-500 font-mono">
                    {trader.address.slice(0, 6)}...{trader.address.slice(-4)}
                  </p>
                </div>
              </div>

              <div className="grid grid-cols-3 gap-4 mb-4">
                <div className="text-center">
                  <p className="text-xs text-gray-500">Win Rate</p>
                  <p className="font-semibold text-green-600">{trader.winRate}%</p>
                </div>
                <div className="text-center">
                  <p className="text-xs text-gray-500">Trades</p>
                  <p className="font-semibold">{trader.totalTrades}</p>
                </div>
                <div className="text-center">
                  <p className="text-xs text-gray-500">Followers</p>
                  <p className="font-semibold">{trader.followers.toLocaleString()}</p>
                </div>
              </div>

              <button className="w-full py-2 px-4 rounded-lg font-medium bg-blue-600 text-white hover:bg-blue-700">
                Copy Trade
              </button>
            </div>
          ))}
        </div>
      )}

      {activeTab === 'positions' && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {positions.length === 0 ? (
            <div className="col-span-full text-center py-12 text-gray-500">
              No active positions. Start copying a trader!
            </div>
          ) : null}
        </div>
      )}

      {activeTab === 'settings' && (
        <div className="bg-white rounded-lg shadow-md p-6 max-w-lg">
          <h3 className="text-lg font-semibold mb-4">Copy Settings</h3>
          
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Copy Amount Type
              </label>
              <select
                value={settings.copyAmountType}
                onChange={(e) => setSettings({ ...settings, copyAmountType: e.target.value as 'fixed' | 'percentage' })}
                className="w-full px-3 py-2 border rounded-lg"
              >
                <option value="fixed">Fixed Amount</option>
                <option value="percentage">Percentage of Portfolio</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                {settings.copyAmountType === 'fixed' ? 'Copy Amount ($)' : 'Copy Percentage (%)'}
              </label>
              <input
                type="number"
                value={settings.copyAmount}
                onChange={(e) => setSettings({ ...settings, copyAmount: parseFloat(e.target.value) })}
                className="w-full px-3 py-2 border rounded-lg"
              />
            </div>

            <div className="flex items-center gap-3">
              <input
                type="checkbox"
                id="autoCopy"
                checked={settings.autoCopy}
                onChange={(e) => setSettings({ ...settings, autoCopy: e.target.checked })}
                className="w-4 h-4"
              />
              <label htmlFor="autoCopy" className="text-sm text-gray-700">
                Auto-copy new trades
              </label>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default CopyTradingDashboard;
