/**
 * Settings Page - Wallet settings and preferences
 */

import React, { useState } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { useAuth } from '../contexts/AuthContext';

function SettingsPage() {
  const { theme, toggleTheme } = useTheme();
  const { user, logout } = useAuth();
  const [activeTab, setActiveTab] = useState('account');

  const settings = {
    security: [
      { name: 'Biometric Login', enabled: true, description: 'Use fingerprint or face ID' },
      { name: 'Auto-Lock', enabled: true, description: 'Lock after 5 minutes of inactivity' },
      { name: 'Show Private Key', enabled: false, description: 'Reveal private key temporarily' },
    ],
    notifications: [
      { name: 'Push Notifications', enabled: true, description: 'Transaction alerts' },
      { name: 'Price Alerts', enabled: false, description: 'Get notified of price changes' },
      { name: 'Marketing Emails', enabled: false, description: 'Receive updates and news' },
    ],
    network: [
      { name: 'Testnet Mode', enabled: false, description: 'Use test networks' },
      { name: 'RPC Auto-Fallback', enabled: true, description: 'Switch RPC on failure' },
    ],
  };

  return (
    <div className="p-6 max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Settings</h1>

      <div className="flex gap-2 mb-6">
        {['account', 'security', 'notifications', 'network'].map(tab => (
          <button key={tab} onClick={() => setActiveTab(tab)} className={`px-4 py-2 rounded-lg ${activeTab === tab ? 'bg-amber-500 text-black' : theme === 'dark' ? 'bg-slate-800' : 'bg-gray-200'}`}>
            {tab.charAt(0).toUpperCase() + tab.slice(1)}
          </button>
        ))}
      </div>

      {activeTab === 'account' && (
        <div className="space-y-4">
          <div className={`card ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
            <h3 className="font-semibold mb-4">Profile</h3>
            <div className="space-y-4">
              <div><label className="label">Email</label><input type="email" defaultValue={user?.email || ''} className="input" /></div>
              <div><label className="label">Username</label><input type="text" defaultValue={user?.username || ''} className="input" /></div>
            </div>
          </div>

          <div className={`card ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
            <h3 className="font-semibold mb-4">Appearance</h3>
            <div className="flex items-center justify-between">
              <span>Dark Mode</span>
              <button onClick={toggleTheme} className={`w-12 h-6 rounded-full ${theme === 'dark' ? 'bg-amber-500' : 'bg-gray-400'}`}>
                <div className={`w-5 h-5 bg-white rounded-full transition ${theme === 'dark' ? 'translate-x-6' : 'translate-x-0.5'}`} />
              </button>
            </div>
          </div>

          <div className={`card ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
            <h3 className="font-semibold mb-4 text-red-500">Danger Zone</h3>
            <button onClick={logout} className="btn btn-danger w-full">Sign Out</button>
          </div>
        </div>
      )}

      {activeTab === 'security' && (
        <div className="space-y-4">
          {settings.security.map((s, i) => (
            <div key={i} className={`card flex justify-between items-center ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
              <div>
                <h3 className="font-semibold">{s.name}</h3>
                <p className="text-sm opacity-60">{s.description}</p>
              </div>
              <button className={`w-12 h-6 rounded-full ${s.enabled ? 'bg-amber-500' : 'bg-gray-400'}`}>
                <div className={`w-5 h-5 bg-white rounded-full transition ${s.enabled ? 'translate-x-6' : 'translate-x-0.5'}`} />
              </button>
            </div>
          ))}
        </div>
      )}

      {activeTab === 'notifications' && (
        <div className="space-y-4">
          {settings.notifications.map((s, i) => (
            <div key={i} className={`card flex justify-between items-center ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
              <div>
                <h3 className="font-semibold">{s.name}</h3>
                <p className="text-sm opacity-60">{s.description}</p>
              </div>
              <button className={`w-12 h-6 rounded-full ${s.enabled ? 'bg-amber-500' : 'bg-gray-400'}`}>
                <div className={`w-5 h-5 bg-white rounded-full transition ${s.enabled ? 'translate-x-6' : 'translate-x-0.5'}`} />
              </button>
            </div>
          ))}
        </div>
      )}

      {activeTab === 'network' && (
        <div className="space-y-4">
          {settings.network.map((s, i) => (
            <div key={i} className={`card flex justify-between items-center ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
              <div>
                <h3 className="font-semibold">{s.name}</h3>
                <p className="text-sm opacity-60">{s.description}</p>
              </div>
              <button className={`w-12 h-6 rounded-full ${s.enabled ? 'bg-amber-500' : 'bg-gray-400'}`}>
                <div className={`w-5 h-5 bg-white rounded-full transition ${s.enabled ? 'translate-x-6' : 'translate-x-0.5'}`} />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default SettingsPage;
