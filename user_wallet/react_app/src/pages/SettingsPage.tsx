// Settings Page
import React from 'react';
import { useTheme } from '../stores/ThemeStore';
import './SettingsPage.css';

const SettingsPage: React.FC = () => {
  const { theme, toggleTheme } = useTheme();

  const settingsSections = [
    {
      title: 'Security',
      items: [
        { icon: '🔐', label: 'Change Password', description: 'Update your wallet password' },
        { icon: '📝', label: 'Recovery Phrase', description: 'View your recovery phrase' },
        { icon: '👆', label: 'Biometric Login', description: 'Use fingerprint or face ID' },
        { icon: '🔑', label: 'Auto-Lock', description: 'Lock wallet after inactivity' },
      ],
    },
    {
      title: 'Network',
      items: [
        { icon: '⛓️', label: 'Networks', description: 'Manage blockchain networks' },
        { icon: '🔌', label: 'RPC Nodes', description: 'Configure RPC endpoints' },
      ],
    },
    {
      title: 'Preferences',
      items: [
        { icon: '🌙', label: 'Dark Mode', description: 'Toggle dark/light theme', toggle: true },
        { icon: '🔔', label: 'Notifications', description: 'Transaction alerts' },
        { icon: '🌐', label: 'Language', description: 'English' },
        { icon: '💱', label: 'Currency', description: 'USD' },
      ],
    },
    {
      title: 'Advanced',
      items: [
        { icon: '🔧', label: 'Developer Options', description: 'Debug and test features' },
        { icon: '📦', label: 'Clear Cache', description: 'Free up storage space' },
        { icon: '🗑️', label: 'Reset Wallet', description: 'Delete wallet data' },
      ],
    },
  ];

  return (
    <div className="settings-page">
      <div className="page-header">
        <h1>Settings</h1>
      </div>

      {settingsSections.map((section, index) => (
        <div key={index} className="settings-section">
          <h2>{section.title}</h2>
          <div className="settings-list">
            {section.items.map((item, idx) => (
              <div key={idx} className="settings-item">
                <div className="item-icon">{item.icon}</div>
                <div className="item-info">
                  <span className="item-label">{item.label}</span>
                  <span className="item-description">{item.description}</span>
                </div>
                {item.toggle ? (
                  <label className="toggle">
                    <input
                      type="checkbox"
                      checked={item.label === 'Dark Mode' ? theme === 'dark' : false}
                      onChange={item.label === 'Dark Mode' ? toggleTheme : () => {}}
                    />
                    <span className="toggle-slider"></span>
                  </label>
                ) : (
                  <span className="item-arrow">›</span>
                )}
              </div>
            ))}
          </div>
        </div>
      ))}

      {/* App Info */}
      <div className="app-info">
        <div className="app-logo">🐯</div>
        <div className="app-details">
          <span className="app-name">TigerWallet</span>
          <span className="app-version">Version 1.0.0</span>
        </div>
      </div>
    </div>
  );
};

export default SettingsPage;
