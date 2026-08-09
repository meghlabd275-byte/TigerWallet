'use client';

import React, { useState, useCallback, useEffect } from 'react';
import { stakingApi, walletApi } from '../api/service';

interface StakingPosition {
  id: string;
  chainId: number;
  chainName: string;
  token: string;
  stakedAmount: string;
  reward: string;
  apy: number;
  validator: string;
  status: 'active' | 'unbonding' | 'claimed';
  startTime: number;
  unlockTime?: number;
}

interface StakingPool {
  id: string;
  name: string;
  token: string;
  chainId: number;
  chainName: string;
  apy: number;
  minStake: string;
  lockPeriod: number;
  totalStaked: string;
  rewardToken: string;
  description: string;
}

const CHAIN_NAMES: Record<number, string> = {
  1: 'Ethereum',
  56: 'BNB Chain',
  137: 'Polygon',
  42161: 'Arbitrum',
  10: 'Optimism',
  8453: 'Base',
  43114: 'Avalanche',
};

const STAKING_POOLS: StakingPool[] = [
  { id: 'lido', name: 'Lido Liquid Staking', token: 'ETH', chainId: 1, chainName: 'Ethereum', apy: 4.2, minStake: '0.01', lockPeriod: 0, totalStaked: '15.2B', rewardToken: 'stETH', description: 'Liquid staking - get stETH immediately' },
  { id: 'rocketpool', name: 'Rocket Pool Liquid Staking', token: 'ETH', chainId: 1, chainName: 'Ethereum', apy: 3.8, minStake: '0.01', lockPeriod: 0, totalStaked: '2.1B', rewardToken: 'rETH', description: 'Decentralized liquid staking' },
  { id: 'aave', name: 'Aave Staking', token: 'AAVE', chainId: 1, chainName: 'Ethereum', apy: 5.5, minStake: '1', lockPeriod: 0, totalStaked: '180M', rewardToken: 'AAVE', description: 'Stake AAVE for rewards' },
  { id: 'compound', name: 'Compound Staking', token: 'COMP', chainId: 1, chainName: 'Ethereum', apy: 4.8, minStake: '0.1', lockPeriod: 0, totalStaked: '120M', rewardToken: 'COMP', description: 'Stake COMP for governance rewards' },
  { id: 'solana', name: 'Solana Staking', token: 'SOL', chainId: 101, chainName: 'Solana', apy: 6.5, minStake: '1', lockPeriod: 2, totalStaked: '12.8B', rewardToken: 'SOL', description: 'Stake SOL with validators' },
  { id: 'polygon', name: 'Polygon Staking', token: 'MATIC', chainId: 137, chainName: 'Polygon', apy: 5.2, minStake: '10', lockPeriod: 4, totalStaked: '1.8B', rewardToken: 'MATIC', description: 'Stake Polygon' },
  { id: 'avalanche', name: 'Avalanche Staking', token: 'AVAX', chainId: 43114, chainName: 'Avalanche', apy: 8.1, minStake: '25', lockPeriod: 14, totalStaked: '2.5B', rewardToken: 'AVAX', description: 'Stake Avalanche' },
  { id: 'bsc', name: 'BNB Chain Staking', token: 'BNB', chainId: 56, chainName: 'BNB Chain', apy: 4.8, minStake: '1', lockPeriod: 7, totalStaked: '2.8B', rewardToken: 'BNB', description: 'Stake BNB' },
];

