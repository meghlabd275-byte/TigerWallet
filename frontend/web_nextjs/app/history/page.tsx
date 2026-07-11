'use client';

import React, { useState, useEffect, useCallback } from 'react';

interface Transaction {
  id: string;
  hash: string;
  from: string;
  to: string;
  amount: string;
  token: string;
  tokenAddress: string;
  chainId: number;
  chainName: string;
  status: 'pending' | 'confirmed' | 'failed';
  timestamp: number;
  type: 'send' | 'receive' | 'swap' | 'approve' | 'contract' | 'stake' | 'unstake' | 'claim' | 'bridge' | 'nft_transfer';
  fee: string;
  blockNumber: number;
  gasUsed?: string;
  gasPrice?: string;
  nonce?: number;
  metadata?: Record<string, string>;
}

interface FilterOptions {
  type: string[];
  status: string[];
  chainId: number | null;
  dateRange: { start: number; end: number };
  searchQuery: string;
}

interface PriceAlert {
  id: string;
  symbol: string;
  targetPrice: number;
  condition: 'above' | 'below';
  currentPrice: number;
  isActive: boolean;
  triggered: boolean;
  createdAt: number;
}

const MOCK_TRANSACTIONS: Transaction[] = [
  { id: 'tx_1', hash: '0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E1234567890abcdef1234567890', from: '0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E', to: '0x1234567890abcdef1234567890abcdef12345678', amount: '1.5', token: 'ETH', tokenAddress: '0x0000000000000000000000000000000000000000', chainId: 1, chainName: 'Ethereum', status: 'confirmed', timestamp: Date.now() - 300000, type: 'send', fee: '0.005', blockNumber: 18500000, gasUsed: '21000', gasPrice: '0.00000002', nonce: 15 },
  { id: 'tx_2', hash: '0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890', from: '0xabcdef1234567890abcdef1234567890abcdef12', to: '0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E', amount: '5000', token: 'USDT', tokenAddress: '0xdAC17F958D2ee523a2206206994597C13D831ec7', chainId: 1, chainName: 'Ethereum', status: 'confirmed', timestamp: Date.now() - 3600000, type: 'receive', fee: '0.003', blockNumber: 18499500, gasUsed: '65000', gasPrice: '0.000000015', nonce: 14 },
  { id: 'tx_3', hash: '0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef', from: '0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E', to: 'Uniswap V3', amount: '2.0', token: 'ETH', tokenAddress: '0x0000000000000000000000000000000000000000', chainId: 1, chainName: 'Ethereum', status: 'confirmed', timestamp: Date.now() - 86400000, type: 'swap', fee: '0.015', blockNumber: 18490000, gasUsed: '150000', gasPrice: '0.00000002', nonce: 13, metadata: { outputToken: 'USDC', outputAmount: '7000' } },
  { id: 'tx_4', hash: '0x9876543210fedcba9876543210fedcba9876543210fedcba9876543210fedc', from: '0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E', to: '0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E', amount: '32.5', token: 'MATIC', tokenAddress: '0x0000000000000000000000000000000000000000', chainId: 137, chainName: 'Polygon', status: 'confirmed', timestamp: Date.now() - 172800000, type: 'send', fee: '0.001', blockNumber: 45000000, gasUsed: '21000', nonce: 12 },
  { id: 'tx_5', hash: '0xfedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210', from: '0xabcd1234efgh5678ijkl9012mnop3456qrst7890', to: '0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E', amount: '0.5', token: 'BTC', tokenAddress: '', chainId: 0, chainName: 'Bitcoin', status: 'confirmed', timestamp: Date.now() - 259200000, type: 'receive', fee: '0.0001', blockNumber: 850000 },
  { id: 'tx_6', hash: '0x5678901234abcd5678901234abcd5678901234abcd5678901234abcd5678901234', from: '0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E', to: 'Staking Contract', amount: '10', token: 'ETH', tokenAddress: '0x0000000000000000000000000000000000000000', chainId: 1, chainName: 'Ethereum', status: 'confirmed', timestamp: Date.now() - 604800000, type: 'stake', fee: '0.008', blockNumber: 18450000, nonce: 10, metadata: { validator: 'Lido', reward: '4.2% APY' } },
  { id: 'tx_7', hash: '0xabcd5678901234efgh5678901234ijkl5678901234mnop5678901234qrst5678901234', from: '0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E', to: '0xdef4567890abcdef4567890abcdef4567890abcdef', amount: '1000', token: 'USDC', tokenAddress: '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48', chainId: 56, chainName: 'BNB Chain', status: 'failed', timestamp: Date.now() - 7200000, type: 'send', fee: '0.0005', blockNumber: 32000000, gasUsed: '21000', nonce: 5 },
  { id: 'tx_8', hash: '0xefgh9012345678ijkl9012345678mnop9012345678qrst9012345678uvwx9012345678', from: '0xBridge', to: '0x742d35Cc6634C0532925a3b844Bc9e7595f0eB1E', amount: '250', token: 'BNB', tokenAddress: '', chainId: 56, chainName: 'BNB Chain', status: 'confirmed', timestamp: Date.now() - 432000000, type: 'bridge', fee: '0.002', blockNumber: 31500000, metadata: { sourceChain: 'Ethereum', destinationChain: 'BNB Chain' } },
];

