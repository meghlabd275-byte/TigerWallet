'use client';

import React, { useState, useEffect } from 'react';

interface P2PAdvert {
  id: string;
  userId: string;
  username: string;
  avatar: string;
  side: 'BUY' | 'SELL';
  token: string;
  fiatCurrency: string;
  paymentMethod: string;
  price: number;
  minAmount: number;
  maxAmount: number;
  availableAmount: number;
  ordersCompleted: number;
  completionRate: number;
  avgReleaseTime: number;
  isOnline: boolean;
}

interface P2POrder {
  id: string;
  advertId: string;
  side: string;
  token: string;
  fiatCurrency: string;
  paymentMethod: string;
  price: number;
  amount: number;
  fiatAmount: number;
  status: string;
  createTime: Date;
}

const TOKENS = ['USDT', 'BTC', 'ETH', 'USDC', 'BNB'];
const FIAT_CURRENCIES = ['USD', 'EUR', 'GBP', 'CNY', 'INR'];
const PAYMENT_METHODS = ['Bank Transfer', 'PayPal', 'AliPay', 'WeChat Pay', 'UPI', 'Gift Card'];

const generateAdverts = (): P2PAdvert[] => {
  const users = [
    { username: 'CryptoTrader1', avatar: '🧑‍💼', online: true },
    { username: 'BitSeller', avatar: '👨‍💻', online: true },
    { username: 'FastTrade', avatar: '⚡', online: false },
    { username: 'P2PPro', avatar: '🎯', online: true },
    { username: 'SecureDeal', avatar: '🔒', online: true },
  ];
  
  const basePrices: Record<string, number> = {
    'USDT': 1.0, 'BTC': 43250, 'ETH': 2280, 'USDC': 1.0, 'BNB': 312.5
  };
  
  const adverts: P2PAdvert[] = [];
  let id = 0;
  
  for (const user of users) {
    for (const token of TOKENS) {
      for (const fiat of FIAT_CURRENCIES) {
        const priceVariation = (Math.random() * 10 - 5) / 1000;
        const basePrice = basePrices[token] || 10;
        
        adverts.push({
          id: `advert_${id++}`,
          userId: `user_${users.indexOf(user)}`,
          username: user.username,
          avatar: user.avatar,
          side: id % 2 === 0 ? 'BUY' : 'SELL',
          token,
          fiatCurrency: fiat,
          paymentMethod: PAYMENT_METHODS[id % PAYMENT_METHODS.length],
          price: basePrice * (1 + priceVariation),
          minAmount: fiat === 'USD' ? 10 : 100,
          maxAmount: fiat === 'USD' ? 5000 : 50000,
          availableAmount: (id * 0.5 + 1) * basePrice,
          ordersCompleted: 50 + id * 10,
          completionRate: 95 + (id % 5),
          avgReleaseTime: 2 + (id % 10),
          isOnline: user.online,
        });
      }
    }
  }
  return adverts;
};

