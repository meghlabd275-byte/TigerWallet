'use client';

import React, { useState } from 'react';

interface GiftCard {
  id: string;
  brand: string;
  logo: string;
  amount: number;
  value: number;
  discount: number;
  expiresAt: number;
  status: 'available' | 'redeemed' | 'expired';
}

interface MyGiftCard {
  id: string;
  brand: string;
  code: string;
  amount: number;
  purchasedAt: number;
  expiresAt: number;
  status: 'active' | 'redeemed';
}

const AVAILABLE_CARDS: GiftCard[] = [
  { id: '1', brand: 'Amazon', logo: '🛒', amount: 25, value: 22.50, discount: 10, expiresAt: Date.now() + 86400000 * 365, status: 'available' },
  { id: '2', brand: 'Apple', logo: '🍎', amount: 50, value: 45, discount: 10, expiresAt: Date.now() + 86400000 * 365, status: 'available' },
  { id: '3', brand: 'Google Play', logo: '🎮', amount: 100, value: 85, discount: 15, expiresAt: Date.now() + 86400000 * 365, status: 'available' },
  { id: '4', brand: 'Steam', logo: '🎮', amount: 50, value: 42.50, discount: 15, expiresAt: Date.now() + 86400000 * 365, status: 'available' },
  { id: '5', brand: 'Spotify', logo: '🎵', amount: 30, value: 25.50, discount: 15, expiresAt: Date.now() + 86400000 * 365, status: 'available' },
  { id: '6', brand: 'Netflix', logo: '🎬', amount: 30, value: 27, discount: 10, expiresAt: Date.now() + 86400000 * 365, status: 'available' },
  { id: '7', brand: 'Walmart', logo: '🛒', amount: 100, value: 90, discount: 10, expiresAt: Date.now() + 86400000 * 365, status: 'available' },
  { id: '8', brand: 'Target', logo: '🛍️', amount: 50, value: 42.50, discount: 15, expiresAt: Date.now() + 86400000 * 365, status: 'available' },
  { id: '9', brand: 'Visa', logo: '💳', amount: 200, value: 180, discount: 10, expiresAt: Date.now() + 86400000 * 365, status: 'available' },
  { id: '10', brand: 'Mastercard', logo: '💳', amount: 100, value: 85, discount: 15, expiresAt: Date.now() + 86400000 * 365, status: 'available' },
];

const MY_CARDS: MyGiftCard[] = [
  { id: '1', brand: 'Amazon', code: 'AMZN-XXXX-XXXX-1234', amount: 25, purchasedAt: Date.now() - 86400000 * 30, expiresAt: Date.now() + 86400000 * 300, status: 'active' },
  { id: '2', brand: 'Google Play', code: 'GPLY-XXXX-XXXX-5678', amount: 50, purchasedAt: Date.now() - 86400000 * 15, expiresAt: Date.now() + 86400000 * 315, status: 'active' },
];

