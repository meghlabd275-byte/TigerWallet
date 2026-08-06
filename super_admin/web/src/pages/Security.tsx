/**
 * TigerWallet Super Admin - Security Page
 * Complete security management with 2FA, IP Whitelist, Rate Limiting
 */

import React, { useState, useEffect } from 'react';
import { useTheme } from '../context/ThemeContext';
import superAdminApi from '../services/api';

interface IPWhitelistItem {
  id: string;
  ip: string;
  description: string;
  created_at: string;
}

interface RateLimitItem {
  id: string;
  endpoint: string;
  requests_per_minute: number;
  burst: number;
}

interface TwoFASettings {
  enabled: boolean;
  method: string;
  backup_codes_remaining: number;
}

export default function Security() {
  const { resolvedTheme } = useTheme();
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('2fa');
  
  // 2FA State
  const [twoFAEnabled, setTwoFAEnabled] = useState(false);
  const [setupStep, setSetupStep] = useState(1);
  const [secret, setSecret] = useState('');
  const [qrCode, setQrCode] = useState('');
  const [verifyCode, setVerifyCode] = useState('');
  const [backupCodes, setBackupCodes] = useState<string[]>([]);
  const [twoFALoading, setTwoFALoading] = useState(false);
  
  // IP Whitelist State
  const [ipWhitelist, setIPWhitelist] = useState<IPWhitelistItem[]>([]);
  const [newIP, setNewIP] = useState('');
  const [newIPDesc, setNewIPDesc] = useState('');
  const [ipLoading, setIPLoading] = useState(false);
  
  // Rate Limiting State
  const [rateLimits, setRateLimits] = useState<RateLimitItem[]>([]);
  const [newEndpoint, setNewEndpoint] = useState('');
  const [newRate, setNewRate] = useState(100);
  const [rateLoading, setRateLoading] = useState(false);
  
  // 2FA Setup
  const handleSetup2FA = async () => {
    setTwoFALoading(true);
    try {
      const result = await superAdminApi.setup2FA('current_user');
      setSecret(result.secret);
      setQrCode(result.qr_code_url);
      setSetupStep(2);
    } catch (error) {
      console.error('Failed to setup 2FA:', error);
      alert('Failed to setup 2FA. Please try again.');
    }
    setTwoFALoading(false);
  };
  
  const handleVerify2FA = async () => {
    setTwoFALoading(true);
    try {
      const result = await superAdminApi.verify2FASetup('current_user', secret, verifyCode);
      setBackupCodes(result.backup_codes);
      setTwoFAEnabled(true);
      setSetupStep(3);
    } catch (error) {
      console.error('Failed to verify 2FA:', error);
      alert('Invalid code. Please try again.');
    }
    setTwoFALoading(false);
  };
  
  const handleDisable2FA = async () => {
    const code = prompt('Enter your 2FA code to disable:');
    if (!code) return;
    
    setTwoFALoading(true);
    try {
      await superAdminApi.disable2FA('current_user', code);
      setTwoFAEnabled(false);
      setSetupStep(1);
      setBackupCodes([]);
      alert('2FA has been disabled.');
    } catch (error) {
      console.error('Failed to disable 2FA:', error);
      alert('Failed to disable 2FA. Invalid code?');
    }
    setTwoFALoading(false);
  };
  
  // IP Whitelist
  const loadIPWhitelist = async () => {
    try {
      const result = await superAdminApi.getIPWhitelist();
      setIPWhitelist(result.ips || []);
    } catch (error) {
      console.error('Failed to load IP whitelist:', error);
    }
  };
  
  const handleAddIP = async () => {
    if (!newIP) return;
    
    setIPLoading(true);
    try {
      await superAdminApi.addToIPWhitelist(newIP, newIPDesc);
      setNewIP('');
      setNewIPDesc('');
      await loadIPWhitelist();
    } catch (error) {
      console.error('Failed to add IP:', error);
      alert('Failed to add IP to whitelist.');
    }
    setIPLoading(false);
  };
  
  const handleRemoveIP = async (ip: string) => {
    if (!confirm(`Remove ${ip} from whitelist?`)) return;
    
    try {
      await superAdminApi.removeFromIPWhitelist(ip);
      await loadIPWhitelist();
    } catch (error) {
      console.error('Failed to remove IP:', error);
    }
  };
  
  // Rate Limiting
  const loadRateLimits = async () => {
    try {
      const result = await superAdminApi.getRateLimits();
      setRateLimits(result.limits || []);
    } catch (error) {
      console.error('Failed to load rate limits:', error);
    }
  };
  
  const handleAddRateLimit = async () => {
    if (!newEndpoint) return;
    
    setRateLoading(true);
    try {
      await superAdminApi.setRateLimit({
        endpoint: newEndpoint,
        requests_per_minute: newRate,
      });
      setNewEndpoint('');
      setNewRate(100);
      await loadRateLimits();
    } catch (error) {
      console.error('Failed to add rate limit:', error);
      alert('Failed to add rate limit.');
    }
    setRateLoading(false);
  };
  
  useEffect(() => {
    const loadData = async () => {
      setLoading(false);
      await Promise.all([
        loadIPWhitelist(),
        loadRateLimits(),
      ]);
    };
    
    if (!loading) {
      loadData();
    }
  }, [loading]);
  
  const tabs = [
    { id: '2fa', label: 'Two-Factor Authentication', icon: '🔐' },
    { id: 'ip-whitelist', label: 'IP Whitelist', icon: '🌐' },
    { id: 'rate-limiting', label: 'Rate Limiting', icon: '⚡' },
    { id: 'sessions', label: 'Active Sessions', icon: '💻' },
  ];
  
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6" style={{ color: 'var(--text-primary)' }}>
        Security Settings
      </h1>
      
      {/* Tabs */}
      <div className="flex gap-2 mb-6 overflow-x-auto">
        {tabs.map(tab => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`px-4 py-2 rounded-lg font-medium transition-colors whitespace-nowrap ${
              activeTab === tab.id
                ? 'bg-blue-600 text-white'
                : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600'
            }`}
            style={{
              backgroundColor: activeTab === tab.id ? 'var(--accent-primary)' : undefined,
              color: activeTab === tab.id ? '#fff' : 'var(--text-primary)',
            }}
          >
            <span className="mr-2">{tab.icon}</span>
            {tab.label}
          </button>
        ))}
      </div>
      
      {/* 2FA Tab */}
      {activeTab === '2fa' && (
        <div className="card">
          <div className="card-body">
            <div className="flex items-center justify-between mb-6">
              <div>
                <h2 className="text-xl font-semibold" style={{ color: 'var(--text-primary)' }}>
                  Two-Factor Authentication
                </h2>
                <p style={{ color: 'var(--text-secondary)' }}>
                  Add an extra layer of security to your account
                </p>
              </div>
              <div className={`px-3 py-1 rounded-full text-sm font-medium ${
                twoFAEnabled 
                  ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                  : 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200'
              }`}>
                {twoFAEnabled ? 'Enabled' : 'Disabled'}
              </div>
            </div>
            
            {setupStep === 1 && (
              <div className="text-center py-8">
                <div className="mb-4 text-6xl">🔐</div>
                <p className="mb-4" style={{ color: 'var(--text-secondary)' }}>
                  {twoFAEnabled 
                    ? '2FA is currently enabled on your account.'
                    : 'Enable two-factor authentication to secure your account.'}
                </p>
                {twoFAEnabled ? (
                  <button
                    onClick={handleDisable2FA}
                    disabled={twoFALoading}
                    className="px-6 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50"
                  >
                    {twoFALoading ? 'Processing...' : 'Disable 2FA'}
                  </button>
                ) : (
                  <button
                    onClick={handleSetup2FA}
                    disabled={twoFALoading}
                    className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
                  >
                    {twoFALoading ? 'Loading...' : 'Enable 2FA'}
                  </button>
                )}
              </div>
            )}
            
            {setupStep === 2 && (
              <div className="py-4">
                <h3 className="text-lg font-semibold mb-4" style={{ color: 'var(--text-primary)' }}>
                  Step 1: Scan QR Code
                </h3>
                <div className="flex flex-col md:flex-row gap-6 mb-6">
                  <div className="bg-white p-4 rounded-lg border">
                    <img src={qrCode} alt="QR Code" className="w-48 h-48" />
                  </div>
                  <div className="flex-1">
                    <p className="mb-2" style={{ color: 'var(--text-secondary)' }}>
                      Scan this QR code with your authenticator app (Google Authenticator, Authy, etc.)
                    </p>
                    <p className="text-sm" style={{ color: 'var(--text-tertiary)' }}>
                      Or enter this code manually: <code className="bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded">{secret}</code>
                    </p>
                  </div>
                </div>
                
                <h3 className="text-lg font-semibold mb-4" style={{ color: 'var(--text-primary)' }}>
                  Step 2: Verify
                </h3>
                <div className="flex gap-4">
                  <input
                    type="text"
                    value={verifyCode}
                    onChange={e => setVerifyCode(e.target.value)}
                    placeholder="Enter 6-digit code"
                    className="flex-1 px-4 py-2 border rounded-lg"
                    style={{ 
                      backgroundColor: 'var(--bg-secondary)',
                      borderColor: 'var(--border-primary)',
                      color: 'var(--text-primary)',
                    }}
                    maxLength={6}
                  />
                  <button
                    onClick={handleVerify2FA}
                    disabled={twoFALoading || verifyCode.length < 6}
                    className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
                  >
                    {twoFALoading ? 'Verifying...' : 'Verify & Enable'}
                  </button>
                </div>
              </div>
            )}
            
            {setupStep === 3 && backupCodes.length > 0 && (
              <div className="py-4">
                <div className="bg-yellow-50 dark:bg-yellow-900 border border-yellow-200 dark:border-yellow-700 rounded-lg p-4 mb-4">
                  <h3 className="font-semibold text-yellow-800 dark:text-yellow-200 mb-2">
                    ⚠️ Save Your Backup Codes
                  </h3>
                  <p className="text-sm text-yellow-700 dark:text-yellow-300">
                    Save these codes in a secure place. You can use them to access your account if you lose your device.
                  </p>
                </div>
                
                <div className="grid grid-cols-2 gap-2 mb-4 p-4 bg-gray-50 dark:bg-gray-900 rounded-lg">
                  {backupCodes.map((code, index) => (
                    <code key={index} className="font-mono text-lg">{code}</code>
                  ))}
                </div>
                
                <button
                  onClick={() => { setSetupStep(1); }}
                  className="px-6 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700"
                >
                  I've Saved My Codes
                </button>
              </div>
            )}
          </div>
        </div>
      )}
      
      {/* IP Whitelist Tab */}
      {activeTab === 'ip-whitelist' && (
        <div className="card">
          <div className="card-body">
            <h2 className="text-xl font-semibold mb-4" style={{ color: 'var(--text-primary)' }}>
              IP Whitelist
            </h2>
            <p className="mb-4" style={{ color: 'var(--text-secondary)' }}>
              Restrict access to specific IP addresses
            </p>
            
            <div className="flex gap-2 mb-6">
              <input
                type="text"
                value={newIP}
                onChange={e => setNewIP(e.target.value)}
                placeholder="Enter IP address (e.g., 192.168.1.1)"
                className="flex-1 px-4 py-2 border rounded-lg"
                style={{ 
                  backgroundColor: 'var(--bg-secondary)',
                  borderColor: 'var(--border-primary)',
                  color: 'var(--text-primary)',
                }}
              />
              <input
                type="text"
                value={newIPDesc}
                onChange={e => setNewIPDesc(e.target.value)}
                placeholder="Description (optional)"
                className="flex-1 px-4 py-2 border rounded-lg"
                style={{ 
                  backgroundColor: 'var(--bg-secondary)',
                  borderColor: 'var(--border-primary)',
                  color: 'var(--text-primary)',
                }}
              />
              <button
                onClick={handleAddIP}
                disabled={ipLoading || !newIP}
                className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
              >
                {ipLoading ? 'Adding...' : 'Add IP'}
              </button>
            </div>
            
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b" style={{ borderColor: 'var(--border-primary)' }}>
                    <th className="text-left py-3 px-4" style={{ color: 'var(--text-primary)' }}>IP Address</th>
                    <th className="text-left py-3 px-4" style={{ color: 'var(--text-primary)' }}>Description</th>
                    <th className="text-left py-3 px-4" style={{ color: 'var(--text-primary)' }}>Added</th>
                    <th className="text-right py-3 px-4" style={{ color: 'var(--text-primary)' }}>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {ipWhitelist.map(item => (
                    <tr key={item.id} className="border-b" style={{ borderColor: 'var(--border-primary)' }}>
                      <td className="py-3 px-4 font-mono" style={{ color: 'var(--text-primary)' }}>{item.ip}</td>
                      <td className="py-3 px-4" style={{ color: 'var(--text-secondary)' }}>{item.description || '-'}</td>
                      <td className="py-3 px-4" style={{ color: 'var(--text-secondary)' }}>
                        {new Date(item.created_at).toLocaleDateString()}
                      </td>
                      <td className="py-3 px-4 text-right">
                        <button
                          onClick={() => handleRemoveIP(item.ip)}
                          className="text-red-600 hover:text-red-800"
                        >
                          Remove
                        </button>
                      </td>
                    </tr>
                  ))}
                  {ipWhitelist.length === 0 && (
                    <tr>
                      <td colSpan={4} className="py-8 text-center" style={{ color: 'var(--text-secondary)' }}>
                        No IP addresses in whitelist
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}
      
      {/* Rate Limiting Tab */}
      {activeTab === 'rate-limiting' && (
        <div className="card">
          <div className="card-body">
            <h2 className="text-xl font-semibold mb-4" style={{ color: 'var(--text-primary)' }}>
              Rate Limiting
            </h2>
            <p className="mb-4" style={{ color: 'var(--text-secondary)' }}>
              Configure rate limits for API endpoints
            </p>
            
            <div className="flex gap-2 mb-6">
              <input
                type="text"
                value={newEndpoint}
                onChange={e => setNewEndpoint(e.target.value)}
                placeholder="Endpoint (e.g., /api/v1/withdraw)"
                className="flex-1 px-4 py-2 border rounded-lg"
                style={{ 
                  backgroundColor: 'var(--bg-secondary)',
                  borderColor: 'var(--border-primary)',
                  color: 'var(--text-primary)',
                }}
              />
              <input
                type="number"
                value={newRate}
                onChange={e => setNewRate(parseInt(e.target.value))}
                placeholder="Requests/minute"
                className="w-40 px-4 py-2 border rounded-lg"
                style={{ 
                  backgroundColor: 'var(--bg-secondary)',
                  borderColor: 'var(--border-primary)',
                  color: 'var(--text-primary)',
                }}
              />
              <button
                onClick={handleAddRateLimit}
                disabled={rateLoading || !newEndpoint}
                className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
              >
                {rateLoading ? 'Adding...' : 'Add'}
              </button>
            </div>
            
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b" style={{ borderColor: 'var(--border-primary)' }}>
                    <th className="text-left py-3 px-4" style={{ color: 'var(--text-primary)' }}>Endpoint</th>
                    <th className="text-left py-3 px-4" style={{ color: 'var(--text-primary)' }}>Requests/min</th>
                    <th className="text-left py-3 px-4" style={{ color: 'var(--text-primary)' }}>Burst</th>
                    <th className="text-right py-3 px-4" style={{ color: 'var(--text-primary)' }}>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {rateLimits.map(item => (
                    <tr key={item.id} className="border-b" style={{ borderColor: 'var(--border-primary)' }}>
                      <td className="py-3 px-4 font-mono" style={{ color: 'var(--text-primary)' }}>{item.endpoint}</td>
                      <td className="py-3 px-4" style={{ color: 'var(--text-secondary)' }}>{item.requests_per_minute}</td>
                      <td className="py-3 px-4" style={{ color: 'var(--text-secondary)' }}>{item.burst}</td>
                      <td className="py-3 px-4 text-right">
                        <button className="text-red-600 hover:text-red-800">Remove</button>
                      </td>
                    </tr>
                  ))}
                  {rateLimits.length === 0 && (
                    <tr>
                      <td colSpan={4} className="py-8 text-center" style={{ color: 'var(--text-secondary)' }}>
                        No rate limits configured
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}
      
      {/* Sessions Tab */}
      {activeTab === 'sessions' && (
        <div className="card">
          <div className="card-body">
            <h2 className="text-xl font-semibold mb-4" style={{ color: 'var(--text-primary)' }}>
              Active Sessions
            </h2>
            <p className="mb-4" style={{ color: 'var(--text-secondary)' }}>
              Manage your active sessions
            </p>
            
            <div className="text-center py-8" style={{ color: 'var(--text-secondary)' }}>
              <div className="text-4xl mb-4">💻</div>
              <p>Loading sessions...</p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
