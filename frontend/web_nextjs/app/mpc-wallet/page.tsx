'use client';

import React, { useState, useEffect } from 'react';
import { useWallet } from '../wallet';

// ================================================================================
// Types
// ================================================================================

interface MPCKeyShare {
  shareId: string;
  holderId: string;
  index: number;
  status: 'generated' | 'distributed' | 'active';
}

interface MPCSession {
  sessionId: string;
  challenge: string;
  expiresAt: number;
  requiredShares: number;
  collectedShares: number;
}

interface SocialLoginProvider {
  id: string;
  name: string;
  icon: string;
  color: string;
}

interface WalletDevice {
  id: string;
  name: string;
  type: 'mobile' | 'desktop' | 'tablet' | 'hardware';
  lastActive: number;
  trusted: boolean;
}

// ================================================================================
// Social Login Providers
// ================================================================================

const socialProviders: SocialLoginProvider[] = [
  { id: 'google', name: 'Google', icon: 'G', color: '#4285F4' },
  { id: 'apple', name: 'Apple', icon: '', color: '#000000' },
  { id: 'twitter', name: 'X (Twitter)', icon: 'X', color: '#000000' },
  { id: 'discord', name: 'Discord', icon: 'D', color: '#5865F2' },
  { id: 'telegram', name: 'Telegram', icon: 'T', color: '#0088CC' },
  { id: 'email', name: 'Email', icon: '✉', color: '#EA4335' },
];

// ================================================================================
// API Service
// ================================================================================

const API_BASE_URL = process.env.NEXT_PUBLIC_MPC_API_URL || 'http://localhost:8081/api/v1';

class MPCService {
  private baseUrl: string;

  constructor(baseUrl: string = API_BASE_URL) {
    this.baseUrl = baseUrl;
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Request failed' }));
      throw new Error(error.error || `HTTP ${response.status}`);
    }

    return response.json();
  }

  // OAuth Social Login
  async socialLogin(provider: string, idToken: string): Promise<{ walletAddress: string; sessionToken: string }> {
    return this.request('/auth/social', {
      method: 'POST',
      body: JSON.stringify({ provider, idToken }),
    });
  }

  // Key Share Management
  async generateKeyShares(threshold: number, totalShares: number): Promise<{ publicKey: string; shares: MPCKeyShare[] }> {
    return this.request('/keys/generate', {
      method: 'POST',
      body: JSON.stringify({ threshold, totalShares }),
    });
  }

  async distributeShare(shareId: string, deviceId: string): Promise<{ encryptedShare: string }> {
    return this.request(`/keys/distribute/${shareId}`, {
      method: 'POST',
      body: JSON.stringify({ deviceId }),
    });
  }

  // Signing Sessions
  async createSigningSession(messageHash: string, threshold: number): Promise<MPCSession> {
    return this.request('/signing/session', {
      method: 'POST',
      body: JSON.stringify({ messageHash, threshold }),
    });
  }

  async submitSignatureShare(sessionId: string, shareId: string, signature: string): Promise<{ collected: number; required: number }> {
    return this.request(`/signing/submit/${sessionId}`, {
      method: 'POST',
      body: JSON.stringify({ shareId, signature }),
    });
  }

  // Device Management
  async getDevices(): Promise<WalletDevice[]> {
    return this.request('/devices');
  }

  async addDevice(name: string, deviceType: string): Promise<WalletDevice> {
    return this.request('/devices', {
      method: 'POST',
      body: JSON.stringify({ name, type: deviceType }),
    });
  }

  async removeDevice(deviceId: string): Promise<void> {
    await this.request(`/devices/${deviceId}`, { method: 'DELETE' });
  }

  async trustDevice(deviceId: string): Promise<void> {
    await this.request(`/devices/${deviceId}/trust`, { method: 'POST' });
  }

  // Recovery
  async initiateRecovery(): Promise<{ recoveryId: string; backupCodes: string[] }> {
    return this.request('/recovery/initiate', { method: 'POST' });
  }

  async recoverWithBackupCode(recoveryId: string, backupCode: string): Promise<{ walletAddress: string }> {
    return this.request('/recovery/backup', {
      method: 'POST',
      body: JSON.stringify({ recoveryId, backupCode }),
    });
  }
}

