'use client';

import React, { useState } from 'react';
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

interface CouponResult {
  valid: boolean;
  code?: string;
  type?: string;
  value?: string;
  discount_amount?: string;
  min_amount?: string;
  max_uses?: number;
  used_count?: number;
  valid_from?: number;
  valid_until?: number;
  status?: string;
  error?: string;
}

export default function CouponPage() {
  const { isDark } = useTheme();
  const [code, setCode] = useState('');
  const [amount, setAmount] = useState('');
  const [chainId, setChainId] = useState('');
  const [pair, setPair] = useState('');
  const [result, setResult] = useState<CouponResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleValidate = async () => {
    if (!code.trim()) {
      setError('Please enter a coupon code');
      return;
    }
    setLoading(true);
    setError('');
    setResult(null);
    try {
      const body: any = { code: code.trim() };
      if (amount) body.amount = amount;
      if (chainId) body.chain_id = parseInt(chainId);
      if (pair) body.pair = pair;
      const data = await fetchAPI<CouponResult>('/coupon/validate', {
        method: 'POST',
        body: JSON.stringify(body),
      });
      setResult(data);
    } catch (e: any) {
      setError(e.message || 'Failed to validate coupon');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className={isDark ? 'min-h-screen bg-gray-900 text-white p-6' : 'min-h-screen bg-gray-50 text-gray-900 p-6'}>
      <div className="max-w-2xl mx-auto">
        <h1 className="text-3xl font-bold mb-6">Validate Coupon</h1>

        {error && (
          <div className={isDark ? 'mb-4 p-3 bg-red-900/50 border border-red-700 rounded-lg text-red-200' : 'mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700'}>
            {error}
          </div>
        )}

        <div className={isDark ? 'bg-gray-800 rounded-xl p-6 border border-gray-700 space-y-4' : 'bg-white rounded-xl p-6 border border-gray-200 shadow-sm space-y-4'}>
          <div>
            <label className={isDark ? 'block text-sm text-gray-400 mb-1' : 'block text-sm text-gray-500 mb-1'}>Coupon Code</label>
            <input
              type="text"
              placeholder="e.g. TIGER20"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              className={isDark ? 'w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg' : 'w-full px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg'}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className={isDark ? 'block text-sm text-gray-400 mb-1' : 'block text-sm text-gray-500 mb-1'}>Order Amount (optional)</label>
              <input
                type="text"
                placeholder="e.g. 100.00"
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                className={isDark ? 'w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg' : 'w-full px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg'}
              />
            </div>
            <div>
              <label className={isDark ? 'block text-sm text-gray-400 mb-1' : 'block text-sm text-gray-500 mb-1'}>Chain ID (optional)</label>
              <input
                type="text"
                placeholder="e.g. 1"
                value={chainId}
                onChange={(e) => setChainId(e.target.value)}
                className={isDark ? 'w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg' : 'w-full px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg'}
              />
            </div>
          </div>

          <div>
            <label className={isDark ? 'block text-sm text-gray-400 mb-1' : 'block text-sm text-gray-500 mb-1'}>Trading Pair (optional)</label>
            <input
              type="text"
              placeholder="e.g. ETH/USDT"
              value={pair}
              onChange={(e) => setPair(e.target.value)}
              className={isDark ? 'w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg' : 'w-full px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg'}
            />
          </div>

          <button
            onClick={handleValidate}
            disabled={loading}
            className="w-full px-4 py-3 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:opacity-50 font-medium"
          >
            {loading ? 'Validating...' : 'Validate Coupon'}
          </button>
        </div>

        {result && (
          <div className={`mt-6 rounded-xl p-6 border ${result.valid
            ? (isDark ? 'bg-green-900/30 border-green-700' : 'bg-green-50 border-green-200')
            : (isDark ? 'bg-red-900/30 border-red-700' : 'bg-red-50 border-red-200')
          }`}>
            <div className="flex items-center gap-2 mb-3">
              <span className={result.valid ? 'text-green-500 text-2xl' : 'text-red-500 text-2xl'}>
                {result.valid ? '✓' : '✗'}
              </span>
              <h2 className={`text-xl font-semibold ${result.valid ? (isDark ? 'text-green-300' : 'text-green-700') : (isDark ? 'text-red-300' : 'text-red-700')}`}>
                {result.valid ? 'Coupon Valid' : 'Coupon Invalid'}
              </h2>
            </div>

            {result.valid && (
              <div className="space-y-1 text-sm">
                {result.type && <div><span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Type:</span> {result.type}</div>}
                {result.value && <div><span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Value:</span> {result.value}</div>}
                {result.discount_amount && <div><span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Discount:</span> {result.discount_amount}</div>}
                {result.min_amount && <div><span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Min Amount:</span> {result.min_amount}</div>}
                {result.max_uses !== undefined && <div><span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Uses:</span> {result.used_count || 0} / {result.max_uses}</div>}
                {result.valid_until && <div><span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Valid Until:</span> {new Date(result.valid_until * 1000).toLocaleDateString()}</div>}
                {result.status && <div><span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Status:</span> {result.status}</div>}
              </div>
            )}

            {!result.valid && result.error && (
              <p className={isDark ? 'text-sm text-red-300' : 'text-sm text-red-600'}>{result.error}</p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
