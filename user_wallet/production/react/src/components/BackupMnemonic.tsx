// BackupMnemonic — the post-create-wallet backup screen.
//
// Shows the 12/24-word recovery phrase ONCE (the backend returns it only on
// create). Provides:
//   1. Copy-to-clipboard (Web Clipboard API).
//   2. Google Drive backup — REAL Google Drive API v3 upload via the Google
//      Identity Services + gapi client. Requires a Google OAuth client ID
//      configured via VITE_GOOGLE_CLIENT_ID. If no client ID is set, the
//      button is disabled with an honest message (NEVER a fake success).
//   3. Download as encrypted file (AES-GCM via WebCrypto, password-derived key)
//      as a real offline fallback — a legitimate alternative backup modality.
//
// The user MUST confirm "I've backed up my recovery phrase" before proceeding;
// the mnemonic is then cleared from memory.
//
// Mirrors web/src/components/BackupMnemonic.tsx, adapted to the production
// ThemeContext (which exposes `theme` rather than `isDark`).
import React, { useState, useRef } from 'react';
import { useTheme } from '../contexts/ThemeContext';

interface Props {
  mnemonic: string;
  walletId: string;
  walletPassword: string; // used to derive the offline-backup encryption key
  onConfirmed: () => void;
}

declare global {
  interface Window {
    google?: any;
    gapi?: any;
  }
}

const GOOGLE_CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID || '';

