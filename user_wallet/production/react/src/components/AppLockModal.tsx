/**
 * App Lock setup modal.
 *
 * Lets the user configure an app lock for a wallet using either a passcode
 * or a WebAuthn passkey. Calls walletService.setupLock() with the chosen
 * credential. Uses the real navigator.credentials API via utils/passkey.
 */

import React, { useState } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { WalletService } from '../services/WalletService';
import { passkeySupported, createPasskey } from '../utils/passkey';

interface AppLockModalProps {
  walletId: string;
  walletLabel?: string;
  onClose: () => void;
  onSetupComplete?: (result: { hasPasscode: boolean; hasPasskey: boolean }) => void;
}

function AppLockModal({ walletId, walletLabel, onClose, onSetupComplete }: AppLockModalProps) {
  const { theme } = useTheme();
  const [walletService] = useState(() => new WalletService());

  const [passcode, setPasscode] = useState('');
  const [confirmPasscode, setConfirmPasscode] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const webauthnSupported = passkeySupported();

  const handlePasscodeSetup = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);

    if (!passcode || passcode.length < 4) {
      setError('Passcode must be at least 4 characters');
      return;
    }
    if (passcode !== confirmPasscode) {
      setError('Passcodes do not match');
      return;
    }

    setBusy(true);
    try {
      const res: any = await walletService.setupLock(walletId, { passcode });
      setSuccess(res?.message || 'App lock passcode configured');
      onSetupComplete?.({
        hasPasscode: typeof res?.has_passcode === 'boolean' ? res.has_passcode : true,
        hasPasskey: typeof res?.has_passkey === 'boolean' ? res.has_passkey : false,
      });
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to set passcode lock');
    } finally {
      setBusy(false);
    }
  };

  const handlePasskeySetup = async () => {
    setError(null);
    setSuccess(null);
    setBusy(true);
    try {
      const passkey = await createPasskey(walletLabel || `wallet-${walletId}`);
      const res: any = await walletService.setupLock(walletId, {
        passkeyCredentialId: passkey.credentialId,
        passkeyPublicKey: passkey.publicKey,
      });
      setSuccess(res?.message || 'App lock passkey configured');
      onSetupComplete?.({
        hasPasscode: typeof res?.has_passcode === 'boolean' ? res.has_passcode : false,
        hasPasskey: typeof res?.has_passkey === 'boolean' ? res.has_passkey : true,
      });
    } catch (err: any) {
      setError(err?.message || err?.response?.data?.error || 'Failed to set passkey lock');
    } finally {
      setBusy(false);
    }
  };

  const inputClass = `input w-full ${theme === 'dark' ? 'bg-slate-900 border-slate-700' : 'bg-white'}`;
  const modalBg = theme === 'dark' ? 'bg-slate-800' : 'bg-white';

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className={`w-full max-w-md rounded-2xl p-6 ${modalBg}`}>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold">
            Setup App Lock
            {walletLabel ? <span className="opacity-60 text-sm font-normal"> · {walletLabel}</span> : null}
          </h3>
          <button onClick={onClose} className="text-2xl leading-none opacity-60 hover:opacity-100" aria-label="Close">×</button>
        </div>

        {error && (
          <div className="mb-4 p-3 rounded bg-red-500/20 border border-red-500 text-red-500 text-sm">
            {error}
          </div>
        )}
        {success && (
          <div className="mb-4 p-3 rounded bg-green-500/20 border border-green-500 text-green-500 text-sm">
            {success}
          </div>
        )}

        <form onSubmit={handlePasscodeSetup} className="mb-6">
          <div className="mb-3">
            <label className="label">Passcode</label>
            <input
              type="password"
              value={passcode}
              onChange={(e) => setPasscode(e.target.value)}
              placeholder="Choose a passcode"
              className={inputClass}
              autoComplete="new-password"
            />
          </div>
          <div className="mb-4">
            <label className="label">Confirm Passcode</label>
            <input
              type="password"
              value={confirmPasscode}
              onChange={(e) => setConfirmPasscode(e.target.value)}
              placeholder="Confirm passcode"
              className={inputClass}
              autoComplete="new-password"
            />
          </div>
          <button type="submit" disabled={busy} className="btn btn-primary w-full">
            {busy ? 'Setting...' : 'Set Passcode Lock'}
          </button>
        </form>

        <div className="relative my-4">
          <div className="absolute inset-0 flex items-center">
            <div className="w-full border-t opacity-20"></div>
          </div>
          <div className="relative flex justify-center">
            <span className={`px-2 text-xs ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'} opacity-60`}>or</span>
          </div>
        </div>

        <button
          onClick={handlePasskeySetup}
          disabled={busy || !webauthnSupported}
          className="btn btn-secondary w-full"
          title={webauthnSupported ? 'Use device passkey' : 'WebAuthn not supported in this browser'}
        >
          {webauthnSupported ? 'Use Passkey' : 'Passkey not supported'}
        </button>

        {!webauthnSupported && (
          <p className="text-xs opacity-50 mt-2 text-center">
            WebAuthn is not supported in this browser. Use a passcode instead.
          </p>
        )}
      </div>
    </div>
  );
}

export default AppLockModal;
