'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../components/ThemeProvider';

const API_BASE_URL = typeof window !== 'undefined' ? '' : (process.env.BACKEND_URL || 'http://localhost:8443');

const fetchAPI = async <T,>(endpoint: string, options?: RequestInit): Promise<T> => {
  const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
  const response = await fetch(`${API_BASE_URL}/api/v1${endpoint}`, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}), ...options?.headers },
  });
  if (!response.ok) throw new Error(`API Error: ${response.statusText}`);
  const data = await response.json();
  return data.data;
};

interface Proposal {
  id: string;
  title: string;
  description: string;
  type: 'parameter' | 'treasury' | 'upgrade' | 'governance';
  status: 'active' | 'passed' | 'failed' | 'executed';
  votesFor: number;
  votesAgainst: number;
  totalVoters: number;
  startTime: number;
  endTime: number;
  proposer: string;
  quorum: number;
}

interface Delegate {
  address: string;
  name: string;
  votes: number;
  proposals: number;
  rank: number;
  avatar: string;
}

interface BackendProposal {
  id: string;
  title: string;
  description: string;
  proposer: string;
  proposer_name: string;
  for_votes: string;
  against_votes: string;
  abstain_votes: string;
  status: string;
  start_time: number;
  end_time: number;
  executed: boolean;
}

interface BackendDelegate {
  id: string;
  address: string;
  name: string;
  voting_power: string;
  proposals_count: number;
  delegated_to?: string;
}

const PROPOSAL_AVATARS = ['🐯', '🐋', '👑', '🎯', '⚡'];

function mapProposal(p: BackendProposal, idx: number): Proposal {
  const now = Date.now();
  const endMs = p.end_time * 1000;
  let status: Proposal['status'] = 'active';
  if (p.executed) status = 'executed';
  else if (now >= endMs) status = parseFloat(p.for_votes) >= parseFloat(p.against_votes) ? 'passed' : 'failed';
  const votesFor = parseFloat(p.for_votes) || 0;
  const votesAgainst = parseFloat(p.against_votes) || 0;
  return {
    id: p.id,
    title: p.title,
    description: p.description,
    type: 'governance',
    status,
    votesFor,
    votesAgainst,
    totalVoters: 0,
    startTime: p.start_time * 1000,
    endTime: endMs,
    proposer: p.proposer_name || p.proposer,
    quorum: 5000000,
  };
}

function mapDelegate(d: BackendDelegate, idx: number): Delegate {
  return {
    address: d.address,
    name: d.name,
    votes: parseFloat(d.voting_power) || 0,
    proposals: d.proposals_count,
    rank: idx + 1,
    avatar: PROPOSAL_AVATARS[idx % PROPOSAL_AVATARS.length],
  };
}

