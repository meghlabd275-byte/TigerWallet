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

interface EarnProduct {
  id: string;
  name: string;
  description: string;
  product_type: string;
  chain_id: number;
  token_address: string;
  apy: string;
  apy_type: string;
  min_deposit: string;
  max_deposit: string;
  min_term: number;
  max_term: number;
  total_deposited: string;
  total_value: string;
  max_capacity: string;
  status: string;
  features?: string[];
  created_at: number;
}

interface EarnDeposit {
  id: string;
  product_id: string;
  user_id: string;
  amount: string;
  status: string;
  created_at: number;
}

export default function EarnPage() {
  const { isDark } = useTheme();
  const [products, setProducts] = useState<EarnProduct[]>([]);
  const [deposits, setDeposits] = useState<EarnDeposit[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [successMsg, setSuccessMsg] = useState('');
  const [actionProduct, setActionProduct] = useState<EarnProduct | null>(null);
  const [actionType, setActionType] = useState<'deposit' | 'withdraw' | 'claim'>('deposit');
  const [actionAmount, setActionAmount] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [prodData, depData] = await Promise.all([
        fetchAPI<{ products?: EarnProduct[] } | EarnProduct[]>('/earn/products').catch(() => ({ products: [] })),
        fetchAPI<{ deposits?: EarnDeposit[] } | EarnDeposit[]>(`/earn/deposits?user_id=${getUserIdFromToken()}`).catch(() => ({ deposits: [] })),
      ]);
      const prods = Array.isArray(prodData) ? prodData : (prodData as any).products || [];
      const deps = Array.isArray(depData) ? depData : (depData as any).deposits || [];
      setProducts(prods);
      setDeposits(deps);
    } catch (e: any) {
      setError(e.message || 'Failed to load earn data');
      setProducts([]);
      setDeposits([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleAction = async () => {
    if (!actionProduct || !actionAmount) return;
    setSubmitting(true);
    setError('');
    try {
      const body: any = {
        product_id: actionProduct.id,
        user_id: getUserIdFromToken(),
        amount: actionAmount,
      };
      await fetchAPI(`/earn/${actionType}`, {
        method: 'POST',
        body: JSON.stringify(body),
      });
      setSuccessMsg(`${actionType.charAt(0).toUpperCase() + actionType.slice(1)} of ${actionAmount} submitted successfully`);
      setActionProduct(null);
      setActionAmount('');
      loadData();
    } catch (e: any) {
      setError(e.message || `Failed to ${actionType}`);
    } finally {
      setSubmitting(false);
    }
  };

  const fmtTerm = (seconds: number) => {
    if (!seconds) return 'Flexible';
    const days = seconds / 86400;
    if (days >= 1) return `${days} days`;
    const hours = seconds / 3600;
    return `${hours} hours`;
  };

  return (
    <div className={isDark ? 'min-h-screen bg-gray-900 text-white p-6' : 'min-h-screen bg-gray-50 text-gray-900 p-6'}>
      <div className="max-w-6xl mx-auto">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-3xl font-bold">Earn Products</h1>
          <button
            onClick={loadData}
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
        ) : (
          <>
            <div className="mb-8">
              <h2 className="text-xl font-semibold mb-4">Available Products</h2>
              {products.length === 0 ? (
                <div className={isDark ? 'text-center py-10 text-gray-400' : 'text-center py-10 text-gray-500'}>
                  No earn products available.
                </div>
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {products.map((product) => (
                    <div key={product.id} className={isDark ? 'bg-gray-800 rounded-xl p-5 border border-gray-700' : 'bg-white rounded-xl p-5 border border-gray-200 shadow-sm'}>
                      <div className="flex justify-between items-start mb-2">
                        <h3 className="text-lg font-semibold">{product.name}</h3>
                        <span className={isDark ? 'px-2 py-1 bg-green-900/50 text-green-300 text-xs rounded-full' : 'px-2 py-1 bg-green-100 text-green-700 text-xs rounded-full'}>
                          {product.apy} APY
                        </span>
                      </div>
                      <p className={isDark ? 'text-sm text-gray-400 mb-3' : 'text-sm text-gray-500 mb-3'}>{product.description}</p>
                      <div className="space-y-1 text-sm mb-4">
                        <div><span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Type:</span> {product.product_type} ({product.apy_type})</div>
                        <div><span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Min Deposit:</span> {product.min_deposit}</div>
                        <div><span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Term:</span> {fmtTerm(product.min_term)} - {fmtTerm(product.max_term)}</div>
                        <div><span className={isDark ? 'text-gray-400' : 'text-gray-500'}>Total Deposited:</span> {product.total_deposited}</div>
                      </div>
                      {product.status === 'active' && (
                        <div className="flex gap-2">
                          <button
                            onClick={() => { setActionProduct(product); setActionType('deposit'); setActionAmount(''); }}
                            className="flex-1 px-3 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 text-sm"
                          >
                            Deposit
                          </button>
                          <button
                            onClick={() => { setActionProduct(product); setActionType('withdraw'); setActionAmount(''); }}
                            className={isDark ? 'flex-1 px-3 py-2 bg-gray-700 rounded-lg hover:bg-gray-600 text-sm' : 'flex-1 px-3 py-2 bg-gray-100 rounded-lg hover:bg-gray-200 text-sm'}
                          >
                            Withdraw
                          </button>
                          <button
                            onClick={() => { setActionProduct(product); setActionType('claim'); setActionAmount('0'); }}
                            className={isDark ? 'flex-1 px-3 py-2 bg-green-800 rounded-lg hover:bg-green-700 text-sm' : 'flex-1 px-3 py-2 bg-green-100 text-green-700 rounded-lg hover:bg-green-200 text-sm'}
                          >
                            Claim
                          </button>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div>
              <h2 className="text-xl font-semibold mb-4">My Deposits</h2>
              {deposits.length === 0 ? (
                <div className={isDark ? 'text-center py-10 text-gray-400' : 'text-center py-10 text-gray-500'}>
                  No active deposits.
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className={isDark ? 'w-full text-sm' : 'w-full text-sm'}>
                    <thead>
                      <tr className={isDark ? 'border-b border-gray-700 text-left' : 'border-b border-gray-200 text-left'}>
                        <th className="py-2 px-3">Product ID</th>
                        <th className="py-2 px-3">Amount</th>
                        <th className="py-2 px-3">Status</th>
                        <th className="py-2 px-3">Date</th>
                      </tr>
                    </thead>
                    <tbody>
                      {deposits.map((dep) => (
                        <tr key={dep.id} className={isDark ? 'border-b border-gray-800' : 'border-b border-gray-100'}>
                          <td className="py-2 px-3 font-mono text-xs">{dep.product_id}</td>
                          <td className="py-2 px-3">{dep.amount}</td>
                          <td className="py-2 px-3">
                            <span className={isDark ? 'px-2 py-0.5 bg-blue-900/50 text-blue-300 text-xs rounded-full' : 'px-2 py-0.5 bg-blue-100 text-blue-700 text-xs rounded-full'}>
                              {dep.status}
                            </span>
                          </td>
                          <td className="py-2 px-3 text-xs">{dep.created_at ? new Date(dep.created_at * 1000).toLocaleDateString() : '—'}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </>
        )}

        {actionProduct && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setActionProduct(null)}>
            <div className={isDark ? 'bg-gray-800 rounded-xl p-6 max-w-md w-full mx-4 border border-gray-700' : 'bg-white rounded-xl p-6 max-w-md w-full mx-4 border border-gray-200'} onClick={(e) => e.stopPropagation()}>
              <h3 className="text-lg font-semibold mb-2">{actionType.charAt(0).toUpperCase() + actionType.slice(1)}: {actionProduct.name}</h3>
              <p className={isDark ? 'text-sm text-gray-400 mb-4' : 'text-sm text-gray-500 mb-4'}>
                {actionType === 'claim' ? 'Claim your rewards for this product.' : `Min: ${actionProduct.min_deposit} · Max: ${actionProduct.max_deposit}`}
              </p>
              {actionType !== 'claim' && (
                <input
                  type="text"
                  placeholder="Amount"
                  value={actionAmount}
                  onChange={(e) => setActionAmount(e.target.value)}
                  className={isDark ? 'w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg mb-4' : 'w-full px-3 py-2 bg-gray-50 border border-gray-200 rounded-lg mb-4'}
                />
              )}
              <div className="flex gap-2">
                <button
                  onClick={handleAction}
                  disabled={submitting}
                  className="flex-1 px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:opacity-50"
                >
                  {submitting ? 'Submitting...' : `Confirm ${actionType}`}
                </button>
                <button
                  onClick={() => setActionProduct(null)}
                  className={isDark ? 'flex-1 px-4 py-2 bg-gray-700 rounded-lg hover:bg-gray-600' : 'flex-1 px-4 py-2 bg-gray-100 rounded-lg hover:bg-gray-200'}
                >
                  Cancel
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
