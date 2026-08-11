'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../components/ThemeProvider';

// ============================================================================
// Types
// ============================================================================

interface BugReport {
  id: string;
  title: string;
  description: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
  status: 'submitted' | 'triaged' | 'accepted' | 'rejected' | 'fixed' | 'rewarded' | 'verified';
  reward: string;
  reporter: string;
  date: number;
  program_id: string;
  cvss_score: number;
  impact: string;
  poc_url: string;
}

interface Program {
  id: string;
  name: string;
  description: string;
  status: string;
  total_pool: string;
  paid_out: string;
  rewards: {
    critical: string;
    high: string;
    medium: string;
    low: string;
  };
}

interface Stats {
  total_programs: number;
  total_reports: number;
  accepted_reports: number;
  rewarded_reports: number;
  pending_reports: number;
  total_paid_out: string;
  remaining_pool: string;
  active_hackers: number;
}

// ============================================================================
// API Functions
// ============================================================================

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'https://api.tigerwallet.io';

async function fetchStats(): Promise<Stats | null> {
  try {
    const res = await fetch(`${API_BASE}/api/v1/bug-bounty/stats`);
    const data = await res.json();
    return data.data;
  } catch (error) {
    console.error('Failed to fetch stats:', error);
    return null;
  }
}

async function fetchPrograms(): Promise<Program[]> {
  try {
    const res = await fetch(`${API_BASE}/api/v1/bug-bounty/programs`);
    const data = await res.json();
    return data.data || [];
  } catch (error) {
    console.error('Failed to fetch programs:', error);
    return [];
  }
}

async function fetchReports(programId?: string): Promise<BugReport[]> {
  try {
    const url = programId 
      ? `${API_BASE}/api/v1/bug-bounty/reports?program_id=${programId}`
      : `${API_BASE}/api/v1/bug-bounty/reports`;
    const res = await fetch(url);
    const data = await res.json();
    return data.data || [];
  } catch (error) {
    console.error('Failed to fetch reports:', error);
    return [];
  }
}

async function submitReport(report: Partial<BugReport>): Promise<BugReport | null> {
  try {
    const res = await fetch(`${API_BASE}/api/v1/bug-bounty/reports`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(report),
    });
    const data = await res.json();
    return data.data;
  } catch (error) {
    console.error('Failed to submit report:', error);
    return null;
  }
}

// ============================================================================
// All reports must come from the authenticated bug-bounty service.
const REPORTS: BugReport[] = [];

const SEVERITY_COLORS = {
  critical: 'bg-red-100 text-red-800 border-red-200',
  high: 'bg-orange-100 text-orange-800 border-orange-200',
  medium: 'bg-yellow-100 text-yellow-800 border-yellow-200',
  low: 'bg-green-100 text-green-800 border-green-200',
};

const REWARD_RANGES = [
  { severity: 'Critical', range: '$25,000 - $100,000', examples: 'Smart contract bugs, wallet drainage, privilege escalation' },
  { severity: 'High', range: '$5,000 - $25,000', examples: 'XSS, CSRF, authentication bypass' },
  { severity: 'Medium', range: '$1,000 - $5,000', examples: 'Information disclosure, minor security issues' },
  { severity: 'Low', range: '$100 - $1,000', examples: 'Minor bugs, UI issues, documentation' },
];

