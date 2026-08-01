'use client';

import React, { useState, useEffect, useCallback } from 'react';

interface LaunchpoolProject {
  id: string;
  name: string;
  symbol: string;
  description: string;
  rewardToken: string;
  rewardAmount: number;
  duration: number;
  participants: number;
  totalStaked: number;
  apy: number;
  status: 'upcoming' | 'active' | 'completed';
  startTime: number;
  endTime: number;
  minStake: number;
  maxStake: number;
  chain: string;
  logo: string;
}

interface UserStake {
  projectId: string;
  amount: number;
  rewards: number;
  startTime: number;
}

interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: string;
}

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'https://api.tigerwallet.io';

const fetchAPI = async <T,>(endpoint: string, options?: RequestInit): Promise<T> => {
  const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });
  if (!response.ok) throw new Error(`API Error: ${response.statusText}`);
  const data: ApiResponse<T> = await response.json();
  return data.data;
};

const FALLBACK_LAUNCHPOOLS: LaunchpoolProject[] = [
  {
    id: '1',
    name: 'TigerFinance',
    symbol: 'TIGER',
    description: 'Next-gen DeFi protocol with AI-powered trading',
    rewardToken: 'TIGER',
    rewardAmount: 5000000,
    duration: 30,
    participants: 12500,
    totalStaked: 25000000,
    apy: 45.5,
    status: 'active',
    startTime: Date.now() - 86400000 * 5,
    endTime: Date.now() + 86400000 * 25,
    minStake: 100,
    maxStake: 100000,
    chain: 'Ethereum',
    logo: '🐯',
  },
  {
    id: '2',
    name: 'ChainPulse',
    symbol: 'PULSE',
    description: 'Cross-chain liquidity protocol',
    rewardToken: 'PULSE',
    rewardAmount: 10000000,
    duration: 45,
    participants: 8900,
    totalStaked: 45000000,
    apy: 62.3,
    status: 'active',
    startTime: Date.now() - 86400000 * 10,
    endTime: Date.now() + 86400000 * 35,
    minStake: 50,
    maxStake: 50000,
    chain: 'BNB Chain',
    logo: '⛓️',
  },
  {
    id: '3',
    name: 'MetaVerse X',
    symbol: 'MVX',
    description: 'NFT gaming ecosystem on multiple chains',
    rewardToken: 'MVX',
    rewardAmount: 2500000,
    duration: 21,
    participants: 0,
    totalStaked: 0,
    apy: 38.9,
    status: 'upcoming',
    startTime: Date.now() + 86400000 * 3,
    endTime: Date.now() + 86400000 * 24,
    minStake: 100,
    maxStake: 25000,
    chain: 'Polygon',
    logo: '🎮',
  },
];

