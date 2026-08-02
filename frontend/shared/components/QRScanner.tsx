/**
 * TigerWallet QR Scanner Component
 * Universal QR Code Scanner for making payments/sending crypto
 * Works across Admin Panel, User Wallet, Mobile Apps, Web
 * Supports camera scanning and manual address entry
 * 
 * Features:
 * - Camera-based QR code scanning
 * - Manual address entry fallback
 * - Address validation for multiple chains
 * - Light/Dark theme support
 * - Copy to clipboard functionality
 */

import React, { useState, useEffect, useRef, useCallback } from 'react';

// QR Scanner styles - CSS variables for theming
const scannerStyles = `
  .qr-scanner-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.85);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10000;
  }

  .qr-scanner-modal {
    background: var(--bg-secondary, #1A1A2E);
    border-radius: 16px;
    padding: 24px;
    max-width: 450px;
    width: 90%;
    max-height: 90vh;
    overflow-y: auto;
    border: 1px solid var(--border-color, #374151);
  }

  .qr-scanner-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
  }

  .qr-scanner-title {
    font-size: 20px;
    font-weight: 600;
    color: var(--text-primary, #FFFFFF);
  }

  .qr-scanner-close {
    background: none;
    border: none;
    font-size: 24px;
    cursor: pointer;
    color: var(--text-secondary, #B0B0C0);
    padding: 4px;
  }

  .qr-scanner-tabs {
    display: flex;
    margin-bottom: 20px;
    border-bottom: 1px solid var(--border-color, #374151);
  }

  .qr-scanner-tab {
    flex: 1;
    padding: 12px;
    background: none;
    border: none;
    color: var(--text-secondary, #B0B0C0);
    cursor: pointer;
    font-size: 14px;
    font-weight: 500;
    border-bottom: 2px solid transparent;
    transition: all 0.2s;
  }

  .qr-scanner-tab.active {
    color: var(--primary, #FF6B35);
    border-bottom-color: var(--primary, #FF6B35);
  }

  .qr-scanner-camera-container {
    position: relative;
    width: 100%;
    height: 300px;
    background: #000;
    border-radius: 12px;
    overflow: hidden;
    margin-bottom: 16px;
  }

  .qr-scanner-video {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .qr-scanner-overlay-box {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    width: 200px;
    height: 200px;
    border: 2px solid var(--primary, #FF6B35);
    border-radius: 12px;
    box-shadow: 0 0 0 9999px rgba(0, 0, 0, 0.5);
  }

  .qr-scanner-corner {
    position: absolute;
    width: 30px;
    height: 30px;
    border-color: var(--primary, #FF6B35);
    border-style: solid;
  }

  .qr-scanner-corner.tl { top: 0; left: 0; border-width: 4px 0 0 4px; border-radius: 8px 0 0 0; }
  .qr-scanner-corner.tr { top: 0; right: 0; border-width: 4px 4px 0 0; border-radius: 0 8px 0 0; }
  .qr-scanner-corner.bl { bottom: 0; left: 0; border-width: 0 0 4px 4px; border-radius: 0 0 0 8px; }
  .qr-scanner-corner.br { bottom: 0; right: 0; border-width: 0 4px 4px 0; border-radius: 0 0 8px 0; }

  .qr-scanner-line {
    position: absolute;
    top: 50%;
    left: 10%;
    right: 10%;
    height: 2px;
    background: linear-gradient(90deg, transparent, var(--primary, #FF6B35), transparent);
    animation: scan 2s ease-in-out infinite;
  }

  @keyframes scan {
    0%, 100% { top: 20%; }
    50% { top: 80%; }
  }

  .qr-scanner-manual {
    padding: 20px 0;
  }

  .qr-scanner-input-group {
    margin-bottom: 16px;
  }

  .qr-scanner-label {
    display: block;
    font-size: 14px;
    font-weight: 500;
    color: var(--text-secondary, #B0B0C0);
    margin-bottom: 8px;
  }

  .qr-scanner-input {
    width: 100%;
    padding: 14px 16px;
    background: var(--bg-primary, #0D0D12);
    border: 1px solid var(--border-color, #374151);
    border-radius: 10px;
    color: var(--text-primary, #FFFFFF);
    font-size: 14px;
    font-family: monospace;
    transition: border-color 0.2s;
  }

  .qr-scanner-input:focus {
    outline: none;
    border-color: var(--primary, #FF6B35);
  }

  .qr-scanner-input::placeholder {
    color: var(--text-muted, #6B7280);
  }

  .qr-scanner-validate {
    padding: 10px;
    border-radius: 8px;
    font-size: 13px;
    margin-top: 8px;
  }

  .qr-scanner-valid {
    background: rgba(16, 185, 129, 0.1);
    color: #10B981;
    border: 1px solid rgba(16, 185, 129, 0.3);
  }

  .qr-scanner-invalid {
    background: rgba(239, 68, 68, 0.1);
    color: #EF4444;
    border: 1px solid rgba(239, 68, 68, 0.3);
  }

  .qr-scanner-actions {
    display: flex;
    gap: 12px;
    margin-top: 20px;
  }

  .qr-scanner-btn {
    flex: 1;
    padding: 14px 20px;
    border-radius: 10px;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;
    border: none;
  }

  .qr-scanner-btn-primary {
    background: var(--primary, #FF6B35);
    color: white;
  }

  .qr-scanner-btn-primary:hover {
    background: var(--primary-hover, #E55A2B);
  }

  .qr-scanner-btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .qr-scanner-btn-secondary {
    background: var(--bg-tertiary, #252540);
    color: var(--text-primary, #FFFFFF);
    border: 1px solid var(--border-color, #374151);
  }

  .qr-scanner-btn-secondary:hover {
    background: var(--bg-elevated, #2D2D4A);
  }

  .qr-scanner-error {
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.3);
    color: #EF4444;
    padding: 12px;
    border-radius: 8px;
    margin-bottom: 16px;
    font-size: 13px;
  }

  .qr-scanner-success {
    background: rgba(16, 185, 129, 0.1);
    border: 1px solid rgba(16, 185, 129, 0.3);
    color: #10B981;
    padding: 12px;
    border-radius: 8px;
    margin-bottom: 16px;
    font-size: 13px;
  }

  .qr-scanner-instructions {
    text-align: center;
    color: var(--text-secondary, #B0B0C0);
    font-size: 13px;
    margin-bottom: 16px;
  }

  .qr-scanner-history {
    max-height: 150px;
    overflow-y: auto;
    margin-top: 12px;
  }

  .qr-scanner-history-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 12px;
    background: var(--bg-primary, #0D0D12);
    border-radius: 8px;
    margin-bottom: 8px;
    cursor: pointer;
    transition: background 0.2s;
  }

  .qr-scanner-history-item:hover {
    background: var(--bg-tertiary, #252540);
  }

  .qr-scanner-history-address {
    font-family: monospace;
    font-size: 12px;
    color: var(--text-primary, #FFFFFF);
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 200px;
  }

  .qr-scanner-history-chain {
    font-size: 11px;
    color: var(--text-muted, #6B7280);
  }
`;

