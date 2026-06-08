'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { ThemeToggle } from '../../components/ThemeToggle';
import { useTheme } from '../../components/ThemeProvider';

// ============================================================================
// Types
// ============================================================================

interface WhiteLabelRegistration {
  client_name: string;
  brand_name: string;
  contact_email: string;
  website: string;
  tier: 'basic' | 'pro' | 'enterprise';
  domain: string;
  cloud_provider: string;
}

interface WhiteLabelAPIKey {
  id: string;
  key: string;
  name: string;
  createdAt: number;
  expiresAt: number;
  lastUsed: number;
  status: 'active' | 'expired' | 'revoked';
}

interface WhiteLabelClient {
  client_id: string;
  client_name: string;
  brand_name: string;
  status: 'pending' | 'approved' | 'suspended';
  tier: string;
  apiKeys: WhiteLabelAPIKey[];
  swap_fee_share_bps: number;
  trading_fee_share_bps: number;
  createdAt: number;
}

// ============================================================================
// API Functions
// ============================================================================

const api = {
  async registerWhiteLabel(data: WhiteLabelRegistration): Promise<{ clientId: string; status: string }> {
    const response = await fetch('/api/v1/white-label/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'Registration failed');
    }
    
    return response.json();
  },
  
  async getClientStatus(clientId: string): Promise<WhiteLabelClient> {
    const response = await fetch(`/api/v1/white-label/client/${clientId}`, {
      method: 'GET',
    });
    
    if (!response.ok) {
      throw new Error('Failed to get client status');
    }
    
    return response.json();
  },
  
  async generateAPIKey(clientId: string, name: string): Promise<{ apiKey: string; keyId: string }> {
    const response = await fetch(`/api/v1/white-label/client/${clientId}/api-keys`, {
      method: 'POST',
      headers: { 
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
      body: JSON.stringify({ name }),
    });
    
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'API key generation failed');
    }
    
    return response.json();
  },
  
  async validateAPIKey(apiKey: string): Promise<{ valid: boolean; clientId: string; message: string }> {
    const response = await fetch('/api/v1/white-label/validate-key', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ apiKey }),
    });
    
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message || 'API key validation failed');
    }
    
    return response.json();
  },
  
  async getFeeConfig(clientId: string): Promise<{ swap_fee: number; trading_fee: number; bridge_fee: number }> {
    const response = await fetch(`/api/v1/white-label/client/${clientId}/fees`, {
      method: 'GET',
    });
    
    if (!response.ok) {
      throw new Error('Failed to get fee config');
    }
    
    return response.json();
  },
};

// ============================================================================
// Validation Functions
// ============================================================================

const validateEmail = (email: string): boolean => {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email);
};

const validateDomain = (domain: string): boolean => {
  const domainRegex = /^[a-zA-Z0-9][a-zA-Z0-9-]{1,61}[a-zA-Z0-9]?\.[a-zA-Z]{2,}$/;
  return domainRegex.test(domain);
};

const validateWebsite = (website: string): boolean => {
  try {
    const url = new URL(website);
    return url.protocol === 'https:';
  } catch {
    return false;
  }
};

// ============================================================================
// Main Component
// ============================================================================

