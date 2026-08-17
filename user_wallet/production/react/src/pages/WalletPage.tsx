/**
 * Wallet Page - Token Balances & Management
 *
 * Shows the active wallet's tokens plus a wallet management panel:
 *   - Per-wallet "Setup App Lock" button -> AppLockModal (passcode/passkey)
 *   - "Create Wallet" flow with a "Create with Passkey" option that uses the
 *     real WebAuthn navigator.credentials.create API via utils/passkey and
 *     calls walletService.passkeyCreateWallet; the returned mnemonic is shown
 *     with a Copy button.
 */

import React, { useState, useEffect } from 'react';
import { useWallet } from '../contexts/WalletContext';
import { useTheme } from '../contexts/ThemeContext';
import { Token, Chain } from '../services/WalletService';
import { WalletService } from '../services/WalletService';
import AppLockModal from '../components/AppLockModal';
import { passkeySupported, createPasskey } from '../utils/passkey';
import { backupToDrive } from '../services/googleDriveBackup';

interface PasskeyWalletResult {
  walletId: string;
  label: string;
  address: string;
  mnemonic: string;
}

function WalletPage() {
  const { activeWallet, wallets, refreshBalances, isLoading } = useWallet();
  const { theme, toggleTheme } = useTheme();
  const [tokens, setTokens] = useState<Token[]>([]);
  const [searchQuery, setSearchQuery] = useState('');

  // App lock modal state
  const [lockWalletId, setLockWalletId] = useState<string | null>(null);
  const [lockWalletLabel, setLockWalletLabel] = useState<string | undefined>(undefined);

  // Passkey wallet creation state
  const [walletService] = useState(() => new WalletService());
  const [passkeyLabel, setPasskeyLabel] = useState('');
  const [passkeyChainId, setPasskeyChainId] = useState(1);
  const [creatingPasskey, setCreatingPasskey] = useState(false);
  const [passkeyResult, setPasskeyResult] = useState<PasskeyWalletResult | null>(null);
  const [passkeyError, setPasskeyError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [driveBusy, setDriveBusy] = useState(false);
  const [driveMsg, setDriveMsg] = useState('');

  const webauthnSupported = passkeySupported();

  useEffect(() => {
    if (activeWallet?.tokens) {
      setTokens(activeWallet.tokens);
    }
  }, [activeWallet]);

  const filteredTokens = tokens.filter(token =>
    token.symbol.toLowerCase().includes(searchQuery.toLowerCase()) ||
    token.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const totalBalance = tokens.reduce((acc, token) => acc + token.balanceUSD, 0);

  const openLockModal = (walletId: string, label?: string) => {
    setLockWalletId(walletId);
    setLockWalletLabel(label);
  };

  const closeLockModal = () => {
    setLockWalletId(null);
    setLockWalletLabel(undefined);
  };

  const handlePasskeyCreate = async () => {
    setPasskeyError(null);
    setPasskeyResult(null);
    setCreatingPasskey(true);
    try {
      // 1. Create a real WebAuthn passkey via the browser.
      const passkey = await createPasskey(passkeyLabel || 'tiger-wallet');

      // 2. Provision the wallet on the backend with the passkey credential.
      const res = await walletService.passkeyCreateWallet({
        label: passkeyLabel || `wallet-${Date.now()}`,
        chainId: passkeyChainId,
        credentialId: passkey.credentialId,
        publicKey: passkey.publicKey,
      });

      setPasskeyResult({
        walletId: res.wallet_id,
        label: res.label,
        address: res.address,
        mnemonic: res.mnemonic,
      });

      // Refresh wallet balances so the new wallet appears in the list.
      try {
        await refreshBalances();
      } catch {
        /* refresh is best-effort */
      }
    } catch (err: any) {
      setPasskeyError(err?.message || 'Passkey wallet creation failed');
    } finally {
      setCreatingPasskey(false);
    }
  };

  const copyMnemonic = async () => {
    if (!passkeyResult?.mnemonic) return;
    try {
      await navigator.clipboard.writeText(passkeyResult.mnemonic);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* clipboard may be blocked; ignore */
    }
  };

  // Upload the wallet recovery phrase to Google Drive appDataFolder.
  const handleBackupToDrive = async () => {
    if (!passkeyResult?.mnemonic) return;
    setDriveBusy(true);
    setDriveMsg('');
    try {
      await backupToDrive(passkeyResult.mnemonic);
      setDriveMsg('✓ Backed up to Google Drive');
    } catch (err: unknown) {
      setDriveMsg(err instanceof Error ? err.message : 'Google Drive backup failed');
    } finally {
      setDriveBusy(false);
    }
  };

  const inputClass = `input w-full ${theme === 'dark' ? 'bg-slate-900 border-slate-700' : 'bg-white'}`;

  return (
    <div className="p-6">
      {/* Balance Card */}
      <div className={`card mb-6 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-xl font-semibold">Total Balance</h2>
          <button onClick={toggleTheme} className="p-2 rounded-lg bg-amber-500 text-black">
            {theme === 'dark' ? '☀️' : '🌙'}
          </button>
        </div>
        <div className="text-4xl font-bold text-amber-500">
          ${totalBalance.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
        </div>
        <div className="flex gap-2 mt-4">
          <button className="btn btn-primary flex-1">Send</button>
          <button className="btn btn-secondary flex-1">Receive</button>
          <button className="btn btn-secondary flex-1">Swap</button>
        </div>
      </div>

      {/* Chain Selector */}
      <div className={`card mb-6 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
        <h3 className="font-semibold mb-3">Select Network</h3>
        <div className="flex flex-wrap gap-2">
          {['Ethereum', 'Polygon', 'BNB Chain', 'Arbitrum', 'Solana', 'Avalanche'].map(chain => (
            <button key={chain} className={`px-4 py-2 rounded-lg text-sm ${
              activeWallet?.chain.name === chain 
                ? 'bg-amber-500 text-black' 
                : theme === 'dark' ? 'bg-slate-700' : 'bg-gray-200'
            }`}>
              {chain}
            </button>
          ))}
        </div>
      </div>

      {/* Wallet list with per-wallet Setup App Lock */}
      <div className={`card mb-6 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
        <h3 className="font-semibold mb-3">Your Wallets</h3>
        {wallets.length === 0 ? (
          <p className="text-sm opacity-60">No wallets yet. Create one below.</p>
        ) : (
          <div className="space-y-2">
            {wallets.map((w) => (
              <div
                key={w.id}
                className={`flex items-center justify-between p-3 rounded-lg ${theme === 'dark' ? 'bg-slate-900' : 'bg-gray-100'}`}
              >
                <div className="min-w-0">
                  <p className="font-mono text-sm truncate">{w.address}</p>
                  <p className="text-xs opacity-60">
                    {w.chain?.name || '—'} · {w.balance || '0'}
                  </p>
                </div>
                <button
                  onClick={() => openLockModal(w.id, w.chain?.name)}
                  className="btn btn-secondary text-sm"
                >
                  Setup App Lock
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Create Wallet with Passkey */}
      <div className={`card mb-6 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
        <h3 className="font-semibold mb-3">Create Wallet with Passkey</h3>
        <div className="mb-3">
          <label className="label">Label (optional)</label>
          <input
            type="text"
            value={passkeyLabel}
            onChange={(e) => setPasskeyLabel(e.target.value)}
            placeholder="My Passkey Wallet"
            className={inputClass}
          />
        </div>
        <div className="mb-4">
          <label className="label">Chain</label>
          <select
            value={passkeyChainId}
            onChange={(e) => setPasskeyChainId(Number(e.target.value))}
            className={inputClass}
          >
            <option value={1}>Ethereum</option>
            <option value={137}>Polygon</option>
            <option value={56}>BNB Chain</option>
            <option value={42161}>Arbitrum</option>
            <option value={43114}>Avalanche</option>
          </select>
        </div>

        {passkeyError && (
          <div className="mb-3 p-3 rounded bg-red-500/20 border border-red-500 text-red-500 text-sm">
            {passkeyError}
          </div>
        )}

        {passkeyResult && (
          <div className="mb-4 p-4 rounded bg-green-500/10 border border-green-500">
            <p className="text-sm text-green-500 font-medium mb-1">
              ✓ Passkey wallet created: {passkeyResult.label}
            </p>
            <p className="font-mono text-xs break-all opacity-80 mb-3">{passkeyResult.address}</p>
            <p className="text-xs opacity-70 mb-1">Recovery phrase (keep this safe):</p>
            <div className="flex items-start gap-2">
              <p className="font-mono text-sm break-all flex-1 p-2 rounded bg-black/20">
                {passkeyResult.mnemonic}
              </p>
              <button onClick={copyMnemonic} className="btn btn-secondary text-sm shrink-0">
                {copied ? 'Copied!' : 'Copy'}
              </button>
              <button
                onClick={handleBackupToDrive}
                disabled={driveBusy}
                className="btn btn-secondary text-sm shrink-0"
              >
                {driveBusy ? 'Uploading...' : '☁️ Backup to Google Drive'}
              </button>
            </div>
            {driveMsg && (
              <p className={`text-xs mt-2 ${driveMsg.startsWith('✓') ? 'text-green-500' : 'text-red-500'}`}>
                {driveMsg}
              </p>
            )}
          </div>
        )}

        <button
          onClick={handlePasskeyCreate}
          disabled={creatingPasskey || !webauthnSupported}
          className="btn btn-primary w-full"
          title={webauthnSupported ? 'Create wallet secured by a device passkey' : 'WebAuthn not supported in this browser'}
        >
          {creatingPasskey ? 'Creating...' : webauthnSupported ? 'Create with Passkey' : 'Passkey not supported'}
        </button>
        {!webauthnSupported && (
          <p className="text-xs opacity-50 mt-2 text-center">
            WebAuthn is not supported in this browser.
          </p>
        )}
      </div>

      {/* Search */}
      <div className="mb-4">
        <input
          type="text"
          placeholder="Search tokens..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className={`input w-full ${theme === 'dark' ? 'bg-slate-800 border-slate-700' : 'bg-white'}`}
        />
      </div>

      {/* Token List */}
      <div className="space-y-3">
        {filteredTokens.map(token => (
          <div key={token.address} className={`token-item ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`}>
            <div className="token-info">
              <div className="token-icon bg-amber-500 rounded-full w-10 h-10 flex items-center justify-center text-black font-bold">
                {token.symbol.slice(0, 2)}
              </div>
              <div>
                <div className="font-semibold">{token.name}</div>
                <div className="text-sm opacity-60">{token.symbol}</div>
              </div>
            </div>
            <div className="token-balance text-right">
              <div className="font-semibold">{parseFloat(token.balance).toFixed(6)}</div>
              <div className="text-sm opacity-60">${token.balanceUSD.toFixed(2)}</div>
            </div>
          </div>
        ))}
      </div>

      {isLoading && (
        <div className="flex justify-center py-4">
          <div className="spinner"></div>
        </div>
      )}

      {lockWalletId && (
        <AppLockModal
          walletId={lockWalletId}
          walletLabel={lockWalletLabel}
          onClose={closeLockModal}
        />
      )}
    </div>
  );
}

export default WalletPage;
