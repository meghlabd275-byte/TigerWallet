// Receive Page
import React, { useState } from 'react';
import { QRDisplay } from '../../../frontend/shared/components/QRScanner';
import './ReceivePage.css';

const ReceivePage: React.FC = () => {
  const [selectedChain, setSelectedChain] = useState('Ethereum');
  const [walletAddress] = useState('0x742d35Cc6634C0532925a3b844Bc9e7595f1234');

  const chains = [
    { id: 'ethereum', name: 'Ethereum', symbol: 'ETH', icon: '🔷' },
    { id: 'bsc', name: 'BNB Chain', symbol: 'BNB', icon: '🟡' },
    { id: 'polygon', name: 'Polygon', symbol: 'MATIC', icon: '🟣' },
    { id: 'arbitrum', name: 'Arbitrum', symbol: 'ETH', icon: '🔵' },
    { id: 'solana', name: 'Solana', symbol: 'SOL', icon: '☀️' },
  ];

  const copyAddress = () => {
    navigator.clipboard.writeText(walletAddress);
    alert('Address copied!');
  };

  // Generate QR code URL
  const qrCodeUrl = `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(walletAddress)}`;

  return (
    <div className="receive-page">
      <div className="page-header">
        <h1>Receive</h1>
      </div>

      {/* Chain Selector */}
      <div className="chain-selector">
        <label>Select Network</label>
        <div className="chains-grid">
          {chains.map(chain => (
            <button
              key={chain.id}
              className={`chain-option ${selectedChain === chain.name ? 'selected' : ''}`}
              onClick={() => setSelectedChain(chain.name)}
            >
              <span className="chain-icon">{chain.icon}</span>
              <span className="chain-name">{chain.name}</span>
            </button>
          ))}
        </div>
      </div>

      {/* QR Code */}
      <div className="qr-section">
        <div className="qr-code">
          <img src={qrCodeUrl} alt="QR Code" style={{ borderRadius: '12px' }} />
        </div>
        <p className="qr-hint">Scan to receive {chains.find(c => c.name === selectedChain)?.symbol}</p>
      </div>

      {/* Address */}
      <div className="address-section">
        <label>Your Address</label>
        <div className="address-display">
          <span className="address">{walletAddress}</span>
          <button className="copy-btn" onClick={copyAddress}>📋 Copy</button>
        </div>
      </div>

      {/* Warning */}
      <div className="warning-box">
        ⚠️ Only send {chains.find(c => c.name === selectedChain)?.symbol} to this address
      </div>
    </div>
  );
};

export default ReceivePage;
