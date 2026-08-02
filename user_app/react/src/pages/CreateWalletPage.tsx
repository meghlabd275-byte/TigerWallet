/**
 * TigerWallet - Create Wallet Page
 * Complete wallet creation with seed phrase generation and backup
 */

import React, { useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { WalletService } from '../services/walletService';
import { useTheme } from '../stores/ThemeStore';

const CreateWalletPage: React.FC = () => {
  const navigate = useNavigate();
  const { theme } = useTheme();
  
  const [step, setStep] = useState(1);
  const [walletName, setWalletName] = useState('');
  const [selectedChain, setSelectedChain] = useState('ethereum');
  const [seedPhrase, setSeedPhrase] = useState<string[]>([]);
  const [confirmedBackup, setConfirmedBackup] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showSeedPhrase, setShowSeedPhrase] = useState(false);

  const chains = WalletService.getDefaultChains();

  const handleCreateWallet = useCallback(async () => {
    if (!walletName.trim()) {
      setError('Please enter a wallet name');
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      // Generate wallet with seed phrase
      const result = await WalletService.createWallet({
        name: walletName,
        chain: selectedChain,
      });

      setSeedPhrase(result.seedPhrase.split(' '));
      setStep(2);
    } catch (err: any) {
      setError(err.message || 'Failed to create wallet');
    } finally {
      setIsLoading(false);
    }
  }, [walletName, selectedChain]);

  const handleConfirmBackup = useCallback(() => {
    if (!confirmedBackup) {
      setError('Please confirm you have backed up your seed phrase');
      return;
    }
    setStep(3);
  }, [confirmedBackup]);

  const handleComplete = useCallback(() => {
    navigate('/wallet');
  }, [navigate]);

  const copySeedPhrase = useCallback(() => {
    navigator.clipboard.writeText(seedPhrase.join(' '));
  }, [seedPhrase]);

  return (
    <div className={`create-wallet-page ${theme}`}>
      <div className="create-wallet-container">
        {/* Header */}
        <div className="page-header">
          <h1>🐯 Create New Wallet</h1>
          <div className="step-indicator">
            <span className={`step ${step >= 1 ? 'active' : ''}`}>1</span>
            <span className="step-line"></span>
            <span className={`step ${step >= 2 ? 'active' : ''}`}>2</span>
            <span className="step-line"></span>
            <span className={`step ${step >= 3 ? 'active' : ''}`}>3</span>
          </div>
        </div>

        {/* Step 1: Wallet Details */}
        {step === 1 && (
          <div className="step-content">
            <div className="form-group">
              <label>Wallet Name</label>
              <input
                type="text"
                value={walletName}
                onChange={(e) => setWalletName(e.target.value)}
                placeholder="My Wallet"
                className="form-input"
              />
            </div>

            <div className="form-group">
              <label>Select Blockchain</label>
              <div className="chain-grid">
                {chains.map((chain) => (
                  <button
                    key={chain.id}
                    className={`chain-option ${selectedChain === chain.id ? 'selected' : ''}`}
                    onClick={() => setSelectedChain(chain.id)}
                  >
                    <span className="chain-icon">{chain.symbol}</span>
                    <span className="chain-name">{chain.name}</span>
                  </button>
                ))}
              </div>
            </div>

            {error && <div className="error-message">{error}</div>}

            <button
              className="btn btn-primary btn-large"
              onClick={handleCreateWallet}
              disabled={isLoading}
            >
              {isLoading ? 'Creating...' : 'Create Wallet'}
            </button>
          </div>
        )}

        {/* Step 2: Backup Seed Phrase */}
        {step === 2 && (
          <div className="step-content">
            <div className="warning-banner">
              ⚠️ Important: Write down your seed phrase and store it safely
            </div>

            <div className="seed-phrase-container">
              <div className="seed-phrase-grid">
                {seedPhrase.map((word, index) => (
                  <div key={index} className="seed-word">
                    <span className="word-number">{index + 1}.</span>
                    <span className="word-text">{word}</span>
                  </div>
                ))}
              </div>

              <button
                className="btn btn-secondary"
                onClick={copySeedPhrase}
              >
                📋 Copy to Clipboard
              </button>
            </div>

            <div className="backup-confirmation">
              <label className="checkbox-label">
                <input
                  type="checkbox"
                  checked={confirmedBackup}
                  onChange={(e) => setConfirmedBackup(e.target.checked)}
                />
                I have securely written down my seed phrase and understand it is the only way to recover my wallet
              </label>
            </div>

            {error && <div className="error-message">{error}</div>}

            <div className="button-group">
              <button
                className="btn btn-secondary"
                onClick={() => setStep(1)}
              >
                Back
              </button>
              <button
                className="btn btn-primary"
                onClick={handleConfirmBackup}
              >
                I've Backed Up My Seed Phrase
              </button>
            </div>
          </div>
        )}

        {/* Step 3: Success */}
        {step === 3 && (
          <div className="step-content success-step">
            <div className="success-icon">✅</div>
            <h2>Wallet Created Successfully!</h2>
            <p>Your wallet has been created and is ready to use.</p>
            
            <div className="wallet-summary">
              <div className="summary-row">
                <span>Name:</span>
                <span>{walletName}</span>
              </div>
              <div className="summary-row">
                <span>Network:</span>
                <span>{chains.find(c => c.id === selectedChain)?.name}</span>
              </div>
            </div>

            <button
              className="btn btn-primary btn-large"
              onClick={handleComplete}
            >
              Go to Wallet
            </button>
          </div>
        )}

        {/* Back to Login */}
        <div className="page-footer">
          <span>Already have a wallet?</span>
          <a href="/import">Import Existing Wallet</a>
        </div>
      </div>
    </div>
  );
};

export default CreateWalletPage;
