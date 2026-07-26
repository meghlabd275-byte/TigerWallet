/**
 * TigerWallet Approval Manager
 * Token approval management and revocation system
 * 
 * Features:
 * - Scan all token approvals for an address
 * - Batch revocation
 * - Risk assessment
 * - Real-time monitoring
 */

'use client';

import React, { useState, useEffect, useCallback } from 'react';

// Types
interface TokenApproval {
  id: string;
  owner: string;
  spender: string;
  token: TokenInfo;
  amount: string;
  valueUSD: number;
  isInfinite: boolean;
  lastUpdated: number;
  risk: 'low' | 'medium' | 'high' | 'critical';
  txHash: string;
}

interface TokenInfo {
  address: string;
  symbol: string;
  name: string;
  decimals: number;
  logo?: string;
  price?: number;
}

interface ApprovalStats {
  totalApprovals: number;
  highRisk: number;
  mediumRisk: number;
  lowRisk: number;
  totalValue: number;
  infiniteApprovals: number;
}

interface RevokeResult {
  success: boolean;
  txHash?: string;
  error?: string;
}

// API Base URL
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:9098';

export default function ApprovalManagerPage() {
  const [address, setAddress] = useState('');
  const [loading, setLoading] = useState(false);
  const [scanning, setScanning] = useState(false);
  const [approvals, setApprovals] = useState<TokenApproval[]>([]);
  const [stats, setStats] = useState<ApprovalStats>({
    totalApprovals: 0,
    highRisk: 0,
    mediumRisk: 0,
    lowRisk: 0,
    totalValue: 0,
    infiniteApprovals: 0,
  });
  const [selectedApprovals, setSelectedApprovals] = useState<Set<string>>(new Set());
  const [filter, setFilter] = useState<'all' | 'high' | 'medium' | 'low'>('all');
  const [darkMode, setDarkMode] = useState(false);

  // Check dark mode
  useEffect(() => {
    if (typeof window !== 'undefined') {
      setDarkMode(document.documentElement.classList.contains('dark'));
    }
  }, []);

  // Scan approvals for address
  const scanApprovals = useCallback(async (walletAddress: string) => {
    if (!walletAddress) return;
    
    setScanning(true);
    setLoading(true);
    
    try {
      const res = await fetch(`${API_BASE}/api/v1/approvals/scan/${walletAddress}`);
      if (res.ok) {
        const data = await res.json();
        setApprovals(data.approvals);
        calculateStats(data.approvals);
      }
    } catch (err) {
      console.error('Failed to scan approvals:', err);
    } finally {
      setScanning(false);
      setLoading(false);
    }
  }, []);

  // Calculate statistics
  const calculateStats = (approvalList: TokenApproval[]) => {
    const newStats: ApprovalStats = {
      totalApprovals: approvalList.length,
      highRisk: 0,
      mediumRisk: 0,
      lowRisk: 0,
      totalValue: 0,
      infiniteApprovals: 0,
    };

    approvalList.forEach(approval => {
      if (approval.risk === 'high') newStats.highRisk++;
      else if (approval.risk === 'medium') newStats.mediumRisk++;
      else newStats.lowRisk++;
      
      if (approval.isInfinite) newStats.infiniteApprovals++;
      newStats.totalValue += approval.valueUSD;
    });

    setStats(newStats);
  };

  // Revoke single approval
  const revokeApproval = async (approval: TokenApproval): Promise<RevokeResult> => {
    try {
      const res = await fetch(`${API_BASE}/api/v1/approvals/revoke`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          owner: approval.owner,
          spender: approval.spender,
          token: approval.token.address,
        }),
      });

      if (res.ok) {
        const data = await res.json();
        // Remove from list
        setApprovals(prev => prev.filter(a => a.id !== approval.id));
        calculateStats(approvals.filter(a => a.id !== approval.id));
        return { success: true, txHash: data.txHash };
      }
      
      const error = await res.json();
      return { success: false, error: error.message };
    } catch (err) {
      return { success: false, error: 'Failed to revoke approval' };
    }
  };

  // Batch revoke
  const batchRevoke = async () => {
    const selected = Array.from(selectedApprovals);
    const results: RevokeResult[] = [];
    
    for (const id of selected) {
      const approval = approvals.find(a => a.id === id);
      if (approval) {
        const result = await revokeApproval(approval);
        results.push(result);
      }
    }
    
    // Update selection
    setSelectedApprovals(new Set());
    
    return results;
  };

  // Toggle selection
  const toggleSelection = (id: string) => {
    const newSelected = new Set(selectedApprovals);
    if (newSelected.has(id)) {
      newSelected.delete(id);
    } else {
      newSelected.add(id);
    }
    setSelectedApprovals(newSelected);
  };

  // Filter approvals
  const filteredApprovals = approvals.filter(approval => {
    if (filter === 'all') return true;
    return approval.risk === filter;
  });

  // Format currency
  const formatUSD = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
    }).format(value);
  };

  // Format address
  const formatAddress = (addr: string) => {
    if (!addr || addr.length < 10) return addr;
    return `${addr.slice(0, 6)}...${addr.slice(-4)}`;
  };

  // Get risk color
  const getRiskColor = (risk: string) => {
    switch (risk) {
      case 'critical':
        return 'text-red-600 bg-red-100 dark:bg-red-900 dark:text-red-200';
      case 'high':
        return 'text-orange-600 bg-orange-100 dark:bg-orange-900 dark:text-orange-200';
      case 'medium':
        return 'text-yellow-600 bg-yellow-100 dark:bg-yellow-900 dark:text-yellow-200';
      default:
        return 'text-green-600 bg-green-100 dark:bg-green-900 dark:text-green-200';
    }
  };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900">
      <header className="bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
        <div className="max-w-7xl mx-auto px-4">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-4">
              <a href="/" className="text-2xl">🐯</a>
              <h1 className="text-xl font-bold">Approval Manager</h1>
            </div>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto px-4 py-8">
        {/* Search Section */}
        <div className="bg-white dark:bg-slate-800 rounded-lg p-6 mb-6 border border-slate-200 dark:border-slate-700">
          <h2 className="text-lg font-semibold mb-4">Scan Wallet Approvals</h2>
          <div className="flex gap-4">
            <input
              type="text"
              value={address}
              onChange={(e) => setAddress(e.target.value)}
              placeholder="Enter wallet address (0x...)"
              className="flex-1 px-4 py-2 rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-700"
            />
            <button
              onClick={() => scanApprovals(address)}
              disabled={scanning || !address}
              className="px-6 py-2 bg-blue-600 text-white rounded-lg font-medium disabled:opacity-50"
            >
              {scanning ? 'Scanning...' : 'Scan'}
            </button>
          </div>
        </div>

        {/* Stats Grid */}
        {loading ? (
          <div className="text-center py-8">Loading...</div>
        ) : approvals.length > 0 ? (
          <>
            <div className="grid grid-cols-6 gap-4 mb-6">
              <div className="bg-white dark:bg-slate-800 rounded-lg p-4 border border-slate-200 dark:border-slate-700">
                <p className="text-sm text-slate-500">Total</p>
                <p className="text-2xl font-bold">{stats.totalApprovals}</p>
              </div>
              <div className="bg-white dark:bg-slate-800 rounded-lg p-4 border border-slate-200 dark:border-slate-700">
                <p className="text-sm text-slate-500">High Risk</p>
                <p className="text-2xl font-bold text-red-600">{stats.highRisk}</p>
              </div>
              <div className="bg-white dark:bg-slate-800 rounded-lg p-4 border border-slate-200 dark:border-slate-700">
                <p className="text-sm text-slate-500">Medium Risk</p>
                <p className="text-2xl font-bold text-orange-600">{stats.mediumRisk}</p>
              </div>
              <div className="bg-white dark:bg-slate-800 rounded-lg p-4 border border-slate-200 dark:border-slate-700">
                <p className="text-sm text-slate-500">Low Risk</p>
                <p className="text-2xl font-bold text-green-600">{stats.lowRisk}</p>
              </div>
              <div className="bg-white dark:bg-slate-800 rounded-lg p-4 border border-slate-200 dark:border-slate-700">
                <p className="text-sm text-slate-500">Infinite</p>
                <p className="text-2xl font-bold text-purple-600">{stats.infiniteApprovals}</p>
              </div>
              <div className="bg-white dark:bg-slate-800 rounded-lg p-4 border border-slate-200 dark:border-slate-700">
                <p className="text-sm text-slate-500">Total Value</p>
                <p className="text-2xl font-bold">{formatUSD(stats.totalValue)}</p>
              </div>
            </div>

            {/* Filter & Actions */}
            <div className="flex justify-between items-center mb-4">
              <div className="flex gap-2">
                {(['all', 'high', 'medium', 'low'] as const).map(f => (
                  <button
                    key={f}
                    onClick={() => setFilter(f)}
                    className={`px-4 py-2 rounded-lg ${
                      filter === f
                        ? 'bg-blue-600 text-white'
                        : 'bg-slate-200 dark:bg-slate-700'
                    }`}
                  >
                    {f.charAt(0).toUpperCase() + f.slice(1)}
                  </button>
                ))}
              </div>
              
              {selectedApprovals.size > 0 && (
                <button
                  onClick={batchRevoke}
                  className="px-4 py-2 bg-red-600 text-white rounded-lg"
                >
                  Revoke Selected ({selectedApprovals.size})
                </button>
              )}
            </div>

            {/* Approvals List */}
            <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700">
              <table className="w-full">
                <thead className="bg-slate-50 dark:bg-slate-700">
                  <tr>
                    <th className="px-4 py-3 text-left">
                      <input
                        type="checkbox"
                        onChange={(e) => {
                          if (e.target.checked) {
                            setSelectedApprovals(new Set(filteredApprovals.map(a => a.id)));
                          } else {
                            setSelectedApprovals(new Set());
                          }
                        }}
                        checked={selectedApprovals.size === filteredApprovals.length}
                      />
                    </th>
                    <th className="px-4 py-3 text-left">Token</th>
                    <th className="px-4 py-3 text-left">Spender</th>
                    <th className="px-4 py-3 text-right">Amount</th>
                    <th className="px-4 py-3 text-right">Value</th>
                    <th className="px-4 py-3 text-center">Risk</th>
                    <th className="px-4 py-3 text-center">Action</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredApprovals.map(approval => (
                    <tr key={approval.id} className="border-t border-slate-200 dark:border-slate-700">
                      <td className="px-4 py-3">
                        <input
                          type="checkbox"
                          checked={selectedApprovals.has(approval.id)}
                          onChange={() => toggleSelection(approval.id)}
                        />
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          {approval.token.logo && (
                            <img src={approval.token.logo} alt="" className="w-6 h-6 rounded-full" />
                          )}
                          <span className="font-medium">{approval.token.symbol}</span>
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <a
                          href={`https://etherscan.io/address/${approval.spender}`}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-blue-600 hover:underline"
                        >
                          {formatAddress(approval.spender)}
                        </a>
                      </td>
                      <td className="px-4 py-3 text-right font-mono">
                        {approval.isInfinite ? '∞' : approval.amount}
                      </td>
                      <td className="px-4 py-3 text-right">
                        {formatUSD(approval.valueUSD)}
                      </td>
                      <td className="px-4 py-3 text-center">
                        <span className={`px-2 py-1 rounded-full text-xs font-medium ${getRiskColor(approval.risk)}`}>
                          {approval.risk.toUpperCase()}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-center">
                        <button
                          onClick={() => revokeApproval(approval)}
                          className="px-3 py-1 text-sm bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-200 rounded"
                        >
                          Revoke
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </>
        ) : (
          <div className="text-center py-12 text-slate-500">
            {address ? 'No approvals found' : 'Enter an address to scan'}
          </div>
        )}
      </div>
    </div>
  );
}
