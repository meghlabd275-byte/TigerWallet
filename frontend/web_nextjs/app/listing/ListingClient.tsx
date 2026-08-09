'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from '../components/ThemeProvider';

// API Base URL
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8443';

// Types
interface User {
  id: string;
  email: string;
  username: string;
  role: string;
}

interface Token {
  id?: string;
  token_symbol: string;
  token_name: string;
  contract_address: string;
  chain_id: number;
  chain_name: string;
  quote_token: string;
  logo_url?: string;
  website_url?: string;
  twitter_url?: string;
  telegram_url?: string;
  discord_url?: string;
  whitepaper_url?: string;
}

interface Listing extends Token {
  id: string;
  user_id: string;
  tier: string;
  fee_amount: number;
  fee_token: string;
  status: string;
  rejection_reason?: string;
  listed_at?: string;
  approved_by?: string;
  approved_at?: string;
  created_at: string;
  updated_at: string;
}

interface Chain {
  id: number;
  name: string;
}

interface Tier {
  id: string;
  name: string;
  fee: string;
  feeUsd: string;
}

interface AuthResponse {
  success: boolean;
  token: string;
  user: User;
  expires_at: string;
}

// API Functions
const api = {
  async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const token = typeof window !== 'undefined' ? localStorage.getItem('token') : '';
    
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
    };
    
    if (token) {
      (headers as Record<string, string>)['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE}${endpoint}`, {
      ...options,
      headers,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Request failed' }));
      throw new Error(error.error || 'Request failed');
    }

    return response.json();
  },

  // Auth
  async register(email: string, username: string, password: string): Promise<AuthResponse> {
    return this.request<AuthResponse>('/api/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, username, password }),
    });
  },

  async login(email: string, password: string): Promise<AuthResponse> {
    return this.request<AuthResponse>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
  },

  // Listings
  async createListing(listing: Token, tier: string): Promise<{ success: boolean; data: Listing }> {
    return this.request<{ success: boolean; data: Listing }>('/api/listings', {
      method: 'POST',
      body: JSON.stringify({ ...listing, tier }),
    });
  },

  async getMyListings(): Promise<{ success: boolean; data: Listing[] }> {
    return this.request<{ success: boolean; data: Listing[] }>('/api/listings');
  },

  async getListing(id: string): Promise<{ success: boolean; data: Listing }> {
    return this.request<{ success: boolean; data: Listing }>(`/api/listings/${id}`);
  },

  // Payments (Crypto only)
  async createCryptoPayment(listingId: string, currency: string, amount: number, network: string): Promise<any> {
    return this.request('/api/payments/crypto', {
      method: 'POST',
      body: JSON.stringify({
        listing_id: listingId,
        currency,
        amount,
        network,
      }),
    });
  },

  async getPaymentStatus(paymentId: string): Promise<any> {
    return this.request(`/api/payments/crypto/${paymentId}`);
  },

  // Admin
  async getAllListings(status?: string, tier?: string): Promise<{ success: boolean; data: Listing[] }> {
    const params = new URLSearchParams();
    if (status) params.append('status', status);
    if (tier) params.append('tier', tier);
    const query = params.toString() ? `?${params.toString()}` : '';
    return this.request<{ success: boolean; data: Listing[] }>(`/api/admin/listings${query}`);
  },

  async approveListing(id: string, notes?: string): Promise<any> {
    return this.request(`/api/admin/listings/${id}/approve`, {
      method: 'POST',
      body: JSON.stringify({ notes }),
    });
  },

  async rejectListing(id: string, reason: string): Promise<any> {
    return this.request(`/api/admin/listings/${id}/reject`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  },

  async getListingStats(): Promise<any> {
    return this.request('/api/admin/stats');
  },
};

// Available chains
const chains: Chain[] = [
  { id: 1, name: 'Ethereum' },
  { id: 56, name: 'BNB Chain' },
  { id: 137, name: 'Polygon' },
  { id: 42161, name: 'Arbitrum' },
  { id: 10, name: 'Optimism' },
  { id: 43114, name: 'Avalanche' },
  { id: 250, name: 'Fantom' },
  { id: 8453, name: 'Base' },
  { id: 42220, name: 'Celo' },
  { id: 11155111, name: 'Sepolia (Testnet)' },
];

