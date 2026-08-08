'use client';

import React, { useState, useEffect, useCallback } from 'react';
import api, { FiatRate, FiatProvider } from '../../src/lib/api/client';

// Types
interface FiatCurrency {
  code: string;
  name: string;
  symbol: string;
  icon: string;
}

interface CryptoCurrency {
  symbol: string;
  name: string;
  network: string;
  address?: string;
}

interface Order {
  id: string;
  fiatAmount: number;
  cryptoAmount: number;
  cryptoSymbol: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  createdAt: number;
}

const FIAT_CURRENCIES: FiatCurrency[] = [
  { code: 'USD', name: 'US Dollar', symbol: '$', icon: '🇺🇸' },
  { code: 'EUR', name: 'Euro', symbol: '€', icon: '🇪🇺' },
  { code: 'GBP', name: 'British Pound', symbol: '£', icon: '🇬🇧' },
  { code: 'JPY', name: 'Japanese Yen', symbol: '¥', icon: '🇯🇵' },
  { code: 'KRW', name: 'Korean Won', symbol: '₩', icon: '🇰🇷' },
  { code: 'INR', name: 'Indian Rupee', symbol: '₹', icon: '🇮🇳' },
  { code: 'BRL', name: 'Brazilian Real', symbol: 'R$', icon: '🇧🇷' },
  { code: 'AUD', name: 'Australian Dollar', symbol: 'A$', icon: '🇦🇺' },
];

const CRYPTO_CURRENCIES: CryptoCurrency[] = [
  { symbol: 'ETH', name: 'Ethereum', network: 'Ethereum' },
  { symbol: 'BTC', name: 'Bitcoin', network: 'Bitcoin' },
  { symbol: 'USDT', name: 'Tether USD', network: 'Ethereum' },
  { symbol: 'USDC', name: 'USD Coin', network: 'Ethereum' },
  { symbol: 'BNB', name: 'BNB', network: 'BNB Chain' },
  { symbol: 'SOL', name: 'Solana', network: 'Solana' },
  { symbol: 'TRX', name: 'Tron', network: 'Tron' },
  { symbol: 'PI', name: 'Pi Network', network: 'Pi Network' },
  { symbol: 'TON', name: 'Toncoin', network: 'Toncoin' },
  { symbol: 'DOGE', name: 'Dogecoin', network: 'Dogecoin' },
];

const PAYMENT_METHODS = [
  { id: 'apple_pay', name: 'Apple Pay', icon: '🍎', description: 'Instant purchase' },
  { id: 'google_pay', name: 'Google Pay', icon: '🔵', description: 'Fast checkout' },
  { id: 'debit_card', name: 'Debit Card', icon: '💳', description: 'Visa/Mastercard' },
  { id: 'credit_card', name: 'Credit Card', icon: '💳', description: 'Visa/Mastercard' },
  { id: 'bank_transfer', name: 'Bank Transfer', icon: '🏦', description: '3-5 business days' },
  { id: 'ideal', name: 'iDEAL', icon: '🇳🇱', description: 'Netherlands' },
  { id: 'sofort', name: 'Sofort', icon: '🇩🇪', description: 'Germany/Austria' },
  { id: 'giropay', name: 'Giropay', icon: '🇩🇪', description: 'Germany' },
];

