'use client';

import React, { useState } from 'react';
import { useTheme } from '../components/ThemeProvider';

interface Approval {
  id: string;
  token: string;
  tokenSymbol: string;
  spender: string;
  spenderName: string;
  chainId: number;
  chainName: string;
  amount: string;
  allowance: string;
  isUnlimited: boolean;
  dateApproved: number;
  risk: 'low' | 'medium' | 'high' | 'critical';
  verified: boolean;
}

const MOCK_APPROVALS: Approval[] = [
  { id: '1', token: '0xdac17f958d2ee523a2206206994597c13d831ec7', tokenSymbol: 'USDT', spender: '0x7a250d5630b4cf539739df2c5dacb4c659f2488d', spenderName: 'Uniswap V3', chainId: 1, chainName: 'Ethereum', amount: '10000', allowance: '10000', isUnlimited: false, dateApproved: Date.now() - 86400000 * 30, risk: 'low', verified: true },
  { id: '2', token: '0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48', tokenSymbol: 'USDC', spender: '0xe592427a0aece92de3edee1f18e0157c05861564', spenderName: 'Uniswap V3', chainId: 1, chainName: 'Ethereum', amount: 'unlimited', allowance: 'unlimited', isUnlimited: true, dateApproved: Date.now() - 86400000 * 60, risk: 'high', verified: true },
  { id: '3', token: '0x6b175474e89094c44da98b954eedeac495271d0f', tokenSymbol: 'DAI', spender: '0x5c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f', spenderName: 'Uniswap V2', chainId: 1, chainName: 'Ethereum', amount: '5000', allowance: '5000', isUnlimited: false, dateApproved: Date.now() - 86400000 * 15, risk: 'low', verified: true },
  { id: '4', token: '0x2260fac5e5542a773aa44fbcfedf7c193bc2c599', tokenSymbol: 'WBTC', spender: '0xbb0e17ef65f82ab018d8edd776e8dd940327b28b', spenderName: 'Aave V3', chainId: 1, chainName: 'Ethereum', amount: 'unlimited', allowance: 'unlimited', isUnlimited: true, dateApproved: Date.now() - 86400000 * 90, risk: 'critical', verified: true },
  { id: '5', token: '0x7fc66500c84a76ad7e9c93437bfc5ac33e2ddae9', tokenSymbol: 'AAVE', spender: '0x87870bca3f3fd6335c3fbd83f7c6bdd4c0d823ce', spenderName: 'Aave V3', chainId: 1, chainName: 'Ethereum', amount: '1000', allowance: '1000', isUnlimited: false, dateApproved: Date.now() - 86400000 * 7, risk: 'medium', verified: true },
  { id: '6', token: '0x514910771af9ca656af840dff83e8264ecf986ca', tokenSymbol: 'LINK', spender: '0xa6cc7c5ec1e4b7e3e2f5e5d2e5f5e5d2e5f5e5', spenderName: 'Unknown Contract', chainId: 1, chainName: 'Ethereum', amount: 'unlimited', allowance: 'unlimited', isUnlimited: true, dateApproved: Date.now() - 86400000 * 45, risk: 'critical', verified: false },
  { id: '7', token: '0x1f9840a85d5af5bf1d1762f925bdaddc4201f984', tokenSymbol: 'UNI', spender: '0x1f9840a85d5af5bf1d1762f925bdaddc4201f984', spenderName: 'Uniswap V3', chainId: 137, chainName: 'Polygon', amount: '5000', allowance: '5000', isUnlimited: false, dateApproved: Date.now() - 86400000 * 20, risk: 'low', verified: true },
  { id: '8', token: '0x2791bca1f2de4661ed88a30c99a7a9449aa84174', tokenSymbol: 'USDC', spender: '0x1b02da8cb01d0e0f526c3887ca6d2ae054c7a14d', spenderName: 'QuickSwap', chainId: 137, chainName: 'Polygon', amount: 'unlimited', allowance: 'unlimited', isUnlimited: true, dateApproved: Date.now() - 86400000 * 10, risk: 'high', verified: true },
];