export default function P2PTradingPage() {
  const [adverts, setAdverts] = useState<P2PAdvert[]>([]);
  const [selectedToken, setSelectedToken] = useState('USDT');
  const [selectedFiat, setSelectedFiat] = useState('USD');
  const [selectedSide, setSelectedSide] = useState<'BUY' | 'SELL'>('BUY');
  const [selectedPayment, setSelectedPayment] = useState('All');
  const [activeTab, setActiveTab] = useState<'orders' | 'create'>('orders');
  const [orders, setOrders] = useState<P2POrder[]>([]);

  useEffect(() => {
    setAdverts(generateAdverts());
  }, []);

  const filteredAdverts = adverts.filter(a => 
    a.token === selectedToken && 
    a.fiatCurrency === selectedFiat &&
    a.side === selectedSide &&
    (selectedPayment === 'All' || a.paymentMethod === selectedPayment)
  );

  const handleBuy = (advert: P2PAdvert) => {
    const order: P2POrder = {
      id: `order_${Date.now()}`,
      advertId: advert.id,
      side: advert.side,
      token: advert.token,
      fiatCurrency: advert.fiatCurrency,
      paymentMethod: advert.paymentMethod,
      price: advert.price,
      amount: advert.minAmount,
      fiatAmount: advert.minAmount * advert.price,
      status: 'PENDING',
      createTime: new Date(),
    };
    setOrders([order, ...orders]);
    alert(`Order created! Send ${order.fiatAmount.toFixed(2)} ${advert.fiatCurrency} to complete the trade.`);
  };

  return (
    <div className="min-h-screen bg-gray-900 text-white p-6">
      <div className="max-w-7xl mx-auto">
        <div className="mb-8">
          <h1 className="text-3xl font-bold">P2P Trading</h1>
          <p className="text-gray-400 mt-1">Buy and sell crypto directly with other users</p>
        </div>

        {/* Filters */}
        <div className="bg-gray-800 rounded-xl p-6 mb-6">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div>
              <label className="block text-sm text-gray-400 mb-2">Token</label>
              <select
                value={selectedToken}
                onChange={(e) => setSelectedToken(e.target.value)}
                className="w-full bg-gray-700 border border-gray-600 rounded-lg px-4 py-3"
              >
                {TOKENS.map(t => <option key={t} value={t}>{t}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm text-gray-400 mb-2">Fiat Currency</label>
              <select
                value={selectedFiat}
                onChange={(e) => setSelectedFiat(e.target.value)}
                className="w-full bg-gray-700 border border-gray-600 rounded-lg px-4 py-3"
              >
                {FIAT_CURRENCIES.map(f => <option key={f} value={f}>{f}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm text-gray-400 mb-2">Payment Method</label>
              <select
                value={selectedPayment}
                onChange={(e) => setSelectedPayment(e.target.value)}
                className="w-full bg-gray-700 border border-gray-600 rounded-lg px-4 py-3"
              >
                <option value="All">All Methods</option>
                {PAYMENT_METHODS.map(p => <option key={p} value={p}>{p}</option>)}
              </select>
            </div>
            <div>
              <label className="block text-sm text-gray-400 mb-2">Type</label>
              <div className="flex space-x-2">
                <button
                  onClick={() => setSelectedSide('BUY')}
                  className={`flex-1 py-3 rounded-lg font-bold ${selectedSide === 'BUY' ? 'bg-green-600' : 'bg-gray-700'}`}
                >
                  Buy
                </button>
                <button
                  onClick={() => setSelectedSide('SELL')}
                  className={`flex-1 py-3 rounded-lg font-bold ${selectedSide === 'SELL' ? 'bg-red-600' : 'bg-gray-700'}`}
                >
                  Sell
                </button>
              </div>
            </div>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex space-x-4 mb-6 border-b border-gray-700">
          <button
            onClick={() => setActiveTab('orders')}
            className={`pb-3 px-4 font-medium ${activeTab === 'orders' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400'}`}
          >
            Browse Ads
          </button>
          <button
            onClick={() => setActiveTab('create')}
            className={`pb-3 px-4 font-medium ${activeTab === 'create' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400'}`}
          >
            Create Ad
          </button>
        </div>

        {activeTab === 'orders' && (
          <div className="space-y-4">
            {filteredAdverts.map(advert => (
              <div key={advert.id} className="bg-gray-800 rounded-xl p-6">
                <div className="flex justify-between items-start">
                  <div className="flex items-center space-x-4">
                    <div className="w-12 h-12 bg-gray-700 rounded-full flex items-center justify-center text-2xl">
                      {advert.avatar}
                    </div>
                    <div>
                      <div className="font-bold flex items-center gap-2">
                        {advert.username}
                        {advert.isOnline && <span className="w-2 h-2 bg-green-500 rounded-full"></span>}
                      </div>
                      <div className="text-sm text-gray-400">
                        {advert.ordersCompleted} orders • {advert.completionRate}% completion
                      </div>
                    </div>
                  </div>
                  <div className="text-right">
                    <div className={`text-xl font-bold ${advert.side === 'BUY' ? 'text-green-400' : 'text-red-400'}`}>
                      {advert.side === 'BUY' ? 'Buy' : 'Sell'} {advert.token}
                    </div>
                    <div className="text-2xl font-bold">{advert.price.toFixed(2)} {advert.fiatCurrency}</div>
                  </div>
                </div>
                <div className="mt-4 flex justify-between items-center">
                  <div className="text-sm text-gray-400">
                    <div>Payment: {advert.paymentMethod}</div>
                    <div>Limits: {advert.minAmount} - {advert.maxAmount} {advert.fiatCurrency}</div>
                    <div>Available: {advert.availableAmount.toFixed(4)} {advert.token}</div>
                    <div>Release: ~{advert.avgReleaseTime} min</div>
                  </div>
                  <button
                    onClick={() => handleBuy(advert)}
                    className={`px-6 py-3 rounded-lg font-bold ${advert.side === 'BUY' ? 'bg-green-600 hover:bg-green-700' : 'bg-red-600 hover:bg-red-700'}`}
                  >
                    {advert.side === 'BUY' ? 'Buy Now' : 'Sell Now'}
                  </button>
                </div>
              </div>
            ))}
            {filteredAdverts.length === 0 && (
              <div className="text-center py-12 text-gray-400">No ads found for this selection</div>
            )}
          </div>
        )}

        {activeTab === 'create' && (
          <div className="bg-gray-800 rounded-xl p-6">
            <h3 className="text-xl font-bold mb-6">Create P2P Advertisement</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label className="block text-sm text-gray-400 mb-2">I want to</label>
                <select className="w-full bg-gray-700 border border-gray-600 rounded-lg px-4 py-3">
                  <option>Buy</option>
                  <option>Sell</option>
                </select>
              </div>
              <div>
                <label className="block text-sm text-gray-400 mb-2">Token</label>
                <select className="w-full bg-gray-700 border border-gray-600 rounded-lg px-4 py-3">
                  {TOKENS.map(t => <option key={t}>{t}</option>)}
                </select>
              </div>
              <div>
                <label className="block text-sm text-gray-400 mb-2">Fiat Currency</label>
                <select className="w-full bg-gray-700 border border-gray-600 rounded-lg px-4 py-3">
                  {FIAT_CURRENCIES.map(f => <option key={f}>{f}</option>)}
                </select>
              </div>
              <div>
                <label className="block text-sm text-gray-400 mb-2">Price (with spread)</label>
                <input type="number" className="w-full bg-gray-700 border border-gray-600 rounded-lg px-4 py-3" placeholder="0.00" />
              </div>
              <div>
                <label className="block text-sm text-gray-400 mb-2">Min Amount</label>
                <input type="number" className="w-full bg-gray-700 border border-gray-600 rounded-lg px-4 py-3" placeholder="0.00" />
              </div>
              <div>
                <label className="block text-sm text-gray-400 mb-2">Max Amount</label>
                <input type="number" className="w-full bg-gray-700 border border-gray-600 rounded-lg px-4 py-3" placeholder="0.00" />
              </div>
              <div className="md:col-span-2">
                <label className="block text-sm text-gray-400 mb-2">Payment Methods</label>
                <div className="grid grid-cols-3 gap-2">
                  {PAYMENT_METHODS.map(p => (
                    <label key={p} className="flex items-center space-x-2 bg-gray-700 p-3 rounded-lg cursor-pointer hover:bg-gray-600">
                      <input type="checkbox" />
                      <span>{p}</span>
                    </label>
                  ))}
                </div>
              </div>
            </div>
            <button className="w-full bg-blue-600 py-4 rounded-lg font-bold text-lg mt-6 hover:bg-blue-700">
              Create Advertisement
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
