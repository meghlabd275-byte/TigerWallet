'use client';

import React, { useState, useEffect } from 'react';
import { useTheme } from '../components/ThemeProvider'

interface FiatProvider {
  id: string;
  name: string;
  logo: string;
  supportedFiat: string[];
  supportedCrypto: string[];
  minAmount: number;
  maxAmount: number;
  feePercent: number;
  processingTime: string;
  paymentMethods: string[];
  isAvailable: boolean;
}

interface FiatOrder {
  id: string;
  provider: string;
  side: 'BUY' | 'SELL';
  fiatCurrency: string;
  cryptoCurrency: string;
  fiatAmount: number;
  cryptoAmount: number;
  status: string;
}

const PROVIDERS: FiatProvider[] = [
  { id: '1', name: 'MoonPay', logo: '🌙', supportedFiat: ['USD', 'EUR', 'GBP', 'AUD', 'CAD'], supportedCrypto: ['BTC', 'ETH', 'USDT', 'BNB', 'SOL'], minAmount: 30, maxAmount: 50000, feePercent: 2.5, processingTime: '5-30 min', paymentMethods: ['Card', 'Bank'], isAvailable: true },
  { id: '2', name: 'Simplex', logo: '💳', supportedFiat: ['USD', 'EUR', 'GBP'], supportedCrypto: ['BTC', 'ETH', 'USDT'], minAmount: 50, maxAmount: 25000, feePercent: 3.5, processingTime: '10-60 min', paymentMethods: ['Card', 'Apple Pay'], isAvailable: true },
  { id: '3', name: 'Transak', logo: '🔄', supportedFiat: ['USD', 'EUR', 'GBP', 'INR'], supportedCrypto: ['BTC', 'ETH', 'USDT', 'MATIC', 'AVAX'], minAmount: 20, maxAmount: 100000, feePercent: 2.0, processingTime: '15-45 min', paymentMethods: ['Bank', 'UPI', 'Card'], isAvailable: true },
  { id: '4', name: 'OnRamper', logo: '📱', supportedFiat: ['USD', 'EUR', 'GBP', 'AUD'], supportedCrypto: ['BTC', 'ETH', 'USDT', 'ADA', 'DOT'], minAmount: 25, maxAmount: 75000, feePercent: 1.8, processingTime: '5-20 min', paymentMethods: ['Card', 'Apple Pay'], isAvailable: true },
];

const CRYPTO_TOKENS = ['BTC', 'ETH', 'USDT', 'USDC', 'BNB', 'SOL', 'ADA', 'MATIC'];
const FIAT_CURRENCIES = ['USD', 'EUR', 'GBP', 'INR', 'AUD'];

