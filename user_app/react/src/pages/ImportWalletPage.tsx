/**
 * TigerWallet - Import Wallet Page
 * Import existing wallet from seed phrase
 */

import React, { useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { WalletService } from '../services/walletService';
import { useTheme } from '../stores/ThemeStore';

const ImportWalletPage: React.FC = () => {
  const navigate = useNavigate();
  const { theme } = useTheme();
  
  const [step, setStep] = useState(1);
  const [walletName, setWalletName] = useState('');
  const [selectedChain, setSelectedChain] = useState('ethereum');
  const [seedPhrase, setSeedPhrase] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const chains = WalletService.getDefaultChains();

  const validateSeedPhrase = (phrase: string): boolean => {
    const words = phrase.trim().split(/\s+/);
    return words.length === 12 || words.length === 24;
  };

  const handleImportWallet = useCallback(async () => {
    if (!walletName.trim()) {
      setError('Please enter a wallet name');
      return;
    }

    if (!validateSeedPhrase(seedPhrase)) {
      setError('Invalid seed phrase. Must be 12 or 24 words');
      return;
    }

    if (password !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      await WalletService.importWallet({
        seedPhrase: seedPhrase.trim(),
        chain: selectedChain,
        name: walletName,
        password,
      });

      setStep(2);
    } catch (err: any) {
      setError(err.message || 'Failed to import wallet');
    } finally {
      setIsLoading(false);
    }
  }, [walletName, seedPhrase, selectedChain, password, confirmPassword]);

  const handleComplete = useCallback(() => {
    navigate('/wallet');
  }, [navigate]);

  return (
    <div className={`import-wallet-page ${theme}`}>
      <div className="import-wallet-container">
        {/* Header */}
        <div className="page-header">
          <h1>🐯 Import Wallet</h1>
          <p>Restore your wallet using seed phrase</p>
        </div>

        {/* Step 1: Import Details */}
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

            <div className="form-group">
              <label>Seed Phrase</label>
              <textarea
                value={seedPhrase}
                onChange={(e) => setSeedPhrase(e.target.value)}
                placeholder="Enter your 12 or 24 word seed phrase"
                className="form-textarea"
                rows={4}
              />
              <p className="form-hint">Separate words with spaces</p>
            </div>

            <div className="form-group">
              <label>Wallet Password (Optional)</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Enter password to encrypt wallet"
                className="form-input"
              />
            </div>

            <div className="form-group">
              <label>Confirm Password</label>
              <input
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder="Confirm password"
                className="form-input"
              />
            </div>

            {error && <div className="error-message">{error}</div>}

            <button
              className="btn btn-primary btn-large"
              onClick={handleImportWallet}
              disabled={isLoading}
            >
              {isLoading ? 'Importing...' : 'Import Wallet'}
            </button>
          </div>
        )}

        {/* Step 2: Success */}
        {step === 2 && (
          <div className="step-content success-step">
            <div className="success-icon">✅</div>
            <h2>Wallet Imported Successfully!</h2>
            <p>Your wallet has been imported and is ready to use.</p>
            
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
          <span>Don't have a wallet?</span>
          <a href="/create">Create New Wallet</a>
        </div>
      </div>
    </div>
  );
};

export default ImportWalletPage;