export default function GiftCardsPage() {
  const [activeTab, setActiveTab] = useState<'buy' | 'my_cards'>('buy');
  const [selectedCard, setSelectedCard] = useState<GiftCard | null>(null);
  const [buyAmount, setBuyAmount] = useState(25);
  const [loading, setLoading] = useState(false);

  const handleBuy = async () => {
    if (!selectedCard) return;
    setLoading(true);
    await new Promise(r => setTimeout(r, 2000));
    alert(`Successfully purchased $${buyAmount} ${selectedCard.brand} gift card!`);
    setSelectedCard(null);
    setLoading(false);
  };

  const stats = {
    totalSaved: 45,
    cardsOwned: 2,
    totalValue: 75,
  };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900">
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
        {/* Stats */}
        <div className="grid grid-cols-3 gap-4 mb-6">
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4 border">
            <p className="text-sm text-slate-500">Total Saved</p>
            <p className="text-2xl font-bold text-green-600">${stats.totalSaved}</p>
          </div>
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4 border">
            <p className="text-sm text-slate-500">Cards Owned</p>
            <p className="text-2xl font-bold">{stats.cardsOwned}</p>
          </div>
          <div className="bg-white dark:bg-slate-800 rounded-lg p-4 border">
            <p className="text-sm text-slate-500">Total Value</p>
            <p className="text-2xl font-bold">${stats.totalValue}</p>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex gap-2 mb-6">
          <button
            onClick={() => setActiveTab('buy')}
            className={`px-6 py-2 rounded-lg font-medium ${activeTab === 'buy' ? 'bg-pink-600 text-white' : 'bg-white dark:bg-slate-800'}`}
          >
            Buy Gift Cards
          </button>
          <button
            onClick={() => setActiveTab('my_cards')}
            className={`px-6 py-2 rounded-lg font-medium ${activeTab === 'my_cards' ? 'bg-pink-600 text-white' : 'bg-white dark:bg-slate-800'}`}
          >
            My Cards
          </button>
        </div>

        {activeTab === 'buy' && (
          <div className="grid grid-cols-4 gap-4">
            {AVAILABLE_CARDS.map(card => (
              <div
                key={card.id}
                onClick={() => setSelectedCard(card)}
                className="bg-white dark:bg-slate-800 rounded-xl p-4 border hover:border-pink-500 transition-colors cursor-pointer"
              >
                <div className="text-4xl mb-3">{card.logo}</div>
                <h3 className="font-bold mb-2">{card.brand}</h3>
                <div className="flex justify-between items-end">
                  <div>
                    <p className="text-sm text-slate-500">From</p>
                    <p className="text-xl font-bold">${card.value}</p>
                  </div>
                  <div className="text-right">
                    <p className="text-xs text-green-600">Save {card.discount}%</p>
                    <p className="text-sm text-slate-400">${card.amount} value</p>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {activeTab === 'my_cards' && (
          <div className="space-y-4">
            {MY_CARDS.map(card => (
              <div key={card.id} className="bg-white dark:bg-slate-800 rounded-xl p-6 border">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-4">
                    <span className="text-4xl">🎁</span>
                    <div>
                      <h3 className="font-bold text-lg">{card.brand}</h3>
                      <p className="font-mono text-sm text-slate-500">{card.code}</p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="text-2xl font-bold">${card.amount}</p>
                    <p className="text-sm text-slate-500">Expires: {new Date(card.expiresAt).toLocaleDateString()}</p>
                  </div>
                </div>
                <div className="flex gap-2 mt-4">
                  <button className="flex-1 py-2 bg-pink-600 text-white rounded-lg hover:bg-pink-700">
                    View Code
                  </button>
                  <button className="px-4 py-2 bg-slate-100 dark:bg-slate-700 rounded-lg">
                    Share
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Buy Modal */}
      {selectedCard && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-slate-800 rounded-xl p-6 max-w-md w-full mx-4">
            <div className="flex items-center gap-4 mb-4">
              <span className="text-4xl">{selectedCard.logo}</span>
              <div>
                <h3 className="text-xl font-bold">{selectedCard.brand} Gift Card</h3>
                <p className="text-green-600">Save {selectedCard.discount}%</p>
              </div>
            </div>
            
            <div className="mb-4">
              <label className="block text-sm font-medium mb-2">Select Amount</label>
              <div className="grid grid-cols-4 gap-2">
                {[25, 50, 100, 200].map(amt => (
                  <button
                    key={amt}
                    onClick={() => setBuyAmount(amt)}
                    className={`py-2 rounded-lg border ${buyAmount === amt ? 'bg-pink-600 text-white border-pink-600' : 'border-slate-300'}`}
                  >
                    ${amt}
                  </button>
                ))}
              </div>
            </div>

            <div className="p-4 bg-slate-50 dark:bg-slate-700 rounded-lg mb-4">
              <div className="flex justify-between mb-2">
                <span className="text-slate-500">Gift Card Value</span>
                <span>${selectedCard.value * (buyAmount / selectedCard.amount)}</span>
              </div>
              <div className="flex justify-between mb-2">
                <span className="text-slate-500">You Pay</span>
                <span className="font-bold">${buyAmount}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">You Save</span>
                <span className="text-green-600">${buyAmount - (selectedCard.value * (buyAmount / selectedCard.amount))}</span>
              </div>
            </div>

            <div className="flex gap-4">
              <button onClick={() => setSelectedCard(null)} className="flex-1 py-3 bg-slate-200 rounded-lg">
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
