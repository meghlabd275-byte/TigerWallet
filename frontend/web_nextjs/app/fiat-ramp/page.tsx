'use client';

import React, { useState, useEffect } from 'react';
import { useWallet } from '../wallet';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8451';

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
  const data = await response.json();
  return data.data || data;
};

// ================================================================================
// Types
// ================================================================================

interface FiatProvider {
  id: string;
  name: string;
  logo: string;
  supportedMethods: string[];
  supportedFiat: string[];
  supportedCrypto: string[];
  minAmount: number;
  maxAmount: number;
  fees: number;
  processingTime: string;
  available: boolean;
}

interface Transaction {
  id: string;
  provider: string;
  type: 'buy' | 'sell';
  fiatAmount: number;
  cryptoAmount: number;
  cryptoCurrency: string;
  fiatCurrency: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  createdAt: number;
  expiresAt: number;
  paymentMethod: string;
}

interface KYCStatus {
  level: number;
  status: 'none' | 'pending' | 'verified' | 'rejected';
  limits: {
    daily: number;
    monthly: number;
    yearly: number;
  };
}

// Fiat providers
const FIAT_PROVIDERS: FiatProvider[] = [
  {
    id: 'moonpay',
    name: 'MoonPay',
    logo: '🌙',
    supportedMethods: ['card', 'bank', 'applepay', 'googlepay'],
    supportedFiat: ['USD', 'EUR', 'GBP', 'AUD', 'CAD'],
    supportedCrypto: ['ETH', 'BTC', 'SOL', 'MATIC', 'USDT', 'USDC'],
    minAmount: 30,
    maxAmount: 5000,
    fees: 4.5,
    processingTime: '5-30 min',
    available: true,
  },
  {
    id: 'ramp',
    name: 'Ramp Network',
    logo: '💳',
    supportedMethods: ['card', 'bank', 'swish', 'blik'],
    supportedFiat: ['USD', 'EUR', 'GBP'],
    supportedCrypto: ['ETH', 'BTC', 'MATIC', 'USDC'],
    minAmount: 50,
    maxAmount: 10000,
    fees: 3.5,
    processingTime: '10-30 min',
    available: true,
  },
  {
    id: 'transak',
    name: 'Transak',
    logo: '🔄',
    supportedMethods: ['card', 'bank', 'upi', 'pix'],
    supportedFiat: ['USD', 'EUR', 'GBP', 'INR', 'BRL'],
    supportedCrypto: ['ETH', 'BTC', 'SOL', 'AVAX', 'USDT'],
    minAmount: 20,
    maxAmount: 5000,
    fees: 4.0,
    processingTime: '15-45 min',
    available: true,
  },
  {
    id: 'stripe',
    name: 'Stripe',
    logo: '💵',
    supportedMethods: ['card', 'applepay', 'googlepay'],
    supportedFiat: ['USD', 'EUR', 'GBP'],
    supportedCrypto: ['USDC', 'USDT'],
    minAmount: 100,
    maxAmount: 25000,
    fees: 2.9,
    processingTime: '1-2 min',
    available: true,
  },
  {
    id: 'safepay',
    name: 'SafePay',
    logo: '🔒',
    supportedMethods: ['sepa', 'fps', 'pix'],
    supportedFiat: ['EUR', 'GBP', 'BRL'],
    supportedCrypto: ['ETH', 'BTC', 'USDT'],
    minAmount: 100,
    maxAmount: 50000,
    fees: 1.5,
    processingTime: '1-3 days',
    available: true,
  },
];

// Payment methods with icons
const PAYMENT_METHODS = [
  { id: 'card', name: 'Credit/Debit Card', icon: '💳', popular: true },
  { id: 'applepay', name: 'Apple Pay', icon: '🍎', popular: true },
  { id: 'googlepay', name: 'Google Pay', icon: '🔷', popular: true },
  { id: 'bank', name: 'Bank Transfer', icon: '🏦', popular: false },
  { id: 'sepa', name: 'SEPA Instant', icon: '🇪🇺', popular: false },
  { id: 'fps', name: 'FPS (UK)', icon: '🇬🇧', popular: false },
  { id: 'pix', name: 'PIX (Brazil)', icon: '🇧🇷', popular: false },
  { id: 'upi', name: 'UPI (India)', icon: '🇮🇳', popular: false },
  { id: 'blik', name: 'BLIK (Poland)', icon: '🇵🇱', popular: false },
];