export default function FiatOnRamp() {
  const [activeTab, setActiveTab] = useState<'buy' | 'sell'>('buy');
  const [fiatCurrency, setFiatCurrency] = useState<FiatCurrency>(FIAT_CURRENCIES[0]);
  const [cryptoCurrency, setCryptoCurrency] = useState<CryptoCurrency>(CRYPTO_CURRENCIES[0]);
  const [fiatAmount, setFiatAmount] = useState('');
  const [cryptoAmount, setCryptoAmount] = useState('');
  const [selectedPaymentMethod, setSelectedPaymentMethod] = useState('apple_pay');
  const [walletAddress, setWalletAddress] = useState('');
  const [orders, setOrders] = useState<Order[]>([]);
  const [rates, setRates] = useState<FiatRate[]>([]);
  const [ratesLoading, setRatesLoading] = useState(true);
  const [providers, setProviders] = useState<FiatProvider[]>([]);
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  // Current crypto exchange rate (fiat per 1 crypto) derived from API rates.
  // FiatRate.price is denominated in the rate's currency; we match the selected fiat currency.
  const currentRate = rates.find(
    r => r.symbol === cryptoCurrency.symbol && r.currency === fiatCurrency.code
  );
  const exchangeRate = currentRate?.price ?? 0;

  // Fetch live exchange rates from the fiat onramp service
  const fetchRates = useCallback(async () => {
    setRatesLoading(true);
    try {
      const res = await api.getFiatRates({ currency: fiatCurrency.code });
      if (res.success && res.data) {
        setRates(res.data);
      } else {
        setRates([]);
      }
    } catch (err) {
      console.error('Failed to load fiat rates:', err);
      setRates([]);
    } finally {
      setRatesLoading(false);
    }
  }, [fiatCurrency.code]);

  // Fetch available fiat providers for the selected currency pair
  const fetchProviders = useCallback(async () => {
    try {
      const res = await api.getFiatProviders({ fiat: fiatCurrency.code, crypto: cryptoCurrency.symbol });
      if (res.success && res.data) {
        setProviders(res.data);
      } else {
        setProviders([]);
      }
    } catch (err) {
      console.error('Failed to load fiat providers:', err);
      setProviders([]);
    }
  }, [fiatCurrency.code, cryptoCurrency.symbol]);

  useEffect(() => {
    fetchRates();
  }, [fetchRates]);

  useEffect(() => {
    fetchProviders();
  }, [fetchProviders]);

  // Recalculate crypto amount when fiat amount or rate changes
  useEffect(() => {
    if (fiatAmount && parseFloat(fiatAmount) > 0 && exchangeRate > 0) {
      const cryptoValue = parseFloat(fiatAmount) / exchangeRate;
      setCryptoAmount(cryptoValue.toFixed(6));
    } else if (!fiatAmount) {
      setCryptoAmount('');
    }
  }, [fiatAmount, exchangeRate]);

  const handleFiatAmountChange = (value: string) => {
    setFiatAmount(value);
  };

  const handleCryptoAmountChange = (value: string) => {
    setCryptoAmount(value);
    if (value && parseFloat(value) > 0 && exchangeRate > 0) {
      const fiatValue = parseFloat(value) * exchangeRate;
      setFiatAmount(fiatValue.toFixed(2));
    } else {
      setFiatAmount('');
    }
  };

  const handleBuy = async () => {
    if (!fiatAmount || parseFloat(fiatAmount) < 50) {
      setMessage({ type: 'error', text: 'Minimum purchase amount is $50' });
      return;
    }

    if (!walletAddress) {
      setMessage({ type: 'error', text: 'Please enter your wallet address' });
      return;
    }

    if (exchangeRate <= 0) {
      setMessage({ type: 'error', text: 'Exchange rate unavailable. Please try again.' });
      return;
    }

    const provider = providers.find(p => p.available);
    if (!provider) {
      setMessage({ type: 'error', text: 'No provider available for this currency pair.' });
      return;
    }

    setLoading(true);
    setMessage(null);

    try {
      const res = await api.createFiatOrder({
        providerId: provider.id,
        type: 'buy',
        fiatAmount: parseFloat(fiatAmount),
        cryptoCurrency: cryptoCurrency.symbol,
        fiatCurrency: fiatCurrency.code,
        paymentMethod: selectedPaymentMethod,
        walletAddress,
      });

      if (res.success && res.data) {
        const newOrder: Order = {
          id: res.data.id,
          fiatAmount: parseFloat(fiatAmount),
          cryptoAmount: parseFloat(cryptoAmount),
          cryptoSymbol: cryptoCurrency.symbol,
          status: res.data.status,
          createdAt: res.data.createdAt,
        };
        setOrders(prev => [newOrder, ...prev]);
        setMessage({ type: 'success', text: 'Purchase initiated! You will receive your crypto shortly.' });
        setFiatAmount('');
        setCryptoAmount('');
      } else {
        setMessage({ type: 'error', text: res.error || 'Failed to create order. Please try again.' });
      }
    } catch (err: any) {
      setMessage({ type: 'error', text: err?.response?.data?.error || err?.message || 'Failed to create order. Please try again.' });
    } finally {
      setLoading(false);
    }
  };

  const handleSell = async () => {
    if (!fiatAmount || parseFloat(fiatAmount) < 50) {
      setMessage({ type: 'error', text: 'Minimum sell amount is $50' });
      return;
    }

    if (exchangeRate <= 0) {
      setMessage({ type: 'error', text: 'Exchange rate unavailable. Please try again.' });
      return;
    }

    const provider = providers.find(p => p.available);
    if (!provider) {
      setMessage({ type: 'error', text: 'No provider available for this currency pair.' });
      return;
    }

    setLoading(true);
    setMessage(null);

    try {
      const res = await api.createFiatOrder({
        providerId: provider.id,
        type: 'sell',
        fiatAmount: parseFloat(fiatAmount),
        cryptoCurrency: cryptoCurrency.symbol,
        fiatCurrency: fiatCurrency.code,
        paymentMethod: selectedPaymentMethod,
      });

      if (res.success && res.data) {
        const newOrder: Order = {
          id: res.data.id,
          fiatAmount: parseFloat(fiatAmount),
          cryptoAmount: parseFloat(cryptoAmount),
          cryptoSymbol: cryptoCurrency.symbol,
          status: res.data.status,
          createdAt: res.data.createdAt,
        };
        setOrders(prev => [newOrder, ...prev]);
        setMessage({ type: 'success', text: 'Sell order initiated!' });
        setFiatAmount('');
        setCryptoAmount('');
      } else {
        setMessage({ type: 'error', text: res.error || 'Failed to create sell order. Please try again.' });
      }
    } catch (err: any) {
      setMessage({ type: 'error', text: err?.response?.data?.error || err?.message || 'Failed to create sell order. Please try again.' });
    } finally {
      setLoading(false);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed':
        return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
      case 'processing':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200';
      case 'pending':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200';
      case 'failed':
        return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
      default:
        return 'bg-slate-100 text-slate-800';
    }
  };

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-900 text-slate-900 dark:text-slate-50">
      {/* Header */}
      <header className="bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center gap-4">
              <a href="/" className="text-2xl">🐯</a>
              <h1 className="text-xl font-bold">Fiat On-Ramp</h1>
            </div>
            <nav className="flex gap-4">
              <a href="/wallet" className="text-slate-600 dark:text-slate-400 hover:text-orange-500">Wallet</a>
              <a href="/swap" className="text-slate-600 dark:text-slate-400 hover:text-orange-500">Swap</a>
            </nav>
          </div>
        </div>
      </header>

      {/* Message */}
      {message && (
        <div className="max-w-7xl mx-auto px-4 pt-4">
          <div className={`p-3 rounded-lg ${message.type === 'success' ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' : 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'}`}>
            {message.text}
          </div>
        </div>
      )}

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Main Form */}
          <div className="lg:col-span-2 space-y-6">
            {/* Buy/Sell Tabs */}
            <div className="bg-white dark:bg-slate-800 rounded-lg p-6 shadow-sm">
              <div className="flex border-b border-slate-200 dark:border-slate-700 mb-6">
                <button
                  onClick={() => setActiveTab('buy')}
                  className={`px-6 py-3 font-semibold ${activeTab === 'buy' ? 'border-b-2 border-orange-500 text-orange-500' : 'text-slate-500 dark:text-slate-400'}`}
                >
                  Buy Crypto
                </button>
                <button
                  onClick={() => setActiveTab('sell')}
                  className={`px-6 py-3 font-semibold ${activeTab === 'sell' ? 'border-b-2 border-orange-500 text-orange-500' : 'text-slate-500 dark:text-slate-400'}`}
                >
                  Sell Crypto
                </button>
              </div>

              {/* Currency Selection */}
              <div className="grid grid-cols-2 gap-4 mb-6">
                <div>
                  <label className="block text-sm text-slate-500 dark:text-slate-400 mb-2">You Pay</label>
                  <select
                    value={fiatCurrency.code}
                    onChange={(e) => setFiatCurrency(FIAT_CURRENCIES.find(c => c.code === e.target.value) || FIAT_CURRENCIES[0])}
                    className="w-full bg-slate-100 dark:bg-slate-700 border-0 rounded-lg px-4 py-3"
                  >
                    {FIAT_CURRENCIES.map(c => (
                      <option key={c.code} value={c.code}>{c.icon} {c.code} - {c.name}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-sm text-slate-500 dark:text-slate-400 mb-2">You Receive</label>
                  <select
                    value={cryptoCurrency.symbol}
                    onChange={(e) => setCryptoCurrency(CRYPTO_CURRENCIES.find(c => c.symbol === e.target.value) || CRYPTO_CURRENCIES[0])}
                    className="w-full bg-slate-100 dark:bg-slate-700 border-0 rounded-lg px-4 py-3"
                  >
                    {CRYPTO_CURRENCIES.map(c => (
                      <option key={c.symbol} value={c.symbol}>{c.symbol} - {c.name} ({c.network})</option>
                    ))}
                  </select>
                </div>
              </div>

              {/* Amount Input */}
              <div className="grid grid-cols-2 gap-4 mb-6">
                <div>
                  <label className="block text-sm text-slate-500 dark:text-slate-400 mb-2">Amount ({fiatCurrency.code})</label>
                  <input
                    type="number"
                    value={fiatAmount}
                    onChange={(e) => handleFiatAmountChange(e.target.value)}
                    placeholder="0.00"
                    className="w-full bg-slate-100 dark:bg-slate-700 border-0 rounded-lg px-4 py-3 text-2xl font-semibold"
                  />
                </div>
                <div>
                  <label className="block text-sm text-slate-500 dark:text-slate-400 mb-2">Amount ({cryptoCurrency.symbol})</label>
                  <input
                    type="number"
                    value={cryptoAmount}
                    onChange={(e) => handleCryptoAmountChange(e.target.value)}
                    placeholder="0.000000"
                    className="w-full bg-slate-100 dark:bg-slate-700 border-0 rounded-lg px-4 py-3 text-2xl font-semibold"
                  />
                </div>
              </div>

              {/* Quick Amount Buttons */}
              <div className="flex gap-2 mb-6">
                {[100, 250, 500, 1000, 2500, 5000].map(amount => (
                  <button
                    key={amount}
                    onClick={() => setFiatAmount(amount.toString())}
                    className="flex-1 py-2 px-3 bg-slate-100 dark:bg-slate-700 rounded-lg text-sm hover:bg-slate-200 dark:hover:bg-slate-600 transition-colors"
                  >
                    ${amount}
                  </button>
                ))}
              </div>

              {/* Payment Method */}
              <div className="mb-6">
                <label className="block text-sm text-slate-500 dark:text-slate-400 mb-2">Payment Method</label>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
                  {PAYMENT_METHODS.map((method) => (
                    <button
                      key={method.id}
                      onClick={() => setSelectedPaymentMethod(method.id)}
                      className={`p-3 rounded-lg text-center transition-colors ${
                        selectedPaymentMethod === method.id
                          ? 'bg-orange-100 dark:bg-orange-900 border-2 border-orange-500'
                          : 'bg-slate-100 dark:bg-slate-700 hover:bg-slate-200 dark:hover:bg-slate-600'
                      }`}
                    >
                      <div className="text-2xl mb-1">{method.icon}</div>
                      <div className="text-xs font-medium">{method.name}</div>
                    </button>
                  ))}
                </div>
              </div>

              {/* Wallet Address (for buy) */}
              {activeTab === 'buy' && (
                <div className="mb-6">
                  <label className="block text-sm text-slate-500 dark:text-slate-400 mb-2">Your {cryptoCurrency.network} Address</label>
                  <input
                    type="text"
                    value={walletAddress}
                    onChange={(e) => setWalletAddress(e.target.value)}
                    placeholder="0x..."
                    className="w-full bg-slate-100 dark:bg-slate-700 border-0 rounded-lg px-4 py-3 font-mono"
                  />
                </div>
              )}

              {/* Exchange Rate Info */}
              <div className="bg-slate-100 dark:bg-slate-700 rounded-lg p-4 mb-6">
                <div className="flex justify-between text-sm">
                  <span className="text-slate-500 dark:text-slate-400">Exchange Rate</span>
                  <span className="font-semibold">
                    {ratesLoading
                      ? 'Loading...'
                      : exchangeRate > 0
                      ? `1 ${cryptoCurrency.symbol} = ${fiatCurrency.symbol}${exchangeRate.toFixed(2)} ${fiatCurrency.code}`
                      : 'Unavailable'}
                  </span>
                </div>
                <div className="flex justify-between text-sm mt-2">
                  <span className="text-slate-500 dark:text-slate-400">Network Fee</span>
                  <span className="font-semibold">~{fiatCurrency.symbol}1.00</span>
                </div>
                <div className="flex justify-between text-sm mt-2">
                  <span className="text-slate-500 dark:text-slate-400">Processing Time</span>
                  <span className="font-semibold">{selectedPaymentMethod.includes('apple') || selectedPaymentMethod.includes('google') ? 'Instant' : '1-5 minutes'}</span>
                </div>
              </div>

              {/* Submit Button */}
              <button
                onClick={activeTab === 'buy' ? handleBuy : handleSell}
                disabled={loading || !fiatAmount || parseFloat(fiatAmount) < 50}
                className="w-full bg-orange-500 hover:bg-orange-600 disabled:bg-slate-400 text-white py-4 rounded-lg font-semibold text-lg transition-colors"
              >
                {loading ? 'Processing...' : activeTab === 'buy' ? `Buy ${cryptoCurrency.symbol}` : `Sell ${cryptoCurrency.symbol}`}
              </button>
            </div>
          </div>

          {/* Sidebar */}
          <div className="space-y-6">
            {/* Recent Orders */}
            <div className="bg-white dark:bg-slate-800 rounded-lg p-6 shadow-sm">
              <h3 className="font-semibold mb-4">Recent Orders</h3>
              {orders.length === 0 ? (
                <div className="text-center py-8 text-slate-500 dark:text-slate-400">
                  <div className="text-4xl mb-2">📋</div>
                  <p>No orders yet</p>
                </div>
              ) : (
                <div className="space-y-3">
                  {orders.slice(0, 5).map((order) => (
                    <div key={order.id} className="flex items-center justify-between p-3 bg-slate-100 dark:bg-slate-700 rounded-lg">
                      <div>
                        <div className="font-semibold">{order.cryptoSymbol}</div>
                        <div className="text-sm text-slate-500 dark:text-slate-400">
                          {fiatCurrency.symbol}{order.fiatAmount.toFixed(2)}
                        </div>
                      </div>
                      <span className={`px-2 py-1 rounded text-xs ${getStatusColor(order.status)}`}>
                        {order.status.toUpperCase()}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Supported Currencies */}
            <div className="bg-white dark:bg-slate-800 rounded-lg p-6 shadow-sm">
              <h3 className="font-semibold mb-4">Supported Currencies</h3>
              <div className="space-y-2">
                <div className="text-sm text-slate-500 dark:text-slate-400">Fiat</div>
                <div className="flex flex-wrap gap-1">
                  {FIAT_CURRENCIES.map(c => (
                    <span key={c.code} className="px-2 py-1 bg-slate-100 dark:bg-slate-700 rounded text-xs">
                      {c.icon} {c.code}
                    </span>
                  ))}
                </div>
              </div>
              <div className="mt-4 space-y-2">
                <div className="text-sm text-slate-500 dark:text-slate-400">Crypto</div>
                <div className="flex flex-wrap gap-1">
                  {CRYPTO_CURRENCIES.map(c => (
                    <span key={c.symbol} className="px-2 py-1 bg-slate-100 dark:bg-slate-700 rounded text-xs">
                      {c.symbol}
                    </span>
                  ))}
                </div>
              </div>
            </div>

            {/* Limits */}
            <div className="bg-white dark:bg-slate-800 rounded-lg p-6 shadow-sm">
              <h3 className="font-semibold mb-4">Limits</h3>
              <div className="space-y-3 text-sm">
                <div className="flex justify-between">
                  <span className="text-slate-500 dark:text-slate-400">Min. Purchase</span>
                  <span className="font-semibold">{fiatCurrency.symbol}50</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-500 dark:text-slate-400">Max. Purchase (Daily)</span>
                  <span className="font-semibold">{fiatCurrency.symbol}20,000</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-500 dark:text-slate-400">Max. Purchase (Monthly)</span>
                  <span className="font-semibold">{fiatCurrency.symbol}50,000</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
