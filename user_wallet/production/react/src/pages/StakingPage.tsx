/**
 * Staking Page - Stake tokens and earn rewards
 */

import React, { useState } from 'react';
import { useTheme } from '../contexts/ThemeContext';

function StakingPage() {
  const { theme } = useTheme();
  const [activeTab, setActiveTab] = useState('staking');

  const stakingPools = [
    { token: 'ETH', apy: '4.2%', staked: '12.5 ETH', rewards: '0.52 ETH', lockPeriod: 'None' },
    { token: 'MATIC', apy: '5.8%', staked: '5,000 MATIC', rewards: '290 MATIC', lockPeriod: '30 days' },
    { token: 'SOL', apy: '6.5%', staked: '150 SOL', rewards: '9.75 SOL', lockPeriod: 'None' },
    { token: 'BNB', apy: '4.8%', staked: '25 BNB', rewards: '1.2 BNB', lockPeriod: 'None' },
  ];

  const validators = [
    { name: 'Lido', apy: '4.2%', tvl: '$15.2B', fee: '10%' },
    { name: 'Rocket Pool', apy: '4.0%', tvl: '$2.1B', fee: '15%' },
    { name: 'Stakewise', apy: '4.1%', tvl: '$890M', fee: '10%' },
  ];

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Staking</h1>

      <div className="flex gap-2 mb-6">
        <button onClick={() => setActiveTab('mystakes')} className={`px-4 py-2 rounded-lg ${activeTab === 'mystakes' ? 'bg-amber-500 text-black' : theme === 'dark' ? 'bg-slate-800' : 'bg-gray-200'}`}>My Stakes</button>
        <button onClick={() => setActiveTab('validators')} className={`px-4 py-2 rounded-lg ${activeTab === 'validators' ? 'bg-amber-500 text-black' : theme === 'dark' ? 'bg-slate-800' : 'bg-gray-200'}`}>Validators</button>
        <button onClick={() => setActiveTab('earn')} className={`px-4 py-2 rounded-lg ${activeTab === 'earn' ? 'bg-amber-500 text-black' : theme === 'dark' ? 'bg-slate-800' : 'bg-gray-200'}`}>Earn</button>
      </div>

      {activeTab === 'mystakes' && (
        <div className="space-y-4">
          {stakingPools.map((pool, i) => (
            <div key={i} className={`card ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
              <div className="flex justify-between items-start mb-4">
                <div>
                  <h3 className="text-xl font-bold text-amber-500">{pool.token} Staking</h3>
                  <p className="text-sm opacity-60">APY: {pool.apy}</p>
                </div>
                <span className="badge badge-success">Active</span>
              </div>
              <div className="grid grid-cols-2 gap-4 mb-4">
                <div><p className="text-sm opacity-60">Staked</p><p className="font-semibold">{pool.staked}</p></div>
                <div><p className="text-sm opacity-60">Rewards</p><p className="font-semibold text-green-500">{pool.rewards}</p></div>
              </div>
              <div className="flex gap-2">
                <button className="btn btn-primary flex-1">Claim</button>
                <button className="btn btn-secondary flex-1">Stake More</button>
              </div>
            </div>
          ))}
        </div>
      )}

      {activeTab === 'validators' && (
        <div className="space-y-4">
          {validators.map((v, i) => (
            <div key={i} className={`card ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
              <div className="flex justify-between items-center">
                <div><h3 className="font-bold">{v.name}</h3><p className="text-sm opacity-60">TVL: {v.tvl}</p></div>
                <div className="text-right"><p className="text-xl font-bold text-green-500">{v.apy}</p><p className="text-sm opacity-60">Fee: {v.fee}</p></div>
              </div>
              <button className="btn btn-primary w-full mt-4">Stake with {v.name}</button>
            </div>
          ))}
        </div>
      )}

      {activeTab === 'earn' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className={`card ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
            <h3 className="font-bold text-lg mb-2">Liquid Staking</h3>
            <p className="text-sm opacity-60 mb-4">Stake and get liquid tokens</p>
            <p className="text-2xl font-bold text-green-500 mb-4">4.2% APY</p>
            <button className="btn btn-primary w-full">Learn More</button>
          </div>
          <div className={`card ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
            <h3 className="font-bold text-lg mb-2">Lock Staking</h3>
            <p className="text-sm opacity-60 mb-4">Lock for higher APY</p>
            <p className="text-2xl font-bold text-green-500 mb-4">6.5% APY</p>
            <button className="btn btn-primary w-full">Learn More</button>
          </div>
        </div>
      )}
    </div>
  );
}

export default StakingPage;
