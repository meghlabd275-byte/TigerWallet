'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../../components/ThemeProvider';

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
    cache: 'no-store',
  });
  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error((data as { error?: string }).error || `API Error: ${response.statusText}`);
  }
  return data as T;
};

interface AdminChain {
  id?: string;
  chain_id: number;
  name: string;
  symbol: string;
  rpc_url: string;
  explorer_url: string;
  status: string;
  is_default: boolean;
}

interface ChainForm {
  chain_id: string;
  name: string;
  symbol: string;
  rpc_url: string;
  explorer_url: string;
  status: string;
  is_default: boolean;
}

const EMPTY_FORM: ChainForm = {
  chain_id: '',
  name: '',
  symbol: '',
  rpc_url: '',
  explorer_url: '',
  status: 'active',
  is_default: false,
};

export default function AdminChainsPage() {
  const { isDark } = useTheme();
  const [chains, setChains] = useState<AdminChain[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form, setForm] = useState<ChainForm>(EMPTY_FORM);
  const [submitting, setSubmitting] = useState(false);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');

  const loadChains = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await fetchAPI<{ chains: AdminChain[]; count: number }>('/admin/chains');
      setChains(data.chains || []);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load chains');
      setChains([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadChains();
  }, [loadChains]);

  const startEdit = (chain: AdminChain) => {
    setEditingId(chain.chain_id);
    setForm({
      chain_id: String(chain.chain_id),
      name: chain.name,
      symbol: chain.symbol,
      rpc_url: chain.rpc_url,
      explorer_url: chain.explorer_url,
      status: chain.status || 'active',
      is_default: chain.is_default || false,
    });
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const resetForm = () => {
    setEditingId(null);
    setForm(EMPTY_FORM);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.chain_id || !form.name || !form.symbol || !form.rpc_url) {
      setError('Chain ID, name, symbol, and RPC URL are required');
      return;
    }
    setSubmitting(true);
    setError('');
    try {
      const body = {
        chain_id: parseInt(form.chain_id, 10),
        name: form.name,
        symbol: form.symbol.toUpperCase(),
        rpc_url: form.rpc_url,
        explorer_url: form.explorer_url,
        status: form.status,
        is_default: form.is_default,
      };
      if (editingId !== null) {
        await fetchAPI(`/admin/chains/${editingId}`, {
          method: 'PUT',
          body: JSON.stringify(body),
        });
      } else {
        await fetchAPI('/admin/chains', {
          method: 'POST',
          body: JSON.stringify(body),
        });
      }
      resetForm();
      await loadChains();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save chain');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (chainId: number, name: string) => {
    if (!confirm(`Remove "${name}" (chain ${chainId}) from the active registry? This disables it for all clients.`)) {
      return;
    }
    try {
      await fetchAPI(`/admin/chains/${chainId}`, { method: 'DELETE' });
      await loadChains();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to delete chain');
    }
  };

  const filtered = chains.filter((c) => {
    const matchesSearch =
      !search ||
      c.name.toLowerCase().includes(search.toLowerCase()) ||
      c.symbol.toLowerCase().includes(search.toLowerCase()) ||
      String(c.chain_id).includes(search);
    const matchesStatus = statusFilter === 'all' || c.status === statusFilter;
    return matchesSearch && matchesStatus;
  });

  const cardClass = isDark
    ? 'bg-gray-800 border border-gray-700 rounded-lg p-6'
    : 'bg-white border border-gray-200 rounded-lg p-6 shadow-sm';
  const inputClass = isDark
    ? 'w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white placeholder-gray-400 focus:outline-none focus:border-blue-500'
    : 'w-full px-3 py-2 bg-gray-50 border border-gray-300 rounded text-gray-900 placeholder-gray-400 focus:outline-none focus:border-blue-500';
  const btnPrimary = 'px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded font-medium disabled:opacity-50';
  const btnSecondary = isDark
    ? 'px-4 py-2 bg-gray-700 hover:bg-gray-600 text-white rounded font-medium'
    : 'px-4 py-2 bg-gray-200 hover:bg-gray-300 text-gray-900 rounded font-medium';
  const btnDanger = isDark
    ? 'px-3 py-1 bg-red-900 hover:bg-red-800 text-red-100 rounded text-sm'
    : 'px-3 py-1 bg-red-100 hover:bg-red-200 text-red-800 rounded text-sm';

  return (
    <div className={isDark ? 'min-h-screen bg-gray-900 text-white p-6' : 'min-h-screen bg-gray-50 text-gray-900 p-6'}>
      <div className="max-w-7xl mx-auto">
        <header className="mb-6">
          <h1 className="text-3xl font-bold mb-2">Chain Management</h1>
          <p className={isDark ? 'text-gray-400' : 'text-gray-600'}>
            Administer the blockchain registry consumed by all clients (web, desktop, mobile, extension).
            Add, edit, or disable chains. Changes propagate to GET /api/v1/chains immediately.
          </p>
        </header>

        {error && (
          <div className="mb-6 px-4 py-3 rounded border-l-4 border-red-500 bg-red-50 text-red-700">
            {error}
          </div>
        )}

        <div className={`${cardClass} mb-8`}>
          <h2 className="text-xl font-semibold mb-4">
            {editingId !== null ? `Edit Chain #${editingId}` : 'Add New Chain'}
          </h2>
          <form onSubmit={handleSubmit} className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">Chain ID *</label>
              <input
                type="number"
                value={form.chain_id}
                onChange={(e) => setForm({ ...form, chain_id: e.target.value })}
                disabled={editingId !== null}
                className={inputClass}
                placeholder="e.g. 1 (Ethereum), 9000000501 (Solana)"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Name *</label>
              <input
                type="text"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                className={inputClass}
                placeholder="Ethereum Mainnet"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Symbol *</label>
              <input
                type="text"
                value={form.symbol}
                onChange={(e) => setForm({ ...form, symbol: e.target.value })}
                className={inputClass}
                placeholder="ETH"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Status</label>
              <select
                value={form.status}
                onChange={(e) => setForm({ ...form, status: e.target.value })}
                className={inputClass}
              >
                <option value="active">active</option>
                <option value="disabled">disabled</option>
                <option value="maintenance">maintenance</option>
              </select>
            </div>
            <div className="md:col-span-2">
              <label className="block text-sm font-medium mb-1">RPC URL *</label>
              <input
                type="url"
                value={form.rpc_url}
                onChange={(e) => setForm({ ...form, rpc_url: e.target.value })}
                className={inputClass}
                placeholder="https://mainnet.infura.io/v3/..."
              />
            </div>
            <div className="md:col-span-2">
              <label className="block text-sm font-medium mb-1">Explorer URL</label>
              <input
                type="url"
                value={form.explorer_url}
                onChange={(e) => setForm({ ...form, explorer_url: e.target.value })}
                className={inputClass}
                placeholder="https://etherscan.io"
              />
            </div>
            <div className="md:col-span-2 flex items-center gap-2">
              <input
                type="checkbox"
                id="is_default"
                checked={form.is_default}
                onChange={(e) => setForm({ ...form, is_default: e.target.checked })}
                className="w-4 h-4"
              />
              <label htmlFor="is_default" className="text-sm font-medium">
                Default chain (shown first in client chain pickers)
              </label>
            </div>
            <div className="md:col-span-2 flex gap-2">
              <button type="submit" disabled={submitting} className={btnPrimary}>
                {submitting ? 'Saving...' : editingId !== null ? 'Update Chain' : 'Add Chain'}
              </button>
              {editingId !== null && (
                <button type="button" onClick={resetForm} className={btnSecondary}>
                  Cancel Edit
                </button>
              )}
            </div>
          </form>
        </div>

        <div className={`${cardClass} mb-4`}>
          <div className="flex flex-col md:flex-row gap-4">
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className={inputClass}
              placeholder="Search by name, symbol, or chain ID..."
            />
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className={`${inputClass} md:w-48`}
            >
              <option value="all">All statuses</option>
              <option value="active">active</option>
              <option value="disabled">disabled</option>
              <option value="maintenance">maintenance</option>
            </select>
            <div className={`flex items-center text-sm ${isDark ? 'text-gray-400' : 'text-gray-600'} md:ml-auto`}>
              {filtered.length} of {chains.length} chains
            </div>
          </div>
        </div>

        <div className={cardClass}>
          {loading ? (
            <div className="text-center py-8">Loading chains...</div>
          ) : filtered.length === 0 ? (
            <div className="text-center py-8 opacity-60">
              {chains.length === 0 ? 'No chains registered.' : 'No chains match your filters.'}
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className={isDark ? 'border-b border-gray-700 text-gray-400' : 'border-b border-gray-200 text-gray-500'}>
                    <th className="text-left py-2 px-3">Chain ID</th>
                    <th className="text-left py-2 px-3">Name</th>
                    <th className="text-left py-2 px-3">Symbol</th>
                    <th className="text-left py-2 px-3">RPC URL</th>
                    <th className="text-left py-2 px-3">Status</th>
                    <th className="text-left py-2 px-3">Default</th>
                    <th className="text-right py-2 px-3">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {filtered.map((chain) => (
                    <tr
                      key={chain.id || chain.chain_id}
                      className={isDark ? 'border-b border-gray-800' : 'border-b border-gray-100'}
                    >
                      <td className="py-2 px-3 font-mono">{chain.chain_id}</td>
                      <td className="py-2 px-3">{chain.name}</td>
                      <td className="py-2 px-3 font-mono">{chain.symbol}</td>
                      <td className="py-2 px-3 max-w-xs truncate" title={chain.rpc_url}>
                        {chain.rpc_url || '—'}
                      </td>
                      <td className="py-2 px-3">
                        <span
                          className={
                            chain.status === 'active'
                              ? 'px-2 py-0.5 rounded text-xs bg-green-100 text-green-800'
                              : chain.status === 'maintenance'
                                ? 'px-2 py-0.5 rounded text-xs bg-yellow-100 text-yellow-800'
                                : 'px-2 py-0.5 rounded text-xs bg-gray-200 text-gray-600'
                          }
                        >
                          {chain.status || 'active'}
                        </span>
                      </td>
                      <td className="py-2 px-3">{chain.is_default ? '✓' : ''}</td>
                      <td className="py-2 px-3 text-right whitespace-nowrap">
                        <button
                          onClick={() => startEdit(chain)}
                          className={
                            isDark
                              ? 'px-3 py-1 bg-gray-700 hover:bg-gray-600 text-white rounded text-sm mr-1'
                              : 'px-3 py-1 bg-gray-200 hover:bg-gray-300 text-gray-900 rounded text-sm mr-1'
                          }
                        >
                          Edit
                        </button>
                        <button onClick={() => handleDelete(chain.chain_id, chain.name)} className={btnDanger}>
                          Delete
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