// Listing tiers
const tiers: Tier[] = [
  { id: 'tier1', name: 'Tier 1 - Major Pairs', fee: '5000', feeUsd: '2500' },
  { id: 'tier2', name: 'Tier 2 - Established', fee: '2000', feeUsd: '1000' },
  { id: 'tier3', name: 'Tier 3 - New Tokens', fee: '1000', feeUsd: '500' },
  { id: 'tier4', name: 'Tier 4 - Community', fee: '500', feeUsd: '250' },
];

// Quote tokens
const quoteTokens = ['USDT', 'USDC', 'ETH', 'BTC', 'BNB', 'SOL'];

// Supported crypto for payment
const cryptoCurrencies = [
  { symbol: 'USDT', name: 'Tether USD', networks: ['ETH', 'BSC', 'ARB', 'AVAX', 'POLYGON', 'OPTIMISM', 'TRON'] },
  { symbol: 'USDC', name: 'USD Coin', networks: ['ETH', 'BSC', 'ARB', 'AVAX', 'POLYGON', 'OPTIMISM'] },
  { symbol: 'ETH', name: 'Ethereum', networks: ['ETH'] },
  { symbol: 'BTC', name: 'Bitcoin', networks: ['BTC', 'BTC (SegWit)'] },
  { symbol: 'BNB', name: 'BNB', networks: ['BSC'] },
  { symbol: 'SOL', name: 'Solana', networks: ['SOLANA'] },
];

