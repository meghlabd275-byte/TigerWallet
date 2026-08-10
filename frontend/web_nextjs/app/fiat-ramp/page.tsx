'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../components/ThemeProvider'
import { useWallet } from '../wallet';
import api, { FiatProvider, FiatOrder } from '../../src/lib/api/client';

// ================================================================================
// Types
// ================================================================================

type Transaction = FiatOrder;

interface KYCStatus {
  level: number;
  status: 'none' | 'pending' | 'verified' | 'rejected';
  limits: {
    daily: number;
    monthly: number;
    yearly: number;
  };
}

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
  const { isDark } = useTheme()
  const { address, isConnected } = useWallet();
  
  // Buy/Sell tabs
  const [activeTab, setActiveTab] = useState<'buy' | 'sell'>('buy');
  
  // Providers from API
  const [providers, setProviders] = useState<FiatProvider[]>([]);
  const [providersLoading, setProvidersLoading] = useState(true);
  
  // Form state
  const [selectedProvider, setSelectedProvider] = useState<FiatProvider | null>(null);
  const [fiatCurrency, setFiatCurrency] = useState('USD');
  const [cryptoCurrency, setCryptoCurrency] = useState('ETH');
  const [fiatAmount, setFiatAmount] = useState('');
  const [cryptoAmount, setCryptoAmount] = useState('');
  const [paymentMethod, setPaymentMethod] = useState('');
  const [email, setEmail] = useState('');
  
  // Live crypto price (in fiat currency) fetched from price API
  const [cryptoPrice, setCryptoPrice] = useState<number | null>(null);
  const [priceLoading, setPriceLoading] = useState(false);
  
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
      setProvidersLoading(true);
      setError(null);
      try {
        const res = await api.getFiatProviders({ fiat: fiatCurrency, crypto: cryptoCurrency });
        if (res.success && res.data) {
          setProviders(res.data);
        } else {
          setError(res.error || 'Failed to load fiat providers');
        }
      } catch (err: any) {
        setError(err.message || 'Failed to load fiat providers');
      } finally {
        setProvidersLoading(false);
      }

      // Fetch KYC status
      if (address) {
        try {
          const kycRes = await api.getKycStatus();
          if (kycRes.success && kycRes.data) {
            setKycStatus(kycRes.data as unknown as KYCStatus);
          }
        } catch {
          // KYC status is optional; leave defaults on failure
        }
      }
    };
    loadData();
  }, [address, fiatCurrency, cryptoCurrency]);

  // Fetch live crypto price whenever the selected crypto/fiat pair changes
  const fetchPrice = useCallback(async () => {
    setPriceLoading(true);
    try {
      const res = await api.getTokenPrice(cryptoCurrency, fiatCurrency);
      if (res.success && res.data) {
        setCryptoPrice(res.data.price);
      } else {
        setCryptoPrice(null);
      }
    } catch {
      setCryptoPrice(null);
    } finally {
      setPriceLoading(false);
    }
  }, [cryptoCurrency, fiatCurrency]);

  useEffect(() => {
    fetchPrice();
  }, [fetchPrice]);

  // Calculate crypto amount based on fiat using live price
  const calculateCryptoAmount = (fiat: string) => {
    if (!fiat || cryptoPrice === null) {
      setCryptoAmount('');
      return;
    }
    const crypto = parseFloat(fiat) / cryptoPrice;
    setCryptoAmount(crypto.toFixed(6));
  };

  // Calculate fiat amount based on crypto using live price
  const calculateFiatAmount = (crypto: string) => {
    if (!crypto || cryptoPrice === null) {
      setFiatAmount('');
      return;
    }
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
      const res = await api.createFiatOrder({
        providerId: selectedProvider.id,
        type: activeTab,
        fiatAmount: parseFloat(fiatAmount),
        cryptoCurrency,
        fiatCurrency,
        paymentMethod,
        email,
        walletAddress: address || undefined,
      });

      if (res.success && res.data) {
        setCurrentTransaction(res.data);
        setSuccess(`Order created!${res.data.redirectUrl ? ` Redirecting to ${selectedProvider.name}...` : ''}`);
      } else {
        setError(res.error || 'Failed to create order');
      }

    } catch (err: any) {
      setError(err.message || 'Failed to create order');
    } finally {
      setIsLoading(false);
    }
  };

  // Check KYC
  const checkKYC = async () => {
    setIsLoading(true);
    try {
      const res = await api.getKycStatus();
      if (res.success && res.data) {
        setKycStatus(res.data);
        setSuccess('KYC verified! You can now buy up to $5,000 daily.');
      } else {
        setError(res.error || 'Failed to verify KYC status');
      }
    } catch (err: any) {
      setError(err.message || 'Failed to verify KYC status');
    } finally {
      setIsLoading(false);
    }
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
    <div className={`'min-h-screen bg-gradient-to-br' ${isDark ? 'from-slate-900' : 'from-slate-50'} ${isDark ? 'to-slate-800' : 'to-slate-100'} 'p-8'`}>
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <h1 className={`'text-4xl font-bold' ${isDark ? 'text-white' : 'text-slate-900'} 'mb-2'`}>Fiat On-Ramp</h1>
          <p className={`${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Buy crypto with fiat or sell crypto for fiat</p>
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
                <p className={`${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Daily Limit</p>
                <p className={`${isDark ? 'text-white' : 'text-slate-900'} 'font-medium'`}>{formatCurrency(kycStatus.limits.daily, 'USD')}</p>
              </div>
              <div>
                <p className={`${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Monthly Limit</p>
                <p className={`${isDark ? 'text-white' : 'text-slate-900'} 'font-medium'`}>{formatCurrency(kycStatus.limits.monthly, 'USD')}</p>
              </div>
              <div>
                <p className={`${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Yearly Limit</p>
                <p className={`${isDark ? 'text-white' : 'text-slate-900'} 'font-medium'`}>{formatCurrency(kycStatus.limits.yearly, 'USD')}</p>
              </div>
            </div>
          </div>
        )}

        {/* Provider Selection */}
        <div className={`${isDark ? 'bg-slate-800' : 'bg-white'} 'rounded-2xl p-6 border' ${isDark ? 'border-slate-700' : 'border-slate-200'} 'mb-6'`}>
          <h2 className={`'text-xl font-semibold' ${isDark ? 'text-white' : 'text-slate-900'} 'mb-4'`}>Select Provider</h2>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {providersLoading ? (
              <div className={`'col-span-full text-center py-8' ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
                Loading providers...
              </div>
            ) : getAvailableProviders().length === 0 ? (
              <div className={`'col-span-full text-center py-8' ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
                No providers available for {fiatCurrency} {'->'} {cryptoCurrency}
              </div>
            ) : (
            getAvailableProviders().map(provider => (
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
                  <p className={`${isDark ? 'text-white' : 'text-slate-900'} 'font-medium'`}>{provider.name}</p>
                  <p className={`${isDark ? 'text-slate-400' : 'text-slate-500'} 'text-sm'`}>Fee: {provider.fees}% - {provider.processingTime}</p>
                </div>
                {provider.fees < 2 && (
                  <span className="bg-green-500/20 text-green-400 text-xs px-2 py-1 rounded">Best Rate</span>
                )}
              </button>
            ))
            )}
          </div>
        </div>

        {/* Currency Selection */}
        <div className="grid grid-cols-2 gap-4 mb-6">
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white'} 'rounded-xl p-4 border' ${isDark ? 'border-slate-700' : 'border-slate-200'}`}>
            <label className={`'block' ${isDark ? 'text-slate-400' : 'text-slate-500'} 'text-sm mb-2'`}>Fiat Currency</label>
            <select
              value={fiatCurrency}
              onChange={(e) => setFiatCurrency(e.target.value)}
              className={`'w-full' ${isDark ? 'bg-slate-700' : 'bg-slate-100'} 'border' ${isDark ? 'border-slate-600' : 'border-slate-300'} 'rounded-lg px-4 py-3' ${isDark ? 'text-white' : 'text-slate-900'}`}
            >
              {['USD', 'EUR', 'GBP', 'AUD', 'CAD', 'INR', 'BRL'].map(c => (
                <option key={c} value={c}>{c}</option>
              ))}
            </select>
          </div>
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white'} 'rounded-xl p-4 border' ${isDark ? 'border-slate-700' : 'border-slate-200'}`}>
            <label className={`'block' ${isDark ? 'text-slate-400' : 'text-slate-500'} 'text-sm mb-2'`}>Crypto Currency</label>
            <select
              value={cryptoCurrency}
              onChange={(e) => setCryptoCurrency(e.target.value)}
              className={`'w-full' ${isDark ? 'bg-slate-700' : 'bg-slate-100'} 'border' ${isDark ? 'border-slate-600' : 'border-slate-300'} 'rounded-lg px-4 py-3' ${isDark ? 'text-white' : 'text-slate-900'}`}
            >
              {['ETH', 'BTC', 'SOL', 'MATIC', 'AVAX', 'USDT', 'USDC'].map(c => (
                <option key={c} value={c}>{c}</option>
              ))}
            </select>
          </div>
        </div>

        {/* Amount Input */}
        <div className="grid grid-cols-2 gap-4 mb-6">
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white'} 'rounded-xl p-4 border' ${isDark ? 'border-slate-700' : 'border-slate-200'}`}>
            <label className={`'block' ${isDark ? 'text-slate-400' : 'text-slate-500'} 'text-sm mb-2'`}>
              {activeTab === 'buy' ? 'Fiat Amount' : 'Crypto Amount'} ({activeTab === 'buy' ? fiatCurrency : cryptoCurrency})
            </label>
            <input
              type="number"
              value={activeTab === 'buy' ? fiatAmount : cryptoAmount}
              onChange={(e) => activeTab === 'buy' ? handleFiatChange(e.target.value) : handleCryptoChange(e.target.value)}
              placeholder="0.00"
              className={`'w-full bg-transparent text-2xl' ${isDark ? 'text-white' : 'text-slate-900'} 'font-bold outline-none'`}
            />
            {selectedProvider && (
              <p className={`${isDark ? 'text-slate-400' : 'text-slate-500'} 'text-sm mt-2'`}>
                Min: {selectedProvider.minAmount} {fiatCurrency} • Max: {selectedProvider.maxAmount} {fiatCurrency}
              </p>
            )}
            <p className="text-slate-500 text-xs mt-2">
              {priceLoading
                ? 'Fetching live price...'
                : cryptoPrice !== null
                ? `1 ${cryptoCurrency} = ${cryptoPrice.toLocaleString()} ${fiatCurrency}`
                : 'Live price unavailable'}
            </p>
          </div>
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white'} 'rounded-xl p-4 border' ${isDark ? 'border-slate-700' : 'border-slate-200'}`}>
            <label className={`'block' ${isDark ? 'text-slate-400' : 'text-slate-500'} 'text-sm mb-2'`}>
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
              <p className={`${isDark ? 'text-slate-400' : 'text-slate-500'} 'text-sm mt-2'`}>
                Fee: {selectedProvider.fees}% included
              </p>
            )}
          </div>
        </div>

        {/* Payment Method */}
        {selectedProvider && (
          <div className={`${isDark ? 'bg-slate-800' : 'bg-white'} 'rounded-2xl p-6 border' ${isDark ? 'border-slate-700' : 'border-slate-200'} 'mb-6'`}>
            <h2 className={`'text-xl font-semibold' ${isDark ? 'text-white' : 'text-slate-900'} 'mb-4'`}>Payment Method</h2>
            
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
                  <span className={`${isDark ? 'text-white' : 'text-slate-900'} 'text-sm'`}>{method.name}</span>
                  {method.popular && (
                    <span className="ml-auto text-xs bg-blue-500/20 text-blue-400 px-2 py-0.5 rounded">Popular</span>
                  )}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Email */}
        <div className={`${isDark ? 'bg-slate-800' : 'bg-white'} 'rounded-xl p-4 border' ${isDark ? 'border-slate-700' : 'border-slate-200'} 'mb-6'`}>
          <label className={`'block' ${isDark ? 'text-slate-400' : 'text-slate-500'} 'text-sm mb-2'`}>Email Address</label>
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="your@email.com"
            className={`'w-full' ${isDark ? 'bg-slate-700' : 'bg-slate-100'} 'border' ${isDark ? 'border-slate-600' : 'border-slate-300'} 'rounded-lg px-4 py-3' ${isDark ? 'text-white' : 'text-slate-900'}`}
          />
          <p className={`${isDark ? 'text-slate-400' : 'text-slate-500'} 'text-sm mt-2'`}>Receipt and transaction updates will be sent here</p>
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
          <div className={`'mt-6' ${isDark ? 'bg-slate-800' : 'bg-white'} 'rounded-2xl p-6 border' ${isDark ? 'border-slate-700' : 'border-slate-200'}`}>
            <h2 className={`'text-xl font-semibold' ${isDark ? 'text-white' : 'text-slate-900'} 'mb-4'`}>Transaction Status</h2>
            
            <div className="flex items-center gap-4 mb-4">
              <div className={`w-4 h-4 rounded-full ${
                currentTransaction.status === 'completed' ? 'bg-green-500' :
                currentTransaction.status === 'processing' ? 'bg-blue-500 animate-pulse' :
                currentTransaction.status === 'failed' ? 'bg-red-500' :
                'bg-yellow-500 animate-pulse'
              }`}></div>
              <span className={`${isDark ? 'text-white' : 'text-slate-900'} 'font-medium capitalize'`}>{currentTransaction.status}</span>
            </div>
            
            <div className="space-y-3">
              <div className="flex justify-between text-sm">
                <span className={`${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Provider</span>
                <span className={`${isDark ? 'text-white' : 'text-slate-900'}`}>{currentTransaction.provider}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className={`${isDark ? 'text-slate-400' : 'text-slate-500'}`}>You {currentTransaction.type === 'buy' ? 'pay' : 'get'}</span>
                <span className={`${isDark ? 'text-white' : 'text-slate-900'}`}>{currentTransaction.fiatAmount} {currentTransaction.fiatCurrency}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className={`${isDark ? 'text-slate-400' : 'text-slate-500'}`}>You {currentTransaction.type === 'buy' ? 'receive' : 'sell'}</span>
                <span className={`${isDark ? 'text-white' : 'text-slate-900'}`}>{currentTransaction.cryptoAmount} {currentTransaction.cryptoCurrency}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className={`${isDark ? 'text-slate-400' : 'text-slate-500'}`}>Payment</span>
                <span className={`${isDark ? 'text-white' : 'text-slate-900'} 'capitalize'`}>{currentTransaction.paymentMethod}</span>
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
