'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../components/ThemeProvider';

// Types - matching backend API
interface FundStats {
  totalFund: number;
  availableFund: number;
  currentCoverage: number;
  protectedUsers: number;
  claimsPaid: number;
  pendingClaims: number;
  annualBudget: number;
  reserveRatio: number;
  avgClaimTime: number;
}

interface FundPool {
  id: number;
  name: string;
  totalBalance: number;
  availableBalance: number;
  reservedBalance: number;
  annualBudget: number;
  currency: string;
  walletAddress: string;
  isActive: boolean;
}

interface Claim {
  claimId: string;
  userAddress: string;
  amount: number;
  currency: string;
  reason: string;
  description: string;
  status: 'pending' | 'review' | 'approved' | 'rejected' | 'paid';
  priority: number;
  createdAt: string;
  updatedAt: string;
}

// API Base URL - configurable
const API_BASE = process.env.NEXT_PUBLIC_API_URL || '';

export default function ProtectionFundPage() {
  const [activeTab, setActiveTab] = useState<'overview' | 'claims' | 'coverage'>('overview');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [stats, setStats] = useState<FundStats>({
    totalFund: 0,
    availableFund: 0,
    currentCoverage: 0,
    protectedUsers: 0,
    claimsPaid: 0,
    pendingClaims: 0,
    annualBudget: 0,
    reserveRatio: 0,
    avgClaimTime: 0,
  });
  const [fundPool, setFundPool] = useState<FundPool | null>(null);
  const [claims, setClaims] = useState<Claim[]>([]);
  const [darkMode, setDarkMode] = useState(false);

  // Check for dark mode
  useEffect(() => {
    if (typeof window !== 'undefined') {
      setDarkMode(document.documentElement.classList.contains('dark'));
    }
  }, []);

  const { isDark } = useTheme();

  // Fetch data from backend
  const fetchStats = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      
      const [statsRes, poolRes] = await Promise.all([
        fetch(`${API_BASE}/api/v1/protection/stats`).catch(() => null),
        fetch(`${API_BASE}/api/v1/protection/fund`).catch(() => null),
      ]);
      
      if (statsRes?.ok) {
        const data = await statsRes.json();
        setStats(data);
      }
      
      if (poolRes?.ok) {
        const data = await poolRes.json();
        setFundPool(data);
      }
    } catch (err) {
      console.error('Failed to fetch protection fund stats:', err);
      setError('Failed to load protection fund data');
    } finally {
      setLoading(false);
    }
  }, []);

  // Fetch claims
  const fetchClaims = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/api/v1/protection/claims`).catch(() => null);
      if (res?.ok) {
        const data = await res.json();
        setClaims(data.slice(0, 10)); // Limit to 10 recent
      }
    } catch (err) {
      console.error('Failed to fetch claims:', err);
    }
  }, []);

  useEffect(() => {
    fetchStats();
    fetchClaims();
    
    // Refresh every 30 seconds
    const interval = setInterval(fetchStats, 30000);
    return () => clearInterval(interval);
  }, [fetchStats, fetchClaims]);

  // Format currency
  const formatCurrency = (amount: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(amount);
  };

  // Format address
  const formatAddress = (address: string) => {
    if (!address || address.length < 10) return address;
    return `${address.slice(0, 6)}...${address.slice(-4)}`;
  };

  // Format date
  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    });
  };

  // Status badge color
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'paid':
        return isDark ? 'bg-green-900 text-green-200' : 'bg-green-100 text-green-800';
      case 'approved':
        return isDark ? 'bg-blue-900 text-blue-200' : 'bg-blue-100 text-blue-800';
      case 'pending':
      case 'review':
        return isDark ? 'bg-yellow-900 text-yellow-200' : 'bg-yellow-100 text-yellow-800';
      case 'rejected':
        return isDark ? 'bg-red-900 text-red-200' : 'bg-red-100 text-red-800';
      default:
        return isDark ? 'bg-gray-800 text-gray-200' : 'bg-gray-100 text-gray-800';
    }
  };

  return (
    <div className={`min-h-screen ${isDark ? 'bg-slate-900' : 'bg-slate-50'}`}>
      <header className={`border-b ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}>
        <div className="max-w-7xl mx-auto px-4">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-4">
              <a href="/" className="text-2xl">🐯</a>
              <h1 className="text-xl font-bold">Protection Fund</h1>
            </div>
            <div className="flex items-center gap-2">
              <span className="px-3 py-1 bg-green-100 text-green-800 rounded-full text-sm font-medium">
                Active
              </span>
            </div>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto px-4 py-8">
        {/* Hero Section */}
        <div className="bg-gradient-to-r from-blue-600 to-purple-600 rounded-2xl p-8 mb-8 text-white">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-3xl font-bold mb-2">TigerWallet Protection Fund</h2>
              <p className="text-blue-100 mb-4">Your assets are protected by our comprehensive security fund</p>
              <div className="flex gap-4">
                <div className="bg-white/20 rounded-lg px-4 py-2">
                  <p className="text-sm text-blue-100">Total Fund</p>
                  <p className="text-2xl font-bold">{loading ? '...' : formatCurrency(stats.totalFund)}</p>
                </div>
                <div className="bg-white/20 rounded-lg px-4 py-2">
                  <p className="text-sm text-blue-100">Protected Users</p>
                  <p className="text-2xl font-bold">{stats.protectedUsers.toLocaleString()}</p>
                </div>
                <div className="bg-white/20 rounded-lg px-4 py-2">
                  <p className="text-sm text-blue-100">Claims Paid</p>
                  <p className="text-2xl font-bold">{loading ? '...' : formatCurrency(stats.claimsPaid)}</p>
                </div>
              </div>
            </div>
            <div className="text-8xl">🛡️</div>
          </div>
        </div>

        {/* Stats Grid */}
        <div className="grid grid-cols-3 gap-4 mb-8">
          <div className={`rounded-lg p-6 border ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Current Coverage</p>
            <p className="text-3xl font-bold text-green-600">{loading ? '...' : formatCurrency(stats.currentCoverage)}</p>
            <p className={`text-sm mt-1 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Available for claims</p>
          </div>
          <div className={`rounded-lg p-6 border ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Annual Budget</p>
            <p className="text-3xl font-bold">{loading ? '...' : formatCurrency(stats.annualBudget)}</p>
            <p className={`text-sm mt-1 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>2026 allocation</p>
          </div>
          <div className={`rounded-lg p-6 border ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}>
            <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Reserve Ratio</p>
            <p className="text-3xl font-bold text-blue-600">{loading ? '...' : `${(stats.reserveRatio * 100).toFixed(0)}%`}</p>
            <p className={`text-sm mt-1 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Of total fund maintained</p>
          </div>
        </div>

        {/* Tabs */}
        <div className={`rounded-lg border ${isDark ? 'bg-slate-800 border-gray-700' : 'bg-white border-gray-200'}`}>
          <div className={`flex border-b ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
            <button
              onClick={() => setActiveTab('overview')}
              className={`px-6 py-4 font-medium ${activeTab === 'overview' ? 'text-blue-600 border-b-2 border-blue-600' : (isDark ? 'text-gray-400' : 'text-gray-500')}`}
            >
              Coverage Details
            </button>
            <button
              onClick={() => setActiveTab('claims')}
              className={`px-6 py-4 font-medium ${activeTab === 'claims' ? 'text-blue-600 border-b-2 border-blue-600' : (isDark ? 'text-gray-400' : 'text-gray-500')}`}
            >
              Recent Claims
            </button>
            <button
              onClick={() => setActiveTab('coverage')}
              className={`px-6 py-4 font-medium ${activeTab === 'coverage' ? 'text-blue-600 border-b-2 border-blue-600' : (isDark ? 'text-gray-400' : 'text-gray-500')}`}
            >
              How It Works
            </button>
          </div>

          <div className="p-6">
            {activeTab === 'overview' && (
              <div className="space-y-6">
                <div className="grid grid-cols-2 gap-6">
                  <div className={`p-4 rounded-lg border ${isDark ? 'bg-green-900/20 border-green-800' : 'bg-green-50 border-green-200'}`}>
                    <h3 className={`font-semibold mb-3 ${isDark ? 'text-green-200' : 'text-green-800'}`}>Covered Events</h3>
                    <ul className="space-y-2 text-sm">
                      <li className="flex items-center gap-2">
                        <span className="text-green-500">✓</span> Smart contract exploits
                      </li>
                      <li className="flex items-center gap-2">
                        <span className="text-green-500">✓</span> Phishing attacks (with proof)
                      </li>
                      <li className="flex items-center gap-2">
                        <span className="text-green-500">✓</span> Flash loan attacks
                      </li>
                      <li className="flex items-center gap-2">
                        <span className="text-green-500">✓</span> Bridge exploits
                      </li>
                      <li className="flex items-center gap-2">
                        <span className="text-green-500">✓</span> Oracle manipulation
                      </li>
                    </ul>
                  </div>
                  <div className={`p-4 rounded-lg border ${isDark ? 'bg-red-900/20 border-red-800' : 'bg-red-50 border-red-200'}`}>
                    <h3 className={`font-semibold mb-3 ${isDark ? 'text-red-200' : 'text-red-800'}`}>Not Covered</h3>
                    <ul className="space-y-2 text-sm">
                      <li className="flex items-center gap-2">
                        <span className="text-red-500">✗</span> User negligence
                      </li>
                      <li className="flex items-center gap-2">
                        <span className="text-red-500">✗</span> Sharing private keys
                      </li>
                      <li className="flex items-center gap-2">
                        <span className="text-red-500">✗</span> Trading losses
                      </li>
                      <li className="flex items-center gap-2">
                        <span className="text-red-500">✗</span> Impermanent loss
                      </li>
                      <li className="flex items-center gap-2">
                        <span className="text-red-500">✗</span> Centralized exchange hacks
                      </li>
                    </ul>
                  </div>
                </div>

                <div className={`p-4 rounded-lg border ${isDark ? 'bg-blue-900/20 border-blue-800' : 'bg-blue-50 border-blue-200'}`}>
                  <h3 className={`font-semibold mb-3 ${isDark ? 'text-blue-200' : 'text-blue-800'}`}>Coverage Limits</h3>
                  <div className="grid grid-cols-3 gap-4 text-sm">
                    <div>
                      <p className="font-medium">Standard Users</p>
                      <p className="text-2xl font-bold">$10,000</p>
                      <p className={isDark ? 'text-gray-400' : 'text-gray-500'}>Max per user</p>
                    </div>
                    <div>
                      <p className="font-medium">VIP Users</p>
                      <p className="text-2xl font-bold">$50,000</p>
                      <p className={isDark ? 'text-gray-400' : 'text-gray-500'}>With KYC verification</p>
                    </div>
                    <div>
                      <p className="font-medium">Institutional</p>
                      <p className="text-2xl font-bold">$500,000</p>
                      <p className={isDark ? 'text-gray-400' : 'text-gray-500'}>Custom coverage</p>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {activeTab === 'claims' && (
              <div className="space-y-4">
                {claims.map(claim => (
                  <div key={claim.claimId} className={`flex items-center justify-between p-4 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-slate-50'}`}>
                    <div className="flex items-center gap-4">
                      <div className="w-10 h-10 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center text-white font-bold">
                        {claim.userAddress.slice(2, 4).toUpperCase()}
                      </div>
                      <div>
                        <p className="font-medium">{claim.userAddress}</p>
                        <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{claim.reason}</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <p className="font-bold">{claim.amount}</p>
                      <span className={`px-3 py-1 rounded-full text-xs font-medium ${
                        claim.status === 'paid' ? 'bg-green-100 text-green-800' :
                        claim.status === 'approved' ? 'bg-blue-100 text-blue-800' :
                        'bg-yellow-100 text-yellow-800'
                      }`}>
                        {claim.status.toUpperCase()}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {activeTab === 'coverage' && (
              <div className="space-y-6">
                <div className="flex gap-4">
                  <div className={`flex-1 p-4 rounded-lg text-center ${isDark ? 'bg-slate-700' : 'bg-slate-50'}`}>
                    <div className="text-4xl mb-2">1</div>
                    <h3 className="font-semibold mb-2">Report Incident</h3>
                    <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Contact support within 7 days of incident</p>
                  </div>
                  <div className={`flex-1 p-4 rounded-lg text-center ${isDark ? 'bg-slate-700' : 'bg-slate-50'}`}>
                    <div className="text-4xl mb-2">2</div>
                    <h3 className="font-semibold mb-2">Submit Evidence</h3>
                    <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Provide proof of unauthorized transaction</p>
                  </div>
                  <div className={`flex-1 p-4 rounded-lg text-center ${isDark ? 'bg-slate-700' : 'bg-slate-50'}`}>
                    <div className="text-4xl mb-2">3</div>
                    <h3 className="font-semibold mb-2">Review Process</h3>
                    <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Team reviews claim within 5 business days</p>
                  </div>
                  <div className={`flex-1 p-4 rounded-lg text-center ${isDark ? 'bg-slate-700' : 'bg-slate-50'}`}>
                    <div className="text-4xl mb-2">4</div>
                    <h3 className="font-semibold mb-2">Receive Compensation</h3>
                    <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Funds transferred to your wallet</p>
                  </div>
                </div>

                <div className={`p-4 border rounded-lg ${isDark ? 'bg-yellow-900/20 border-yellow-800' : 'bg-yellow-50 border-yellow-200'}`}>
                  <p className={isDark ? 'text-yellow-200' : 'text-yellow-800'}>
                    <strong>Note:</strong> Claims must be reported within 7 days of the incident. 
                    Maximum coverage depends on your account level. False claims will result in account termination.
                  </p>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
