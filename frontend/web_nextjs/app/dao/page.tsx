'use client';

import React, { useState } from 'react';
import { useTheme } from '../components/ThemeProvider';

interface Proposal {
  id: number;
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

const MOCK_PROPOSALS: Proposal[] = [
  {
    id: 42,
    title: 'Increase TGR Staking Rewards to 25%',
    description: 'Proposal to increase the TGR staking rewards from 20% to 25% APY to encourage more staking participation and network security.',
    type: 'parameter',
    status: 'active',
    votesFor: 2500000,
    votesAgainst: 800000,
    totalVoters: 450,
    startTime: Date.now() - 86400000 * 2,
    endTime: Date.now() + 86400000 * 5,
    proposer: '0x7a23...8d4f',
    quorum: 5000000,
  },
  {
    id: 41,
    title: 'Add RUSD as Collateral for Loans',
    description: 'Enable RUSD stablecoin as collateral for the lending protocol to increase liquidity and utility.',
    type: 'parameter',
    status: 'active',
    votesFor: 1800000,
    votesAgainst: 200000,
    totalVoters: 280,
    startTime: Date.now() - 86400000,
    endTime: Date.now() + 86400000 * 6,
    proposer: '0x3f8b...2c1e',
    quorum: 5000000,
  },
  {
    id: 40,
    title: 'Treasury Diversification - Buy BTC',
    description: 'Use 10% of treasury (~$2M) to purchase Bitcoin as a reserve asset.',
    type: 'treasury',
    status: 'passed',
    votesFor: 4200000,
    votesAgainst: 600000,
    totalVoters: 680,
    startTime: Date.now() - 86400000 * 10,
    endTime: Date.now() - 86400000 * 3,
    proposer: '0x9b2d...5a4c',
    quorum: 5000000,
  },
  {
    id: 39,
    title: 'Upgrade DEX Fee Distribution',
    description: 'Change DEX fee distribution from 50/50 to 70/30 (protocol/stakers).',
    type: 'parameter',
    status: 'passed',
    votesFor: 3800000,
    votesAgainst: 1200000,
    totalVoters: 520,
    startTime: Date.now() - 86400000 * 15,
    endTime: Date.now() - 86400000 * 8,
    proposer: '0x1a2b...3c4d',
    quorum: 5000000,
  },
  {
    id: 38,
    title: 'Add New Bridge Partner',
    description: 'Integrate LayerZero as additional bridge partner for cross-chain swaps.',
    type: 'upgrade',
    status: 'failed',
    votesFor: 1500000,
    votesAgainst: 3800000,
    totalVoters: 380,
    startTime: Date.now() - 86400000 * 20,
    endTime: Date.now() - 86400000 * 13,
    proposer: '0x5c6d...7e8f',
    quorum: 5000000,
  },
];

const MOCK_DELEGATES: Delegate[] = [
  { address: '0x8a9c...d2e1', name: 'TigerDAO Core', votes: 5200000, proposals: 12, rank: 1, avatar: '🐯' },
  { address: '0x3b2d...f4a5', name: 'DeFi Whale', votes: 3800000, proposals: 8, rank: 2, avatar: '🐋' },
  { address: '0x7c4e...b8d9', name: 'Staking King', votes: 2500000, proposals: 5, rank: 3, avatar: '👑' },
  { address: '0x2f1a...c3e7', name: 'Community Lead', votes: 1800000, proposals: 15, rank: 4, avatar: '🎯' },
  { address: '0x9d5f...a1b2', name: 'Validator Pro', votes: 1200000, proposals: 3, rank: 5, avatar: '⚡' },
];

export default function DAOPage() {
  const { isDark } = useTheme();
  const [activeTab, setActiveTab] = useState<'proposals' | 'delegates' | 'treasury'>('proposals');
  const [voteAmount, setVoteAmount] = useState('');
  const [votingProposal, setVotingProposal] = useState<Proposal | null>(null);
  const [voteSide, setVoteSide] = useState<'for' | 'against'>('for');

  const handleVote = async () => {
    if (!votingProposal || !voteAmount) return;
    alert(`Successfully voted with ${voteAmount} TGR!`);
    setVoteAmount('');
    setVotingProposal(null);
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
        {/* Stats */}
        <div className="grid grid-cols-4 gap-4 mb-6">
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-4`}>
            <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Total Proposals</p>
            <p className="text-2xl font-bold">42</p>
          </div>
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-4`}>
            <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Active</p>
            <p className="text-2xl font-bold text-blue-600">2</p>
          </div>
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-4`}>
            <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Total Voters</p>
            <p className="text-2xl font-bold">1,250</p>
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
            {MOCK_PROPOSALS.map(proposal => {
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
            })}
          </div>
        )}

        {/* Delegates Tab */}
        {activeTab === 'delegates' && (
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-xl p-6`}>
            <h3 className="text-lg font-bold mb-4">Top Delegates</h3>
            <div className="space-y-4">
              {MOCK_DELEGATES.map(delegate => (
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
              ))}
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
                disabled={!voteAmount}
                className="flex-1 py-3 bg-indigo-600 text-white rounded-lg disabled:opacity-50"
              >
                Submit Vote
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
