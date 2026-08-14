'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../components/ThemeProvider';

const API_BASE_URL = typeof window !== 'undefined' ? '' : (process.env.BACKEND_URL || 'http://localhost:8443');

const fetchAPI = async <T,>(endpoint: string, options?: RequestInit): Promise<T> => {
  const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
  const response = await fetch(`${API_BASE_URL}/api/v1${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });
  if (!response.ok) {
    const err = await response.json().catch(() => ({}));
    throw new Error(err.error || err.message || `API Error: ${response.statusText}`);
  }
  const data = await response.json();
  return data?.data ?? data;
};

function getUserIdFromToken(): string {
  if (typeof window === 'undefined') return '';
  const token = localStorage.getItem('tigerwallet-token');
  if (!token) return '';
  try {
    const payload = JSON.parse(atob(token.split('.')[1] || ''));
    return payload.user_id || payload.sub || payload.userId || '';
  } catch {
    return '';
  }
}

interface AirdropCampaign {
  id: string;
  name: string;
  description: string;
  token_address: string;
  chain_id: number;
  total_amount: string;
  claimed_amount: string;
  start_time: number;
  end_time: number;
  status: string;
  claim_type: string;
  merkle_root: string;
  rules: string;
  created_at: number;
}

export default function AirdropPage() {
  const { isDark } = useTheme();
  const [campaigns, setCampaigns] = useState<AirdropCampaign[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [claimingId, setClaimingId] = useState<string | null>(null);
  const [claimAddress, setClaimAddress] = useState('');
  const [successMsg, setSuccessMsg] = useState('');

  const loadCampaigns = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await fetchAPI<{ campaigns?: AirdropCampaign[] } | AirdropCampaign[]>('/airdrop/campaigns');
      const arr = Array.isArray(data) ? data : (data as any).campaigns || [];
      setCampaigns(arr);
    } catch (e: any) {
      setError(e.message || 'Failed to load airdrop campaigns');
      setCampaigns([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadCampaigns();
  }, [loadCampaigns]);

  const handleClaim = async (campaignId: string) => {
    if (!claimAddress) {
      setError('Please enter your wallet address to claim');
      return;
    }
    setClaimingId(campaignId);
    setError('');
    try {
      const result = await fetchAPI('/airdrop/claim', {
        method: 'POST',
        body: JSON.stringify({
          campaign_id: campaignId,
          user_id: getUserIdFromToken(),
          address: claimAddress,
        }),
      });
      setSuccessMsg(`Claim submitted! Claim ID: ${(result as any).id || (result as any).claim_id || 'pending'}`);
      loadCampaigns();
    } catch (e: any) {
      setError(e.message || 'Failed to submit claim');
    } finally {
      setClaimingId(null);
    }
  };

  const fmtTime = (ts: number) => {
    if (!ts) return '—';
    return new Date(ts * 1000).toLocaleDateString();
  };

  const fmtProgress = (claimed: string, total: string) => {
    const c = parseFloat(claimed) || 0;
    const t = parseFloat(total) || 0;
    if (t === 0) return 0;
    return Math.min(100, (c / t) * 100);
  };

  return (
    <div className={isDark ? 'min-h-screen bg-gray-900 text-white p-6' : 'min-h-screen bg-gray-50 text-gray-900 p-6'}>
      <div className="max-w-6xl mx-auto">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-3xl font-bold">Airdrop Campaigns</h1>
          <button
            onClick={loadCampaigns}
            className={isDark ? 'px-4 py-2 bg-blue-600 rounded-lg hover:bg-blue-700' : 'px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600'}
          >
            Refresh
          </button>
        </div>

        {error && (
          <div className={isDark ? 'mb-4 p-3 bg-red-900/50 border border-red-700 rounded-lg text-red-200' : 'mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700'}>
            {error}
          </div>
        )}
        {successMsg && (
          <div className={isDark ? 'mb-4 p-3 bg-green-900/50 border border-green-700 rounded-lg text-green-200' : 'mb-4 p-3 bg-green-50 border border-green-200 rounded-lg text-green-700'}>
            {successMsg}
          </div>
        )}

        {loading ? (
          <div className="flex justify-center py-20">
            <div className={isDark ? 'animate-spin rounded-full h-12 w-12 border-4 border-blue-500 border-t-transparent' : 'animate-spin rounded-full h-12 w-12 border-4 border-blue-400 border-t-transparent'}></div>
          </div>
        ) : campaigns.length === 0 ? (
          <div className={isDark ? 'text-center py-20 text-gray-400' : 'text-center py-20 text-gray-500'}>
            No airdrop campaigns available. Check back later.
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {campaigns.map((campaign) => (
              <div key={campaign.id} className={isDark ? 'bg-gray-800 rounded-xl p-6 border border-gray-700' : 'bg-white rounded-xl p-6 border border-gray-200 shadow-sm'}>
                <div className="flex justify-between items-start mb-3">
                  <div>
                    <h2 className="text-xl font-semibold">{campaign.name}</h2>
                    <p className={isDark ? 'text-sm text-gray-400 mt-1' : 'text-sm text-gray-500 mt-1'}>
                      Chain ID: {campaign.chain_id} · Type: {campaign.claim_type}
                    </p>
                  </div>
                  <span className={
                    campaign.status === 'active' ? (isDark ? 'px-2 py-1 bg-green-900/50 text-green-300 text-xs rounded-full' : 'px-2 py-1 bg-green-100 text-green-700 text-xs rounded-full')
                    : campaign.status === 'upcoming' ? (isDark ? 'px-2 py-1 bg-blue-900/50 text-blue-300 text-xs rounded-full' : 'px-2 py-1 bg-blue-100 text-blue-700 text-xs rounded-full')
                    : (isDark ? 'px-2 py-1 bg-gray-700 text-gray-300 text-xs rounded-full' : 'px-2 py-1 bg-gray-100 text-gray-600 text-xs rounded-full')
                  }>
                    {campaign.status}
                  </span>
                </div>

                {campaign.description && (
                  <p className={isDark ? 'text-sm text-gray-300 mb-3' : 'text-sm text-gray-600 mb-3'}>{campaign.description}</p>
                )}

                <div className="grid grid-cols-2 gap-3 mb-4 text-sm">
                  <div>
                    <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Total:</span> {campaign.total_amount}
                  </div>
                  <div>
                    <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Claimed:</span> {campaign.claimed_amount}
                  </div>
                  <div>
                    <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Start:</span> {fmtTime(campaign.start_time)}
                  </div>
                  <div>
                    <span className={isDark ? 'text-gray-400' : 'text-gray-500'}>End:</span> {fmtTime(campaign.end_time)}
                  </div>
                </div>

                <div className="mb-4">
                  <div className={isDark ? 'w-full bg-gray-700 rounded-full h-2' : 'w-full bg-gray-200 rounded-full h-2'}>
                    <div className="bg-blue-500 h-2 rounded-full" style={{ width: `${fmtProgress(campaign.claimed_amount, campaign.total_amount)}%` }}></div>
                  </div>
                </div>

                {campaign.rules && (
                  <p className={isDark ? 'text-xs text-gray-400 mb-3' : 'text-xs text-gray-500 mb-3'}>Rules: {campaign.rules}</p>
                )}

                {campaign.status === 'active' && (
                  <div className="flex gap-2">
                    <input
                      type="text"
                      placeholder="Your wallet address"
                      value={claimAddress}
                      onChange={(e) => setClaimAddress(e.target.value)}
                      className={isDark ? 'flex-1 px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg text-sm' : 'flex-1 px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg text-sm'}
                    />
                    <button
                      onClick={() => handleClaim(campaign.id)}
                      disabled={claimingId === campaign.id}
                      className="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:opacity-50 text-sm"
                    >
                      {claimingId === campaign.id ? 'Claiming...' : 'Claim'}
                    </button>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