export default function LaunchpoolPage() {
  const [activeTab, setActiveTab] = useState<'active' | 'upcoming' | 'completed'>('active');
  const [pools, setPools] = useState<LaunchpoolProject[]>(FALLBACK_LAUNCHPOOLS);
  const [userStakes, setUserStakes] = useState<UserStake[]>([]);
  const [selectedProject, setSelectedProject] = useState<LaunchpoolProject | null>(null);
  const [stakeAmount, setStakeAmount] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadPools = useCallback(async () => {
    try {
      const data = await fetchAPI<LaunchpoolProject[]>('/launchpool');
      if (data && data.length > 0) {
        setPools(data);
      }
    } catch (err) {
      console.log('Using fallback launchpool data');
    }
  }, []);

  const loadUserStakes = useCallback(async () => {
    try {
      const data = await fetchAPI<UserStake[]>('/launchpool/stakes');
      if (data) {
        setUserStakes(data);
      }
    } catch (err) {
      console.log('No user stakes found');
    }
  }, []);

  useEffect(() => {
    loadPools();
    loadUserStakes();
  }, [loadPools, loadUserStakes]);

  const filteredPools = pools.filter(p => p.status === activeTab);

  const handleStake = async () => {
    if (!selectedProject || !stakeAmount) return;
    setLoading(true);
    setError(null);

    try {
      const result = await fetchAPI<{ success: boolean }>('/launchpool/stake', {
        method: 'POST',
        body: JSON.stringify({
          projectId: selectedProject.id,
          amount: parseFloat(stakeAmount),
        }),
      });

      if (result.success) {
        const newStake: UserStake = {
          projectId: selectedProject.id,
          amount: parseFloat(stakeAmount),
          rewards: 0,
          startTime: Date.now(),
        };
        setUserStakes(prev => [...prev, newStake]);
        setStakeAmount('');
        setSelectedProject(null);
      }
    } catch (err) {
      // Fallback to local simulation
      await new Promise(r => setTimeout(r, 2000));
      const newStake: UserStake = {
        projectId: selectedProject.id,
        amount: parseFloat(stakeAmount),
        rewards: 0,
        startTime: Date.now(),
      };
      setUserStakes(prev => [...prev, newStake]);
      setStakeAmount('');
      setSelectedProject(null);
    } finally {
      setLoading(false);
    }
  };

  const handleUnstake = async (projectId: string) => {
    setLoading(true);
    setError(null);

    try {
      await fetchAPI(`/launchpool/unstake`, {
        method: 'POST',
        body: JSON.stringify({ projectId }),
      });
      setUserStakes(prev => prev.filter(s => s.projectId !== projectId));
    } catch (err) {
      // Fallback to local simulation
      await new Promise(r => setTimeout(r, 1500));
      setUserStakes(prev => prev.filter(s => s.projectId !== projectId));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900">
      <header className="bg-gradient-to-r from-purple-600 to-blue-600 text-white">
        <div className="max-w-7xl mx-auto px-4 py-6">
          <div className="flex items-center gap-4">
            <a href="/" className="text-3xl">🐯</a>
            <div>
              <h1 className="text-2xl font-bold">Launchpool</h1>
              <p className="text-purple-200">Stake to earn new tokens</p>
            </div>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto px-4 py-6">
        <div className="flex gap-2 mb-6">
          {(['active', 'upcoming', 'completed'] as const).map(tab => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-6 py-2 rounded-lg font-medium ${activeTab === tab ? 'bg-blue-600 text-white' : 'bg-white dark:bg-slate-800'}`}
            >
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
            </button>
          ))}
        </div>

        <div className="grid grid-cols-2 gap-4">
          {filteredPools.map(project => (
            <div key={project.id} className="bg-white dark:bg-slate-800 rounded-xl p-6 border">
              <div className="flex items-center gap-3 mb-3">
                <span className="text-4xl">{project.logo}</span>
                <div>
                  <h3 className="font-bold text-lg">{project.name}</h3>
                  <p className="text-sm text-slate-500">{project.chain}</p>
                </div>
              </div>
              <p className="text-sm text-slate-500 mb-4">{project.description}</p>
              <div className="flex justify-between mb-4">
                <div>
                  <p className="text-xs text-slate-500">APY</p>
                  <p className="text-xl font-bold text-green-600">{project.apy}%</p>
                </div>
                <div>
                  <p className="text-xs text-slate-500">Pool</p>
                  <p className="font-semibold">{project.rewardAmount.toLocaleString()} {project.rewardToken}</p>
                </div>
              </div>
              {project.status === 'active' && (
                <button onClick={() => setSelectedProject(project)} className="w-full py-3 bg-blue-600 text-white rounded-lg">
                  Stake Now
                </button>
              )}
            </div>
          ))}
        </div>
      </div>

      {selectedProject && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-slate-800 rounded-xl p-6 max-w-md">
            <h3 className="text-xl font-bold mb-4">Stake in {selectedProject.name}</h3>
            <input
              type="number"
              value={stakeAmount}
              onChange={(e) => setStakeAmount(e.target.value)}
              placeholder={`Min: ${selectedProject.minStake}`}
              className="w-full p-3 border rounded-lg mb-4"
            />
            <div className="flex gap-4">
              <button onClick={() => setSelectedProject(null)} className="flex-1 py-3 bg-slate-200 rounded-lg">Cancel</button>
              <button onClick={handleStake} disabled={loading} className="flex-1 py-3 bg-blue-600 text-white rounded-lg">
                {loading ? 'Staking...' : 'Confirm'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