export default function WhiteLabelRegisterPage() {
  const router = useRouter();
  const { theme } = useTheme();
  
  // State
  const [mode, setMode] = useState<'register' | 'pending' | 'approved' | 'api_keys'>('register');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  
  // Form state
  const [formData, setFormData] = useState<WhiteLabelRegistration>({
    client_name: '',
    brand_name: '',
    contact_email: '',
    website: '',
    tier: 'basic',
    domain: '',
    cloud_provider: 'aws',
  });
  
  // Client state
  const [client, setClient] = useState<WhiteLabelClient | null>(null);
  const [clientId, setClientId] = useState<string | null>(null);
  const [apiKeys, setAPIKeys] = useState<WhiteLabelAPIKey[]>([]);
  const [newAPIKeyName, setNewAPIKeyName] = useState('');
  const [showAPIKey, setShowAPIKey] = useState<string | null>(null);
  
  // ============================================================================
  // Handlers
  // ============================================================================
  
  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    
    // Validate inputs
    if (!formData.client_name || !formData.brand_name || !formData.contact_email) {
      setError('Please fill in all required fields');
      return;
    }
    
    if (!validateEmail(formData.contact_email)) {
      setError('Please enter a valid email address');
      return;
    }
    
    if (formData.website && !validateWebsite(formData.website)) {
      setError('Please enter a valid HTTPS website URL');
      return;
    }
    
    if (formData.domain && !validateDomain(formData.domain)) {
      setError('Please enter a valid domain (e.g., wallet.yourbrand.com)');
      return;
    }
    
    setLoading(true);
    
    try {
      const result = await api.registerWhiteLabel(formData);
      setClientId(result.clientId);
      setSuccess('Registration submitted! Pending approval from TigerWallet admin.');
      setMode('pending');
    } catch (err: any) {
      setError(err.message || 'Registration failed');
    } finally {
      setLoading(false);
    }
  };
  
  const checkStatus = async () => {
    if (!clientId) return;
    
    setLoading(true);
    
    try {
      const clientData = await api.getClientStatus(clientId);
      setClient(clientData);
      
      if (clientData.status === 'approved') {
        setMode('approved');
        setAPIKeys(clientData.apiKeys || []);
      } else if (clientData.status === 'pending') {
        setMode('pending');
      }
    } catch (err: any) {
      setError(err.message || 'Failed to check status');
    } finally {
      setLoading(false);
    }
  };
  
  const generateAPIKey = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!newAPIKeyName.trim()) {
      setError('Please enter a name for the API key');
      return;
    }
    
    if (!clientId) return;
    
    setLoading(true);
    
    try {
      const result = await api.generateAPIKey(clientId, newAPIKeyName);
      
      const newKey: WhiteLabelAPIKey = {
        id: result.keyId,
        key: result.apiKey,
        name: newAPIKeyName,
        createdAt: Date.now(),
        expiresAt: Date.now() + 365 * 24 * 60 * 60 * 1000, // 1 year
        lastUsed: 0,
        status: 'active',
      };
      
      setAPIKeys([...apiKeys, newKey]);
      setShowAPIKey(result.apiKey);
      setNewAPIKeyName('');
      setMode('api_keys');
    } catch (err: any) {
      setError(err.message || 'Failed to generate API key');
    } finally {
      setLoading(false);
    }
  };
  
  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setSuccess('Copied to clipboard!');
  };
  
  // ============================================================================
  // Render
  // ============================================================================
  
  return (
    <div className="min-h-screen bg-slate-950">
      <div className="absolute top-0 left-0 right-0 p-4 flex justify-between items-center z-10">
        <div className="logo text-2xl font-bold text-orange-500">🐯 TigerWallet</div>
        <ThemeToggle />
      </div>
      
      <div className="min-h-screen flex items-center justify-center p-4">
        <div className="w-full max-w-2xl">
          {/* Error/Success Messages */}
          {error && (
            <div className="mb-4 p-4 bg-red-500/10 border border-red-500/50 rounded-lg text-red-500">
              {error}
              <button onClick={() => setError(null)} className="float-right">✕</button>
            </div>
          )}
          {success && (
            <div className="mb-4 p-4 bg-green-500/10 border border-green-500/50 rounded-lg text-green-500">
              {success}
              <button onClick={() => setSuccess(null)} className="float-right">✕</button>
            </div>
          )}
          
          {/* Registration Form */}
          {mode === 'register' && (
            <div className="bg-slate-900/80 rounded-2xl p-8 backdrop-blur-xl border border-slate-800">
              <div className="text-center mb-8">
                <h1 className="text-3xl font-bold text-white mb-2">White Label Program</h1>
                <p className="text-slate-400">
                  Create your branded TigerWallet product with 20% fee sharing
                </p>
              </div>
              
              <form onSubmit={handleRegister}>
                <div className="grid grid-cols-2 gap-4 mb-4">
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-2">
                      Company Name *
                    </label>
                    <input
                      type="text"
                      value={formData.client_name}
                      onChange={(e) => setFormData({ ...formData, client_name: e.target.value })}
                      className="w-full px-4 py-3 bg-slate-800/50 border border-slate-700 rounded-lg text-white focus:outline-none focus:border-orange-500"
                      placeholder="Your Company LLC"
                      required
                    />
                  </div>
                  
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-2">
                      Brand Name *
                    </label>
                    <input
                      type="text"
                      value={formData.brand_name}
                      onChange={(e) => setFormData({ ...formData, brand_name: e.target.value })}
                      className="w-full px-4 py-3 bg-slate-800/50 border border-slate-700 rounded-lg text-white focus:outline-none focus:border-orange-500"
                      placeholder="MyWallet"
                      required
                    />
                  </div>
                </div>
                
                <div className="mb-4">
                  <label className="block text-sm font-medium text-slate-300 mb-2">
                    Contact Email *
                  </label>
                  <input
                    type="email"
                    value={formData.contact_email}
                    onChange={(e) => setFormData({ ...formData, contact_email: e.target.value })}
                    className="w-full px-4 py-3 bg-slate-800/50 border border-slate-700 rounded-lg text-white focus:outline-none focus:border-orange-500"
                    placeholder="contact@yourcompany.com"
                    required
                  />
                </div>
                
                <div className="mb-4">
                  <label className="block text-sm font-medium text-slate-300 mb-2">
                    Website URL
                  </label>
                  <input
                    type="url"
                    value={formData.website}
                    onChange={(e) => setFormData({ ...formData, website: e.target.value })}
                    className="w-full px-4 py-3 bg-slate-800/50 border border-slate-700 rounded-lg text-white focus:outline-none focus:border-orange-500"
                    placeholder="https://www.yourcompany.com"
                  />
                </div>
                
                <div className="grid grid-cols-2 gap-4 mb-4">
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-2">
                      Domain for Your Wallet
                    </label>
                    <input
                      type="text"
                      value={formData.domain}
                      onChange={(e) => setFormData({ ...formData, domain: e.target.value })}
                      className="w-full px-4 py-3 bg-slate-800/50 border border-slate-700 rounded-lg text-white focus:outline-none focus:border-orange-500"
                      placeholder="wallet.yourbrand.com"
                    />
                  </div>
                  
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-2">
                      Cloud Provider
                    </label>
                    <select
                      value={formData.cloud_provider}
                      onChange={(e) => setFormData({ ...formData, cloud_provider: e.target.value })}
                      className="w-full px-4 py-3 bg-slate-800/50 border border-slate-700 rounded-lg text-white focus:outline-none focus:border-orange-500"
                    >
                      <option value="aws">AWS</option>
                      <option value="gcp">Google Cloud</option>
                      <option value="azure">Azure</option>
                      <option value="custom">Custom</option>
                    </select>
                  </div>
                </div>
                
                <div className="mb-6">
                  <label className="block text-sm font-medium text-slate-300 mb-2">
                    Plan Tier
                  </label>
                  <div className="grid grid-cols-3 gap-4">
                    {[
                      { value: 'basic', label: 'Basic', price: '$499/mo', features: '5,000 users' },
                      { value: 'pro', label: 'Pro', price: '$999/mo', features: '50,000 users' },
                      { value: 'enterprise', label: 'Enterprise', price: 'Custom', features: 'Unlimited users' },
                    ].map((tier) => (
                      <div
                        key={tier.value}
                        onClick={() => setFormData({ ...formData, tier: tier.value as any })}
                        className={`p-4 rounded-lg border cursor-pointer ${
                          formData.tier === tier.value
                            ? 'border-orange-500 bg-orange-500/10'
                            : 'border-slate-700 hover:border-slate-600'
                        }`}
                      >
                        <div className="font-bold text-white">{tier.label}</div>
                        <div className="text-orange-500">{tier.price}</div>
                        <div className="text-xs text-slate-400">{tier.features}</div>
                      </div>
                    ))}
                  </div>
                </div>
                
                <button
                  type="submit"
                  disabled={loading}
                  className="w-full py-3 bg-orange-500 hover:bg-orange-600 disabled:bg-slate-600 rounded-lg text-white font-semibold transition-colors"
                >
                  {loading ? 'Submitting...' : 'Apply for White Label'}
                </button>
                
                <div className="mt-4 text-center text-sm text-slate-500">
                  <p>20% of all fees will be shared with TigerWallet</p>
                  <p>Requires TigerWallet admin approval</p>
                </div>
              </form>
            </div>
          )}
          
          {/* Pending Status */}
          {mode === 'pending' && (
            <div className="bg-slate-900/80 rounded-2xl p-8 backdrop-blur-xl border border-slate-800 text-center">
              <div className="text-6xl mb-4">⏳</div>
              <h1 className="text-2xl font-bold text-white mb-2">Pending Approval</h1>
              <p className="text-slate-400 mb-8">
                Your white label application is being reviewed by TigerWallet admin.
                You'll be notified once approved.
              </p>
              
              <button
                onClick={checkStatus}
                disabled={loading}
                className="px-6 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg text-white"
              >
                {loading ? 'Checking...' : 'Check Status'}
              </button>
            </div>
          )}
          
          {/* Approved Status */}
          {mode === 'approved' && (
            <div className="bg-slate-900/80 rounded-2xl p-8 backdrop-blur-xl border border-slate-800">
              <div className="text-center mb-8">
                <div className="text-6xl mb-4">🎉</div>
                <h1 className="text-2xl font-bold text-white mb-2">Approved!</h1>
                <p className="text-slate-400">
                  Your white label product is now active
                </p>
              </div>
              
              {client && (
                <div className="mb-6 p-4 bg-slate-800/50 rounded-lg">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <span className="text-slate-400">Client ID</span>
                      <p className="font-mono text-sm">{client.client_id}</p>
                    </div>
                    <div>
                      <span className="text-slate-400">Brand</span>
                      <p className="font-bold">{client.brand_name}</p>
                    </div>
                    <div>
                      <span className="text-slate-400">Swap Fee</span>
                      <p className="text-orange-500">{client.swap_fee_share_bps / 100}%</p>
                    </div>
                    <div>
                      <span className="text-slate-400">Trading Fee</span>
                      <p className="text-orange-500">{client.trading_fee_share_bps / 100}%</p>
                    </div>
                  </div>
                </div>
              )}
              
              <button
                onClick={() => setMode('api_keys')}
                className="w-full py-3 bg-orange-500 hover:bg-orange-600 rounded-lg text-white font-semibold"
              >
                Manage API Keys
              </button>
            </div>
          )}
          
          {/* API Keys Management */}
          {mode === 'api_keys' && (
            <div className="bg-slate-900/80 rounded-2xl p-8 backdrop-blur-xl border border-slate-800">
              <div className="flex justify-between items-center mb-6">
                <h1 className="text-2xl font-bold text-white">API Keys</h1>
                <button
                  onClick={() => setMode('approved')}
                  className="text-slate-400 hover:text-white"
                >
                  ← Back
                </button>
              </div>
              
              {/* Show new API key */}
              {showAPIKey && (
                <div className="mb-6 p-4 bg-green-500/10 border border-green-500/50 rounded-lg">
                  <p className="text-green-500 font-bold mb-2">API Key Generated!</p>
                  <p className="text-sm text-slate-400 mb-2">
                    Save this key - it won't be shown again:
                  </p>
                  <code className="block p-2 bg-slate-800 rounded text-sm break-all">
                    {showAPIKey}
                  </code>
                  <button
                    onClick={() => copyToClipboard(showAPIKey)}
                    className="mt-2 text-sm text-orange-500 hover:underline"
                  >
                    Copy to clipboard
                  </button>
                </div>
              )}
              
              {/* Existing API keys */}
              <div className="mb-6">
                <h2 className="text-lg font-semibold text-white mb-4">Your API Keys</h2>
                {apiKeys.length === 0 ? (
                  <p className="text-slate-400">No API keys yet</p>
                ) : (
                  <div className="space-y-2">
                    {apiKeys.map((key) => (
                      <div
                        key={key.id}
                        className="p-3 bg-slate-800/50 rounded-lg flex justify-between items-center"
                      >
                        <div>
                          <div className="font-medium">{key.name}</div>
                          <div className="text-xs text-slate-400">
                            Created: {new Date(key.createdAt).toLocaleDateString()}
                          </div>
                        </div>
                        <div className={`px-2 py-1 rounded text-xs ${
                          key.status === 'active' ? 'bg-green-500' : 'bg-red-500'
                        }`}>
                          {key.status}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
              
              {/* Generate new key */}
              <form onSubmit={generateAPIKey} className="flex gap-2">
                <input
                  type="text"
                  value={newAPIKeyName}
                  onChange={(e) => setNewAPIKeyName(e.target.value)}
                  placeholder="Key name (e.g., Production)"
                  className="flex-1 px-4 py-2 bg-slate-800/50 border border-slate-700 rounded-lg text-white"
                />
                <button
                  type="submit"
                  disabled={loading}
                  className="px-4 py-2 bg-orange-500 hover:bg-orange-600 rounded-lg text-white"
                >
                  Generate
                </button>
              </form>
              
              <div className="mt-6 p-4 bg-yellow-500/10 border border-yellow-500/50 rounded-lg">
                <p className="text-yellow-500 font-bold mb-2">Important</p>
                <p className="text-sm text-slate-400">
                  Your white label product requires valid API keys to work. 
                  Without authorized API keys, your product will show 
                  "Please input authorized API keys" error.
                </p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}