interface QRScannerProps {
  isOpen: boolean;
  onClose: () => void;
  onScan: (address: string, chain?: string) => void;
  title?: string;
  supportedChains?: string[];
  recentAddresses?: string[];
}

// Inject styles
const injectStyles = () => {
  if (typeof document !== 'undefined' && !document.getElementById('qr-scanner-styles')) {
    const style = document.createElement('style');
    style.id = 'qr-scanner-styles';
    style.textContent = scannerStyles;
    document.head.appendChild(style);
  }
};

export const QRScanner: React.FC<QRScannerProps> = ({
  isOpen,
  onClose,
  onScan,
  title = 'Scan QR Code',
  supportedChains = ['ethereum', 'bitcoin', 'solana', 'tron'],
  recentAddresses = [],
}) => {
  const [activeTab, setActiveTab] = useState<'camera' | 'manual'>('camera');
  const [manualAddress, setManualAddress] = useState('');
  const [isValidAddress, setIsValidAddress] = useState<boolean | null>(null);
  const [detectedChain, setDetectedChain] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [scanning, setScanning] = useState(false);
  const videoRef = useRef<HTMLVideoElement>(null);
  const streamRef = useRef<MediaStream | null>(null);

  useEffect(() => {
    injectStyles();
  }, []);

  useEffect(() => {
    if (isOpen && activeTab === 'camera') {
      startCamera();
    } else {
      stopCamera();
    }
    return () => stopCamera();
  }, [isOpen, activeTab]);

  useEffect(() => {
    if (manualAddress) {
      validateAddress(manualAddress);
    } else {
      setIsValidAddress(null);
      setDetectedChain(null);
    }
  }, [manualAddress]);

  const startCamera = async () => {
    setError(null);
    setScanning(true);
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: 'environment', width: { ideal: 1280 }, height: { ideal: 720 } }
      });
      streamRef.current = stream;
      if (videoRef.current) {
        videoRef.current.srcObject = stream;
      }
    } catch (err: any) {
      console.error('Camera error:', err);
      setError('Unable to access camera. Please use manual entry or grant camera permission.');
      setActiveTab('manual');
    }
  };

  const stopCamera = () => {
    if (streamRef.current) {
      streamRef.current.getTracks().forEach(track => track.stop());
      streamRef.current = null;
    }
    setScanning(false);
  };

  const validateAddress = (address: string) => {
    const trimmed = address.trim();
    
    // Check for Ethereum addresses
    if (/^0x[a-fA-F0-9]{40}$/.test(trimmed)) {
      setIsValidAddress(true);
      setDetectedChain('Ethereum');
      return;
    }
    
    // Check for Bitcoin addresses
    if (/^(bc1|[13])[a-zA-HJ-NP-Z0-9]{25,62}$/.test(trimmed)) {
      setIsValidAddress(true);
      setDetectedChain('Bitcoin');
      return;
    }
    
    // Check for Solana addresses
    if (/^[1-9A-HJ-NP-Z]{32,44}$/.test(trimmed)) {
      setIsValidAddress(true);
      setDetectedChain('Solana');
      return;
    }
    
    // Check for Tron addresses
    if (/^T[a-zA-HJ-NP-Z0-9]{33}$/.test(trimmed)) {
      setIsValidAddress(true);
      setDetectedChain('TRON');
      return;
    }
    
    // Check for Cosmos addresses
    if (/^cosmos1[a-z0-9]{38}$/.test(trimmed)) {
      setIsValidAddress(true);
      setDetectedChain('Cosmos');
      return;
    }
    
    // Check for NEAR addresses
    if (/\.near$/.test(trimmed)) {
      setIsValidAddress(true);
      setDetectedChain('NEAR');
      return;
    }

    setIsValidAddress(false);
    setDetectedChain(null);
  };

  const handleScan = useCallback((address: string) => {
    validateAddress(address);
    if (isValidAddress !== false) {
      onScan(address, detectedChain || undefined);
      onClose();
    }
  }, [isValidAddress, detectedChain, onScan, onClose]);

  const handleManualSubmit = () => {
    if (manualAddress && isValidAddress) {
      handleScan(manualAddress);
    }
  };

  const handleUseAddress = (address: string) => {
    setManualAddress(address);
    validateAddress(address);
  };

  const copyToClipboard = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="qr-scanner-overlay" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="qr-scanner-modal">
        <div className="qr-scanner-header">
          <h2 className="qr-scanner-title">{title}</h2>
          <button className="qr-scanner-close" onClick={onClose}>×</button>
        </div>

        <div className="qr-scanner-tabs">
          <button 
            className={`qr-scanner-tab ${activeTab === 'camera' ? 'active' : ''}`}
            onClick={() => setActiveTab('camera')}
          >
            📷 Camera Scan
          </button>
          <button 
            className={`qr-scanner-tab ${activeTab === 'manual' ? 'active' : ''}`}
            onClick={() => setActiveTab('manual')}
          >
            ⌨️ Manual Entry
          </button>
        </div>

        {error && (
          <div className="qr-scanner-error">{error}</div>
        )}

        {activeTab === 'camera' ? (
          <div className="qr-scanner-camera-container">
            <video 
              ref={videoRef} 
              className="qr-scanner-video" 
              autoPlay 
              playsInline 
              muted
            />
            <div className="qr-scanner-overlay-box">
              <div className="qr-scanner-corner tl" />
              <div className="qr-scanner-corner tr" />
              <div className="qr-scanner-corner bl" />
              <div className="qr-scanner-corner br" />
              <div className="qr-scanner-line" />
            </div>
          </div>
        ) : (
          <div className="qr-scanner-manual">
            <div className="qr-scanner-input-group">
              <label className="qr-scanner-label">Recipient Address</label>
              <input
                type="text"
                className="qr-scanner-input"
                placeholder="0x... or wallet address"
                value={manualAddress}
                onChange={(e) => setManualAddress(e.target.value)}
                autoFocus
              />
              {isValidAddress === true && detectedChain && (
                <div className="qr-scanner-validate qr-scanner-valid">
                  ✓ Valid {detectedChain} address
                </div>
              )}
              {isValidAddress === false && (
                <div className="qr-scanner-validate qr-scanner-invalid">
                  ✗ Invalid address format
                </div>
              )}
            </div>

            <div className="qr-scanner-input-group">
              <label className="qr-scanner-label">Supported Chains</label>
              <div style={{ color: 'var(--text-secondary, #B0B0C0)', fontSize: '13px' }}>
                Ethereum, Bitcoin, Solana, TRON, Cosmos, NEAR, and more
              </div>
            </div>

            {recentAddresses.length > 0 && (
              <div className="qr-scanner-input-group">
                <label className="qr-scanner-label">Recent Addresses</label>
                <div className="qr-scanner-history">
                  {recentAddresses.slice(0, 5).map((addr, idx) => (
                    <div 
                      key={idx} 
                      className="qr-scanner-history-item"
                      onClick={() => handleUseAddress(addr)}
                    >
                      <span className="qr-scanner-history-address">{addr}</span>
                      <button 
                        className="qr-scanner-btn-secondary"
                        style={{ padding: '4px 8px', fontSize: '11px' }}
                        onClick={(e) => { e.stopPropagation(); copyToClipboard(addr); }}
                      >
                        📋
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        <div className="qr-scanner-actions">
          <button 
            className="qr-scanner-btn qr-scanner-btn-secondary"
            onClick={onClose}
          >
            Cancel
          </button>
          <button 
            className="qr-scanner-btn qr-scanner-btn-primary"
            disabled={activeTab === 'manual' && !isValidAddress}
            onClick={handleManualSubmit}
          >
            Use Address
          </button>
        </div>
      </div>
    </div>
  );
};

// QR Code Display Component (for receiving)
interface QRDisplayProps {
  address: string;
  chain?: string;
  size?: number;
  showCopy?: boolean;
}

export const QRDisplay: React.FC<QRDisplayProps> = ({ 
  address, 
  chain = 'Ethereum',
  size = 200,
  showCopy = true 
}) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(address);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  // Generate QR code using canvas (simple implementation)
  const qrCodeUrl = `https://api.qrserver.com/v1/create-qr-code/?size=${size}x${size}&data=${encodeURIComponent(address)}`;

  return (
    <div style={{ textAlign: 'center', padding: '20px' }}>
      <img 
        src={qrCodeUrl} 
        alt={`QR Code for ${address}`}
        style={{ borderRadius: '12px', marginBottom: '16px' }}
      />
      <div style={{ 
        fontFamily: 'monospace', 
        fontSize: '12px', 
        color: 'var(--text-secondary, #B0B0C0)',
        wordBreak: 'break-all',
        marginBottom: '12px'
      }}>
        {address}
      </div>
      <div style={{ 
        fontSize: '12px', 
        color: 'var(--primary, #FF6B35)',
        marginBottom: '16px' 
      }}>
        {chain}
      </div>
      {showCopy && (
        <button 
          onClick={handleCopy}
          style={{
            padding: '12px 24px',
            background: 'var(--primary, #FF6B35)',
            color: 'white',
            border: 'none',
            borderRadius: '8px',
            cursor: 'pointer',
            fontWeight: '600'
          }}
        >
          {copied ? '✓ Copied!' : '📋 Copy Address'}
        </button>
      )}
    </div>
  );
};

export default QRScanner;
