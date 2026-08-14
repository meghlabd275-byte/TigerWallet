// TigerWallet Admin - Settings Page
// Admin settings and preferences

import React, { useState } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { adminApi } from '../services/api';

const SettingsPage: React.FC = () => {
  const { theme, setTheme, resolvedTheme } = useTheme();
  const [saving, setSaving] = useState(false);
  const [success, setSuccess] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [config, setConfig] = useState({
    platformName: 'TigerWallet',
    supportEmail: 'support@tigerwallet.io',
    maintenanceMode: false,
    registrationEnabled: true,
    kycRequired: true,
    withdrawalMin: '10',
    withdrawalMax: '100000',
    txConfirmations: 12,
    twoFARequired: false,
  });

  const handleSave = async () => {
    try {
      setSaving(true);
      setError(null);
      setSuccess(null);
      
      await adminApi.updateConfig(config);
      setSuccess('Settings saved successfully');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save settings');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="p-6">
      {/* Page Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold" style={{ color: 'var(--text-primary)' }}>
          Settings
        </h1>
        <p style={{ color: 'var(--text-secondary)' }}>
          Configure platform settings and preferences
        </p>
      </div>

      {error && (
        <div className="alert alert-error mb-4">
          {error}
        </div>
      )}

      {success && (
        <div className="alert alert-success mb-4">
          {success}
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Appearance */}
        <div className="card">
          <div className="card-header">
            <h3 className="text-lg font-semibold">Appearance</h3>
          </div>
          <div className="card-body">
            <div className="form-group">
              <label className="form-label">Theme</label>
              <div className="flex gap-4">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    name="theme"
                    value="light"
                    checked={theme === 'light'}
                    onChange={() => setTheme('light')}
                    className="form-radio"
                  />
                  <span style={{ color: 'var(--text-primary)' }}>Light</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    name="theme"
                    value="dark"
                    checked={theme === 'dark'}
                    onChange={() => setTheme('dark')}
                    className="form-radio"
                  />
                  <span style={{ color: 'var(--text-primary)' }}>Dark</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    name="theme"
                    value="system"
                    checked={theme === 'system'}
                    onChange={() => setTheme('system')}
                    className="form-radio"
                  />
                  <span style={{ color: 'var(--text-primary)' }}>System</span>
                </label>
              </div>
            </div>
            
            <div className="mt-4 p-4 rounded" style={{ backgroundColor: 'var(--bg-secondary)' }}>
              <p className="text-sm" style={{ color: 'var(--text-secondary)' }}>
                Current theme: <strong style={{ color: 'var(--text-primary)' }}>{resolvedTheme}</strong>
              </p>
              <div className="mt-2 flex gap-2">
                <button
                  className="btn btn-sm"
                  style={{ 
                    backgroundColor: resolvedTheme === 'light' ? 'var(--color-primary)' : 'var(--bg-tertiary)',
                    color: resolvedTheme === 'light' ? 'white' : 'var(--text-primary)',
                  }}
                  onClick={() => setTheme('light')}
                >
                  Light Preview
                </button>
                <button
                  className="btn btn-sm"
                  style={{ 
                    backgroundColor: resolvedTheme === 'dark' ? 'var(--color-primary)' : 'var(--bg-tertiary)',
                    color: resolvedTheme === 'dark' ? 'white' : 'var(--text-primary)',
                  }}
                  onClick={() => setTheme('dark')}
                >
                  Dark Preview
                </button>
              </div>
            </div>
          </div>
        </div>

        {/* Platform Settings */}
        <div className="card">
          <div className="card-header">
            <h3 className="text-lg font-semibold">Platform Settings</h3>
          </div>
          <div className="card-body">
            <div className="form-group">
              <label className="form-label">Platform Name</label>
              <input
                type="text"
                className="form-input"
                value={config.platformName}
                onChange={(e) => setConfig({ ...config, platformName: e.target.value })}
              />
            </div>
            
            <div className="form-group">
              <label className="form-label">Support Email</label>
              <input
                type="email"
                className="form-input"
                value={config.supportEmail}
                onChange={(e) => setConfig({ ...config, supportEmail: e.target.value })}
              />
            </div>

            <div className="form-group">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={config.registrationEnabled}
                  onChange={(e) => setConfig({ ...config, registrationEnabled: e.target.checked })}
                  className="form-checkbox"
                />
                <span style={{ color: 'var(--text-primary)' }}>Allow User Registration</span>
              </label>
            </div>

            <div className="form-group">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={config.kycRequired}
                  onChange={(e) => setConfig({ ...config, kycRequired: e.target.checked })}
                  className="form-checkbox"
                />
                <span style={{ color: 'var(--text-primary)' }}>Require KYC</span>
              </label>
            </div>

            <div className="form-group">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={config.maintenanceMode}
                  onChange={(e) => setConfig({ ...config, maintenanceMode: e.target.checked })}
                  className="form-checkbox"
                />
                <span style={{ color: 'var(--text-primary)' }}>Maintenance Mode</span>
              </label>
            </div>
          </div>
        </div>

        {/* Transaction Limits */}
        <div className="card">
          <div className="card-header">
            <h3 className="text-lg font-semibold">Transaction Limits</h3>
          </div>
          <div className="card-body">
            <div className="grid grid-cols-2 gap-4">
              <div className="form-group">
                <label className="form-label">Minimum Withdrawal</label>
                <input
                  type="text"
                  className="form-input"
                  value={config.withdrawalMin}
                  onChange={(e) => setConfig({ ...config, withdrawalMin: e.target.value })}
                />
              </div>
              
              <div className="form-group">
                <label className="form-label">Maximum Withdrawal</label>
                <input
                  type="text"
                  className="form-input"
                  value={config.withdrawalMax}
                  onChange={(e) => setConfig({ ...config, withdrawalMax: e.target.value })}
                />
              </div>
            </div>

            <div className="form-group">
              <label className="form-label">Required Confirmations</label>
              <input
                type="number"
                className="form-input"
                value={config.txConfirmations}
                onChange={(e) => setConfig({ ...config, txConfirmations: parseInt(e.target.value) })}
              />
              <p className="text-xs mt-1" style={{ color: 'var(--text-tertiary)' }}>
                Number of block confirmations required for deposits
              </p>
            </div>
          </div>
        </div>

        {/* Security Settings */}
        <div className="card">
          <div className="card-header">
            <h3 className="text-lg font-semibold">Security</h3>
          </div>
          <div className="card-body">
            <div className="space-y-3">
              <div className="flex justify-between items-center py-2 border-b" style={{ borderColor: 'var(--border-primary)' }}>
                <div>
                  <p style={{ color: 'var(--text-primary)' }}>Two-Factor Authentication</p>
                  <p className="text-sm" style={{ color: 'var(--text-tertiary)' }}>Require 2FA for all admins</p>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input type="checkbox" className="sr-only peer" defaultChecked={config.twoFARequired} onChange={(e) => setConfig({ ...config, twoFARequired: e.target.checked })} />
                  <div style={{ width: 44, height: 24, background: 'var(--bg-tertiary)', borderRadius: 999, position: 'relative', transition: 'background .2s' }} className="peer-checked:bg-[var(--accent-primary)]">
                    <span style={{ position: 'absolute', top: 2, left: 2, width: 20, height: 20, background: '#fff', borderRadius: '50%', transition: 'transform .2s' }} className="peer-checked:after:translate-x-5 peer-checked:after:content-['']"></span>
                  </div>
                </label>
              </div>

              <div className="flex justify-between items-center py-2 border-b" style={{ borderColor: 'var(--border-primary)' }}>
                <div>
                  <p style={{ color: 'var(--text-primary)' }}>IP Whitelist</p>
                  <p className="text-sm" style={{ color: 'var(--text-tertiary)' }}>Restrict access by IP</p>
                </div>
                <button className="btn btn-sm btn-outline">Configure</button>
              </div>

              <div className="flex justify-between items-center py-2">
                <div>
                  <p style={{ color: 'var(--text-primary)' }}>API Rate Limiting</p>
                  <p className="text-sm" style={{ color: 'var(--text-tertiary)' }}>Protect against abuse</p>
                </div>
                <button className="btn btn-sm btn-outline">Configure</button>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Save Button */}
      <div className="mt-6 flex justify-end">
        <button
          className="btn btn-primary btn-lg"
          onClick={handleSave}
          disabled={saving}
        >
          {saving ? (
            <>
              <span className="spinner mr-2"></span>
              Saving...
            </>
          ) : (
            'Save Changes'
          )}
        </button>
      </div>
    </div>
  );
};

export default SettingsPage;
