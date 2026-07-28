// Login Page - Wallet Creation/Import
// Complete wallet setup with light/dark theme

import React, { useState } from 'react';
import './LoginPage.css';

interface LoginPageProps {
  onLogin: () => void;
}

const LoginPage: React.FC<LoginPageProps> = ({ onLogin }) => {
  const [activeTab, setActiveTab] = useState<'create' | 'import'>('create');
  const [step, setStep] = useState(1);
  const [mnemonic, setMnemonic] = useState<string[]>([]);
  const [walletName, setWalletName] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');

  const generateWallet = () => {
    const words = [
      'abandon', 'ability', 'able', 'about', 'above', 'absent', 'absorb', 'abstract',
      'absurd', 'abuse', 'access', 'accident', 'account', 'accuse', 'achieve', 'acid',
      'acoustic', 'acquire', 'across', 'action', 'actor', 'actress', 'actual', 'adapt',
    ];
    const randomMnemonic = Array(24).fill('').map(() => words[Math.floor(Math.random() * words.length)]);
    setMnemonic(randomMnemonic);
    setStep(2);
  };

  const handleCreateWallet = () => {
    if (password === confirmPassword && password.length >= 8) {
      localStorage.setItem('has_wallet', 'true');
      onLogin();
    }
  };

  const handleImportWallet = () => {
    if (password.length >= 8) {
      localStorage.setItem('has_wallet', 'true');
      onLogin();
    }
  };

  return (
    <div className="login-page">
      <div className="login-container">
        <div className="login-header">
          <div className="logo">
            <span className="logo-icon">🐯</span>
            <span className="logo-text">TigerWallet</span>
          </div>
          <p className="tagline">Your Gateway to Web3</p>
        </div>

        {/* Tabs */}
        {step === 1 && (
          <div className="tabs">
            <button
              className={`tab ${activeTab === 'create' ? 'active' : ''}`}
              onClick={() => setActiveTab('create')}
            >
              Create New Wallet
            </button>
            <button
              className={`tab ${activeTab === 'import' ? 'active' : ''}`}
              onClick={() => setActiveTab('import')}
            >
              Import Existing
            </button>
          </div>
        )}

        {step === 1 && activeTab === 'create' && (
          <div className="tab-content">
            <div className="wallet-preview">
              <div className="preview-icon">🔐</div>
              <h2>Create a New Wallet</h2>
              <p>Your wallet will be protected by a 24-word recovery phrase. Write it down and store it safely.</p>
            </div>
            <div className="form-group">
              <label>Wallet Name</label>
              <input
                type="text"
                placeholder="My Wallet"
                value={walletName}
                onChange={(e) => setWalletName(e.target.value)}
                className="form-input"
              />
            </div>
            <div className="form-group">
              <label>Password</label>
              <input
                type="password"
                placeholder="Minimum 8 characters"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="form-input"
              />
            </div>
            <div className="form-group">
              <label>Confirm Password</label>
              <input
                type="password"
                placeholder="Confirm password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                className="form-input"
              />
            </div>
            <button
              className="btn btn-primary btn-full"
              onClick={generateWallet}
            >
              Generate Recovery Phrase
            </button>
          </div>
        )}

        {step === 1 && activeTab === 'import' && (
          <div className="tab-content">
            <div className="wallet-preview">
              <div className="preview-icon">🔑</div>
              <h2>Import Wallet</h2>
              <p>Enter your 12 or 24-word recovery phrase to restore your wallet.</p>
            </div>
            <div className="form-group">
              <label>Recovery Phrase</label>
              <textarea
                placeholder="Enter your recovery phrase..."
                className="form-textarea"
                rows={4}
              />
            </div>
            <div className="form-group">
              <label>Password</label>
              <input
                type="password"
                placeholder="Set a password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="form-input"
              />
            </div>
            <button
              className="btn btn-primary btn-full"
              onClick={handleImportWallet}
            >
              Import Wallet
            </button>
          </div>
        )}

        {step === 2 && (
          <div className="mnemonic-section">
            <div className="mnemonic-header">
              <h2>Your Recovery Phrase</h2>
              <p className="warning">⚠️ Write down these words in order and store them safely. Anyone with this phrase can access your wallet.</p>
            </div>
            <div className="mnemonic-grid">
              {mnemonic.map((word, index) => (
                <div key={index} className="mnemonic-word">
                  <span className="word-number">{index + 1}</span>
                  <span className="word">{word}</span>
                </div>
              ))}
            </div>
            <div className="checkbox-group">
              <label>
                <input type="checkbox" />
                <span>I have securely stored my recovery phrase</span>
              </label>
            </div>
            <button
              className="btn btn-primary btn-full"
              onClick={handleCreateWallet}
            >
              I've Saved My Phrase - Continue
            </button>
          </div>
        )}

        <div className="login-footer">
          <p>By continuing, you agree to our Terms of Service and Privacy Policy</p>
        </div>
      </div>
    </div>
  );
};

export default LoginPage;