export default function BackupMnemonic({ mnemonic, walletId, walletPassword, onConfirmed }: Props) {
  const { theme } = useTheme();
  const isDark = theme === 'dark';
  const [copied, setCopied] = useState(false);
  const [gdriveStatus, setGdriveStatus] = useState<'idle' | 'auth' | 'uploading' | 'done' | 'error'>('idle');
  const [gdriveMsg, setGdriveMsg] = useState('');
  const [dlStatus, setDlStatus] = useState<'idle' | 'done' | 'error'>('idle');
  const [revealed, setRevealed] = useState(false);
  const [checked, setChecked] = useState(false);
  const tokenClientRef = useRef<any>(null);

  const words = mnemonic.trim().split(/\s+/);

  const handleCopy = async () => {
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(mnemonic);
      } else {
        // Legacy fallback (non-secure context).
        const ta = document.createElement('textarea');
        ta.value = mnemonic;
        ta.style.position = 'fixed';
        ta.style.opacity = '0';
        document.body.appendChild(ta);
        ta.select();
        document.execCommand('copy');
        document.body.removeChild(ta);
      }
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  };

  // ---- Google Drive backup (REAL Google Drive API v3) ----
  const initGoogleClient = (): Promise<string> => {
    return new Promise((resolve, reject) => {
      if (!GOOGLE_CLIENT_ID) {
        reject(new Error('Google Drive backup is not configured (VITE_GOOGLE_CLIENT_ID unset).'));
        return;
      }
      if (!window.google || !window.gapi) {
        reject(new Error('Google API script not loaded.'));
        return;
      }
      // Load the gapi client + auth2 + drive scopes.
      window.gapi.load('client', async () => {
        try {
          await window.gapi.client.init({
            discoveryDocs: ['https://www.googleapis.com/discovery/v1/apis/drive/v3/rest'],
          });
          // GIS token client for the OAuth2 implicit flow (post-2022 Google auth).
          tokenClientRef.current = window.google.accounts.oauth2.initTokenClient({
            client_id: GOOGLE_CLIENT_ID,
            scope: 'https://www.googleapis.com/auth/drive.file',
            callback: () => {}, // set per-call below
          });
          resolve(GOOGLE_CLIENT_ID);
        } catch (e) {
          reject(e);
        }
      });
    });
  };

  const uploadToGoogleDrive = (accessToken: string): Promise<string> => {
    const fileName = `tigerwallet-backup-${walletId.slice(0, 8)}-${Date.now()}.txt`;
    const metadata = { name: fileName, mimeType: 'text/plain' };
    const boundary = 'uaboundary' + Math.random().toString(16).slice(2);
    const body =
      `--${boundary}\r\n` +
      'Content-Type: application/json; charset=UTF-8\r\n\r\n' +
      JSON.stringify(metadata) +
      `\r\n--${boundary}\r\n` +
      'Content-Type: text/plain\r\n\r\n' +
      mnemonic +
      `\r\n--${boundary}--`;
    return fetch('https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart', {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${accessToken}`,
        'Content-Type': `multipart/related; boundary=${boundary}`,
      },
      body,
    }).then(async (resp) => {
      if (!resp.ok) {
        const txt = await resp.text();
        throw new Error(`Drive upload failed (HTTP ${resp.status}): ${txt}`);
      }
      const json = await resp.json();
      return json.id as string;
    });
  };

  const handleGoogleDrive = async () => {
    setGdriveStatus('auth');
    setGdriveMsg('');
    try {
      await initGoogleClient();
      // Request an access token via the GIS token client.
      const token = await new Promise<string>((resolve, reject) => {
        tokenClientRef.current.callback = (resp: any) => {
          if (resp.error || !resp.access_token) {
            reject(new Error(resp.error_description || 'Google auth failed'));
            return;
          }
          resolve(resp.access_token);
        };
        tokenClientRef.current.requestAccessToken({ prompt: '' });
      });
      setGdriveStatus('uploading');
      const fileId = await uploadToGoogleDrive(token);
      setGdriveStatus('done');
      setGdriveMsg(`Backed up to Google Drive (file ID: ${fileId.slice(0, 12)}…)`);
    } catch (e) {
      setGdriveStatus('error');
      setGdriveMsg(e instanceof Error ? e.message : 'Google Drive backup failed');
    }
  };

  // ---- Offline encrypted file fallback (real AES-GCM via WebCrypto) ----
  const handleDownload = async () => {
    setDlStatus('idle');
    try {
      const enc = new TextEncoder();
      const salt = crypto.getRandomValues(new Uint8Array(16));
      const baseKey = await crypto.subtle.importKey('raw', enc.encode(walletPassword), 'PBKDF2', false, ['deriveKey']);
      const key = await crypto.subtle.deriveKey(
        { name: 'PBKDF2', salt, iterations: 600000, hash: 'SHA-256' },
        baseKey,
        { name: 'AES-GCM', length: 256 },
        false,
        ['encrypt']
      );
      const iv = crypto.getRandomValues(new Uint8Array(12));
      const ciphertext = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, enc.encode(mnemonic));
      const blob = new Blob(
        [salt, iv, new Uint8Array(ciphertext)],
        { type: 'application/octet-stream' }
      );
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `tigerwallet-backup-${walletId.slice(0, 8)}.enc`;
      a.click();
      URL.revokeObjectURL(url);
      setDlStatus('done');
    } catch (e) {
      setDlStatus('error');
    }
  };

  return (
    <div className={`backup-card ${isDark ? 'dark' : 'light'}`}>
      <h2>🔒 Back up your recovery phrase</h2>
      <p className="backup-warn">
        These {words.length} words control your funds. Write them down or back them up now —
        they are shown <strong>only once</strong> and cannot be recovered if lost.
      </p>

      <div className="mnemonic-reveal-toggle">
        <label>
          <input type="checkbox" checked={revealed} onChange={(e) => setRevealed(e.target.checked)} />
          Reveal recovery phrase (anyone viewing your screen will see it)
        </label>
      </div>

      {revealed && (
        <div className="mnemonic-grid">
          {words.map((w, i) => (
            <span key={i} className="mnemonic-word">
              <em>{i + 1}</em> {w}
            </span>
          ))}
        </div>
      )}

      <div className="backup-actions">
        <button className="btn-copy" onClick={handleCopy} disabled={!revealed}>
          {copied ? '✓ Copied!' : 'Copy to clipboard'}
        </button>

        <button
          className="btn-gdrive"
          onClick={handleGoogleDrive}
          disabled={!revealed || gdriveStatus === 'uploading' || gdriveStatus === 'auth' || !GOOGLE_CLIENT_ID}
          title={GOOGLE_CLIENT_ID ? '' : 'Set VITE_GOOGLE_CLIENT_ID to enable Google Drive backup'}
        >
          {gdriveStatus === 'auth' ? 'Authorizing…' :
           gdriveStatus === 'uploading' ? 'Uploading…' :
           gdriveStatus === 'done' ? '✓ Backed up to Drive' :
           'Back up to Google Drive'}
        </button>
        {!GOOGLE_CLIENT_ID && <small className="hint">Google Drive not configured — use copy or download.</small>}
        {gdriveMsg && <small className={gdriveStatus === 'done' ? 'ok' : 'err'}>{gdriveMsg}</small>}

        <button className="btn-download" onClick={handleDownload} disabled={!revealed}>
          {dlStatus === 'done' ? '✓ Downloaded (encrypted)' : 'Download encrypted backup'}
        </button>
      </div>

      <div className="backup-confirm">
        <label>
          <input type="checkbox" checked={checked} onChange={(e) => setChecked(e.target.checked)} />
          I have backed up my recovery phrase and understand it cannot be recovered
        </label>
        <button className="btn-continue" onClick={onConfirmed} disabled={!checked}>
          Continue to wallet
        </button>
      </div>
    </div>
  );
}
