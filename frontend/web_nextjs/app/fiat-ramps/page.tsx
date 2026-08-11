'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../components/ThemeProvider';

interface FiatProvider { 
  id: string; 
  name: string; 
  logo: string; 
  fees: string; 
  methods: string[];
  supportedFiat: string[];
  supportedCrypto: string[];
  minAmount: number;
  maxAmount: number;
}

interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: string;
}

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'https://api.tigerwallet.io';

const fetchAPI = async <T,>(endpoint: string, options?: RequestInit): Promise<T> => {
  const token = typeof window !== 'undefined' ? localStorage.getItem('tigerwallet-token') : null;
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });
  if (!response.ok) throw new Error(`API Error: ${response.statusText}`);
  const data: ApiResponse<T> = await response.json();
  return data.data;
};

const FALLBACK_PROVIDERS: FiatProvider[] = [
  { id: 'banxa', name: 'Banxa', logo: '🅱️', fees: '2.5%', methods: ['Credit Card', 'Debit Card', 'Apple Pay', 'Google Pay', 'Bank Transfer'], supportedFiat: ['USD', 'EUR', 'GBP', 'AUD'], supportedCrypto: ['ETH', 'BTC', 'USDT', 'BNB', 'SOL'], minAmount: 50, maxAmount: 50000 },
  { id: 'moonpay', name: 'MoonPay', logo: '🌙', fees: '3.5%', methods: ['Credit Card', 'Debit Card', 'Apple Pay', 'Google Pay', 'Bank Transfer'], supportedFiat: ['USD', 'EUR', 'GBP'], supportedCrypto: ['ETH', 'BTC', 'USDT', 'BNB'], minAmount: 30, maxAmount: 25000 },
  { id: 'transak', name: 'Transak', logo: '🔄', fees: '2%', methods: ['Credit Card', 'Debit Card', 'Bank Transfer', 'SEPA'], supportedFiat: ['USD', 'EUR', 'GBP', 'INR'], supportedCrypto: ['ETH', 'BTC', 'USDT', 'SOL', 'MATIC'], minAmount: 30, maxAmount: 30000 },
  { id: 'simplex', name: 'Simplex', logo: '💳', fees: '3%', methods: ['Credit Card', 'Debit Card', 'Apple Pay'], supportedFiat: ['USD', 'EUR'], supportedCrypto: ['ETH', 'BTC', 'USDT'], minAmount: 50, maxAmount: 20000 },
];

