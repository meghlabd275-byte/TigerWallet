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
  if (!response.ok) throw new Error(`API Error: ${response.statusText}`);
  const data = await response.json();
  return data?.data ?? data;
};

interface Brand {
  id: string;
  name: string;
  logo: string;
  min_amount: number;
  max_amount: number;
  discount: number;
}

interface MyGiftCard {
  id: string;
  brand: string;
  code: string;
  amount: number;
  status: string;
  created_at?: string;
  expires_at?: string;
}

export default function GiftCardsPage() {
  const { isDark } = useTheme();
  const [activeTab, setActiveTab] = useState<'buy' | 'my_cards'>('buy');
  const [selectedBrand, setSelectedBrand] = useState<Brand | null>(null);
  const [buyAmount, setBuyAmount] = useState(25);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pageLoading, setPageLoading] = useState(true);
  const [brands, setBrands] = useState<Brand[]>([]);
  const [myCards, setMyCards] = useState<MyGiftCard[]>([]);
  const [message, setMessage] = useState<string | null>(null);

  // Fetch the real brand catalog + the user's owned cards from the
  // gift_card_service backend (PostgreSQL-backed, CSPRNG codes). No hardcoded
  // brand list / fake "MY_CARDS" seed.
  const load = useCallback(async () => {
    setPageLoading(true);
    setError(null);
    try {
      let userId = '';
      try {
        const profile = await fetchAPI<any>('/user/profile');
        userId = profile?.id || profile?.user_id || profile?.userId || '';
      } catch (err) { /* not authenticated yet */ }
      const [brandsRes, cardsRes] = await Promise.all([
        fetchAPI<{ brands: Brand[] }>(`/gift-cards/brands`).catch(() => ({ brands: [] })),
        fetchAPI<{ cards: MyGiftCard[] }>(`/gift-cards/list${userId ? `?user_id=${encodeURIComponent(userId)}` : ''}`).catch(() => ({ cards: [] })),
      ]);
      setBrands(brandsRes?.brands ?? []);
      setMyCards(cardsRes?.cards ?? []);
    } catch (err) {
      setError('Failed to load gift cards. Please try again.');
    } finally {
      setPageLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const handleBuy = async () => {
    if (!selectedBrand) return;
    setLoading(true);
    setError(null);
    try {
      let userId = '';
      try {
        const profile = await fetchAPI<any>('/user/profile');
        userId = profile?.id || profile?.user_id || profile?.userId || '';
      } catch (err) { /* not authenticated */ }
      await fetchAPI('/gift-cards/buy', {
        method: 'POST',
        body: JSON.stringify({ brand: selectedBrand.id, token: 'USDC', amount: buyAmount, user_id: userId }),
      });
      setMessage(`Successfully purchased $${buyAmount} ${selectedBrand.name} gift card!`);
      setSelectedBrand(null);
      await load();
    } catch (err) {
      setError('Purchase failed. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleRedeem = async (code: string) => {
    if (!code) return;
    setLoading(true);
    setError(null);
    try {
      let userId = '';
      try {
        const profile = await fetchAPI<any>('/user/profile');
        userId = profile?.id || profile?.user_id || profile?.userId || '';
      } catch (err) { /* not authenticated */ }
      await fetchAPI('/gift-cards/redeem', {
        method: 'POST',
        body: JSON.stringify({ code, user_id: userId }),
      });
      setMessage('Gift card redeemed successfully.');
      await load();
    } catch (err) {
      setError('Redeem failed. Check the code and try again.');
    } finally {
      setLoading(false);
    }
  };

  const stats = {
    totalSaved: myCards.reduce((s, c) => s + (c.amount || 0) * 0.1, 0),
    cardsOwned: myCards.length,
    totalValue: myCards.reduce((s, c) => s + (c.amount || 0), 0),
  };

  return (
    <div className={`min-h-screen ${isDark ? 'bg-slate-900 text-white' : 'bg-slate-50 text-slate-900'}`}>
      <header className="bg-gradient-to-r from-pink-600 to-purple-600 text-white">
        <div className="max-w-7xl mx-auto px-4 py-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <a href="/" className="text-3xl">🎁</a>
              <div>
                <h1 className="text-2xl font-bold">Gift Cards</h1>
                <p className="text-pink-200">Buy crypto-funded gift cards</p>
              </div>
            </div>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto px-4 py-6">
        {error && (
          <div className="rounded-lg p-3 mb-4 bg-red-100 text-red-800 text-sm flex justify-between">
            <span>{error}</span>
            <button onClick={load} className="underline">Retry</button>
          </div>
        )}
        {message && (
          <div className="rounded-lg p-3 mb-4 bg-green-100 text-green-800 text-sm">{message}</div>
        )}
        {pageLoading && (
          <div className={`rounded-lg p-3 mb-4 ${isDark ? 'bg-slate-800 text-slate-400' : 'bg-white border border-gray-200 text-slate-500'}`}>
            Loading gift cards...
          </div>
        )}
        {/* Stats */}
        <div className="grid grid-cols-3 gap-4 mb-6">
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-4`}>
            <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Total Saved</p>
            <p className="text-2xl font-bold text-green-600">${stats.totalSaved}</p>
          </div>
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-4`}>
            <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Cards Owned</p>
            <p className="text-2xl font-bold">{stats.cardsOwned}</p>
          </div>
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-lg p-4`}>
            <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Total Value</p>
            <p className="text-2xl font-bold">${stats.totalValue}</p>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex gap-2 mb-6">
          <button
            onClick={() => setActiveTab('buy')}
            className={`px-6 py-2 rounded-lg font-medium ${activeTab === 'buy' ? 'bg-pink-600 text-white' : `${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'}`}`}
          >
            Buy Gift Cards
          </button>
          <button
            onClick={() => setActiveTab('my_cards')}
            className={`px-6 py-2 rounded-lg font-medium ${activeTab === 'my_cards' ? 'bg-pink-600 text-white' : `${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'}`}`}
          >
            My Cards
          </button>
        </div>

        {activeTab === 'buy' && (
          <div>
            {brands.length === 0 && !pageLoading && (
              <p className={`text-center py-8 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>No brands available.</p>
            )}
            <div className="grid grid-cols-4 gap-4">
              {brands.map(brand => (
                <div
                  key={brand.id}
                  onClick={() => { setSelectedBrand(brand); setBuyAmount(brand.min_amount); }}
                  className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-xl p-4 hover:border-pink-500 transition-colors cursor-pointer`}
                >
                  <div className="text-4xl mb-3">{brand.logo}</div>
                  <h3 className="font-bold mb-2">{brand.name}</h3>
                  <div className="flex justify-between items-end">
                    <div>
                      <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>From</p>
                      <p className="text-xl font-bold">${brand.min_amount}</p>
                    </div>
                    <div className="text-right">
                      <p className="text-xs text-green-600">Save {brand.discount}%</p>
                      <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Up to ${brand.max_amount}</p>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {activeTab === 'my_cards' && (
          <div className="space-y-4">
            {myCards.length === 0 && !pageLoading && (
              <p className={`text-center py-8 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>No gift cards yet. Buy one to get started.</p>
            )}
            {myCards.map(card => (
              <div key={card.id} className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-xl p-6`}>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-4">
                    <span className="text-4xl">🎁</span>
                    <div>
                      <h3 className="font-bold text-lg">{card.brand}</h3>
                      <p className={`font-mono text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>{card.code}</p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="text-2xl font-bold">${card.amount}</p>
                    <p className={`text-sm ${card.status === 'REDEEMED' ? 'text-yellow-600' : 'text-green-600'}`}>{card.status}</p>
                    {card.expires_at && (
                      <p className={`text-sm ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Expires: {new Date(card.expires_at).toLocaleDateString()}</p>
                    )}
                  </div>
                </div>
                <div className="flex gap-2 mt-4">
                  <button
                    onClick={() => handleRedeem(card.code)}
                    disabled={loading || card.status === 'REDEEMED'}
                    className="flex-1 py-2 bg-pink-600 text-white rounded-lg hover:bg-pink-700 disabled:opacity-50"
                  >
                    {card.status === 'REDEEMED' ? 'Redeemed' : 'Redeem'}
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Buy Modal */}
      {selectedBrand && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'} rounded-xl p-6 max-w-md w-full mx-4`}>
            <div className="flex items-center gap-4 mb-4">
              <span className="text-4xl">{selectedBrand.logo}</span>
              <div>
                <h3 className="text-xl font-bold">{selectedBrand.name} Gift Card</h3>
                <p className="text-green-600">Save {selectedBrand.discount}%</p>
              </div>
            </div>

            <div className="mb-4">
              <label className="block text-sm font-medium mb-2">Select Amount</label>
              <div className="grid grid-cols-4 gap-2">
                {[selectedBrand.min_amount, 50, 100, selectedBrand.max_amount].filter((v, i, a) => a.indexOf(v) === i).map(amt => (
                  <button
                    key={amt}
                    onClick={() => setBuyAmount(amt)}
                    className={`py-2 rounded-lg border ${buyAmount === amt ? 'bg-pink-600 text-white border-pink-600' : `${isDark ? 'border-slate-600' : 'border-slate-300'}`}`}
                  >
                    ${amt}
                  </button>
                ))}
              </div>
            </div>

            <div className={`p-4 ${isDark ? 'bg-slate-700' : 'bg-slate-50'} rounded-lg mb-4`}>
              <div className="flex justify-between mb-2">
                <span className={isDark ? 'text-slate-400' : 'text-slate-500'}>Gift Card Value</span>
                <span>${buyAmount}</span>
              </div>
              <div className="flex justify-between mb-2">
                <span className={isDark ? 'text-slate-400' : 'text-slate-500'}>You Pay</span>
                <span className="font-bold">${(buyAmount * (1 - selectedBrand.discount / 100)).toFixed(2)}</span>
              </div>
              <div className="flex justify-between">
                <span className={isDark ? 'text-slate-400' : 'text-slate-500'}>You Save</span>
                <span className="text-green-600">${(buyAmount * selectedBrand.discount / 100).toFixed(2)}</span>
              </div>
            </div>

            <div className="flex gap-4">
              <button onClick={() => setSelectedBrand(null)} className="flex-1 py-3 bg-slate-200 rounded-lg">
                Cancel
              </button>
              <button onClick={handleBuy} disabled={loading} className="flex-1 py-3 bg-pink-600 text-white rounded-lg disabled:opacity-50">
                {loading ? 'Processing...' : 'Buy Now'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
