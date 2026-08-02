/**
 * TigerWallet - Hardware Wallet Settings Page
 * Complete hardware wallet management interface
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useHardwareWallet } from '../services/hardwareWalletService';
import { useTheme } from '../stores/ThemeStore';

const HardwareWalletSettingsPage: React.FC = () => {
  const { theme } = useTheme();
  const {
    wallets,
    isLoading,
    error,
    loadWallets,
    connect,
    disconnect,
    supportedChains
  } = useHardwareWallet();

  const [selectedType, setSelectedType] = useState<'ledger' | 'trezor'>('ledger');
  const [isConnecting, setIsConnecting] = useState(false);

  const handleConnect = useCallback(async () => {
    setIsConnecting(true);
    try {
      await connect(selectedType);
    } catch (err) {
      console.error('Failed to connect:', err);
    } finally {
      setIsConnecting(false);
    }
  }, [selectedType, connect]);

  const handleDisconnect = useCallback(async (walletId: string) => {
    if (confirm('Are you sure you want to disconnect this wallet?')) {
      await disconnect(walletId);
    }
  }, [disconnect]);

  const copyAddress = useCallback((address: string) => {
    navigator.clipboard.writeText(address);
  }, []);

  return (
    <div className={`hardware-wallet-page ${theme}`}>
      <div className="page-header">
        <h1>🔐 Hardware Wallet</h1>
        <p>Connect your Ledger or Trezor device for enhanced security</p>
      </div>

      {/* Error Display */}
      {error && (
        <div className="error-banner">
          {error}
        </div>
      )}

      {/* Connect New Device */}
      <div className="section">
        <h2>Connect New Device</h2>
        
        <div className="device-selection">
          <button
            className={`device-option ${selectedType === 'ledger' ? 'selected' : ''}`}
            onClick={() => setSelectedType('ledger')}
          >
            <div className="device-icon">📱</div>
            <div className="device-name">Ledger</div>
            <div className="device-models">Nano X, Nano S Plus, Nano S</div>
          </button>
          
          <button
            className={`device-option ${selectedType === 'trezor' ? 'selected' : ''}`}
            onClick={() => setSelectedType('trezor')}
          >
            <div className="device-icon">🔐</div>
            <div className="device-name">Trezor</div>
            <div className="device-models">Model T, Model One</div>
          </button>
        </div>

        <div className="connect-instructions">
          <h3>How to connect:</h3>
          <ol>
            <li>Connect your {selectedType === 'ledger' ? 'Ledger' : 'Trezor'} device to your computer</li>
            <li>Unlock your device with your PIN</li>
            <li>Open the Ethereum app on your device</li>
            <li>Click the button below to connect</li>
          </ol>
        </div>

        <button
          className="btn btn-primary btn-large"
          onClick={handleConnect}
          disabled={isConnecting}
        >
          {isConnecting ? 'Connecting...' : `Connect ${selectedType === 'ledger' ? 'Ledger' : 'Trezor'}`}
        </button>
      </div>

      {/* Connected Wallets */}
      <div className="section">
        <h2>Connected Wallets</h2>
        
        {wallets.length === 0 ? (
          <div className="empty-state">
            <p>No hardware wallets connected</p>
          </div>
        ) : (
          <div className="wallet-list">
            {wallets.map((wallet) => (
              <div key={wallet.id} className="wallet-card">
                <div className="wallet-header">
                  <div className="wallet-type">
                    {wallet.type === 'ledger' ? '📱' : '🔐'}
                    <span>{wallet.deviceName}</span>
                  </div>
                  <div className={`connection-status ${wallet.isConnected ? 'connected' : 'disconnected'}`}>
                    {wallet.isConnected ? 'Connected' : 'Disconnected'}
                  </div>
                </div>

                <div className="wallet-addresses">
                  <h4>Addresses</h4>
                  {wallet.addresses.map((addr, index) => (
                    <div key={index} className="address-row">
                      <span className="chain-name">{addr.chain}</span>
                      <span className="address">{addr.address.substring(0, 6)}...{addr.address.substring(38)}</span>
                      <button
                        className="copy-btn"
                        onClick={() => copyAddress(addr.address)}
                        title="Copy address"
                      >
                        📋
                      </button>
                    </div>
                  ))}
                </div>

                <div className="wallet-actions">
                  <button
                    className="btn btn-danger"
                    onClick={() => handleDisconnect(wallet.id)}
                  >
                    Disconnect
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Supported Chains */}
      <div className="section">
        <h2>Supported Networks</h2>
        <div className="chains-grid">
          {supportedChains.map((chain) => (
            <div key={chain} className="chain-badge">
              {chain}
            </div>
          ))}
        </div>
      </div>

      {/* Security Info */}
      <div className="section security-info">
        <h2>🔒 Security Information</h2>
        <ul>
          <li>Your private keys never leave your hardware wallet</li>
          <li>All transactions are signed on-device</li>
          <li>Hardware wallets provide the highest level of security</li>
          <li>Always verify transaction details on your device screen</li>
          <li>Never share your recovery phrase with anyone</li>
        </ul>
      </div>

      <style>{`
        .hardware-wallet-page {
          padding: 20px;
          max-width: 800px;
          margin: 0 auto;
        }

        .page-header {
          margin-bottom: 30px;
        }

        .page-header h1 {
          font-size: 28px;
          margin-bottom: 8px;
        }

        .section {
          background: var(--card-bg, #1e1e2e);
          border-radius: 12px;
          padding: 20px;
          margin-bottom: 20px;
        }

        .section h2 {
          font-size: 20px;
          margin-bottom: 16px;
        }

        .device-selection {
          display: grid;
          grid-template-columns: 1fr 1fr;
          gap: 16px;
          margin-bottom: 20px;
        }

        .device-option {
          background: var(--input-bg, #2a2a3e);
          border: 2px solid transparent;
          border-radius: 12px;
          padding: 20px;
          cursor: pointer;
          text-align: center;
          transition: all 0.2s;
        }

        .device-option:hover {
          border-color: var(--primary-color, #6c5ce7);
        }

        .device-option.selected {
          border-color: var(--primary-color, #6c5ce7);
          background: var(--primary-color, #6c5ce7)22;
        }

        .device-icon {
          font-size: 40px;
          margin-bottom: 8px;
        }

        .device-name {
          font-size: 18px;
          font-weight: 600;
          margin-bottom: 4px;
        }

        .device-models {
          font-size: 12px;
          color: var(--text-muted, #888);
        }

        .connect-instructions {
          background: var(--info-bg, #2a2a4e);
          padding: 16px;
          border-radius: 8px;
          margin-bottom: 20px;
        }

        .connect-instructions ol {
          margin: 12px 0 0 0;
          padding-left: 20px;
        }

        .connect-instructions li {
          margin-bottom: 8px;
          color: var(--text-secondary, #ccc);
        }

        .wallet-list {
          display: flex;
          flex-direction: column;
          gap: 16px;
        }

        .wallet-card {
          background: var(--input-bg, #2a2a3e);
          border-radius: 12px;
          padding: 16px;
        }

        .wallet-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 16px;
        }

        .wallet-type {
          display: flex;
          align-items: center;
          gap: 8px;
          font-size: 18px;
          font-weight: 600;
        }

        .connection-status {
          padding: 4px 12px;
          border-radius: 20px;
          font-size: 12px;
          font-weight: 600;
        }

        .connection-status.connected {
          background: #22c55e33;
          color: #22c55e;
        }

        .connection-status.disconnected {
          background: #ef444433;
          color: #ef4444;
        }

        .address-row {
          display: flex;
          align-items: center;
          gap: 12px;
          padding: 8px 0;
          border-bottom: 1px solid var(--border-color, #333);
        }

        .chain-name {
          font-weight: 600;
          min-width: 80px;
        }

        .address {
          font-family: monospace;
          color: var(--text-secondary, #ccc);
        }

        .wallet-actions {
          margin-top: 16px;
        }

        .chains-grid {
          display: flex;
          flex-wrap: wrap;
          gap: 8px;
        }

        .chain-badge {
          background: var(--input-bg, #2a2a3e);
          padding: 6px 12px;
          border-radius: 20px;
          font-size: 14px;
        }

        .security-info ul {
          padding-left: 20px;
        }

        .security-info li {
          margin-bottom: 8px;
          color: var(--text-secondary, #ccc);
        }

        .empty-state {
          text-align: center;
          padding: 40px;
          color: var(--text-muted, #888);
        }

        .error-banner {
          background: #ef444433;
          color: #ef4444;
          padding: 12px;
          border-radius: 8px;
          margin-bottom: 20px;
        }
      `}</style>
    </div>
  );
};

export default HardwareWalletSettingsPage;
