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

interface RedPacket {
  id: string;
  sender_id: string;
  sender_address: string;
  token_address: string;
  chain_id: number;
  total_amount: string;
  quantity: number;
  remaining_amount: string;
  remaining_qty: number;
  claim_type: string;
  message: string;
  expired_at: number;
  status: string;
  tx_hash: string;
  created_at: number;
}

export default function RedPacketsPage() {
  const { isDark } = useTheme();
  const [packets, setPackets] = useState<RedPacket[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [successMsg, setSuccessMsg] = useState('');
  const [showCreate, setShowCreate] = useState(false);
  const [claimId, setClaimId] = useState('');
  const [claimAddress, setClaimAddress] = useState('');
  const [claimPassword, setClaimPassword] = useState('');
  const [submitting, setSubmitting] = useState(false);

  // Create form state
  const [createForm, setCreateForm] = useState({
    token_address: '',
    chain_id: '1',
    total_amount: '',
    quantity: '10',
    claim_type: 'random',
    message: '',
    password: '',
    sender_address: '',
  });

  const loadPackets = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const userId = getUserIdFromToken();
      const [sentRes, recvRes] = await Promise.all([
        fetchAPI<{ packets?: RedPacket[] } | RedPacket[]>(`/red-packets/sent?user_id=${userId}`).catch(() => ({ packets: [] })),
        fetchAPI<{ packets?: RedPacket[] } | RedPacket[]>(`/red-packets/received?user_id=${userId}`).catch(() => ({ packets: [] })),
      ]);
      const sent = Array.isArray(sentRes) ? sentRes : (sentRes as any).packets || [];
      const recv = Array.isArray(recvRes) ? recvRes : (recvRes as any).packets || [];
      setPackets([...sent, ...recv]);
    } catch (e: any) {
      setError(e.message || 'Failed to load red packets');
      setPackets([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadPackets();
  }, [loadPackets]);

  const handleCreate = async () => {
    setSubmitting(true);
    setError('');
    try {
      await fetchAPI('/red-packets/create', {
        method: 'POST',
        body: JSON.stringify({
          ...createForm,
          sender_id: getUserIdFromToken(),
          chain_id: parseInt(createForm.chain_id),
          quantity: parseInt(createForm.quantity),
        }),
      });
      setSuccessMsg('Red packet created successfully!');
      setShowCreate(false);
      setCreateForm({ token_address: '', chain_id: '1', total_amount: '', quantity: '10', claim_type: 'random', message: '', password: '', sender_address: '' });
      loadPackets();
    } catch (e: any) {
      setError(e.message || 'Failed to create red packet');
    } finally {
      setSubmitting(false);
    }
  };

  const handleClaim = async () => {
    if (!claimId || !claimAddress) {
      setError('Please enter both packet ID and your address');
      return;
    }
    setSubmitting(true);
    setError('');
    try {
      const result = await fetchAPI('/red-packets/claim', {
        method: 'POST',
        body: JSON.stringify({
          packet_id: claimId,
          user_id: getUserIdFromToken(),
          address: claimAddress,
          password: claimPassword || undefined,
        }),
      });
      setSuccessMsg(`Claimed! Amount: ${(result as any).amount || 'check your wallet'}`);
      setClaimId('');
      setClaimAddress('');
      setClaimPassword('');
      loadPackets();
    } catch (e: any) {
      setError(e.message || 'Failed to claim red packet');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className={isDark ? 'min-h-screen bg-gray-900 text-white p-6' : 'min-h-screen bg-gray-50 text-gray-900 p-6'}>
      <div className="max-w-4xl mx-auto">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-3xl font-bold">Red Packets</h1>
          <div className="flex gap-2">
            <button
              onClick={() => setShowCreate(!showCreate)}
              className="px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600"
            >
              {showCreate ? 'Cancel' : 'Create'}
            </button>
            <button
              onClick={loadPackets}
              className={isDark ? 'px-4 py-2 bg-blue-600 rounded-lg hover:bg-blue-700' : 'px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600'}
            >
              Refresh
            </button>
          </div>
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

        {showCreate && (
          <div className={isDark ? 'mb-6 bg-gray-800 rounded-xl p-6 border border-gray-700 space-y-4' : 'mb-6 bg-white rounded-xl p-6 border border-gray-200 shadow-sm space-y-4'}>
            <h2 className="text-xl font-semibold">Create Red Packet</h2>
            <div className="grid grid-cols-2 gap-4">
              <input
                type="text"
                placeholder="Token address"
                value={createForm.token_address}
                onChange={(e) => setCreateForm({ ...createForm, token_address: e.target.value })}
                className={isDark ? 'px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg' : 'px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg'}
              />
              <input
                type="text"
                placeholder="Sender address"
                value={createForm.sender_address}
                onChange={(e) => setCreateForm({ ...createForm, sender_address: e.target.value })}
                className={isDark ? 'px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg' : 'px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg'}
              />
              <input
                type="text"
                placeholder="Total amount"
                value={createForm.total_amount}
                onChange={(e) => setCreateForm({ ...createForm, total_amount: e.target.value })}
                className={isDark ? 'px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg' : 'px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg'}
              />
              <input
                type="number"
                placeholder="Quantity"
                value={createForm.quantity}
                onChange={(e) => setCreateForm({ ...createForm, quantity: e.target.value })}
                className={isDark ? 'px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg' : 'px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg'}
              />
              <select
                value={createForm.chain_id}
                onChange={(e) => setCreateForm({ ...createForm, chain_id: e.target.value })}
                className={isDark ? 'px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg' : 'px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg'}
              >
                <option value="1">Ethereum</option>
                <option value="56">BSC</option>
                <option value="137">Polygon</option>
                <option value="42161">Arbitrum</option>
                <option value="10">Optimism</option>
                <option value="8453">Base</option>
              </select>
              <select
                value={createForm.claim_type}
                onChange={(e) => setCreateForm({ ...createForm, claim_type: e.target.value })}
                className={isDark ? 'px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg' : 'px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg'}
              >
                <option value="random">Random</option>
                <option value="fixed">Fixed</option>
              </select>
            </div>
            <textarea
              placeholder="Message (optional)"
              value={createForm.message}
              onChange={(e) => setCreateForm({ ...createForm, message: e.target.value })}
              className={isDark ? 'w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg' : 'w-full px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg'}
              rows={2}
            />
            <input
              type="password"
              placeholder="Claim password (optional)"
              value={createForm.password}
              onChange={(e) => setCreateForm({ ...createForm, password: e.target.value })}
              className={isDark ? 'w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg' : 'w-full px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg'}
            />
            <button
              onClick={handleCreate}
              disabled={submitting}
              className="w-full px-4 py-3 bg-red-500 text-white rounded-lg hover:bg-red-600 disabled:opacity-50 font-medium"
            >
              {submitting ? 'Creating...' : 'Create Red Packet'}
            </button>
          </div>
        )}

        {/* Claim section */}
        <div className={isDark ? 'mb-6 bg-gray-800 rounded-xl p-6 border border-gray-700' : 'mb-6 bg-white rounded-xl p-6 border border-gray-200 shadow-sm'}>
          <h2 className="text-lg font-semibold mb-3">Claim a Red Packet</h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mb-3">
            <input
              type="text"
              placeholder="Packet ID"
              value={claimId}
              onChange={(e) => setClaimId(e.target.value)}
              className={isDark ? 'px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg' : 'px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg'}
            />
            <input
              type="text"
              placeholder="Your address"
              value={claimAddress}
              onChange={(e) => setClaimAddress(e.target.value)}
              className={isDark ? 'px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg' : 'px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg'}
            />
            <input
              type="password"
              placeholder="Password (if required)"
              value={claimPassword}
              onChange={(e) => setClaimPassword(e.target.value)}
              className={isDark ? 'px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg' : 'px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg'}
            />
          </div>
          <button
            onClick={handleClaim}
            disabled={submitting}
            className="px-4 py-2 bg-green-500 text-white rounded-lg hover:bg-green-600 disabled:opacity-50"
          >
            {submitting ? 'Claiming...' : 'Claim'}
          </button>
        </div>

        {/* List */}
        {loading ? (
          <div className="flex justify-center py-20">
            <div className={isDark ? 'animate-spin rounded-full h-12 w-12 border-4 border-red-500 border-t-transparent' : 'animate-spin rounded-full h-12 w-12 border-4 border-red-400 border-t-transparent'}></div>
          </div>
        ) : packets.length === 0 ? (
          <div className={isDark ? 'text-center py-20 text-gray-400' : 'text-center py-20 text-gray-500'}>
            No red packets yet. Create one to get started!
          </div>
        ) : (
          <div className="space-y-3">
            {packets.map((packet) => (
              <div key={packet.id} className={isDark ? 'bg-gray-800 rounded-xl p-4 border border-gray-700 flex justify-between items-center' : 'bg-white rounded-xl p-4 border border-gray-200 shadow-sm flex justify-between items-center'}>
                <div>
                  <div className="font-mono text-xs text-gray-500 mb-1">{packet.id}</div>
                  <div className="text-sm">
                    {packet.total_amount} · {packet.remaining_qty}/{packet.quantity} remaining · Chain {packet.chain_id}
                  </div>
                  {packet.message && <div className={isDark ? 'text-xs text-gray-400 mt-1' : 'text-xs text-gray-500 mt-1'}>"{packet.message}"</div>}
                </div>
                <div className="text-right">
                  <span className={
                    packet.status === 'active' ? (isDark ? 'px-2 py-1 bg-green-900/50 text-green-300 text-xs rounded-full' : 'px-2 py-1 bg-green-100 text-green-700 text-xs rounded-full')
                    : packet.status === 'expired' ? (isDark ? 'px-2 py-1 bg-gray-700 text-gray-300 text-xs rounded-full' : 'px-2 py-1 bg-gray-100 text-gray-600 text-xs rounded-full')
                    : (isDark ? 'px-2 py-1 bg-blue-900/50 text-blue-300 text-xs rounded-full' : 'px-2 py-1 bg-blue-100 text-blue-700 text-xs rounded-full')
                  }>
                    {packet.status}
                  </span>
                  {packet.claim_type && (
                    <div className={isDark ? 'text-xs text-gray-400 mt-1' : 'text-xs text-gray-500 mt-1'}>{packet.claim_type}</div>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