export default function FiatRamps() {
  const [providers, setProviders] = useState<FiatProvider[]>(FALLBACK_PROVIDERS);
  const [provider, setProvider] = useState('');
  const [fiatAmount, setFiatAmount] = useState('');
  const [cryptoAmount, setCryptoAmount] = useState('');
  const [crypto, setCrypto] = useState('ETH');
  const [fiat, setFiat] = useState('USD');
  const [method, setMethod] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [orderId, setOrderId] = useState<string | null>(null);
  const [priceQuote, setPriceQuote] = useState<{ cryptoAmount: string; rate: string } | null>(null);
  const { isDark } = useTheme();

  const loadProviders = useCallback(async () => {
    try {
      const data = await fetchAPI<FiatProvider[]>('/fiat/providers');
      if (data && data.length > 0) setProviders(data);
    } catch (err) {
      console.log('Using fallback provider data');
    }
  }, []);

  useEffect(() => { loadProviders(); }, [loadProviders]);

  const selectedProvider = providers.find(p => p.id === provider);

  // Fetch price quote when amount changes
  useEffect(() => {
    if (!provider || !fiatAmount || !crypto) return;
    
    const fetchQuote = async () => {
      try {
        const quote = await fetchAPI<{ cryptoAmount: string; rate: string }>('/fiat/quote', {
          method: 'POST',
          body: JSON.stringify({
            providerId: provider,
            fiatAmount: parseFloat(fiatAmount),
            fiatCurrency: fiat,
            cryptoCurrency: crypto,
          }),
        });
        setPriceQuote(quote);
        setCryptoAmount(quote.cryptoAmount);
      } catch (err) {
        // Fallback calculation
        const rates: Record<string, number> = { ETH: 1850, BTC: 42000, USDT: 1, BNB: 310, SOL: 98 };
        const rate = rates[crypto] || 1000;
        setCryptoAmount((parseFloat(fiatAmount) / rate).toFixed(6));
        setPriceQuote({ cryptoAmount: (parseFloat(fiatAmount) / rate).toFixed(6), rate: rate.toString() });
      }
    };

    const timeoutId = setTimeout(fetchQuote, 500);
    return () => clearTimeout(timeoutId);
  }, [provider, fiatAmount, fiat, crypto]);

  const handleBuy = async () => {
    if (!provider || !fiatAmount || !method || !priceQuote) return;
    setLoading(true);
    setError(null);

    try {
      const order = await fetchAPI<{ orderId: string; redirectUrl: string }>('/fiat/create-order', {
        method: 'POST',
        body: JSON.stringify({
          providerId: provider,
          fiatAmount: parseFloat(fiatAmount),
          cryptoAmount: priceQuote.cryptoAmount,
          fiatCurrency: fiat,
          cryptoCurrency: crypto,
          paymentMethod: method,
        }),
      });
      
      if (order.redirectUrl) {
        setOrderId(order.orderId);
        window.location.href = order.redirectUrl;
      }
    } catch (err) {
      setError('Failed to create order. Please try again.');
      setLoading(false);
    }
  };

  return (
    <div className={`min-h-screen ${isDark ? 'bg-slate-900 text-white' : 'bg-slate-50 text-gray-900'}`}>
      <header className={`border-b p-4 ${isDark ? 'bg-slate-800' : 'bg-white border-gray-200'}`}>
        <div className="flex items-center gap-4 max-w-2xl mx-auto">
          <a href="/wallet" className="text-2xl">🐯</a>
          <h1 className="text-xl font-bold">Buy Crypto</h1>
        </div>
      </header>
      
      <div className="max-w-2xl mx-auto p-8">
        {error && (
          <div className={`p-4 rounded-lg mb-6 ${isDark ? 'bg-red-900/20 text-red-400' : 'bg-red-50 text-red-600'}`}>
            {error}
          </div>
        )}
        
        {orderId && (
          <div className={`p-4 rounded-lg mb-6 ${isDark ? 'bg-green-900/20 text-green-400' : 'bg-green-50 text-green-600'}`}>
            Order created: {orderId}. Redirecting to payment...
          </div>
        )}
        
        <div className={`rounded-lg p-6 mb-6 ${isDark ? 'bg-slate-800' : 'bg-white border border-gray-200'}`}>
          <h2 className="font-semibold mb-4">Select Provider</h2>
          <div className="grid grid-cols-2 gap-4 mb-6">
            {providers.map(p => (
              <button 
                key={p.id} 
                onClick={() => { setProvider(p.id); setMethod(''); }}
                className={`p-4 rounded-lg border-2 transition-colors ${provider === p.id ? (isDark ? 'border-orange-500 bg-orange-900/20' : 'border-orange-500 bg-orange-50') : (isDark ? 'border-gray-700 hover:border-orange-300' : 'border-gray-200 hover:border-orange-300')}`}
              >
                <div className="text-3xl mb-2">{p.logo}</div>
                <div className="font-semibold">{p.name}</div>
                <div className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Fees: {p.fees}</div>
                <div className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>${p.minAmount} - ${p.maxAmount.toLocaleString()}</div>
              </button>
            ))}
          </div>
          
          <div className="mb-4">
            <label className={`block text-sm mb-2 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Fiat Currency</label>
            <select 
              value={fiat} 
              onChange={(e) => setFiat(e.target.value)} 
              className={`w-full rounded-lg px-4 py-3 ${isDark ? 'bg-slate-700' : 'bg-slate-100'}`}
            >
              <option value="USD">USD - US Dollar</option>
              <option value="EUR">EUR - Euro</option>
              <option value="GBP">GBP - British Pound</option>
              <option value="AUD">AUD - Australian Dollar</option>
              <option value="INR">INR - Indian Rupee</option>
            </select>
          </div>
          
          <div className="mb-4">
            <label className={`block text-sm mb-2 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Amount ({fiat})</label>
            <input 
              type="number" 
              value={fiatAmount} 
              onChange={(e) => setFiatAmount(e.target.value)} 
              placeholder="100"
              min={selectedProvider?.minAmount || 30}
              max={selectedProvider?.maxAmount || 50000}
              className={`w-full rounded-lg px-4 py-3 text-xl ${isDark ? 'bg-slate-700' : 'bg-slate-100'}`}
            />
            {selectedProvider && (
              <div className={`text-xs mt-1 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
                Min: ${selectedProvider.minAmount} - Max: ${selectedProvider.maxAmount.toLocaleString()}
              </div>
            )}
          </div>
          
          <div className="mb-4">
            <label className={`block text-sm mb-2 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Crypto Currency</label>
            <div className="flex gap-2">
              <select 
                value={crypto} 
                onChange={(e) => setCrypto(e.target.value)} 
                className={`rounded-lg px-4 py-3 ${isDark ? 'bg-slate-700' : 'bg-slate-100'}`}
              >
                <option value="ETH">ETH - Ethereum</option>
                <option value="BTC">BTC - Bitcoin</option>
                <option value="USDT">USDT - Tether</option>
                <option value="BNB">BNB - BNB</option>
                <option value="SOL">SOL - Solana</option>
              </select>
              <input 
                type="text" 
                value={cryptoAmount} 
                readOnly
                placeholder="0.0"
                className={`flex-1 rounded-lg px-4 py-3 text-xl ${isDark ? 'bg-slate-700' : 'bg-slate-100'}`}
              />
            </div>
            {priceQuote && (
              <div className="text-xs text-green-500 mt-1">
                Rate: 1 {crypto} = ${priceQuote.rate} {fiat}
              </div>
            )}
          </div>
          
          {selectedProvider && (
            <div className="mb-6">
              <label className={`block text-sm mb-2 ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>Payment Method</label>
              <select 
                value={method} 
                onChange={(e) => setMethod(e.target.value)} 
                className={`w-full rounded-lg px-4 py-3 ${isDark ? 'bg-slate-700' : 'bg-slate-100'}`}
              >
                <option value="">Select payment method</option>
                {selectedProvider.methods.map(m => (
                  <option key={m} value={m}>{m}</option>
                ))}
              </select>
            </div>
          )}
          
          <button 
            onClick={handleBuy} 
            disabled={loading || !provider || !fiatAmount || !method || !priceQuote} 
            className="w-full bg-orange-500 hover:bg-orange-600 disabled:bg-slate-400 text-white py-4 rounded-lg font-semibold transition-colors"
          >
            {loading ? 'Processing...' : 'Buy Now'}
          </button>
        </div>
      </div>
    </div>
  );
}