const MOCK_ALERTS: PriceAlert[] = [
  { id: 'alert_1', symbol: 'ETH', targetPrice: 4000, condition: 'above', currentPrice: 3500, isActive: true, triggered: false, createdAt: Date.now() - 86400000 },
  { id: 'alert_2', symbol: 'BTC', targetPrice: 70000, condition: 'above', currentPrice: 65000, isActive: true, triggered: false, createdAt: Date.now() - 172800000 },
  { id: 'alert_3', symbol: 'SOL', targetPrice: 100, condition: 'below', currentPrice: 150, isActive: true, triggered: false, createdAt: Date.now() - 259200000 },
];

export default function TransactionHistory() {
  const [transactions, setTransactions] = useState<Transaction[]>(MOCK_TRANSACTIONS);
  const [alerts, setAlerts] = useState<PriceAlert[]>(MOCK_ALERTS);
  const [filters, setFilters] = useState<FilterOptions>({ type: [], status: [], chainId: null, dateRange: { start: 0, end: Date.now() }, searchQuery: '' });
  const [activeTab, setActiveTab] = useState<'transactions' | 'alerts'>('transactions');
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const filteredTransactions = useCallback(() => {
    return transactions.filter(tx => {
      if (filters.type.length > 0 && !filters.type.includes(tx.type)) return false;
      if (filters.status.length > 0 && !filters.status.includes(tx.status)) return false;
      if (filters.chainId && tx.chainId !== filters.chainId) return false;
      if (tx.timestamp < filters.dateRange.start || tx.timestamp > filters.dateRange.end) return false;
      if (filters.searchQuery) {
        const query = filters.searchQuery.toLowerCase();
        const matchesHash = tx.hash.toLowerCase().includes(query);
        const matchesFrom = tx.from.toLowerCase().includes(query);
        const matchesTo = tx.to.toLowerCase().includes(query);
        const matchesToken = tx.token.toLowerCase().includes(query);
        if (!matchesHash && !matchesFrom && !matchesTo && !matchesToken) return false;
      }
      return true;
    });
  }, [transactions, filters]);

  const handleAddAlert = async (symbol: string, targetPrice: number, condition: 'above' | 'below') => {
    setLoading(true);
    await new Promise(resolve => setTimeout(resolve, 500));
    const prices: Record<string, number> = { 'ETH': 3500, 'BTC': 65000, 'SOL': 150, 'BNB': 600, 'MATIC': 0.8, 'USDT': 1, 'USDC': 1 };
    const newAlert: PriceAlert = { id: `alert_${Date.now()}`, symbol, targetPrice, condition, currentPrice: prices[symbol] || 0, isActive: true, triggered: false, createdAt: Date.now() };
    setAlerts(prev => [...prev, newAlert]);
    setMessage({ type: 'success', text: `Price alert set for ${symbol} ${condition} $${targetPrice}` });
    setLoading(false);
  };

  const handleToggleAlert = (alertId: string) => setAlerts(prev => prev.map(a => a.id === alertId ? { ...a, isActive: !a.isActive } : a));
  const handleDeleteAlert = (alertId: string) => { setAlerts(prev => prev.filter(a => a.id !== alertId)); setMessage({ type: 'success', text: 'Alert deleted' }); };

  const formatAddress = (addr: string): string => addr.length <= 12 ? addr : addr.slice(0, 6) + '...' + addr.slice(-4);
  const formatTime = (timestamp: number): string => { const diff = Date.now() - timestamp; const minutes = Math.floor(diff / 60000); const hours = Math.floor(diff / 3600000); const days = Math.floor(diff / 86400000); if (minutes < 1) return 'Just now'; if (minutes < 60) return `${minutes}m ago`; if (hours < 24) return `${hours}h ago`; if (days < 7) return `${days}d ago`; return new Date(timestamp).toLocaleDateString(); };
  const formatCurrency = (value: number): string => new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(value);
  const getCurrentPrice = (symbol: string): number => ({ 'ETH': 3500, 'BTC': 65000, 'SOL': 150, 'BNB': 600, 'MATIC': 0.8 }[symbol] || 0);
  const getTypeIcon = (type: string): string => ({ send: '📤', receive: '📥', swap: '🔄', approve: '✅', contract: '📝', stake: '🎯', unStake: '🎯', claim: '🎁', bridge: '🌉', nft_transfer: '🖼️' }[type] || '💰');
  const getStatusColor = (status: string): string => ({ confirmed: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200', pending: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200', failed: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200' }[status] || 'bg-slate-100 text-slate-800');

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-slate-50">
      <header className="bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-4"><a href="/" className="text-2xl">🐯</a><h1 className="text-xl font-bold">Transaction History</h1></div>
            <nav className="flex gap-4"><a href="/wallet" className="text-slate-600 dark:text-slate-400 hover:text-orange-500">Wallet</a><a href="/portfolio" className="text-slate-600 dark:text-slate-400 hover:text-orange-500">Portfolio</a></nav>
          </div>
        </div>
      </header>
      {message && <div className="fixed top-20 right-4 z-50"><div className={`px-6 py-3 rounded-lg shadow-lg ${message.type === 'success' ? 'bg-green-500' : 'bg-red-500'} text-white`}>{message.text}</div></div>}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="flex border-b border-slate-200 dark:border-slate-700 mb-6">
          <button onClick={() => setActiveTab('transactions')} className={`px-6 py-3 ${activeTab === 'transactions' ? 'border-b-2 border-orange-500 text-orange-500' : 'text-slate-500 dark:text-slate-400'}`}>Transactions ({filteredTransactions().length})</button>
          <button onClick={() => setActiveTab('alerts')} className={`px-6 py-3 ${activeTab === 'alerts' ? 'border-b-2 border-orange-500 text-orange-500' : 'text-slate-500 dark:text-slate-400'}`}>Price Alerts ({alerts.length})</button>
        </div>
        {activeTab === 'transactions' && (
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4 mb-6">
            <div className="flex flex-wrap items-center gap-4">
              <div className="flex-1 min-w-[200px]"><input type="text" placeholder="Search by address, hash, or token..." value={filters.searchQuery} onChange={(e) => setFilters(prev => ({ ...prev, searchQuery: e.target.value }))} className="w-full bg-slate-100 dark:bg-slate-700 border-0 rounded-lg px-4 py-2" /></div>
              <select value={filters.chainId || ''} onChange={(e) => setFilters(prev => ({ ...prev, chainId: e.target.value ? Number(e.target.value) : null }))} className="bg-slate-100 dark:bg-slate-700 border-0 rounded-lg px-4 py-2"><option value="">All Chains</option><option value="1">Ethereum</option><option value="56">BNB Chain</option><option value="137">Polygon</option></select>
              <select value={filters.type.length > 0 ? filters.type[0] : ''} onChange={(e) => setFilters(prev => ({ ...prev, type: e.target.value ? [e.target.value] : [] }))} className="bg-slate-100 dark:bg-slate-700 border-0 rounded-lg px-4 py-2"><option value="">All Types</option><option value="send">Send</option><option value="receive">Receive</option><option value="swap">Swap</option><option value="stake">Stake</option><option value="bridge">Bridge</option></select>
              <select value={filters.status.length > 0 ? filters.status[0] : ''} onChange={(e) => setFilters(prev => ({ ...prev, status: e.target.value ? [e.target.value] : [] }))} className="bg-slate-100 dark:bg-slate-700 border-0 rounded-lg px-4 py-2"><option value="">All Status</option><option value="confirmed">Confirmed</option><option value="pending">Pending</option><option value="failed">Failed</option></select>
            </div>
          </div>
        )}
        {activeTab === 'alerts' && (
          <div className="bg-white dark:bg-slate-800 rounded-lg p-6 mb-6">
            <h3 className="font-semibold mb-4">Create New Alert</h3>
            <div className="flex flex-wrap gap-4">
              <select id="alertSymbol" className="bg-slate-100 dark:bg-slate-700 border-0 rounded-lg px-4 py-2"><option value="ETH">ETH</option><option value="BTC">BTC</option><option value="SOL">SOL</option><option value="BNB">BNB</option></select>
              <select id="alertCondition" className="bg-slate-100 dark:bg-slate-700 border-0 rounded-lg px-4 py-2"><option value="above">Goes Above</option><option value="below">Goes Below</option></select>
              <input type="number" id="alertPrice" placeholder="Target Price ($)" className="bg-slate-100 dark:bg-slate-700 border-0 rounded-lg px-4 py-2 w-48" />
              <button onClick={() => { const symbol = (document.getElementById('alertSymbol') as HTMLSelectElement).value; const condition = (document.getElementById('alertCondition') as HTMLSelectElement).value as 'above' | 'below'; const targetPrice = parseFloat((document.getElementById('alertPrice') as HTMLInputElement).value); if (targetPrice > 0) handleAddAlert(symbol, targetPrice, condition); }} disabled={loading} className="bg-orange-500 hover:bg-orange-600 disabled:bg-slate-400 text-white px-6 py-2 rounded-lg">Create Alert</button>
            </div>
          </div>
        )}
        {activeTab === 'transactions' && (
          <div className="space-y-4">
            {filteredTransactions().length === 0 ? <div className="bg-white dark:bg-slate-800 rounded-lg p-12 text-center"><div className="text-6xl mb-4">📋</div><h3 className="text-xl font-semibold mb-2">No Transactions Found</h3></div> : filteredTransactions().map((tx) => (
              <div key={tx.id} className="bg-white dark:bg-slate-800 rounded-lg p-4 shadow-sm">
                <div className="flex items-start justify-between">
                  <div className="flex items-start gap-4"><div className="text-2xl">{getTypeIcon(tx.type)}</div><div><div className="flex items-center gap-2"><span className="font-semibold capitalize">{tx.type}</span><span className="text-slate-500 dark:text-slate-400">{tx.token}</span></div><div className="text-sm text-slate-500 mt-1">{tx.type !== 'swap' && tx.type !== 'stake' && <span className="font-mono">{formatAddress(tx.type === 'send' ? tx.to : tx.from)}</span>}{tx.type === 'swap' && <span>{tx.metadata?.outputToken}</span>}{tx.type === 'stake' && <span>{tx.metadata?.validator}</span>}</div><div className="text-xs text-slate-400 mt-1">{tx.chainName} • Block #{tx.blockNumber} • {formatTime(tx.timestamp)}</div></div></div>
                  <div className="text-right"><div className="font-semibold">{tx.type === 'receive' ? '+' : tx.type === 'send' ? '-' : ''}{tx.amount} {tx.token}</div><div className="text-sm text-slate-500">≈ {formatCurrency(parseFloat(tx.amount) * getCurrentPrice(tx.token))}</div><span className={`px-2 py-0.5 rounded text-xs ${getStatusColor(tx.status)}`}>{tx.status}</span></div>
                </div>
                <div className="mt-3 pt-3 border-t border-slate-200 dark:border-slate-700"><div className="grid grid-cols-2 md:grid-cols-4 gap-2 text-xs"><div><span className="text-slate-500">Hash: </span><span className="font-mono">{formatAddress(tx.hash)}</span></div><div><span className="text-slate-500">Fee: </span><span>{tx.fee} {tx.token}</span></div>{tx.gasUsed && <div><span className="text-slate-500">Gas: </span><span>{tx.gasUsed}</span></div>}{tx.nonce !== undefined && <div><span className="text-slate-500">Nonce: </span><span>{tx.nonce}</span></div>}</div></div>
              </div>
            ))}
          </div>
        )}
        {activeTab === 'alerts' && (
          <div className="space-y-4">
            {alerts.length === 0 ? <div className="bg-white dark:bg-slate-800 rounded-lg p-12 text-center"><div className="text-6xl mb-4">🔔</div><h3 className="text-xl font-semibold mb-2">No Price Alerts</h3></div> : alerts.map((alert) => (
              <div key={alert.id} className="bg-white dark:bg-slate-800 rounded-lg p-4 shadow-sm">
                <div className="flex items-center justify-between"><div className="flex items-center gap-4"><div className={`w-10 h-10 rounded-full flex items-center justify-center ${alert.condition === 'above' ? 'bg-green-100 text-green-600' : 'bg-red-100 text-red-600'}`}>{alert.condition === 'above' ? '↑' : '↓'}</div><div><div className="font-semibold">{alert.symbol}</div><div className="text-sm text-slate-500">{alert.condition === 'above' ? 'Above' : 'Below'} ${alert.targetPrice.toLocaleString()}</div></div></div><div className="flex items-center gap-4"><div className="text-right"><div className="text-sm text-slate-500">Current</div><div className="font-semibold">${alert.currentPrice.toLocaleString()}</div></div><button onClick={() => handleToggleAlert(alert.id)} className={`px-3 py-1 rounded text-sm ${alert.isActive ? 'bg-green-100 text-green-800' : 'bg-slate-100'}`}>{alert.isActive ? 'Active' : 'Paused'}</button><button onClick={() => handleDeleteAlert(alert.id)} className="text-red-500">Delete</button></div></div>
              </div>
            ))}
          </div>
        )}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mt-8">
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4"><div className="text-slate-500 text-sm">Total Transactions</div><div className="text-2xl font-bold">{transactions.length}</div></div>
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4"><div className="text-slate-500 text-sm">Confirmed</div><div className="text-2xl font-bold text-green-500">{transactions.filter(t => t.status === 'confirmed').length}</div></div>
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4"><div className="text-slate-500 text-sm">Pending</div><div className="text-2xl font-bold text-yellow-500">{transactions.filter(t => t.status === 'pending').length}</div></div>
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4"><div className="text-slate-500 text-sm">Failed</div><div className="text-2xl font-bold text-red-500">{transactions.filter(t => t.status === 'failed').length}</div></div>
        </div>
      </div>
    </div>
  );
}
