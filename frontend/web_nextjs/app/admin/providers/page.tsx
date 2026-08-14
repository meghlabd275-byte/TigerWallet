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

interface Provider {
  id: string;
  name: string;
  fiatCurrency: string;
  cryptoCurrency: string;
  minAmount: number;
  maxAmount: number;
  feePercent: number;
  processingTime: string;
  supportedCountries: string[];
  paymentMethods: string[];
}

interface ProviderKeyStatus {
  providerId: string;
  configured: boolean;
}

export default function AdminProvidersPage() {
  const { isDark } = useTheme();
  const [providers, setProviders] = useState<Provider[]>([]);
  const [keyStatus, setKeyStatus] = useState<Record<string, boolean>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [apiKeyInputs, setApiKeyInputs] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState<string | null>(null);
  const [message, setMessage] = useState('');

  const loadProviders = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const data = await fetchAPI<{ providers: Provider[] }>('/ramp/providers');
      setProviders(data.providers || []);
      const statuses = await Promise.all(
        (data.providers || []).map(async (p) => {
          try {
            const s = await fetchAPI<ProviderKeyStatus>(`/ramp/admin/providers/${p.id}/key`);
            return { id: p.id, configured: s.configured };
          } catch {
            return { id: p.id, configured: false };
          }
        })
      );
      const map: Record<string, boolean> = {};
      statuses.forEach((s) => { map[s.id] = s.configured; });
      setKeyStatus(map);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load providers');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadProviders();
  }, [loadProviders]);

  const handleSaveKey = async (providerId: string) => {
    const key = (apiKeyInputs[providerId] || '').trim();
    if (!key) {
      setMessage('Enter an API key before saving.');
      return;
    }
    setSaving(providerId);
    setMessage('');
    try {
      await fetchAPI(`/ramp/admin/providers/${providerId}/key`, {
        method: 'POST',
        body: JSON.stringify({ apiKey: key }),
      });
      setKeyStatus((prev) => ({ ...prev, [providerId]: true }));
      setApiKeyInputs((prev) => ({ ...prev, [providerId]: '' }));
      setMessage(`${providerId} API key configured.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save key');
    } finally {
      setSaving(null);
    }
  };

  const handleClearKey = async (providerId: string) => {
    setSaving(providerId);
    setMessage('');
    try {
      const res = await fetchAPI<ProviderKeyStatus>(`/ramp/admin/providers/${providerId}/key`, {
        method: 'DELETE',
      });
      setKeyStatus((prev) => ({ ...prev, [providerId]: res.configured }));
      setMessage(`${providerId} API key cleared (falls back to env if set).`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to clear key');
    } finally {
      setSaving(null);
    }
  };

  const containerClass = isDark ? 'bg-gray-900 text-white min-h-screen p-6' : 'bg-gray-50 text-gray-900 min-h-screen p-6';
  const cardClass = isDark ? 'bg-gray-800 border border-gray-700' : 'bg-white border border-gray-200';

  return (
    <div className={containerClass}>
      <div className="max-w-4xl mx-auto">
        <h1 className="text-2xl font-bold mb-2">Fiat Ramp Provider API Keys</h1>
        <p className={isDark ? 'text-gray-400 mb-6' : 'text-gray-600 mb-6'}>
          Configure each on/off-ramp provider&apos;s API key. Keys are stored at runtime in the fiat-ramp service
          and take precedence over environment variables. The key value is never returned by the API.
        </p>

        {loading && <div className={isDark ? 'text-gray-400' : 'text-gray-600'}>Loading providers…</div>}
        {error && (
          <div className={`mb-4 p-3 rounded ${isDark ? 'bg-red-900/40 text-red-300' : 'bg-red-50 text-red-700'}`}>
            {error}
          </div>
        )}
        {message && (
          <div className={`mb-4 p-3 rounded ${isDark ? 'bg-green-900/40 text-green-300' : 'bg-green-50 text-green-700'}`}>
            {message}
          </div>
        )}

        <div className="space-y-4">
          {providers.map((p) => (
            <div key={p.id} className={`rounded-lg p-4 ${cardClass}`}>
              <div className="flex items-center justify-between mb-2">
                <div>
                  <span className="font-semibold">{p.name}</span>
                  <span className={isDark ? 'text-gray-400 ml-2' : 'text-gray-500 ml-2'}>· {p.id}</span>
                </div>
                <span className={`text-xs px-2 py-1 rounded ${keyStatus[p.id] ? (isDark ? 'bg-green-900/50 text-green-300' : 'bg-green-100 text-green-700') : (isDark ? 'bg-gray-700 text-gray-300' : 'bg-gray-100 text-gray-600')}`}>
                  {keyStatus[p.id] ? 'Key configured' : 'No key set'}
                </span>
              </div>
              <div className={`text-sm mb-3 ${isDark ? 'text-gray-400' : 'text-gray-600'}`}>
                Fee: {p.feePercent}% · Range: {p.minAmount}–{p.maxAmount} · {p.processingTime}
              </div>
              <div className="flex gap-2">
                <input
                  type="password"
                  placeholder="Paste provider API key"
                  value={apiKeyInputs[p.id] || ''}
                  onChange={(e) => setApiKeyInputs((prev) => ({ ...prev, [p.id]: e.target.value }))}
                  className={`flex-1 px-3 py-2 rounded border ${isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-300'}`}
                />
                <button
                  onClick={() => handleSaveKey(p.id)}
                  disabled={saving === p.id || !((apiKeyInputs[p.id] || '').trim())}
                  className="px-4 py-2 rounded bg-orange-500 text-white font-semibold disabled:opacity-50"
                >
                  {saving === p.id ? 'Saving…' : 'Save'}
                </button>
                {keyStatus[p.id] && (
                  <button
                    onClick={() => handleClearKey(p.id)}
                    disabled={saving === p.id}
                    className={`px-4 py-2 rounded font-semibold ${isDark ? 'bg-gray-700 text-gray-200' : 'bg-gray-200 text-gray-700'}`}
                  >
                    Clear
                  </button>
                )}
              </div>
            </div>
          ))}
          {!loading && providers.length === 0 && (
            <div className={isDark ? 'text-gray-400' : 'text-gray-600'}>No providers available.</div>
          )}
        </div>
      </div>
    </div>
  );
}