export default function TokenListingPage() {
  const { theme, colors } = useTheme();
  const isDark = theme === 'dark';

  // Auth state
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [user, setUser] = useState<User | null>(null);
  const [showAuthModal, setShowAuthModal] = useState(false);
  const [authMode, setAuthMode] = useState<'login' | 'register'>('login');

  // Form state
  const [step, setStep] = useState(1);
  const [token, setToken] = useState<Token>({
    token_symbol: '',
    token_name: '',
    contract_address: '',
    chain_id: 1,
    chain_name: 'Ethereum',
    quote_token: 'USDT',
    logo_url: '',
    website_url: '',
    twitter_url: '',
    telegram_url: '',
    discord_url: '',
    whitepaper_url: '',
  });
  const [quoteToken, setQuoteToken] = useState('USDT');
  const [selectedTier, setSelectedTier] = useState('tier3');
  const [agreed, setAgreed] = useState(false);

  // Payment state
  const [showPaymentModal, setShowPaymentModal] = useState(false);
  const [paymentData, setPaymentData] = useState<any>(null);
  const [selectedCrypto, setSelectedCrypto] = useState('USDT');
  const [selectedNetwork, setSelectedNetwork] = useState('ETH');

  // Listings state
  const [myListings, setMyListings] = useState<Listing[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // Check auth on mount
  useEffect(() => {
    const storedToken = localStorage.getItem('token');
    const storedUser = localStorage.getItem('user');
    if (storedToken && storedUser) {
      setIsAuthenticated(true);
      setUser(JSON.parse(storedUser));
      loadMyListings();
    }
  }, []);

  // Load user's listings
  const loadMyListings = async () => {
    try {
      setLoading(true);
      const response = await api.getMyListings();
      setMyListings(response.data);
    } catch (err: any) {
      console.error('Failed to load listings:', err);
    } finally {
      setLoading(false);
    }
  };

  // Auth handlers
  const handleAuth = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const formData = new FormData(e.currentTarget);
    const email = formData.get('email') as string;
    const password = formData.get('password') as string;
    const username = formData.get('username') as string;

    try {
      setLoading(true);
      setError(null);

      let response: AuthResponse;
      if (authMode === 'register') {
        response = await api.register(email, username, password);
      } else {
        response = await api.login(email, password);
      }

      localStorage.setItem('token', response.token);
      localStorage.setItem('user', JSON.stringify(response.user));
      setIsAuthenticated(true);
      setUser(response.user);
      setShowAuthModal(false);
      loadMyListings();
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleLogout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    setIsAuthenticated(false);
    setUser(null);
    setMyListings([]);
  };

  // Listing submission
  const handleSubmitListing = async () => {
    if (!isAuthenticated) {
      setShowAuthModal(true);
      return;
    }

    try {
      setLoading(true);
      setError(null);

      const listingData = {
        ...token,
        quote_token: quoteToken,
      };

      await api.createListing(listingData, selectedTier);

      setSuccess('Listing application submitted successfully!');
      setStep(3);
      loadMyListings();
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  // Payment handling (Crypto only)
  const handlePayment = async () => {
    if (!paymentData) return;

    try {
      setLoading(true);
      setError(null);

      const selectedTierData = tiers.find(t => t.id === selectedTier);
      const amount = selectedTierData ? parseFloat(selectedTierData.feeUsd) : 500;

      const response = await api.createCryptoPayment(
        paymentData.id,
        selectedCrypto,
        amount,
        selectedNetwork
      );

      setPaymentData({
        ...paymentData,
        ...response,
        status: 'pending',
      });
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  // Validate form
  const isFormValid = () => {
    return (
      token.token_symbol &&
      token.token_name &&
      token.contract_address &&
      token.contract_address.startsWith('0x') &&
      token.contract_address.length >= 40 &&
      agreed
    );
  };

  // Get selected tier data
  const selectedTierData = tiers.find(t => t.id === selectedTier);
  const selectedChain = chains.find(c => c.id === token.chain_id);

  return (
    <div style={{ background: colors.bgPrimary, minHeight: '100vh', color: colors.textPrimary }}>
      {/* Header */}
      <div style={{ 
        padding: '16px 24px', 
        borderBottom: `1px solid ${colors.border}`,
        background: colors.bgSecondary,
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center'
      }}>
        <div>
          <h1 style={{ fontSize: 24, margin: 0, color: colors.accent }}>🐯 Token Listing</h1>
          <p style={{ margin: '4px 0 0', color: colors.textSecondary, fontSize: 14 }}>
            Apply to list your token on TigerSwap
          </p>
        </div>

        <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
          {isAuthenticated ? (
            <>
              <span style={{ color: colors.textSecondary, fontSize: 14 }}>
                {user?.email}
              </span>
              <button
                onClick={handleLogout}
                style={{
                  padding: '8px 16px',
                  background: colors.bgTertiary,
                  border: `1px solid ${colors.border}`,
                  borderRadius: 8,
                  color: colors.textPrimary,
                  cursor: 'pointer',
                }}
              >
                Logout
              </button>
            </>
          ) : (
            <button
              onClick={() => setShowAuthModal(true)}
              style={{
                padding: '10px 20px',
                background: colors.accent,
                border: 'none',
                borderRadius: 8,
                color: 'white',
                fontWeight: 600,
                cursor: 'pointer',
              }}
            >
              Login / Register
            </button>
          )}
        </div>
      </div>

      <div style={{ maxWidth: 800, margin: '0 auto', padding: '24px' }}>
        {/* Success Message */}
        {success && (
          <div style={{
            padding: '16px',
            background: `${colors.success}20`,
            border: `1px solid ${colors.success}`,
            borderRadius: 12,
            marginBottom: 24,
            color: colors.success,
          }}>
            ✅ {success}
          </div>
        )}

        {/* Error Message */}
        {error && (
          <div style={{
            padding: '16px',
            background: `${colors.error}20`,
            border: `1px solid ${colors.error}`,
            borderRadius: 12,
            marginBottom: 24,
            color: colors.error,
          }}>
            ❌ {error}
          </div>
        )}

        {/* My Listings */}
        {isAuthenticated && myListings.length > 0 && (
          <div style={{ marginBottom: 32 }}>
            <h2 style={{ fontSize: 20, marginBottom: 16, color: colors.textPrimary }}>
              📋 My Listings
            </h2>
            <div style={{ display: 'grid', gap: 12 }}>
              {myListings.map(listing => (
                <div
                  key={listing.id}
                  style={{
                    padding: 16,
                    background: colors.bgCard,
                    border: `1px solid ${colors.border}`,
                    borderRadius: 12,
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                  }}
                >
                  <div>
                    <span style={{ fontWeight: 600, color: colors.textPrimary }}>
                      {listing.token_symbol}
                    </span>
                    <span style={{ color: colors.textSecondary, marginLeft: 8 }}>
                      {listing.token_name}
                    </span>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                    <span style={{
                      padding: '4px 12px',
                      borderRadius: 20,
                      fontSize: 12,
                      fontWeight: 600,
                      background: listing.status === 'approved' ? `${colors.success}20` :
                                  listing.status === 'rejected' ? `${colors.error}20` :
                                  `${colors.warning}20`,
                      color: listing.status === 'approved' ? colors.success :
                             listing.status === 'rejected' ? colors.error :
                             colors.warning,
                    }}>
                      {listing.status.toUpperCase()}
                    </span>
                    <span style={{ color: colors.textSecondary, fontSize: 12 }}>
                      {listing.tier.toUpperCase()}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Progress */}
        <div style={{ display: 'flex', justifyContent: 'center', gap: 16, marginBottom: 32 }}>
          {[1, 2, 3].map(s => (
            <div key={s} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <div style={{
                width: 40,
                height: 40,
                borderRadius: '50%',
                background: step >= s ? colors.accent : colors.bgTertiary,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontWeight: 'bold',
                color: step >= s ? 'white' : colors.textSecondary,
                border: `2px solid ${step >= s ? colors.accent : colors.border}`,
              }}>
                {step > s ? '✓' : s}
              </div>
              <span style={{ color: step >= s ? colors.textPrimary : colors.textSecondary }}>
                {s === 1 ? 'Token' : s === 2 ? 'Tier' : 'Review'}
              </span>
              {s < 3 && (
                <div style={{ width: 40, height: 2, background: colors.border, marginLeft: 8 }} />
              )}
            </div>
          ))}
        </div>

        {/* Step 1: Token Info */}
        {step === 1 && (
          <div style={{ 
            background: colors.bgCard, 
            border: `1px solid ${colors.border}`, 
            borderRadius: 16, 
            padding: 32 
          }}>
            <h2 style={{ marginTop: 0 }}>Token Information</h2>

            {/* Chain Selection */}
            <div style={{ marginBottom: 24 }}>
              <label style={{ display: 'block', marginBottom: 8, color: colors.textSecondary }}>
                Blockchain Network
              </label>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 8 }}>
                {chains.map(chain => (
                  <div
                    key={chain.id}
                    onClick={() => setToken({...token, chain_id: chain.id, chain_name: chain.name})}
                    style={{
                      padding: '12px 16px',
                      borderRadius: 8,
                      border: `2px solid ${token.chain_id === chain.id ? colors.accent : colors.border}`,
                      background: token.chain_id === chain.id ? `${colors.accent}20` : 'transparent',
                      cursor: 'pointer',
                      textAlign: 'center',
                      color: token.chain_id === chain.id ? colors.accent : colors.textPrimary,
                    }}
                  >
                    {chain.name}
                  </div>
                ))}
              </div>
            </div>

            {/* Contract Address */}
            <div style={{ marginBottom: 24 }}>
              <label style={{ display: 'block', marginBottom: 8, color: colors.textSecondary }}>
                Token Contract Address *
              </label>
              <input
                type="text"
                placeholder="0x..."
                value={token.contract_address}
                onChange={(e) => setToken({...token, contract_address: e.target.value})}
                style={{
                  width: '100%',
                  padding: 14,
                  background: colors.bgSecondary,
                  border: `1px solid ${colors.border}`,
                  borderRadius: 8,
                  color: colors.textPrimary,
                  fontSize: 14,
                  fontFamily: 'monospace',
                }}
              />
            </div>

            {/* Token Symbol & Name */}
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 24 }}>
              <div>
                <label style={{ display: 'block', marginBottom: 8, color: colors.textSecondary }}>
                  Token Symbol *
                </label>
                <input
                  type="text"
                  placeholder="e.g., BTC"
                  value={token.token_symbol}
                  onChange={(e) => setToken({...token, token_symbol: e.target.value.toUpperCase()})}
                  style={{
                    width: '100%',
                    padding: 14,
                    background: colors.bgSecondary,
                    border: `1px solid ${colors.border}`,
                    borderRadius: 8,
                    color: colors.textPrimary,
                    fontSize: 14,
                    textTransform: 'uppercase',
                  }}
                />
              </div>
              <div>
                <label style={{ display: 'block', marginBottom: 8, color: colors.textSecondary }}>
                  Token Name *
                </label>
                <input
                  type="text"
                  placeholder="e.g., Bitcoin"
                  value={token.token_name}
                  onChange={(e) => setToken({...token, token_name: e.target.value})}
                  style={{
                    width: '100%',
                    padding: 14,
                    background: colors.bgSecondary,
                    border: `1px solid ${colors.border}`,
                    borderRadius: 8,
                    color: colors.textPrimary,
                    fontSize: 14,
                  }}
                />
              </div>
            </div>

            {/* Quote Token */}
            <div style={{ marginBottom: 24 }}>
              <label style={{ display: 'block', marginBottom: 8, color: colors.textSecondary }}>
                Quote Token (Pair)
              </label>
              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                {quoteTokens.map(qt => (
                  <div
                    key={qt}
                    onClick={() => setQuoteToken(qt)}
                    style={{
                      padding: '10px 20px',
                      borderRadius: 8,
                      border: `2px solid ${quoteToken === qt ? colors.accent : colors.border}`,
                      background: quoteToken === qt ? `${colors.accent}20` : 'transparent',
                      cursor: 'pointer',
                      color: quoteToken === qt ? colors.accent : colors.textPrimary,
                      fontWeight: 500,
                    }}
                  >
                    {qt}
                  </div>
                ))}
              </div>
            </div>

            {/* Optional Links */}
            <div style={{ marginBottom: 24 }}>
              <label style={{ display: 'block', marginBottom: 8, color: colors.textSecondary }}>
                Additional Information (Optional)
              </label>
              <input
                type="text"
                placeholder="Website URL"
                value={token.website_url}
                onChange={(e) => setToken({...token, website_url: e.target.value})}
                style={{
                  width: '100%',
                  padding: 12,
                  background: colors.bgSecondary,
                  border: `1px solid ${colors.border}`,
                  borderRadius: 8,
                  color: colors.textPrimary,
                  fontSize: 14,
                  marginBottom: 8,
                }}
              />
              <input
                type="text"
                placeholder="Twitter/X URL"
                value={token.twitter_url}
                onChange={(e) => setToken({...token, twitter_url: e.target.value})}
                style={{
                  width: '100%',
                  padding: 12,
                  background: colors.bgSecondary,
                  border: `1px solid ${colors.border}`,
                  borderRadius: 8,
                  color: colors.textPrimary,
                  fontSize: 14,
                  marginBottom: 8,
                }}
              />
              <input
                type="text"
                placeholder="Telegram URL"
                value={token.telegram_url}
                onChange={(e) => setToken({...token, telegram_url: e.target.value})}
                style={{
                  width: '100%',
                  padding: 12,
                  background: colors.bgSecondary,
                  border: `1px solid ${colors.border}`,
                  borderRadius: 8,
                  color: colors.textPrimary,
                  fontSize: 14,
                }}
              />
            </div>

            <button
              onClick={() => setStep(2)}
              disabled={!isFormValid()}
              style={{
                width: '100%',
                padding: '16px 32px',
                background: isFormValid() ? colors.accent : colors.bgTertiary,
                border: 'none',
                borderRadius: 8,
                color: isFormValid() ? 'white' : colors.textSecondary,
                fontSize: 16,
                fontWeight: 'bold',
                cursor: isFormValid() ? 'pointer' : 'not-allowed',
                opacity: isFormValid() ? 1 : 0.5,
              }}
            >
              Continue to Tier Selection →
            </button>
          </div>
        )}

        {/* Step 2: Tier Selection */}
        {step === 2 && (
          <div style={{ 
            background: colors.bgCard, 
            border: `1px solid ${colors.border}`, 
            borderRadius: 16, 
            padding: 32 
          }}>
            <h2 style={{ marginTop: 0 }}>Select Listing Tier</h2>
            <p style={{ color: colors.textSecondary, marginBottom: 24 }}>
              Choose the tier that best fits your token.
            </p>

            <div style={{ display: 'grid', gap: 16, marginBottom: 24 }}>
              {tiers.map(tier => (
                <div
                  key={tier.id}
                  onClick={() => setSelectedTier(tier.id)}
                  style={{
                    padding: 24,
                    borderRadius: 12,
                    border: `2px solid ${selectedTier === tier.id ? colors.accent : colors.border}`,
                    background: selectedTier === tier.id ? `${colors.accent}10` : 'transparent',
                    cursor: 'pointer',
                  }}
                >
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div>
                      <h3 style={{ margin: 0, color: colors.textPrimary }}>{tier.name}</h3>
                      <p style={{ margin: '4px 0 0', color: colors.textSecondary, fontSize: 14 }}>
                        Best for {tier.id === 'tier1' ? 'major tokens with high volume' :
                               tier.id === 'tier2' ? 'established projects' :
                               tier.id === 'tier3' ? 'new token projects' :
                               'community tokens'}
                      </p>
                    </div>
                    <div style={{ textAlign: 'right' }}>
                      <div style={{ fontSize: 28, fontWeight: 'bold', color: colors.accent }}>
                        {tier.fee}
                      </div>
                      <div style={{ color: colors.textSecondary }}>TIGER ≈ ${tier.feeUsd}</div>
                    </div>
                  </div>
                </div>
              ))}
            </div>

            <div style={{ display: 'flex', gap: 16 }}>
              <button
                onClick={() => setStep(1)}
                style={{
                  padding: '16px 32px',
                  background: 'transparent',
                  border: `1px solid ${colors.border}`,
                  borderRadius: 8,
                  color: colors.textPrimary,
                  cursor: 'pointer',
                }}
              >
                ← Back
              </button>
              <button
                onClick={() => setStep(3)}
                style={{
                  flex: 1,
                  padding: '16px 32px',
                  background: colors.accent,
                  border: 'none',
                  borderRadius: 8,
                  color: 'white',
                  fontSize: 16,
                  fontWeight: 'bold',
                  cursor: 'pointer',
                }}
              >
                Continue to Review →
              </button>
            </div>
          </div>
        )}

        {/* Step 3: Review */}
        {step === 3 && (
          <div style={{ 
            background: colors.bgCard, 
            border: `1px solid ${colors.border}`, 
            borderRadius: 16, 
            padding: 32 
          }}>
            <h2 style={{ marginTop: 0 }}>Review & Submit</h2>

            {/* Token Details */}
            <div style={{ 
              background: colors.bgSecondary, 
              borderRadius: 12, 
              padding: 24, 
              marginBottom: 24 
            }}>
              <h3 style={{ marginTop: 0, color: colors.textPrimary }}>Token Details</h3>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                <div>
                  <span style={{ color: colors.textSecondary }}>Token: </span>
                  <span style={{ color: colors.textPrimary, fontWeight: 600 }}>
                    {token.token_symbol} ({token.token_name})
                  </span>
                </div>
                <div>
                  <span style={{ color: colors.textSecondary }}>Contract: </span>
                  <span style={{ color: colors.textPrimary, fontFamily: 'monospace', fontSize: 12 }}>
                    {token.contract_address.slice(0, 10)}...{token.contract_address.slice(-8)}
                  </span>
                </div>
                <div>
                  <span style={{ color: colors.textSecondary }}>Chain: </span>
                  <span style={{ color: colors.textPrimary }}>{selectedChain?.name}</span>
                </div>
                <div>
                  <span style={{ color: colors.textSecondary }}>Pair: </span>
                  <span style={{ color: colors.textPrimary }}>{token.token_symbol}/{quoteToken}</span>
                </div>
              </div>
            </div>

            {/* Listing Tier */}
            <div style={{ 
              background: colors.bgSecondary, 
              borderRadius: 12, 
              padding: 24, 
              marginBottom: 24 
            }}>
              <h3 style={{ marginTop: 0, color: colors.textPrimary }}>Listing Tier</h3>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span style={{ fontSize: 18, fontWeight: 600, color: colors.textPrimary }}>
                  {selectedTierData?.name}
                </span>
                <span style={{ fontSize: 28, fontWeight: 'bold', color: colors.accent }}>
                  {selectedTierData?.fee} TIGER
                </span>
              </div>
              <p style={{ color: colors.textSecondary, marginTop: 8 }}>
                ≈ ${selectedTierData?.feeUsd} (payment in crypto only)
              </p>
            </div>

            {/* Payment Info */}
            <div style={{ 
              background: `${colors.warning}10`, 
              border: `1px solid ${colors.warning}`,
              borderRadius: 12, 
              padding: 16, 
              marginBottom: 24 
            }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: colors.warning }}>
                💰
                <strong>Payment: Crypto Only</strong>
              </div>
              <p style={{ color: colors.textSecondary, margin: '8px 0 0', fontSize: 14 }}>
                Accepted: USDT, USDC, ETH, BTC, BNB, SOL
              </p>
            </div>

            {/* Agreement */}
            <div style={{ marginBottom: 24 }}>
              <label style={{ display: 'flex', alignItems: 'center', gap: 12, cursor: 'pointer' }}>
                <input
                  type="checkbox"
                  checked={agreed}
                  onChange={(e) => setAgreed(e.target.checked)}
                  style={{ width: 20, height: 20 }}
                />
                <span style={{ color: colors.textSecondary }}>
                  I agree to the{' '}
                  <a href="#" style={{ color: colors.accent }}>Token Listing Terms</a>
                </span>
              </label>
            </div>

            <div style={{ display: 'flex', gap: 16 }}>
              <button
                onClick={() => setStep(2)}
                style={{
                  padding: '16px 32px',
                  background: 'transparent',
                  border: `1px solid ${colors.border}`,
                  borderRadius: 8,
                  color: colors.textPrimary,
                  cursor: 'pointer',
                }}
              >
                ← Back
              </button>
              <button
                onClick={handleSubmitListing}
                disabled={!agreed || loading}
                style={{
                  flex: 1,
                  padding: '16px 32px',
                  background: agreed && !loading ? colors.accent : colors.bgTertiary,
                  border: 'none',
                  borderRadius: 8,
                  color: agreed && !loading ? 'white' : colors.textSecondary,
                  fontSize: 16,
                  fontWeight: 'bold',
                  cursor: agreed && !loading ? 'pointer' : 'not-allowed',
                }}
              >
                {loading ? 'Submitting...' : 'Submit Application'}
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Auth Modal */}
      {showAuthModal && (
        <div style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          background: colors.overlay,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          zIndex: 1000,
        }}>
          <div style={{
            background: colors.bgCard,
            borderRadius: 16,
            padding: 32,
            width: '90%',
            maxWidth: 400,
          }}>
            <h2 style={{ marginTop: 0, textAlign: 'center' }}>
              {authMode === 'login' ? '🔐 Login' : '📝 Register'}
            </h2>

            <form onSubmit={handleAuth}>
              {authMode === 'register' && (
                <input
                  name="username"
                  type="text"
                  placeholder="Username"
                  required
                  style={{
                    width: '100%',
                    padding: 14,
                    background: colors.bgSecondary,
                    border: `1px solid ${colors.border}`,
                    borderRadius: 8,
                    color: colors.textPrimary,
                    fontSize: 14,
                    marginBottom: 12,
                  }}
                />
              )}
              <input
                name="email"
                type="email"
                placeholder="Email"
                required
                style={{
                  width: '100%',
                  padding: 14,
                  background: colors.bgSecondary,
                  border: `1px solid ${colors.border}`,
                  borderRadius: 8,
                  color: colors.textPrimary,
                  fontSize: 14,
                  marginBottom: 12,
                }}
              />
              <input
                name="password"
                type="password"
                placeholder="Password"
                required
                style={{
                  width: '100%',
                  padding: 14,
                  background: colors.bgSecondary,
                  border: `1px solid ${colors.border}`,
                  borderRadius: 8,
                  color: colors.textPrimary,
                  fontSize: 14,
                  marginBottom: 16,
                }}
              />

              <button
                type="submit"
                disabled={loading}
                style={{
                  width: '100%',
                  padding: 14,
                  background: colors.accent,
                  border: 'none',
                  borderRadius: 8,
                  color: 'white',
                  fontWeight: 'bold',
                  cursor: loading ? 'not-allowed' : 'pointer',
                }}
              >
                {loading ? 'Please wait...' : authMode === 'login' ? 'Login' : 'Register'}
              </button>
            </form>

            <p style={{ textAlign: 'center', marginTop: 16, color: colors.textSecondary }}>
              {authMode === 'login' ? "Don't have an account? " : "Already have an account? "}
              <button
                onClick={() => setAuthMode(authMode === 'login' ? 'register' : 'login')}
                style={{
                  background: 'none',
                  border: 'none',
                  color: colors.accent,
                  cursor: 'pointer',
                  textDecoration: 'underline',
                }}
              >
                {authMode === 'login' ? 'Register' : 'Login'}
              </button>
            </p>

            <button
              onClick={() => setShowAuthModal(false)}
              style={{
                position: 'absolute',
                top: 16,
                right: 16,
                background: 'none',
                border: 'none',
                color: colors.textSecondary,
                cursor: 'pointer',
                fontSize: 20,
              }}
            >
              ✕
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