export default function Staking() {
  const [positions, setPositions] = useState<StakingPosition[]>([]);
  const [pools, setPools] = useState<StakingPool[]>(STAKING_POOLS);
  const [selectedPool, setSelectedPool] = useState<StakingPool | null>(null);
  const [stakeAmount, setStakeAmount] = useState('');
  const [activeTab, setActiveTab] = useState<'stake' | 'pools' | 'positions'>('pools');
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const handleStake = useCallback(async () => {
    if (!selectedPool || !stakeAmount || parseFloat(stakeAmount) < parseFloat(selectedPool.minStake)) {
      setMessage({ type: 'error', text: `Minimum stake is ${selectedPool?.minStake}` });
      return;
    }
    setLoading(true);
    await new Promise(resolve => setTimeout(resolve, 2000));
    const newPosition: StakingPosition = {
      id: `pos_${Date.now()}`,
      chainId: selectedPool.chainId,
      chainName: selectedPool.chainName,
      token: selectedPool.token,
      stakedAmount: stakeAmount,
      reward: '0',
      apy: selectedPool.apy,
      validator: selectedPool.name,
      status: 'active',
      startTime: Date.now(),
    };
    setPositions(prev => [...prev, newPosition]);
    setMessage({ type: 'success', text: `Successfully staked ${stakeAmount} ${selectedPool.token}!` });
    setStakeAmount('');
    setSelectedPool(null);
    setActiveTab('positions');
    setLoading(false);
  }, [selectedPool, stakeAmount]);

  const handleUnstake = useCallback(async (positionId: string) => {
    setLoading(true);
    await new Promise(resolve => setTimeout(resolve, 1500));
    setPositions(prev => prev.map(p => p.id === positionId ? { ...p, status: 'unbonding' as const, unlockTime: Date.now() + 86400000 * 14 } : p));
    setMessage({ type: 'success', text: 'Unstake initiated! Your tokens will be available after the lock period.' });
    setLoading(false);
  }, []);

  const handleClaim = useCallback(async (positionId: string) => {
    setLoading(true);
    await new Promise(resolve => setTimeout(resolve, 1000));
    setPositions(prev => prev.filter(p => p.id !== positionId));
    setMessage({ type: 'success', text: 'Rewards claimed successfully!' });
    setLoading(false);
  }, []);

  const formatTime = (timestamp: number): string => {
    const diff = Date.now() - timestamp;
    const days = Math.floor(diff / 86400000);
    if (days < 1) return 'Today';
    if (days === 1) return 'Yesterday';
    return `${days} days ago`;
  };

  const totalStaked = positions.reduce((acc, p) => acc + parseFloat(p.stakedAmount), 0);
  const totalRewards = positions.reduce((acc, p) => acc + parseFloat(p.reward), 0);

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-slate-50">
      <header className="bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-4"><a href="/" className="text-2xl">🐯</a><h1 className="text-xl font-bold">Staking</h1></div>
            <nav className="flex gap-4"><a href="/wallet" className="text-slate-600 dark:text-slate-400 hover:text-orange-500">Wallet</a></nav>
          </div>
        </div>
      </header>
      {message && <div className="fixed top-20 right-4 z-50"><div className={`px-6 py-3 rounded-lg shadow-lg ${message.type === 'success' ? 'bg-green-500' : 'bg-red-500'} text-white`}>{message.text}</div></div>}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Stats */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4"><div className="text-slate-500 text-sm">Total Staked</div><div className="text-2xl font-bold">{totalStaked.toFixed(2)}</div></div>
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4"><div className="text-slate-500 text-sm">Total Rewards</div><div className="text-2xl font-bold text-green-500">{totalRewards.toFixed(4)}</div></div>
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4"><div className="text-slate-500 text-sm">Active Positions</div><div className="text-2xl font-bold">{positions.filter(p => p.status === 'active').length}</div></div>
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4"><div className="text-slate-500 text-sm">Avg APY</div><div className="text-2xl font-bold text-orange-500">{positions.length > 0 ? (positions.reduce((a, p) => a + p.apy, 0) / positions.length).toFixed(1) : 0}%</div></div>
        </div>
        {/* Tabs */}
        <div className="flex border-b border-slate-200 dark:border-slate-700 mb-6">
          <button onClick={() => setActiveTab('pools')} className={`px-6 py-3 ${activeTab === 'pools' ? 'border-b-2 border-orange-500 text-orange-500' : 'text-slate-500'}`}>Staking Pools</button>
          <button onClick={() => setActiveTab('positions')} className={`px-6 py-3 ${activeTab === 'positions' ? 'border-b-2 border-orange-500 text-orange-500' : 'text-slate-500'}`}>My Positions</button>
        </div>
        {/* Pools Tab */}
        {activeTab === 'pools' && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {STAKING_POOLS.map((pool) => (
              <div key={pool.id} className="bg-white dark:bg-slate-800 rounded-lg p-4 shadow-sm">
                <div className="flex items-center justify-between mb-3"><div><div className="font-semibold">{pool.name}</div><div className="text-xs text-slate-500">{pool.chainName}</div></div><span className="bg-orange-100 text-orange-600 px-2 py-1 rounded text-sm font-semibold">{pool.apy}% APY</span></div>
                <div className="text-sm text-slate-500 mb-3">{pool.description}</div>
                <div className="grid grid-cols-2 gap-2 text-xs mb-3"><div>Min: {pool.minStake} {pool.token}</div><div>Lock: {pool.lockPeriod} days</div><div className="col-span-2">Total: ${pool.totalStaked}</div></div>
                <button onClick={() => { setSelectedPool(pool); setActiveTab('stake'); }} className="w-full bg-orange-500 hover:bg-orange-600 text-white py-2 rounded-lg">Stake</button>
              </div>
            ))}
          </div>
        )}
        {/* Stake Tab */}
        {activeTab === 'stake' && selectedPool && (
          <div className="max-w-md mx-auto bg-white dark:bg-slate-800 rounded-lg p-6">
            <h2 className="text-xl font-semibold mb-4">Stake {selectedPool.token}</h2>
            <div className="bg-slate-100 dark:bg-slate-700 rounded-lg p-3 mb-4"><div className="text-sm text-slate-500">Pool</div><div className="font-semibold">{selectedPool.name}</div></div>
            <div className="mb-4"><label className="block text-sm text-slate-500 mb-2">Amount</label><input type="number" value={stakeAmount} onChange={(e) => setStakeAmount(e.target.value)} placeholder={`Min: ${selectedPool.minStake}`} className="w-full bg-slate-100 dark:bg-slate-700 border-0 rounded-lg px-4 py-3 text-xl" /></div>
            <div className="bg-slate-100 dark:bg-slate-700 rounded-lg p-3 mb-4"><div className="grid grid-cols-2 gap-2 text-sm"><div><span className="text-slate-500">APY</span><div className="font-semibold text-orange-500">{selectedPool.apy}%</div></div><div><span className="text-slate-500">Lock Period</span><div className="font-semibold">{selectedPool.lockPeriod} days</div></div></div></div>
            <div className="flex gap-4"><button onClick={() => setActiveTab('pools')} className="flex-1 bg-slate-200 dark:bg-slate-700 py-3 rounded-lg">Cancel</button><button onClick={handleStake} disabled={loading} className="flex-1 bg-orange-500 hover:bg-orange-600 disabled:bg-slate-400 text-white py-3 rounded-lg">{loading ? 'Staking...' : 'Stake Now'}</button></div>
          </div>
        )}
        {/* Positions Tab */}
        {activeTab === 'positions' && (
          <div className="space-y-4">
            {positions.length === 0 ? <div className="bg-white dark:bg-slate-800 rounded-lg p-12 text-center"><div className="text-6xl mb-4">🎯</div><h3 className="text-xl font-semibold mb-2">No Staking Positions</h3><p className="text-slate-500">Start staking to earn rewards</p><button onClick={() => setActiveTab('pools')} className="mt-4 bg-orange-500 hover:bg-orange-600 text-white px-6 py-2 rounded-lg">View Pools</button></div> : positions.map((pos) => (
              <div key={pos.id} className="bg-white dark:bg-slate-800 rounded-lg p-4 shadow-sm">
                <div className="flex items-center justify-between mb-3"><div><div className="font-semibold">{pos.token} Staking</div><div className="text-sm text-slate-500">{pos.validator} • {pos.chainName}</div></div><span className={`px-2 py-1 rounded text-xs ${pos.status === 'active' ? 'bg-green-100 text-green-600' : 'bg-yellow-100 text-yellow-600'}`}>{pos.status.toUpperCase()}</span></div>
                <div className="grid grid-cols-3 gap-4 mb-3"><div><div className="text-xs text-slate-500">Staked</div><div className="font-semibold">{pos.stakedAmount} {pos.token}</div></div><div><div className="text-xs text-slate-500">Rewards</div><div className="font-semibold text-green-500">{pos.reward} {pos.token}</div></div><div><div className="text-xs text-slate-500">APY</div><div className="font-semibold text-orange-500">{pos.apy}%</div></div></div>
                <div className="flex gap-2"><button onClick={() => handleClaim(pos.id)} disabled={loading || parseFloat(pos.reward) === 0} className="flex-1 bg-green-500 hover:bg-green-600 disabled:bg-slate-400 text-white py-2 rounded-lg">Claim Rewards</button><button onClick={() => handleUnstake(pos.id)} disabled={loading} className="flex-1 bg-red-500 hover:bg-red-600 disabled:bg-slate-400 text-white py-2 rounded-lg">Unstake</button></div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