const mpcService = new MPCService();

// ================================================================================
// Main Component
// ================================================================================

export default function MPCWalletPage() {
  const { address, isConnected } = useWallet();
  
  // Auth state
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [sessionToken, setSessionToken] = useState<string | null>(null);
  const [walletAddress, setWalletAddress] = useState<string | null>(null);
  
  // Key shares
  const [keyShares, setKeyShares] = useState<MPCKeyShare[]>([]);
  const [threshold, setThreshold] = useState(2);
  const [totalShares, setTotalShares] = useState(3);
  
  // Devices
  const [devices, setDevices] = useState<WalletDevice[]>([]);
  
  // Signing
  const [signingSession, setSigningSession] = useState<MPCSession | null>(null);
  const [messageToSign, setMessageToSign] = useState('');
  const [signature, setSignature] = useState('');
  
  // UI State
  const [activeTab, setActiveTab] = useState<'login' | 'keys' | 'devices' | 'signing' | 'recovery'>('login');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    if (isConnected && address) {
      setWalletAddress(address);
      loadDevices();
      loadKeyShares();
    }
  }, [isConnected, address]);

  const loadDevices = async () => {
    try {
      const deviceList = await mpcService.getDevices();
      setDevices(deviceList);
    } catch (err) {
      console.error('Failed to load devices:', err);
    }
  };

  const loadKeyShares = async () => {
    // Load existing key shares info
    setKeyShares([
      { shareId: '1', holderId: 'device-1', index: 1, status: 'active' },
      { shareId: '2', holderId: 'device-2', index: 2, status: 'distributed' },
      { shareId: '3', holderId: 'recovery', index: 3, status: 'generated' },
    ]);
  };

  const handleSocialLogin = async (provider: string) => {
    setIsLoading(true);
    setError(null);
    
    try {
      // In production, this would initiate OAuth flow
      // For demo, simulate with a mock ID token
      const mockIdToken = `mock_token_${provider}_${Date.now()}`;
      
      const result = await mpcService.socialLogin(provider, mockIdToken);
      setSessionToken(result.sessionToken);
      setWalletAddress(result.walletAddress);
      setIsAuthenticated(true);
      setSuccess(`Logged in successfully with ${provider}!`);
      
      // Load devices after login
      loadDevices();
      loadKeyShares();
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  };

  const handleGenerateKeys = async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      const result = await mpcService.generateKeyShares(threshold, totalShares);
      setKeyShares(result.shares.map((share, i) => ({
        ...share,
        index: i + 1,
        status: 'generated' as const,
      })));
      setSuccess(`Generated ${totalShares} key shares with threshold ${threshold}`);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  };

  const handleDistributeShare = async (shareId: string, deviceName: string) => {
    setIsLoading(true);
    
    try {
      await mpcService.distributeShare(shareId, deviceName);
      setKeyShares(prev => prev.map(share => 
        share.shareId === shareId ? { ...share, status: 'distributed' } : share
      ));
      setSuccess('Key share distributed successfully!');
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  };

  const handleAddDevice = async (name: string, type: string) => {
    setIsLoading(true);
    
    try {
      const device = await mpcService.addDevice(name, type);
      setDevices(prev => [...prev, device]);
      setSuccess('Device added successfully!');
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  };

  const handleRemoveDevice = async (deviceId: string) => {
    setIsLoading(true);
    
    try {
      await mpcService.removeDevice(deviceId);
      setDevices(prev => prev.filter(d => d.id !== deviceId));
      setSuccess('Device removed successfully!');
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSignMessage = async () => {
    if (!messageToSign) return;
    
    setIsLoading(true);
    setError(null);
    
    try {
      // Create signing session
      const messageHash = Buffer.from(messageToSign).toString('hex');
      const session = await mpcService.createSigningSession(messageHash, threshold);
      setSigningSession(session);
      
      // In production, this would collect signature shares from devices
      // For demo, simulate completed signature
      setTimeout(async () => {
        const mockSignature = `0x${'a'.repeat(130)}`;
        setSignature(mockSignature);
        setSigningSession(null);
        setSuccess('Message signed successfully!');
      }, 2000);
      
    } catch (err: any) {
      setError(err.message);
      setSigningSession(null);
    } finally {
      setIsLoading(false);
    }
  };

  const handleInitiateRecovery = async () => {
    setIsLoading(true);
    
    try {
      const result = await mpcService.initiateRecovery();
      setSuccess(`Recovery initiated! Save these backup codes: ${result.backupCodes.slice(0, 4).join(', ')}...`);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  };

  const formatAddress = (addr: string | null): string => {
    if (!addr) return '';
    return `${addr.slice(0, 6)}...${addr.slice(-4)}`;
  };

  const getStatusColor = (status: string): string => {
    switch (status) {
      case 'active': return 'bg-green-500/20 text-green-400';
      case 'distributed': return 'bg-blue-500/20 text-blue-400';
      case 'generated': return 'bg-yellow-500/20 text-yellow-400';
      default: return 'bg-slate-500/20 text-slate-400';
    }
  };

  const getDeviceIcon = (type: string): string => {
    switch (type) {
      case 'mobile': return '📱';
      case 'desktop': return '💻';
      case 'tablet': return '📱';
      case 'hardware': return '🔐';
      default: return '❓';
    }
  };

  // ============================================================================
  // Render
  // ============================================================================

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 to-slate-800 p-8">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-4xl font-bold text-white mb-2">MPC Wallet</h1>
          <p className="text-slate-400">Multi-Party Computation with Social Login</p>
        </div>

        {/* Wallet Address */}
        {walletAddress && (
          <div className="bg-slate-800/50 rounded-xl p-4 mb-6 border border-slate-700">
            <div className="flex items-center justify-between">
              <div>
                <span className="text-slate-400 text-sm">Wallet Address:</span>
                <span className="text-white ml-2 font-mono">{formatAddress(walletAddress)}</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
                <span className="text-green-400 text-sm">MPC Protected</span>
              </div>
            </div>
          </div>
        )}

        {/* Tabs */}
        <div className="flex gap-2 mb-6 overflow-x-auto">
          {(['login', 'keys', 'devices', 'signing', 'recovery'] as const).map(tab => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-4 py-2 rounded-lg font-medium transition-colors whitespace-nowrap ${
                activeTab === tab
                  ? 'bg-blue-600 text-white'
                  : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
              }`}
            >
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
            </button>
          ))}
        </div>

        {/* Login Tab */}
        {activeTab === 'login' && (
          <div className="bg-slate-800 rounded-2xl p-6 border border-slate-700">
            <h2 className="text-xl font-semibold text-white mb-6">Social Login</h2>
            
            <div className="grid grid-cols-2 md:grid-cols-3 gap-4 mb-6">
              {socialProviders.map(provider => (
                <button
                  key={provider.id}
                  onClick={() => handleSocialLogin(provider.id)}
                  disabled={isLoading}
                  className="flex items-center gap-3 p-4 bg-slate-700/50 hover:bg-slate-700 rounded-xl transition-colors"
                  style={{ borderColor: provider.color }}
                >
                  <span className="text-2xl">{provider.icon}</span>
                  <span className="text-white font-medium">{provider.name}</span>
                </button>
              ))}
            </div>

            <div className="border-t border-slate-700 pt-6 mt-6">
              <h3 className="text-lg font-medium text-white mb-4">Or connect with Web3</h3>
              <p className="text-slate-400 text-sm">
                Connect your existing wallet to enable MPC key sharing and social recovery features.
              </p>
            </div>
          </div>
        )}

        {/* Key Shares Tab */}
        {activeTab === 'keys' && (
          <div className="bg-slate-800 rounded-2xl p-6 border border-slate-700">
            <h2 className="text-xl font-semibold text-white mb-6">Key Share Management</h2>
            
            <div className="grid grid-cols-2 gap-4 mb-6">
              <div>
                <label className="block text-slate-400 text-sm mb-2">Threshold (signatures needed)</label>
                <input
                  type="number"
                  value={threshold}
                  onChange={(e) => setThreshold(parseInt(e.target.value))}
                  min={1}
                  max={totalShares}
                  className="w-full bg-slate-700 border border-slate-600 rounded-lg px-4 py-3 text-white"
                />
              </div>
              <div>
                <label className="block text-slate-400 text-sm mb-2">Total Shares</label>
                <input
                  type="number"
                  value={totalShares}
                  onChange={(e) => setTotalShares(parseInt(e.target.value))}
                  min={threshold}
                  max={10}
                  className="w-full bg-slate-700 border border-slate-600 rounded-lg px-4 py-3 text-white"
                />
              </div>
            </div>

            <button
              onClick={handleGenerateKeys}
              disabled={isLoading || threshold > totalShares}
              className="bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 text-white px-6 py-3 rounded-xl font-medium transition-colors mb-6"
            >
              {isLoading ? 'Generating...' : 'Generate New Key Shares'}
            </button>

            {keyShares.length > 0 && (
              <div className="space-y-3">
                <h3 className="text-lg font-medium text-white">Current Key Shares</h3>
                {keyShares.map((share, i) => (
                  <div key={share.shareId} className="flex items-center justify-between p-4 bg-slate-700/50 rounded-lg">
                    <div className="flex items-center gap-4">
                      <span className="text-2xl">🔑</span>
                      <div>
                        <p className="text-white font-medium">Share #{share.index}</p>
                        <p className="text-slate-400 text-sm">Holder: {share.holderId}</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <span className={`px-3 py-1 rounded-full text-xs font-medium ${getStatusColor(share.status)}`}>
                        {share.status}
                      </span>
                      {share.status === 'generated' && (
                        <button
                          onClick={() => handleDistributeShare(share.shareId, share.holderId)}
                          className="text-blue-400 hover:text-blue-300 text-sm"
                        >
                          Distribute
                        </button>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Devices Tab */}
        {activeTab === 'devices' && (
          <div className="bg-slate-800 rounded-2xl p-6 border border-slate-700">
            <h2 className="text-xl font-semibold text-white mb-6">Trusted Devices</h2>
            
            <div className="space-y-3 mb-6">
              {devices.length === 0 ? (
                <p className="text-slate-400 text-center py-8">No devices added yet</p>
              ) : (
                devices.map(device => (
                  <div key={device.id} className="flex items-center justify-between p-4 bg-slate-700/50 rounded-lg">
                    <div className="flex items-center gap-4">
                      <span className="text-2xl">{getDeviceIcon(device.type)}</span>
                      <div>
                        <p className="text-white font-medium">{device.name}</p>
                        <p className="text-slate-400 text-sm">
                          Last active: {new Date(device.lastActive).toLocaleDateString()}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <span className={`px-3 py-1 rounded-full text-xs font-medium ${
                        device.trusted ? 'bg-green-500/20 text-green-400' : 'bg-yellow-500/20 text-yellow-400'
                      }`}>
                        {device.trusted ? 'Trusted' : 'Pending'}
                      </span>
                      <button
                        onClick={() => handleRemoveDevice(device.id)}
                        className="text-red-400 hover:text-red-300 text-sm"
                      >
                        Remove
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>

            <div className="border-t border-slate-700 pt-6">
              <h3 className="text-lg font-medium text-white mb-4">Add New Device</h3>
              <div className="flex gap-4">
                <input
                  type="text"
                  placeholder="Device name"
                  id="deviceName"
                  className="flex-1 bg-slate-700 border border-slate-600 rounded-lg px-4 py-3 text-white"
                />
                <select id="deviceType" className="bg-slate-700 border border-slate-600 rounded-lg px-4 py-3 text-white">
                  <option value="mobile">Mobile</option>
                  <option value="desktop">Desktop</option>
                  <option value="tablet">Tablet</option>
                  <option value="hardware">Hardware Wallet</option>
                </select>
                <button
                  onClick={() => {
                    const name = (document.getElementById('deviceName') as HTMLInputElement).value;
                    const type = (document.getElementById('deviceType') as HTMLSelectElement).value;
                    if (name) handleAddDevice(name, type);
                  }}
                  className="bg-blue-600 hover:bg-blue-700 text-white px-6 py-3 rounded-xl font-medium transition-colors"
                >
                  Add
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Signing Tab */}
        {activeTab === 'signing' && (
          <div className="bg-slate-800 rounded-2xl p-6 border border-slate-700">
            <h2 className="text-xl font-semibold text-white mb-6">MPC Signing</h2>
            
            <div className="mb-6">
              <label className="block text-slate-400 text-sm mb-2">Message to Sign</label>
              <textarea
                value={messageToSign}
                onChange={(e) => setMessageToSign(e.target.value)}
                placeholder="Enter message to sign..."
                rows={4}
                className="w-full bg-slate-700 border border-slate-600 rounded-lg px-4 py-3 text-white placeholder-slate-500"
              />
            </div>

            <button
              onClick={handleSignMessage}
              disabled={!messageToSign || isLoading}
              className="bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 text-white px-6 py-3 rounded-xl font-medium transition-colors mb-6"
            >
              {isLoading ? 'Signing...' : 'Sign with MPC'}
            </button>

            {signingSession && (
              <div className="p-4 bg-yellow-500/10 border border-yellow-500/30 rounded-xl mb-4">
                <div className="flex items-center gap-2 text-yellow-400">
                  <span className="w-3 h-3 bg-yellow-500 rounded-full animate-pulse"></span>
                  <span>Collecting signature shares... ({signingSession.collectedShares}/{signingSession.requiredShares})</span>
                </div>
              </div>
            )}

            {signature && (
              <div className="p-4 bg-green-500/10 border border-green-500/30 rounded-xl">
                <p className="text-green-400 font-medium mb-2">Signature:</p>
                <p className="text-green-400/70 text-sm font-mono break-all">{signature}</p>
              </div>
            )}
          </div>
        )}

        {/* Recovery Tab */}
        {activeTab === 'recovery' && (
          <div className="bg-slate-800 rounded-2xl p-6 border border-slate-700">
            <h2 className="text-xl font-semibold text-white mb-6">Account Recovery</h2>
            
            <div className="p-4 bg-blue-500/10 border border-blue-500/30 rounded-xl mb-6">
              <p className="text-blue-400">
                Recover your wallet using backup codes or social guardians. 
                With {threshold}-of-{totalShares} threshold, you need {threshold} recovery methods.
              </p>
            </div>

            <button
              onClick={handleInitiateRecovery}
              disabled={isLoading}
              className="bg-yellow-600 hover:bg-yellow-700 disabled:bg-slate-600 text-white px-6 py-3 rounded-xl font-medium transition-colors mb-6"
            >
              {isLoading ? 'Generating...' : 'Generate Backup Codes'}
            </button>

            <div className="border-t border-slate-700 pt-6">
              <h3 className="text-lg font-medium text-white mb-4">Recover with Backup Code</h3>
              <div className="flex gap-4">
                <input
                  type="text"
                  placeholder="Recovery ID"
                  className="flex-1 bg-slate-700 border border-slate-600 rounded-lg px-4 py-3 text-white"
                />
                <input
                  type="text"
                  placeholder="Backup Code"
                  className="flex-1 bg-slate-700 border border-slate-600 rounded-lg px-4 py-3 text-white"
                />
                <button className="bg-green-600 hover:bg-green-700 text-white px-6 py-3 rounded-xl font-medium transition-colors">
                  Recover
                </button>
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

        {success && (
          <div className="mt-6 p-4 bg-green-500/10 border border-green-500/30 rounded-xl">
            <p className="text-green-400">{success}</p>
          </div>
        )}
      </div>
    </div>
  );
}
