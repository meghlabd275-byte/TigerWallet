'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../components/ThemeProvider';

interface SupportedToken {
  symbol: string;
  name: string;
  decimals: number;
  address?: string;
}

interface ConvertQuote {
  from_token: string;
  to_token: string;
  from_amount: string;
  to_amount: string;
  rate: string;
  price_impact: string;
  min_received: string;
  chain_id: number;
}

interface ConvertHistoryItem {
  id: string;
  from_token: string;
  to_token: string;
  from_amount: string;
  to_amount: string;
  rate: string;
  chain_id: number;
  tx_hash?: string;
  status: string;
  created_at: string;
}

const API_BASE_URL = typeof window !== 'undefined' ? '' : (process.env.BACKEND_URL || 'http://localhost:8443');
const fetchAPI = async <T,>(endpoint: string, options?: RequestInit): Promise<T> => {
  const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
  const res = await fetch(`${API_BASE_URL}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(options?.headers || {}),
    },
  });
  if (!res.ok) {
    let msg = `Request failed (${res.status})`;
    try { const e = await res.json(); msg = e.error || e.message || msg; } catch { /* noop */ }
    throw new Error(msg);
  }
  return res.json();
};

const CHAINS = [
  { id: 1, name: 'Ethereum' },
  { id: 56, name: 'BNB Chain' },
  { id: 137, name: 'Polygon' },
  { id: 42161, name: 'Arbitrum' },
  { id: 10, name: 'Optimism' },
  { id: 8453, name: 'Base' },
];

export default function ConvertPage() {
  const { isDark } = useTheme();
  const [supportedTokens, setSupportedTokens] = useState<SupportedToken[]>([]);
  const [fromSymbol, setFromSymbol] = useState('ETH');
  const [toSymbol, setToSymbol] = useState('USDT');
  const [fromAmount, setFromAmount] = useState('');
  const [quote, setQuote] = useState<ConvertQuote | null>(null);
  const [loadingQuote, setLoadingQuote] = useState(false);
  const [converting, setConverting] = useState(false);
  const [error, setError] = useState('');
  const [activeTab, setActiveTab] = useState<'convert' | 'history'>('convert');
  const [history, setHistory] = useState<ConvertHistoryItem[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [showFromList, setShowFromList] = useState(false);
  const [showToList, setShowToList] = useState(false);
  const [chainId, setChainId] = useState(1);

  const loadSupported = useCallback(async () => {
    try {
      const data = await fetchAPI<{ tokens: SupportedToken[] }>('/api/v1/convert/supported');
      setSupportedTokens(data.tokens || []);
    } catch (e) {
      setError((e as Error).message);
    }
  }, []);

  const loadHistory = useCallback(async () => {
    setHistoryLoading(true);
    try {
      const data = await fetchAPI<{ history: ConvertHistoryItem[] }>('/api/v1/convert/history');
      setHistory(data.history || []);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setHistoryLoading(false);
    }
  }, []);

  useEffect(() => { loadSupported(); }, [loadSupported]);
  useEffect(() => { if (activeTab === 'history') loadHistory(); }, [activeTab, loadHistory]);

  // Fetch a quote whenever inputs change (debounced).
  useEffect(() => {
    if (!fromAmount || parseFloat(fromAmount) <= 0 || !fromSymbol || !toSymbol || fromSymbol === toSymbol) {
      setQuote(null);
      return;
    }
    const t = setTimeout(async () => {
      setLoadingQuote(true);
      setError('');
      try {
        const params = new URLSearchParams({
          from_token: fromSymbol,
          to_token: toSymbol,
          amount: fromAmount,
          chain_id: String(chainId),
        });
        const data = await fetchAPI<ConvertQuote>(`/api/v1/convert/quote?${params}`);
        setQuote(data);
      } catch (e) {
        setQuote(null);
        setError((e as Error).message);
      } finally {
        setLoadingQuote(false);
      }
    }, 400);
    return () => clearTimeout(t);
  }, [fromAmount, fromSymbol, toSymbol, chainId]);

  const handleSwap = () => {
    setFromSymbol(toSymbol);
    setToSymbol(fromSymbol);
  };

  const handleConvert = async () => {
    if (!quote) return;
    setConverting(true);
    setError('');
    try {
      await fetchAPI('/api/v1/convert/execute', {
        method: 'POST',
        body: JSON.stringify({
          from_token: fromSymbol,
          to_token: toSymbol,
          amount: fromAmount,
          chain_id: chainId,
          min_received: quote.min_received,
        }),
      });
      setFromAmount('');
      setQuote(null);
      loadHistory();
      setActiveTab('history');
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setConverting(false);
    }
  };

  const inputCls = isDark ? 'bg-gray-700 border-gray-600 text-white' : 'bg-white border-gray-200 text-gray-900';
  const cardCls = isDark ? 'bg-gray-800' : 'bg-white border border-gray-200 shadow-sm';
  const innerCls = isDark ? 'bg-gray-700' : 'bg-gray-100';
  const subText = isDark ? 'text-gray-400' : 'text-gray-500';

  return (
    <div className={`min-h-screen p-6 ${isDark ? 'bg-gray-900 text-white' : 'bg-gray-50 text-gray-900'}`}>
      <div className="max-w-4xl mx-auto">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold">Convert</h1>
          <p className={`mt-1 ${subText}`}>One-click conversion between crypto assets via on-chain AMM</p>
        </div>

        {error && (
          <div className={`mb-4 rounded-lg p-3 text-sm ${isDark ? 'bg-red-900/40 text-red-300' : 'bg-red-50 text-red-700'}`}>
            {error}
          </div>
        )}

        <div className="flex justify-center mb-6">
          <div className={`rounded-lg p-1 flex ${isDark ? 'bg-gray-800' : 'bg-white border border-gray-200'}`}>
            <button
              onClick={() => setActiveTab('convert')}
              className={`px-6 py-2 rounded-lg ${activeTab === 'convert' ? 'bg-blue-600 text-white' : 'bg-transparent'}`}
            >
              Convert
            </button>
            <button
              onClick={() => setActiveTab('history')}
              className={`px-6 py-2 rounded-lg ${activeTab === 'history' ? 'bg-blue-600 text-white' : 'bg-transparent'}`}
            >
              History
            </button>
          </div>
        </div>

        {activeTab === 'convert' && (
          <div className={`rounded-xl p-6 ${cardCls}`}>
            <div className="space-y-4">
              {/* Chain selector */}
              <div className="flex items-center justify-between">
                <span className={`text-sm ${subText}`}>Network</span>
                <select
                  value={chainId}
                  onChange={(e) => setChainId(Number(e.target.value))}
                  className={`rounded-lg px-3 py-1.5 text-sm border ${inputCls}`}
                >
                  {CHAINS.map((c) => (
                    <option key={c.id} value={c.id}>{c.name}</option>
                  ))}
                </select>
              </div>

              {/* From */}
              <div className={`rounded-lg p-4 ${innerCls}`}>
                <div className="flex justify-between items-center mb-2">
                  <span className={subText}>From</span>
                </div>
                <div className="flex items-center space-x-4">
                  <div className="flex-1">
                    <input
                      type="number"
                      value={fromAmount}
                      onChange={(e) => setFromAmount(e.target.value)}
                      placeholder="0.00"
                      className={`w-full bg-transparent text-2xl font-bold outline-none ${isDark ? 'text-white' : 'text-gray-900'}`}
                    />
                  </div>
                  <div className="relative">
                    <button
                      onClick={() => { setShowFromList(!showFromList); setShowToList(false); }}
                      className={`flex items-center space-x-2 px-4 py-2 rounded-lg ${isDark ? 'bg-gray-600 hover:bg-gray-500' : 'bg-gray-200 hover:bg-gray-300'}`}
                    >
                      <span>{fromSymbol}</span>
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                      </svg>
                    </button>
                    {showFromList && (
                      <div className={`absolute top-full right-0 mt-2 rounded-lg shadow-xl z-20 w-48 max-h-60 overflow-y-auto ${isDark ? 'bg-gray-700' : 'bg-white border border-gray-200'}`}>
                        {(supportedTokens.length ? supportedTokens : CHAINS.flatMap(_ => [])).map((t) => (
                          <button
                            key={t.symbol}
                            onClick={() => { setFromSymbol(t.symbol); setShowFromList(false); }}
                            className={`w-full px-4 py-2 flex items-center justify-between ${isDark ? 'hover:bg-gray-600' : 'hover:bg-gray-100'}`}
                          >
                            <span>{t.symbol}</span>
                            <span className={`text-xs ${subText}`}>{t.name}</span>
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              </div>

              {/* Swap Button */}
              <div className="flex justify-center -my-2 relative z-10">
                <button onClick={handleSwap} className="bg-blue-600 p-2 rounded-full hover:bg-blue-700">
                  <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16V4m0 0L3 8m4-4l4 4m6 0v12m0 0l4-4m-4 4l-4-4" />
                  </svg>
                </button>
              </div>

              {/* To */}
              <div className={`rounded-lg p-4 ${innerCls}`}>
                <div className="flex justify-between items-center mb-2">
                  <span className={subText}>To</span>
                </div>
                <div className="flex items-center space-x-4">
                  <div className="flex-1">
                    <input
                      type="number"
                      value={quote?.to_amount ?? ''}
                      readOnly
                      placeholder={loadingQuote ? 'Fetching quote...' : '0.00'}
                      className={`w-full bg-transparent text-2xl font-bold outline-none ${isDark ? 'text-gray-300' : 'text-gray-600'}`}
                    />
                  </div>
                  <div className="relative">
                    <button
                      onClick={() => { setShowToList(!showToList); setShowFromList(false); }}
                      className={`flex items-center space-x-2 px-4 py-2 rounded-lg ${isDark ? 'bg-gray-600 hover:bg-gray-500' : 'bg-gray-200 hover:bg-gray-300'}`}
                    >
                      <span>{toSymbol}</span>
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                      </svg>
                    </button>
                    {showToList && (
                      <div className={`absolute top-full right-0 mt-2 rounded-lg shadow-xl z-20 w-48 max-h-60 overflow-y-auto ${isDark ? 'bg-gray-700' : 'bg-white border border-gray-200'}`}>
                        {(supportedTokens.length ? supportedTokens : []).map((t) => (
                          <button
                            key={t.symbol}
                            onClick={() => { setToSymbol(t.symbol); setShowToList(false); }}
                            className={`w-full px-4 py-2 flex items-center justify-between ${isDark ? 'hover:bg-gray-600' : 'hover:bg-gray-100'}`}
                          >
                            <span>{t.symbol}</span>
                            <span className={`text-xs ${subText}`}>{t.name}</span>
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              </div>

              {/* Rate Info */}
              {quote && (
                <div className={`rounded-lg p-4 ${innerCls}`}>
                  <div className="flex justify-between mb-2">
                    <span className={subText}>Exchange Rate</span>
                    <span>1 {quote.from_token} = {Number(quote.rate).toFixed(6)} {quote.to_token}</span>
                  </div>
                  <div className="flex justify-between mb-2">
                    <span className={subText}>Price Impact</span>
                    <span>{Number(quote.price_impact).toFixed(2)}%</span>
                  </div>
                  <div className="flex justify-between">
                    <span className={subText}>Minimum Received</span>
                    <span>{Number(quote.min_received).toFixed(6)} {quote.to_token}</span>
                  </div>
                </div>
              )}

              {/* Convert Button */}
              <button
                onClick={handleConvert}
                disabled={!quote || converting}
                className="w-full bg-blue-600 py-4 rounded-lg font-bold text-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {converting ? 'Converting...' : 'Convert Now'}
              </button>
            </div>
          </div>
        )}

        {activeTab === 'history' && (
          <div className={`rounded-xl p-6 ${cardCls}`}>
            <h3 className="text-xl font-bold mb-4">Conversion History</h3>
            {historyLoading ? (
              <div className={`text-center py-12 ${subText}`}>Loading...</div>
            ) : history.length === 0 ? (
              <div className={`text-center py-12 ${subText}`}>No conversions yet</div>
            ) : (
              <div className="space-y-4">
                {history.map((c) => (
                  <div key={c.id} className={`rounded-lg p-4 flex justify-between items-center ${innerCls}`}>
                    <div>
                      <div className="font-bold">{c.from_token} → {c.to_token}</div>
                      <div className={`text-sm ${subText}`}>{new Date(c.created_at).toLocaleString()}</div>
                      <div className={`text-xs ${subText}`}>Chain {c.chain_id}</div>
                    </div>
                    <div className="text-right">
                      <div className="font-bold">{c.from_amount} {c.from_token}</div>
                      <div className="text-green-400">+{Number(c.to_amount).toFixed(6)} {c.to_token}</div>
                      <div className={`text-xs ${subText}`}>{c.status}</div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