export default function FiatRampPage() {
  const { isDark } = useTheme()
  const [side, setSide] = useState<'BUY' | 'SELL'>('BUY');
  const [selectedProvider, setSelectedProvider] = useState<string>('1');
  const [fiatCurrency, setFiatCurrency] = useState('USD');
  const [cryptoCurrency, setCryptoCurrency] = useState('BTC');
  const [fiatAmount, setFiatAmount] = useState('');
  const [orders, setOrders] = useState<FiatOrder[]>([]);
  const [activeTab, setActiveTab] = useState<'buy' | 'orders'>('buy');

  const provider = PROVIDERS.find(p => p.id === selectedProvider) || PROVIDERS[0];
  
  const baseRates: Record<string, number> = {
    'BTC': 43250, 'ETH': 2280, 'USDT': 1, 'USDC': 1, 'BNB': 312.5, 'SOL': 98.75, 'ADA': 0.58, 'MATIC': 0.92
  };

  const cryptoAmount = (() => {
    const amount = parseFloat(fiatAmount) || 0;
    const fee = amount * (provider.feePercent / 100);
    const netAmount = amount - fee;
    return netAmount / (baseRates[cryptoCurrency] || 1);
  })();

  const handleSubmit = () => {
    const order: FiatOrder = {
      id: `order_${Date.now()}`,
      provider: provider.name,
      side,
      fiatCurrency,
      cryptoCurrency,
      fiatAmount: parseFloat(fiatAmount) || 0,
      cryptoAmount,
      status: 'PENDING',
    };
    setOrders([order, ...orders]);
    alert(`Order created! Complete payment to receive your ${cryptoCurrency}.`);
  };

  return (
    <div className={`'min-h-screen' ${isDark ? 'bg-gray-900' : 'bg-gray-50'} ${isDark ? 'text-white' : 'text-gray-900'} 'p-6'`}>
      <div className="max-w-4xl mx-auto">
        <div className="mb-8">
          <h1 className="text-3xl font-bold">Buy Crypto</h1>
          <p className={`${isDark ? 'text-gray-400' : 'text-gray-500'} 'mt-1'`}>Purchase cryptocurrency using fiat currency</p>
        </div>

        {/* Tabs */}
        <div className={`'flex space-x-4 mb-6 border-b' ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
          <button
            onClick={() => setActiveTab('buy')}
            className={`pb-3 px-4 font-medium ${activeTab === 'buy' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400'}`}
          >
            Buy Crypto
          </button>
          <button
            onClick={() => setActiveTab('orders')}
            className={`pb-3 px-4 font-medium ${activeTab === 'orders' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400'}`}
          >
            My Orders
          </button>
        </div>

        {activeTab === 'buy' && (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* Main Form */}
            <div className={`'lg:col-span-2' ${isDark ? 'bg-gray-800' : 'bg-white'} 'rounded-xl p-6'`}>
              {/* Side Selection */}
              <div className="flex space-x-4 mb-6">
                <button
                  onClick={() => setSide('BUY')}
                  className={`flex-1 py-3 rounded-lg font-bold ${side === 'BUY' ? 'bg-green-600' : 'bg-gray-700'}`}
                >
                  Buy
                </button>
                <button
                  onClick={() => setSide('SELL')}
                  className={`flex-1 py-3 rounded-lg font-bold ${side === 'SELL' ? 'bg-red-600' : 'bg-gray-700'}`}
                >
                  Sell
                </button>
              </div>

              {/* Provider Selection */}
              <div className="mb-6">
                <label className={`'block text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'} 'mb-2'`}>Provider</label>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                  {PROVIDERS.map(p => (
                    <button
                      key={p.id}
                      onClick={() => setSelectedProvider(p.id)}
                      className={`p-4 rounded-lg border-2 ${selectedProvider === p.id ? 'border-blue-500 bg-blue-500/20' : 'border-gray-600 hover:border-gray-500'}`}
                    >
                      <div className="text-2xl mb-1">{p.logo}</div>
                      <div className="font-bold text-sm">{p.name}</div>
                    </button>
                  ))}
                </div>
              </div>

              {/* Amount Inputs */}
              <div className="grid grid-cols-2 gap-4 mb-6">
                <div>
                  <label className={`'block text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'} 'mb-2'`}>You Pay</label>
                  <input
                    type="number"
                    value={fiatAmount}
                    onChange={(e) => setFiatAmount(e.target.value)}
                    placeholder="0.00"
                    className={`'w-full' ${isDark ? 'bg-gray-700' : 'bg-gray-100'} 'border border-gray-600 rounded-lg px-4 py-3 text-xl'`}
                  />
                  <select
                    value={fiatCurrency}
                    onChange={(e) => setFiatCurrency(e.target.value)}
                    className={`'w-full' ${isDark ? 'bg-gray-700' : 'bg-gray-100'} 'border border-gray-600 rounded-lg px-4 py-2 mt-2'`}
                  >
                    {FIAT_CURRENCIES.map(f => <option key={f} value={f}>{f}</option>)}
                  </select>
                </div>
                <div>
                  <label className={`'block text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'} 'mb-2'`}>You Receive</label>
                  <input
                    type="text"
                    value={cryptoAmount.toFixed(6)}
                    readOnly
                    className={`'w-full' ${isDark ? 'bg-gray-700' : 'bg-gray-100'} 'border border-gray-600 rounded-lg px-4 py-3 text-xl' ${isDark ? 'text-gray-300' : 'text-gray-600'}`}
                  />
                  <select
                    value={cryptoCurrency}
                    onChange={(e) => setCryptoCurrency(e.target.value)}
                    className={`'w-full' ${isDark ? 'bg-gray-700' : 'bg-gray-100'} 'border border-gray-600 rounded-lg px-4 py-2 mt-2'`}
                  >
                    {CRYPTO_TOKENS.map(c => <option key={c} value={c}>{c}</option>)}
                  </select>
                </div>
              </div>

              {/* Exchange Summary */}
              <div className={`${isDark ? 'bg-gray-700' : 'bg-gray-100'} 'rounded-lg p-4 mb-6'`}>
                <div className="flex justify-between mb-2">
                  <span className={`${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Exchange Rate</span>
                  <span>1 {cryptoCurrency} = {(baseRates[cryptoCurrency]).toFixed(2)} {fiatCurrency}</span>
                </div>
                <div className="flex justify-between mb-2">
                  <span className={`${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Provider Fee</span>
                  <span>{provider.feePercent}%</span>
                </div>
                <div className="flex justify-between">
                  <span className={`${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Processing Time</span>
                  <span>{provider.processingTime}</span>
                </div>
              </div>

              {/* Submit */}
              <button
                onClick={handleSubmit}
                className="w-full bg-blue-600 py-4 rounded-lg font-bold text-lg hover:bg-blue-700"
              >
                {side === 'BUY' ? 'Buy Now' : 'Sell Now'}
              </button>
            </div>

            {/* Provider Info */}
            <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} 'rounded-xl p-6'`}>
              <h3 className="text-lg font-bold mb-4">Provider Details</h3>
              <div className="space-y-4">
                <div className="flex items-center space-x-3">
                  <span className="text-3xl">{provider.logo}</span>
                  <div>
                    <div className="font-bold">{provider.name}</div>
                    <div className={`'text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{provider.processingTime}</div>
                  </div>
                </div>
                <div className={`'border-t' ${isDark ? 'border-gray-700' : 'border-gray-200'} 'pt-4'`}>
                  <div className={`'text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'} 'mb-1'`}>Min / Max</div>
                  <div>{provider.minAmount} - {provider.maxAmount} {fiatCurrency}</div>
                </div>
                <div className={`'border-t' ${isDark ? 'border-gray-700' : 'border-gray-200'} 'pt-4'`}>
                  <div className={`'text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'} 'mb-1'`}>Fee</div>
                  <div className="text-green-400">{provider.feePercent}%</div>
                </div>
                <div className={`'border-t' ${isDark ? 'border-gray-700' : 'border-gray-200'} 'pt-4'`}>
                  <div className={`'text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'} 'mb-2'`}>Payment Methods</div>
                  <div className="flex flex-wrap gap-2">
                    {provider.paymentMethods.map(m => (
                      <span key={m} className={`${isDark ? 'bg-gray-700' : 'bg-gray-100'} 'px-2 py-1 rounded text-sm'`}>{m}</span>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'orders' && (
          <div className={`${isDark ? 'bg-gray-800' : 'bg-white'} 'rounded-xl p-6'`}>
            <h3 className="text-xl font-bold mb-4">Order History</h3>
            {orders.length === 0 ? (
              <div className={`'text-center py-12' ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>No orders yet</div>
            ) : (
              <div className="space-y-4">
                {orders.map(order => (
                  <div key={order.id} className={`${isDark ? 'bg-gray-700' : 'bg-gray-100'} 'rounded-lg p-4 flex justify-between items-center'`}>
                    <div>
                      <div className="font-bold">{order.side} {order.cryptoCurrency}</div>
                      <div className={`'text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{order.provider}</div>
                    </div>
                    <div className="text-right">
                      <div className="font-bold">{order.cryptoAmount.toFixed(6)} {order.cryptoCurrency}</div>
                      <div className={`'text-sm' ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>{order.fiatAmount.toFixed(2)} {order.fiatCurrency}</div>
                    </div>
                    <span className="bg-yellow-600 px-3 py-1 rounded text-sm">{order.status}</span>
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