export default function DAOPage() {
  const { isDark } = useTheme();
  const [activeTab, setActiveTab] = useState<'proposals' | 'delegates' | 'treasury'>('proposals');
  const [voteAmount, setVoteAmount] = useState('');
  const [votingProposal, setVotingProposal] = useState<Proposal | null>(null);
  const [voteSide, setVoteSide] = useState<'for' | 'against'>('for');
  const [proposals, setProposals] = useState<Proposal[]>([]);
  const [delegates, setDelegates] = useState<Delegate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [props, dels] = await Promise.all([
        fetchAPI<BackendProposal[]>('/dao/proposals').catch(() => []),
        fetchAPI<BackendDelegate[]>('/dao/delegates').catch(() => []),
      ]);
      setProposals((props || []).map(mapProposal));
      setDelegates((dels || []).map(mapDelegate));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load DAO data');
      setProposals([]);
      setDelegates([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleVote = async () => {
    if (!votingProposal || !voteAmount) return;
    setSubmitting(true);
    setError(null);
    try {
      await fetchAPI(`/dao/proposals/${votingProposal.id}/vote`, {
        method: 'POST',
        body: JSON.stringify({ choice: voteSide, voting_power: voteAmount }),
      });
      alert(`Successfully voted ${voteSide.toUpperCase()} with ${voteAmount} TGR!`);
      setVoteAmount('');
      setVotingProposal(null);
      await loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit vote');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelegate = async (delegate: Delegate) => {
    alert(`Successfully delegated to ${delegate.name}!`);
  };

  return (
    <div className={`min-h-screen ${isDark ? 'bg-slate-900 text-white' : 'bg-slate-50 text-slate-900'}`}>
      <header className="bg-gradient-to-r from-indigo-600 to-purple-600 text-white">
        <div className="max-w-7xl mx-auto px-4 py-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <a href="/" className="text-3xl">🐯</a>
              <div>
                <h1 className="text-2xl font-bold">TigerDAO Governance</h1>
                <p className="text-indigo-200">Decentralized governance for Tiger Ecosystem</p>
              </div>
            </div>
            <div className="text-right">
              <p className="text-sm text-indigo-200">Your TGR Balance</p>
              <p className="text-2xl font-bold">12,500 TGR</p>
            </div>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto px-4 py-6">
        {error && (
          <div className="mb-4 p-3 rounded-lg bg-red-600 text-white">Error: {error}</div>
        )}
        {/* Stats */}
        <div className="grid grid-cols-4 gap-4 mb-6">
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-4`}>
            <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Total Proposals</p>
            <p className="text-2xl font-bold">{proposals.length}</p>
          </div>
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-4`}>
            <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Active</p>
            <p className="text-2xl font-bold text-blue-600">{proposals.filter(p => p.status === 'active').length}</p>
          </div>
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-4`}>
            <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Total Delegates</p>
            <p className="text-2xl font-bold">{delegates.length}</p>
          </div>
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-4`}>
            <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Treasury</p>
            <p className="text-2xl font-bold">$22.5M</p>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex gap-2 mb-6">
          {(['proposals', 'delegates', 'treasury'] as const).map(tab => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-6 py-2 rounded-lg font-medium ${activeTab === tab ? 'bg-indigo-600 text-white' : `${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'}`}`}
            >
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
            </button>
          ))}
        </div>

        {/* Proposals Tab */}
        {activeTab === 'proposals' && (
          <div className="space-y-4">
            {loading ? (
              <div className={`text-center py-12 animate-pulse ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Loading proposals…</div>
            ) : proposals.length === 0 ? (
              <div className={`text-center py-12 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>No proposals yet</div>
            ) : (
              proposals.map(proposal => {
              const totalVotes = proposal.votesFor + proposal.votesAgainst;
              const forPercent = totalVotes > 0 ? (proposal.votesFor / totalVotes) * 100 : 0;
              const quorumPercent = (totalVotes / proposal.quorum) * 100;
              
              return (
                <div key={proposal.id} className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-xl p-6`}>
                  <div className="flex items-start justify-between mb-4">
                    <div>
                      <div className="flex items-center gap-3 mb-2">
                        <span className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>#{proposal.id}</span>
                        <span className={`px-2 py-1 rounded text-xs ${
                          proposal.type === 'parameter' ? 'bg-blue-100 text-blue-800' :
                          proposal.type === 'treasury' ? 'bg-purple-100 text-purple-800' :
                          proposal.type === 'upgrade' ? 'bg-orange-100 text-orange-800' :
                          'bg-green-100 text-green-800'
                        }`}>
                          {proposal.type.toUpperCase()}
                        </span>
                        <span className={`px-2 py-1 rounded text-xs ${
                          proposal.status === 'active' ? 'bg-green-100 text-green-800' :
                          proposal.status === 'passed' ? 'bg-blue-100 text-blue-800' :
                          proposal.status === 'failed' ? 'bg-red-100 text-red-800' :
                          'bg-purple-100 text-purple-800'
                        }`}>
                          {proposal.status.toUpperCase()}
                        </span>
                      </div>
                      <h3 className="text-lg font-bold">{proposal.title}</h3>
                      <p className={`text-sm mt-1 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>by {proposal.proposer}</p>
                    </div>
                    {proposal.status === 'active' && (
                      <button
                        onClick={() => { setVotingProposal(proposal); setVoteSide('for'); }}
                        className="px-4 py-2 bg-indigo-600 text-white rounded-lg"
                      >
                        Vote
                      </button>
                    )}
                  </div>
                  
                  <p className={`mb-4 ${isDark ? 'text-slate-300' : 'text-slate-600'}`}>{proposal.description}</p>
                  
                  {/* Vote Progress */}
                  <div className="mb-4">
                    <div className="flex justify-between text-sm mb-2">
                      <span className="text-green-600">{proposal.votesFor.toLocaleString()} votes FOR</span>
                      <span className="text-red-600">{proposal.votesAgainst.toLocaleString()} votes AGAINST</span>
                    </div>
                    <div className={`h-3 rounded-full overflow-hidden ${isDark ? 'bg-slate-700' : 'bg-slate-200'}`}>
                      <div className="h-full flex">
                        <div className="bg-green-500" style={{ width: `${forPercent}%` }}></div>
                        <div className="bg-red-500" style={{ width: `${100 - forPercent}%` }}></div>
                      </div>
                    </div>
                  </div>
                  
                  <div className="flex items-center justify-between text-sm">
                    <span className={isDark ? 'text-slate-400' : 'text-slate-500'}>
                      {Math.floor((proposal.endTime - Date.now()) / 86400000)} days remaining
                    </span>
                    <span className={isDark ? 'text-slate-400' : 'text-slate-500'}>
                      Quorum: {quorumPercent.toFixed(1)}% ({totalVotes.toLocaleString()} / {proposal.quorum.toLocaleString()})
                    </span>
                  </div>
                </div>
              );
            })
          )}
          </div>
        )}

        {/* Delegates Tab */}
        {activeTab === 'delegates' && (
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-xl p-6`}>
            <h3 className="text-lg font-bold mb-4">Top Delegates</h3>
            <div className="space-y-4">
              {loading ? (
                <div className={`text-center py-12 animate-pulse ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Loading delegates…</div>
              ) : delegates.length === 0 ? (
                <div className={`text-center py-12 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>No delegates yet</div>
              ) : (
                delegates.map(delegate => (
                <div key={delegate.rank} className={`flex items-center justify-between p-4 ${isDark ? 'bg-slate-700' : 'bg-slate-50'} rounded-lg`}>
                  <div className="flex items-center gap-4">
                    <span className={`w-8 h-8 rounded-full flex items-center justify-center font-bold ${
                      delegate.rank === 1 ? 'bg-yellow-400 text-yellow-900' :
                      delegate.rank === 2 ? 'bg-gray-300 text-gray-700' :
                      delegate.rank === 3 ? 'bg-orange-400 text-orange-900' :
                      isDark ? 'bg-slate-600' : 'bg-slate-200'
                    }`}>
                      {delegate.rank}
                    </span>
                    <span className="text-2xl">{delegate.avatar}</span>
                    <div>
                      <p className="font-semibold">{delegate.name}</p>
                      <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>{delegate.address}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-6">
                    <div className="text-right">
                      <p className="font-bold">{delegate.votes.toLocaleString()}</p>
                      <p className={`text-xs ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>TGR votes</p>
                    </div>
                    <div className="text-right">
                      <p className="font-bold">{delegate.proposals}</p>
                      <p className={`text-xs ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>proposals</p>
                    </div>
                    <button
                      onClick={() => handleDelegate(delegate)}
                      className="px-4 py-2 bg-indigo-600 text-white rounded-lg"
                    >
                      Delegate
                    </button>
                  </div>
                </div>
                ))
            )}
            </div>
          </div>
        )}

        {/* Treasury Tab */}
        {activeTab === 'treasury' && (
          <div className="space-y-4">
            <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-xl p-6`}>
              <h3 className="text-lg font-bold mb-4">Treasury Holdings</h3>
              <div className="grid grid-cols-3 gap-4">
                <div className={`p-4 ${isDark ? 'bg-slate-700' : 'bg-slate-50'} rounded-lg`}>
                  <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>TGR</p>
                  <p className="text-2xl font-bold">15,000,000</p>
                  <p className="text-sm text-green-600">$3,750,000</p>
                </div>
                <div className={`p-4 ${isDark ? 'bg-slate-700' : 'bg-slate-50'} rounded-lg`}>
                  <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>USDT</p>
                  <p className="text-2xl font-bold">8,500,000</p>
                  <p className="text-sm">$8,500,000</p>
                </div>
                <div className={`p-4 ${isDark ? 'bg-slate-700' : 'bg-slate-50'} rounded-lg`}>
                  <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>RUSD</p>
                  <p className="text-2xl font-bold">5,000,000</p>
                  <p className="text-sm">$5,000,000</p>
                </div>
              </div>
            </div>
            
            <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-xl p-6`}>
              <h3 className="text-lg font-bold mb-4">Recent Treasury Transactions</h3>
              <div className="space-y-3">
                {[
                  { type: 'Income', amount: '50,000 USDT', from: 'Swap Fees', date: '2 hours ago' },
                  { type: 'Expense', amount: '25,000 TGR', from: 'Staking Rewards', date: '1 day ago' },
                  { type: 'Income', amount: '100,000 USDT', from: 'Trading Fees', date: '3 days ago' },
                  { type: 'Expense', amount: '500,000 TGR', from: 'Team Vesting', date: '1 week ago' },
                ].map((tx, i) => (
                  <div key={i} className={`flex items-center justify-between p-3 ${isDark ? 'bg-slate-700' : 'bg-slate-50'} rounded-lg`}>
                    <div>
                      <p className="font-medium">{tx.type}</p>
                      <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>{tx.from}</p>
                    </div>
                    <div className="text-right">
                      <p className={`font-bold ${tx.type === 'Income' ? 'text-green-600' : 'text-red-600'}`}>
                        {tx.type === 'Income' ? '+' : '-'}{tx.amount}
                      </p>
                      <p className={`text-xs ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>{tx.date}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Vote Modal */}
      {votingProposal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-xl p-6 max-w-md`}>
            <h3 className="text-xl font-bold mb-4">Vote on Proposal #{votingProposal.id}</h3>
            
            <div className="flex gap-2 mb-4">
              <button
                onClick={() => setVoteSide('for')}
                className={`flex-1 py-3 rounded-lg font-bold ${voteSide === 'for' ? 'bg-green-600 text-white' : isDark ? 'bg-slate-700' : 'bg-slate-200'}`}
              >
                Vote FOR
              </button>
              <button
                onClick={() => setVoteSide('against')}
                className={`flex-1 py-3 rounded-lg font-bold ${voteSide === 'against' ? 'bg-red-600 text-white' : isDark ? 'bg-slate-700' : 'bg-slate-200'}`}
              >
                Vote AGAINST
              </button>
            </div>
            
            <div className="mb-4">
              <label className="block text-sm font-medium mb-2">Vote Weight (TGR)</label>
              <input
                type="number"
                value={voteAmount}
                onChange={(e) => setVoteAmount(e.target.value)}
                placeholder="Enter TGR amount"
                className="w-full p-3 border rounded-lg"
              />
            </div>
            
            <div className="flex gap-4">
              <button
                onClick={() => setVotingProposal(null)}
                className={`flex-1 py-3 rounded-lg ${isDark ? 'bg-slate-700' : 'bg-slate-200'}`}
              >
                Cancel
              </button>
              <button
                onClick={handleVote}
                disabled={!voteAmount || submitting}
                className="flex-1 py-3 bg-indigo-600 text-white rounded-lg disabled:opacity-50"
              >
                {submitting ? 'Submitting…' : 'Submit Vote'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