const RISK_COLORS = {
  low: 'bg-green-100 text-green-800',
  medium: 'bg-yellow-100 text-yellow-800',
  high: 'bg-orange-100 text-orange-800',
  critical: 'bg-red-100 text-red-800',
};

export default function ApprovalsPage() {
  const [approvals, setApprovals] = useState<Approval[]>(MOCK_APPROVALS);
  const [selectedChain, setSelectedChain] = useState<string>('all');
  const [selectedRisk, setSelectedRisk] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [revoking, setRevoking] = useState<string | null>(null);
  const [showRiskWarning, setShowRiskWarning] = useState(false);
  const [selectedApproval, setSelectedApproval] = useState<Approval | null>(null);
  const { isDark } = useTheme();

  const filteredApprovals = approvals.filter(approval => {
    if (selectedChain !== 'all' && approval.chainId.toString() !== selectedChain) return false;
    if (selectedRisk !== 'all' && approval.risk !== selectedRisk) return false;
    if (searchQuery && !approval.tokenSymbol.toLowerCase().includes(searchQuery.toLowerCase()) && 
        !approval.spenderName.toLowerCase().includes(searchQuery.toLowerCase())) return false;
    return true;
  });

  const handleRevoke = async (approval: Approval) => {
    if (approval.risk === 'critical' && !showRiskWarning) {
      setSelectedApproval(approval);
      return;
    }
    setRevoking(approval.id);
    await new Promise(resolve => setTimeout(resolve, 2000));
    setApprovals(prev => prev.filter(a => a.id !== approval.id));
    setRevoking(null);
  };

  const confirmRevoke = async () => {
    if (!selectedApproval) return;
    setRevoking(selectedApproval.id);
    await new Promise(resolve => setTimeout(resolve, 2000));
    setApprovals(prev => prev.filter(a => a.id !== selectedApproval.id));
    setRevoking(null);
    setShowRiskWarning(false);
    setSelectedApproval(null);
  };

  const handleRevokeAllUnlimited = async () => {
    setRevoking('all-unlimited');
    await new Promise(resolve => setTimeout(resolve, 3000));
    setApprovals(prev => prev.filter(a => !a.isUnlimited));
    setRevoking(null);
  };

  const stats = {
    total: approvals.length,
    unlimited: approvals.filter(a => a.isUnlimited).length,
    critical: approvals.filter(a => a.risk === 'critical').length,
    high: approvals.filter(a => a.risk === 'high').length,
  };

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900 text-white' : 'bg-gray-50 text-gray-900'}`}>
      <header className={`${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'} border-b ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
        <div className="max-w-7xl mx-auto px-4">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-4">
              <a href="/" className="text-2xl">🐯</a>
              <h1 className="text-xl font-bold">Token Approvals</h1>
            </div>
            <div className="flex items-center gap-4">
              <button
                onClick={handleRevokeAllUnlimited}
                disabled={revoking !== null || stats.unlimited === 0}
                className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {revoking === 'all-unlimited' ? 'Revoking...' : `Revoke All Unlimited (${stats.unlimited})`}
              </button>
            </div>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto px-4 py-8">
        <div className="grid grid-cols-4 gap-4 mb-8">
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg p-4 border ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Total Approvals</p>
            <p className="text-2xl font-bold">{stats.total}</p>
          </div>
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg p-4 border ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Unlimited Approvals</p>
            <p className="text-2xl font-bold text-orange-600">{stats.unlimited}</p>
          </div>
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg p-4 border ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>High Risk</p>
            <p className="text-2xl font-bold text-red-600">{stats.high + stats.critical}</p>
          </div>
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg p-4 border ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Verified Contracts</p>
            <p className="text-2xl font-bold text-green-600">{approvals.filter(a => a.verified).length}</p>
          </div>
        </div>

        <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg p-4 mb-6 border ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
          <div className="flex gap-4 flex-wrap">
            <input
              type="text"
              placeholder="Search token or contract..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className={`flex-1 min-w-[200px] px-4 py-2 border ${isDark ? 'border-gray-600 bg-gray-700' : 'border-gray-300 bg-white'} rounded-lg focus:ring-2 focus:ring-blue-500 outline-none`}
            />
            <select
              value={selectedChain}
              onChange={(e) => setSelectedChain(e.target.value)}
              className={`px-4 py-2 border ${isDark ? 'border-gray-600 bg-gray-700' : 'border-gray-300 bg-white'} rounded-lg`}
            >
              <option value="all">All Chains</option>
              <option value="1">Ethereum</option>
              <option value="137">Polygon</option>
              <option value="56">BNB Chain</option>
              <option value="42161">Arbitrum</option>
              <option value="10">Optimism</option>
            </select>
            <select
              value={selectedRisk}
              onChange={(e) => setSelectedRisk(e.target.value)}
              className={`px-4 py-2 border ${isDark ? 'border-gray-600 bg-gray-700' : 'border-gray-300 bg-white'} rounded-lg`}
            >
              <option value="all">All Risk Levels</option>
              <option value="critical">Critical</option>
              <option value="high">High</option>
              <option value="medium">Medium</option>
              <option value="low">Low</option>
            </select>
          </div>
        </div>

        <div className="space-y-4">
          {filteredApprovals.map(approval => (
            <div key={approval.id} className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg p-4 border ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <div className="w-12 h-12 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center text-white font-bold">
                    {approval.tokenSymbol.slice(0, 2)}
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <p className="font-semibold">{approval.tokenSymbol}</p>
                      <span className={`px-2 py-0.5 rounded text-xs ${RISK_COLORS[approval.risk]}`}>
                        {approval.risk.toUpperCase()}
                      </span>
                      {approval.verified && (
                        <span className="px-2 py-0.5 rounded text-xs bg-blue-100 text-blue-800">Verified</span>
                      )}
                    </div>
                    <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                      {approval.spenderName} on {approval.chainName}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-4">
                  <div className="text-right">
                    <p className="font-semibold">{approval.isUnlimited ? 'Unlimited' : approval.allowance}</p>
                    <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                      Approved {Math.floor((Date.now() - approval.dateApproved) / (1000 * 60 * 60 * 24))} days ago
                    </p>
                  </div>
                  <button
                    onClick={() => handleRevoke(approval)}
                    disabled={revoking === approval.id}
                    className={`px-4 py-2 ${isDark ? 'bg-red-900/30 text-red-400 hover:bg-red-900/50' : 'bg-red-100 text-red-600 hover:bg-red-200'} rounded-lg disabled:opacity-50 transition-colors`}
                  >
                    {revoking === approval.id ? 'Revoking...' : 'Revoke'}
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>

        {filteredApprovals.length === 0 && (
          <div className={`text-center py-12 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
            <p className="text-lg">No approvals found</p>
          </div>
        )}
      </div>

      {selectedApproval && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'} rounded-xl p-6 max-w-md w-full mx-4`}>
            <div className="flex items-center gap-3 mb-4">
              <span className="text-4xl">Warning</span>
              <h3 className="text-xl font-bold text-red-600">Critical Risk Warning</h3>
            </div>
            <p className={`${isDark ? 'text-gray-300' : 'text-gray-600'} mb-4`}>
              Revoke approval for <strong>{selectedApproval.tokenSymbol}</strong> to <strong>{selectedApproval.spenderName}</strong>?
            </p>
            <p className={`${isDark ? 'text-gray-300' : 'text-gray-600'} mb-6`}>
              Risk level: <strong className="text-red-600">{selectedApproval.risk.toUpperCase()}</strong>
            </p>
            <div className="flex gap-4">
              <button
                onClick={() => setSelectedApproval(null)}
                className={`flex-1 px-4 py-2 ${isDark ? 'bg-gray-700' : 'bg-gray-200'} rounded-lg`}
              >
                Cancel
              </button>
              <button
                onClick={confirmRevoke}
                className="flex-1 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700"
              >
                Yes, Revoke
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