// ================================================================================
// Main Component
// ================================================================================

export default function FiatRampPage() {
  const { address, isConnected } = useWallet();
  
  // Buy/Sell tabs
  const [activeTab, setActiveTab] = useState<'buy' | 'sell'>('buy');
  
  // Providers from API
  const [providers, setProviders] = useState<FiatProvider[]>(FIAT_PROVIDERS);
  
  // Form state
  const [selectedProvider, setSelectedProvider] = useState<FiatProvider | null>(null);
  const [fiatCurrency, setFiatCurrency] = useState('USD');
  const [cryptoCurrency, setCryptoCurrency] = useState('ETH');
  const [fiatAmount, setFiatAmount] = useState('');
  const [cryptoAmount, setCryptoAmount] = useState('');
  const [paymentMethod, setPaymentMethod] = useState('');
  const [email, setEmail] = useState('');
  
  // UI state
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [currentTransaction, setCurrentTransaction] = useState<Transaction | null>(null);
  
  // KYC state
  const [kycStatus, setKycStatus] = useState<KYCStatus>({
    level: 0,
    status: 'none',
    limits: { daily: 0, monthly: 0, yearly: 0 },
  });

  // Load providers and KYC status from backend
  useEffect(() => {
    const loadData = async () => {
      try {
        // Fetch providers from API
        const providersData = await fetchAPI<FiatProvider[]>('/api/v1/fiat-ramp/providers');
        if (providersData && providersData.length > 0) {
          setProviders(providersData);
        }
        
        // Fetch KYC status
        if (address) {
          const kyc = await fetchAPI<KYCStatus>('/api/v1/kyc/status');
          if (kyc) {
            setKycStatus(kyc);
          }
        }
      } catch (err) {
        console.log('Using default providers - API not available');
      }
    };
    loadData();
  }, [address]);

  // Calculate crypto amount based on fiat (mock price)
  const calculateCryptoAmount = (fiat: string) => {
    if (!fiat) {
      setCryptoAmount('');
      return;
    }
    // Mock price: 1 ETH = 3000 USD
    const cryptoPrice = 3000;
    const crypto = parseFloat(fiat) / cryptoPrice;
    setCryptoAmount(crypto.toFixed(6));
  };

  // Calculate fiat amount based on crypto
  const calculateFiatAmount = (crypto: string) => {
    if (!crypto) {
      setFiatAmount('');
      return;
    }
    const cryptoPrice = 3000;
    const fiat = parseFloat(crypto) * cryptoPrice;
    setFiatAmount(fiat.toFixed(2));
  };

  // Handle fiat amount change
  const handleFiatChange = (value: string) => {
    setFiatAmount(value);
    calculateCryptoAmount(value);
  };

  // Handle crypto amount change
  const handleCryptoChange = (value: string) => {
    setCryptoAmount(value);
    calculateFiatAmount(value);
  };

  // Get available providers based on selected currencies
  const getAvailableProviders = () => {
    return providers.filter(p => 
      p.available &&
      p.supportedFiat.includes(fiatCurrency) &&
      p.supportedCrypto.includes(cryptoCurrency)
    );
  };

  // Get available payment methods for selected provider
  const getAvailablePaymentMethods = () => {
    if (!selectedProvider) return [];
    return PAYMENT_METHODS.filter(m => 
      selectedProvider.supportedMethods.includes(m.id)
    );
  };

  // Create order
  const createOrder = async () => {
    if (!selectedProvider || !fiatAmount || !paymentMethod || !email) {
      setError('Please fill in all required fields');
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      // Call backend API to create order
      const orderData = await fetchAPI<Transaction>('/api/v1/fiat-ramp/orders', {
        method: 'POST',
        body: JSON.stringify({
          providerId: selectedProvider.id,
          type: activeTab,
          fiatAmount: parseFloat(fiatAmount),
          cryptoCurrency,
          fiatCurrency,
          paymentMethod,
          email,
          walletAddress: address,
        }),
      });

      setCurrentTransaction(orderData);
      setSuccess(`Order created! Redirecting to ${selectedProvider.name}...`);
      
      // Simulate redirect after 2 seconds
      setTimeout(() => {
        setCurrentTransaction(prev => prev ? { ...prev, status: 'processing' } : null);
      }, 2000);
      
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  };

  // Check KYC
  const checkKYC = () => {
    setIsLoading(true);
    // Simulate KYC check
    setTimeout(() => {
      setKycStatus({
        level: 1,
        status: 'verified',
        limits: { daily: 5000, monthly: 20000, yearly: 100000 },
      });
      setIsLoading(false);
      setSuccess('KYC verified! You can now buy up to $5,000 daily.');
    }, 1500);
  };

  const formatCurrency = (amount: number, currency: string) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency,
    }).format(amount);
  };

  // ============================================================================
  // Render
  // ============================================================================

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 to-slate-800 p-8">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-4xl font-bold text-white mb-2">Fiat On-Ramp</h1>
          <p className="text-slate-400">Buy crypto with fiat or sell crypto for fiat</p>
        </div>

        {/* Buy/Sell Tabs */}
        <div className="flex gap-4 mb-6">
          <button
            onClick={() => setActiveTab('buy')}
            className={`flex-1 py-4 rounded-xl font-semibold text-lg transition-all ${
              activeTab === 'buy'
                ? 'bg-green-600 text-white'
                : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
            }`}
          >
            💰 Buy Crypto
          </button>
          <button
            onClick={() => setActiveTab('sell')}
            className={`flex-1 py-4 rounded-xl font-semibold text-lg transition-all ${
              activeTab === 'sell'
                ? 'bg-blue-600 text-white'
                : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
            }`}
          >
            💵 Sell Crypto
          </button>
        </div>

        {/* KYC Status */}
        {isConnected && kycStatus.status === 'none' && (
          <div className="mb-6 p-4 bg-yellow-500/10 border border-yellow-500/30 rounded-xl">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-yellow-400 font-medium">Complete KYC to unlock higher limits</p>
                <p className="text-yellow-400/70 text-sm">Verify your identity to buy up to $5,000 daily</p>
              </div>
              <button
                onClick={checkKYC}
                disabled={isLoading}
                className="bg-yellow-600 hover:bg-yellow-700 disabled:bg-slate-600 text-white px-6 py-3 rounded-xl font-medium"
              >
                {isLoading ? 'Verifying...' : 'Verify Now'}
              </button>
            </div>
          </div>
        )}

        {kycStatus.status === 'verified' && (
          <div className="mb-6 p-4 bg-green-500/10 border border-green-500/30 rounded-xl">
            <div className="flex items-center gap-2 text-green-400 mb-2">
              <span className="text-xl">✅</span>
              <span className="font-medium">KYC Verified</span>
            </div>
            <div className="grid grid-cols-3 gap-4 text-sm">
              <div>
                <p className="text-slate-400">Daily Limit</p>
                <p className="text-white font-medium">{formatCurrency(kycStatus.limits.daily, 'USD')}</p>
              </div>
              <div>
                <p className="text-slate-400">Monthly Limit</p>
                <p className="text-white font-medium">{formatCurrency(kycStatus.limits.monthly, 'USD')}</p>
              </div>
              <div>
                <p className="text-slate-400">Yearly Limit</p>
                <p className="text-white font-medium">{formatCurrency(kycStatus.limits.yearly, 'USD')}</p>
              </div>
            </div>
          </div>
        )}

        {/* Provider Selection */}
        <div className="bg-slate-800 rounded-2xl p-6 border border-slate-700 mb-6">
          <h2 className="text-xl font-semibold text-white mb-4">Select Provider</h2>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {getAvailableProviders().map(provider => (
              <button
                key={provider.id}
                onClick={() => setSelectedProvider(provider)}
                className={`flex items-center gap-4 p-4 rounded-xl border text-left transition-all ${
                  selectedProvider?.id === provider.id
                    ? 'border-blue-500 bg-blue-500/10'
                    : 'border-slate-600 bg-slate-700/30 hover:border-slate-500'
                }`}
              >
                <span className="text-3xl">{provider.logo}</span>
                <div className="flex-1">
                  <p className="text-white font-medium">{provider.name}</p>
                  <p className="text-slate-400 text-sm">Fee: {provider.fees}% • {provider.processingTime}</p>
                </div>
                {provider.fees < 2 && (
                  <span className="bg-green-500/20 text-green-400 text-xs px-2 py-1 rounded">Best Rate</span>
                )}
              </button>
            ))}
          </div>
        </div>

        {/* Currency Selection */}
        <div className="grid grid-cols-2 gap-4 mb-6">
          <div className="bg-slate-800 rounded-xl p-4 border border-slate-700">
            <label className="block text-slate-400 text-sm mb-2">Fiat Currency</label>
            <select
              value={fiatCurrency}
              onChange={(e) => setFiatCurrency(e.target.value)}
              className="w-full bg-slate-700 border border-slate-600 rounded-lg px-4 py-3 text-white"
            >
              {['USD', 'EUR', 'GBP', 'AUD', 'CAD', 'INR', 'BRL'].map(c => (
                <option key={c} value={c}>{c}</option>
              ))}
            </select>
          </div>
          <div className="bg-slate-800 rounded-xl p-4 border border-slate-700">
            <label className="block text-slate-400 text-sm mb-2">Crypto Currency</label>
            <select
              value={cryptoCurrency}
              onChange={(e) => setCryptoCurrency(e.target.value)}
              className="w-full bg-slate-700 border border-slate-600 rounded-lg px-4 py-3 text-white"
            >
              {['ETH', 'BTC', 'SOL', 'MATIC', 'AVAX', 'USDT', 'USDC'].map(c => (
                <option key={c} value={c}>{c}</option>
              ))}
            </select>
          </div>
        </div>

        {/* Amount Input */}
        <div className="grid grid-cols-2 gap-4 mb-6">
          <div className="bg-slate-800 rounded-xl p-4 border border-slate-700">
            <label className="block text-slate-400 text-sm mb-2">
              {activeTab === 'buy' ? 'Fiat Amount' : 'Crypto Amount'} ({activeTab === 'buy' ? fiatCurrency : cryptoCurrency})
            </label>
            <input
              type="number"
              value={activeTab === 'buy' ? fiatAmount : cryptoAmount}
              onChange={(e) => activeTab === 'buy' ? handleFiatChange(e.target.value) : handleCryptoChange(e.target.value)}
              placeholder="0.00"
              className="w-full bg-transparent text-2xl text-white font-bold outline-none"
            />
            {selectedProvider && (
              <p className="text-slate-400 text-sm mt-2">
                Min: {selectedProvider.minAmount} {fiatCurrency} • Max: {selectedProvider.maxAmount} {fiatCurrency}
              </p>
            )}
          </div>
          <div className="bg-slate-800 rounded-xl p-4 border border-slate-700">
            <label className="block text-slate-400 text-sm mb-2">
              {activeTab === 'buy' ? 'You Receive' : 'You Get'} ({activeTab === 'buy' ? cryptoCurrency : fiatCurrency})
            </label>
            <input
              type="number"
              value={activeTab === 'buy' ? cryptoAmount : fiatAmount}
              onChange={(e) => activeTab === 'buy' ? handleCryptoChange(e.target.value) : handleFiatChange(e.target.value)}
              placeholder="0.00"
              className="w-full bg-transparent text-2xl text-green-400 font-bold outline-none"
            />
            {selectedProvider && (
              <p className="text-slate-400 text-sm mt-2">
                Fee: {selectedProvider.fees}% included
              </p>
            )}
          </div>
        </div>

        {/* Payment Method */}
        {selectedProvider && (
          <div className="bg-slate-800 rounded-2xl p-6 border border-slate-700 mb-6">
            <h2 className="text-xl font-semibold text-white mb-4">Payment Method</h2>
            
            <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
              {getAvailablePaymentMethods().map(method => (
                <button
                  key={method.id}
                  onClick={() => setPaymentMethod(method.id)}
                  className={`flex items-center gap-3 p-3 rounded-xl border text-left transition-all ${
                    paymentMethod === method.id
                      ? 'border-blue-500 bg-blue-500/10'
                      : 'border-slate-600 bg-slate-700/30 hover:border-slate-500'
                  }`}
                >
                  <span className="text-xl">{method.icon}</span>
                  <span className="text-white text-sm">{method.name}</span>
                  {method.popular && (
                    <span className="ml-auto text-xs bg-blue-500/20 text-blue-400 px-2 py-0.5 rounded">Popular</span>
                  )}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Email */}
        <div className="bg-slate-800 rounded-xl p-4 border border-slate-700 mb-6">
          <label className="block text-slate-400 text-sm mb-2">Email Address</label>
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="your@email.com"
            className="w-full bg-slate-700 border border-slate-600 rounded-lg px-4 py-3 text-white"
          />
          <p className="text-slate-400 text-sm mt-2">Receipt and transaction updates will be sent here</p>
        </div>

        {/* Submit */}
        <button
          onClick={createOrder}
          disabled={!selectedProvider || !fiatAmount || !paymentMethod || !email || isLoading}
          className="w-full bg-gradient-to-r from-green-600 to-green-500 hover:from-green-700 hover:to-green-600 disabled:from-slate-600 disabled:to-slate-500 text-white py-4 rounded-xl font-semibold text-lg transition-all"
        >
          {isLoading 
            ? 'Processing...' 
            : `${activeTab === 'buy' ? 'Buy' : 'Sell'} ${cryptoAmount || '0'} ${cryptoCurrency} for ${fiatAmount || '0'} ${fiatCurrency}`
          }
        </button>

        {/* Current Transaction */}
        {currentTransaction && (
          <div className="mt-6 bg-slate-800 rounded-2xl p-6 border border-slate-700">
            <h2 className="text-xl font-semibold text-white mb-4">Transaction Status</h2>
            
            <div className="flex items-center gap-4 mb-4">
              <div className={`w-4 h-4 rounded-full ${
                currentTransaction.status === 'completed' ? 'bg-green-500' :
                currentTransaction.status === 'processing' ? 'bg-blue-500 animate-pulse' :
                currentTransaction.status === 'failed' ? 'bg-red-500' :
                'bg-yellow-500 animate-pulse'
              }`}></div>
              <span className="text-white font-medium capitalize">{currentTransaction.status}</span>
            </div>
            
            <div className="space-y-3">
              <div className="flex justify-between text-sm">
                <span className="text-slate-400">Provider</span>
                <span className="text-white">{currentTransaction.provider}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-slate-400">You {currentTransaction.type === 'buy' ? 'pay' : 'get'}</span>
                <span className="text-white">{currentTransaction.fiatAmount} {currentTransaction.fiatCurrency}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-slate-400">You {currentTransaction.type === 'buy' ? 'receive' : 'sell'}</span>
                <span className="text-white">{currentTransaction.cryptoAmount} {currentTransaction.cryptoCurrency}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-slate-400">Payment</span>
                <span className="text-white capitalize">{currentTransaction.paymentMethod}</span>
              </div>
            </div>
          </div>
        )}

        {/* Messages */}
        {error && (
          <div className="mt-6 p-4 bg-red-500/10 border border-red-500/30 rounded-xl">
            <p className="text-red-400">{error}</p>
          </div>
        )}

        {success && !currentTransaction && (
          <div className="mt-6 p-4 bg-green-500/10 border border-green-500/30 rounded-xl">
            <p className="text-green-400">{success}</p>
          </div>
        )}
      </div>
    </div>
  );
}