export default function BugBountyPage() {
  const [activeTab, setActiveTab] = useState<'programs' | 'leaderboard' | 'report'>('programs');
  const [showReportForm, setShowReportForm] = useState(false);
  const { isDark } = useTheme();

  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900 text-white' : 'bg-gray-50 text-gray-900'}`}>
      <header className={`${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'} border-b ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
        <div className="max-w-7xl mx-auto px-4">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-4">
              <a href="/" className="text-2xl">🐯</a>
              <h1 className="text-xl font-bold">Bug Bounty Program</h1>
            </div>
            <button
              onClick={() => setShowReportForm(true)}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
              Submit Report
            </button>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto px-4 py-8">
        {/* Hero */}
        <div className="bg-gradient-to-r from-purple-600 to-blue-600 rounded-2xl p-8 mb-8 text-white">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-3xl font-bold mb-2">TigerWallet Bug Bounty</h2>
              <p className="text-purple-100 mb-4">Help us secure the future of decentralized finance</p>
              <div className="flex gap-6">
                <div>
                  <p className="text-3xl font-bold">$50,000+</p>
                  <p className="text-purple-200">Max Reward</p>
                </div>
                <div>
                  <p className="text-3xl font-bold">$180,000+</p>
                  <p className="text-purple-200">Total Paid</p>
                </div>
                <div>
                  <p className="text-3xl font-bold">47</p>
                  <p className="text-purple-200">Bugs Fixed</p>
                </div>
              </div>
            </div>
            <div className="text-8xl">🦟</div>
          </div>
        </div>

        {/* Reward Tiers */}
        <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-xl p-6 mb-8 border ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
          <h3 className="text-xl font-bold mb-4">Reward Tiers</h3>
          <div className="grid grid-cols-4 gap-4">
            {REWARD_RANGES.map(tier => (
              <div key={tier.severity} className={`p-4 ${isDark ? 'bg-gray-700' : 'bg-gray-100'} rounded-lg`}>
                <p className="font-bold text-lg mb-1">{tier.severity}</p>
                <p className="text-blue-600 font-bold mb-2">{tier.range}</p>
                <p className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{tier.examples}</p>
              </div>
            ))}
          </div>
        </div>

        {/* Tabs */}
        <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} rounded-lg border ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
          <div className={`flex border-b ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
            <button
              onClick={() => setActiveTab('programs')}
              className={`px-6 py-4 font-medium ${activeTab === 'programs' ? 'text-blue-600 border-b-2 border-blue-600' : isDark ? 'text-gray-400' : 'text-gray-500'}`}
            >
              Active Programs
            </button>
            <button
              onClick={() => setActiveTab('leaderboard')}
              className={`px-6 py-4 font-medium ${activeTab === 'leaderboard' ? 'text-blue-600 border-b-2 border-blue-600' : isDark ? 'text-gray-400' : 'text-gray-500'}`}
            >
              Leaderboard
            </button>
            <button
              onClick={() => setActiveTab('report')}
              className={`px-6 py-4 font-medium ${activeTab === 'report' ? 'text-blue-600 border-b-2 border-blue-600' : isDark ? 'text-gray-400' : 'text-gray-500'}`}
            >
              Rules & Guidelines
            </button>
          </div>

          <div className="p-6">
            {activeTab === 'programs' && (
              <div className="space-y-4">
                {REPORTS.map(report => (
                  <div key={report.id} className={`flex items-center justify-between p-4 ${isDark ? 'bg-gray-700' : 'bg-gray-100'} rounded-lg`}>
                    <div className="flex items-center gap-4">
                      <div className="w-12 h-12 rounded-full bg-gradient-to-br from-purple-500 to-blue-600 flex items-center justify-center text-white font-bold">
                        {report.severity[0].toUpperCase()}
                      </div>
                      <div>
                        <p className="font-medium">{report.title}</p>
                        <p className={`text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Reported by {report.reporter}</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <span className={`px-3 py-1 rounded-full text-xs font-medium border ${SEVERITY_COLORS[report.severity]}`}>
                        {report.severity.toUpperCase()}
                      </span>
                      <span className={`px-3 py-1 rounded-full text-xs font-medium ${
                        report.status === 'rewarded' ? 'bg-green-100 text-green-800' :
                        report.status === 'fixed' ? 'bg-blue-100 text-blue-800' :
                        report.status === 'verified' ? 'bg-purple-100 text-purple-800' :
                        'bg-gray-100 text-gray-800'
                      }`}>
                        {report.status.toUpperCase()}
                      </span>
                      <p className="font-bold text-green-600">{report.reward}</p>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {activeTab === 'leaderboard' && (
              <div className="space-y-4">
                {[
                  { rank: 1, name: 'security_researcher_01', bugs: 8, reward: '$85,000' },
                  { rank: 2, name: 'defi_auditor', bugs: 5, reward: '$62,000' },
                  { rank: 3, name: 'crypto_expert', bugs: 4, reward: '$45,000' },
                  { rank: 4, name: 'web3_bug_hunter', bugs: 3, reward: '$28,000' },
                  { rank: 5, name: 'solidity_dev', bugs: 2, reward: '$15,000' },
                ].map(user => (
                  <div key={user.rank} className={`flex items-center justify-between p-4 ${isDark ? 'bg-gray-700' : 'bg-gray-100'} rounded-lg`}>
                    <div className="flex items-center gap-4">
                      <span className={`w-8 h-8 rounded-full flex items-center justify-center font-bold ${
                        user.rank === 1 ? 'bg-yellow-400 text-yellow-900' :
                        user.rank === 2 ? 'bg-gray-300 text-gray-700' :
                        user.rank === 3 ? 'bg-orange-400 text-orange-900' :
                        isDark ? 'bg-gray-600' : 'bg-gray-200'
                      }`}>
                        {user.rank}
                      </span>
                      <p className="font-medium">{user.name}</p>
                    </div>
                    <div className="flex items-center gap-6">
                      <p className={isDark ? 'text-gray-400' : 'text-gray-500'}>{user.bugs} bugs</p>
                      <p className="font-bold text-green-600">{user.reward}</p>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {activeTab === 'report' && (
              <div className="space-y-6">
                <div className={`p-4 border rounded-lg ${isDark ? 'bg-green-900/20 border-green-800' : 'bg-green-50 border-green-200'}`}>
                  <h4 className={`font-bold mb-2 ${isDark ? 'text-green-200' : 'text-green-800'}`}>In Scope</h4>
                  <ul className="list-disc list-inside space-y-1 text-sm">
                    <li>Smart contract vulnerabilities</li>
                    <li>Wallet security issues</li>
                    <li>Authentication/authorization bypass</li>
                    <li>Cross-site scripting (XSS)</li>
                    <li>Smart contract logic errors</li>
                    <li>Frontend vulnerabilities</li>
                  </ul>
                </div>

                <div className={`p-4 border rounded-lg ${isDark ? 'bg-red-900/20 border-red-800' : 'bg-red-50 border-red-200'}`}>
                  <h4 className={`font-bold mb-2 ${isDark ? 'text-red-200' : 'text-red-800'}`}>Out of Scope</h4>
                  <ul className="list-disc list-inside space-y-1 text-sm">
                    <li>Social engineering attacks</li>
                    <li>Physical security</li>
                    <li>Denial of service attacks</li>
                    <li>Issues in third-party services</li>
                    <li>Previously reported vulnerabilities</li>
                  </ul>
                </div>

                <div className={`p-4 border rounded-lg ${isDark ? 'bg-blue-900/20 border-blue-800' : 'bg-blue-50 border-blue-200'}`}>
                  <h4 className={`font-bold mb-2 ${isDark ? 'text-blue-200' : 'text-blue-800'}`}>How to Report</h4>
                  <ol className="list-decimal list-inside space-y-1 text-sm">
                    <li>Email security@tigerwallet.io with details</li>
                    <li>Include steps to reproduce</li>
                    <li>Provide proof of concept if possible</li>
                    <li>Do not disclose publicly until fixed</li>
                    <li>Response within 48 hours</li>
                  </ol>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Report Form Modal */}
      {showReportForm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'} rounded-xl p-6 max-w-lg w-full mx-4`}>
            <h3 className="text-xl font-bold mb-4">Submit Bug Report</h3>
            <div className="space-y-4">
              <input type="text" placeholder="Vulnerability Title" className="w-full p-3 border rounded-lg" />
              <select className="w-full p-3 border rounded-lg">
                <option>Select Severity</option>
                <option>Critical</option>
                <option>High</option>
                <option>Medium</option>
                <option>Low</option>
              </select>
              <textarea placeholder="Description and steps to reproduce..." className="w-full p-3 border rounded-lg h-32"></textarea>
              <div className="flex gap-4">
                <button onClick={() => setShowReportForm(false)} className="flex-1 p-3 bg-slate-200 rounded-lg">Cancel</button>
                <button className="flex-1 p-3 bg-blue-600 text-white rounded-lg">Submit</button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
