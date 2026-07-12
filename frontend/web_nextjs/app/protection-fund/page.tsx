'use client';

import React, { useState } from 'react';

interface FundStats {
  totalFund: string;
  currentCoverage: string;
  protectedUsers: number;
  claimsPaid: string;
  annualBudget: string;
  reserveRatio: string;
}

interface Claim {
  id: string;
  user: string;
  amount: string;
  reason: string;
  status: 'pending' | 'approved' | 'paid';
  date: number;
}

const MOCK_CLAIMS: Claim[] = [
  { id: '1', user: '0x7a...3d2f', amount: '2,450 USDC', reason: 'Phishing attack', status: 'paid', date: Date.now() - 86400000 * 5 },
  { id: '2', user: '0x3f...8c1e', amount: '1,200 USDC', reason: 'Smart contract exploit', status: 'approved', date: Date.now() - 86400000 * 2 },
  { id: '3', user: '0x9b...2d4a', amount: '5,000 USDT', reason: 'Fake token approval', status: 'pending', date: Date.now() - 86400000 },
];

export default function ProtectionFundPage() {
  const [activeTab, setActiveTab] = useState<'overview' | 'claims' | 'coverage'>('overview');
  const [stats] = useState<FundStats>({
    totalFund: '$10,000,000',
    currentCoverage: '$8,500,000',
    protectedUsers: 245892,
    claimsPaid: '$145,230',
    annualBudget: '$2,000,000',
    reserveRatio: '85%',
  });

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900">
      <header className="bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
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
                  <p className="text-2xl font-bold">{stats.totalFund}</p>
                </div>
                <div className="bg-white/20 rounded-lg px-4 py-2">
                  <p className="text-sm text-blue-100">Protected Users</p>
                  <p className="text-2xl font-bold">{stats.protectedUsers.toLocaleString()}</p>
                </div>
                <div className="bg-white/20 rounded-lg px-4 py-2">
                  <p className="text-sm text-blue-100">Claims Paid</p>
                  <p className="text-2xl font-bold">{stats.claimsPaid}</p>
                </div>
              </div>
            </div>
            <div className="text-8xl">🛡️</div>
          </div>
        </div>

        {/* Stats Grid */}
        <div className="grid grid-cols-3 gap-4 mb-8">
          <div className="bg-white dark:bg-slate-800 rounded-lg p-6 border border-slate-200 dark:border-slate-700">
            <p className="text-sm text-slate-500 dark:text-slate-400">Current Coverage</p>
            <p className="text-3xl font-bold text-green-600">{stats.currentCoverage}</p>
            <p className="text-sm text-slate-500 mt-1">Available for claims</p>
          </div>
          <div className="bg-white dark:bg-slate-800 rounded-lg p-6 border border-slate-200 dark:border-slate-700">
            <p className="text-sm text-slate-500 dark:text-slate-400">Annual Budget</p>
            <p className="text-3xl font-bold">{stats.annualBudget}</p>
            <p className="text-sm text-slate-500 mt-1">2026 allocation</p>
          </div>
          <div className="bg-white dark:bg-slate-800 rounded-lg p-6 border border-slate-200 dark:border-slate-700">
            <p className="text-sm text-slate-500 dark:text-slate-400">Reserve Ratio</p>
            <p className="text-3xl font-bold text-blue-600">{stats.reserveRatio}</p>
            <p className="text-sm text-slate-500 mt-1">Of total fund maintained</p>
          </div>
        </div>

        {/* Tabs */}
        <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700">
          <div className="flex border-b border-slate-200 dark:border-slate-700">
            <button
              onClick={() => setActiveTab('overview')}
              className={`px-6 py-4 font-medium ${activeTab === 'overview' ? 'text-blue-600 border-b-2 border-blue-600' : 'text-slate-500'}`}
            >
              Coverage Details
            </button>
            <button
              onClick={() => setActiveTab('claims')}
              className={`px-6 py-4 font-medium ${activeTab === 'claims' ? 'text-blue-600 border-b-2 border-blue-600' : 'text-slate-500'}`}
            >
              Recent Claims
            </button>
            <button
              onClick={() => setActiveTab('coverage')}
              className={`px-6 py-4 font-medium ${activeTab === 'coverage' ? 'text-blue-600 border-b-2 border-blue-600' : 'text-slate-500'}`}
            >
              How It Works
            </button>
          </div>

          <div className="p-6">
            {activeTab === 'overview' && (
              <div className="space-y-6">
                <div className="grid grid-cols-2 gap-6">
                  <div className="p-4 bg-green-50 dark:bg-green-900/20 rounded-lg border border-green-200 dark:border-green-800">
                    <h3 className="font-semibold text-green-800 dark:text-green-200 mb-3">Covered Events</h3>
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
                  <div className="p-4 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800">
                    <h3 className="font-semibold text-red-800 dark:text-red-200 mb-3">Not Covered</h3>
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

                <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800">
                  <h3 className="font-semibold text-blue-800 dark:text-blue-200 mb-3">Coverage Limits</h3>
                  <div className="grid grid-cols-3 gap-4 text-sm">
                    <div>
                      <p className="font-medium">Standard Users</p>
                      <p className="text-2xl font-bold">$10,000</p>
                      <p className="text-slate-500">Max per user</p>
                    </div>
                    <div>
                      <p className="font-medium">VIP Users</p>
                      <p className="text-2xl font-bold">$50,000</p>
                      <p className="text-slate-500">With KYC verification</p>
                    </div>
                    <div>
                      <p className="font-medium">Institutional</p>
                      <p className="text-2xl font-bold">$500,000</p>
                      <p className="text-slate-500">Custom coverage</p>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {activeTab === 'claims' && (
              <div className="space-y-4">
                {MOCK_CLAIMS.map(claim => (
                  <div key={claim.id} className="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-700 rounded-lg">
                    <div className="flex items-center gap-4">
                      <div className="w-10 h-10 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center text-white font-bold">
                        {claim.user.slice(2, 4).toUpperCase()}
                      </div>
                      <div>
                        <p className="font-medium">{claim.user}</p>
                        <p className="text-sm text-slate-500">{claim.reason}</p>
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
                  <div className="flex-1 p-4 bg-slate-50 dark:bg-slate-700 rounded-lg text-center">
                    <div className="text-4xl mb-2">1</div>
                    <h3 className="font-semibold mb-2">Report Incident</h3>
                    <p className="text-sm text-slate-500">Contact support within 7 days of incident</p>
                  </div>
                  <div className="flex-1 p-4 bg-slate-50 dark:bg-slate-700 rounded-lg text-center">
                    <div className="text-4xl mb-2">2</div>
                    <h3 className="font-semibold mb-2">Submit Evidence</h3>
                    <p className="text-sm text-slate-500">Provide proof of unauthorized transaction</p>
                  </div>
                  <div className="flex-1 p-4 bg-slate-50 dark:bg-slate-700 rounded-lg text-center">
                    <div className="text-4xl mb-2">3</div>
                    <h3 className="font-semibold mb-2">Review Process</h3>
                    <p className="text-sm text-slate-500">Team reviews claim within 5 business days</p>
                  </div>
                  <div className="flex-1 p-4 bg-slate-50 dark:bg-slate-700 rounded-lg text-center">
                    <div className="text-4xl mb-2">4</div>
                    <h3 className="font-semibold mb-2">Receive Compensation</h3>
                    <p className="text-sm text-slate-500">Funds transferred to your wallet</p>
                  </div>
                </div>

                <div className="p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg">
                  <p className="text-yellow-800 dark:text-yellow-200">
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
