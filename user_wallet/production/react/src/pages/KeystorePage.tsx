/**
 * Keystore Page - Export / import keystore JSON.
 *
 * Export: password input + walletService.exportKeystore(walletId, password)
 *   (POST /keystore/export) -> shows the keystore JSON with copy + download.
 * Import: textarea for keystore JSON + password + walletService.importKeystore
 *   (POST /keystore/import).
 * All calls go through WalletService; no mock data.
 */

import React, { useState } from 'react';
import { useTheme } from '../contexts/ThemeContext';
import { useWallet } from '../contexts/WalletContext';
import { WalletService } from '../services/WalletService';

function KeystorePage() {
  const { theme } = useTheme();
  const { activeWallet } = useWallet();
  const [walletService] = useState(() => new WalletService());

  const [activeTab, setActiveTab] = useState<'export' | 'import'>('export');

  // Export state
  const [exportPassword, setExportPassword] = useState('');
  const [exportedKeystore, setExportedKeystore] = useState('');
  const [exporting, setExporting] = useState(false);
  const [exportMessage, setExportMessage] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  // Import state
  const [keystoreJson, setKeystoreJson] = useState('');
  const [importPassword, setImportPassword] = useState('');
  const [importLabel, setImportLabel] = useState('');
  const [importing, setImporting] = useState(false);
  const [importMessage, setImportMessage] = useState<string | null>(null);

  const [error, setError] = useState<string | null>(null);

  const handleExport = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setExportMessage(null);
    setExportedKeystore('');

    if (!activeWallet) {
      setError('No active wallet selected');
      return;
    }
    if (!exportPassword) {
      setError('Wallet password is required to export a keystore');
      return;
    }

    setExporting(true);
    try {
      const result: any = await walletService.exportKeystore({
        walletId: activeWallet.id,
        password: exportPassword,
      });
      const json =
        typeof result === 'string'
          ? result
          : result?.keystore
          ? typeof result.keystore === 'string'
            ? result.keystore
            : JSON.stringify(result.keystore, null, 2)
          : JSON.stringify(result, null, 2);
      setExportedKeystore(json);
      setExportMessage('Keystore exported successfully.');
      setExportPassword('');
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to export keystore');
    } finally {
      setExporting(false);
    }
  };

  const handleCopy = async () => {
    if (!exportedKeystore) return;
    try {
      await navigator.clipboard.writeText(exportedKeystore);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setError('Failed to copy keystore to clipboard');
    }
  };

  const handleDownload = () => {
    if (!exportedKeystore) return;
    const blob = new Blob([exportedKeystore], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `keystore-${activeWallet?.id ?? 'wallet'}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const handleImport = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setImportMessage(null);

    if (!keystoreJson.trim()) {
      setError('Keystore JSON is required');
      return;
    }
    if (!importPassword) {
      setError('Password is required to import a keystore');
      return;
    }

    setImporting(true);
    try {
      const result: any = await walletService.importKeystore({
        keystore: keystoreJson.trim(),
        password: importPassword,
        label: importLabel.trim() || undefined,
      });
      setImportMessage(
        result?.message ||
          result?.status ||
          'Keystore imported successfully.'
      );
      setKeystoreJson('');
      setImportPassword('');
      setImportLabel('');
    } catch (err: any) {
      setError(err?.response?.data?.error || err?.message || 'Failed to import keystore');
    } finally {
      setImporting(false);
    }
  };

  const cardClass = `card mb-6 ${theme === 'dark' ? 'bg-slate-800' : 'bg-white'}`;
  const inputClass = `input w-full ${theme === 'dark' ? 'bg-slate-900 border-slate-700' : 'bg-white'}`;

  return (
    <div className="p-6 max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Keystore</h1>

      {/* Tabs */}
      <div className="flex gap-2 mb-6">
        {(['export', 'import'] as const).map((t) => (
          <button
            key={t}
            onClick={() => { setActiveTab(t); setError(null); }}
            className={`px-4 py-2 rounded-lg text-sm capitalize ${
              activeTab === t
                ? 'bg-amber-500 text-black'
                : theme === 'dark' ? 'bg-slate-800' : 'bg-gray-200'
            }`}
          >
            {t}
          </button>
        ))}
      </div>

      {error && (
        <div className={`card mb-6 ${theme === 'dark' ? 'bg-red-900/30' : 'bg-red-50'}`}>
          <p className="text-sm text-red-500">{error}</p>
        </div>
      )}

      {activeTab === 'export' && (
        <>
          {!activeWallet && (
            <div className={cardClass}>
              <p className="text-sm opacity-60">Select a wallet to export its keystore.</p>
            </div>
          )}

          <form onSubmit={handleExport} className={cardClass}>
            <h3 className="font-semibold mb-4">Export Keystore</h3>
            {activeWallet && (
              <p className="text-xs font-mono opacity-60 mb-4">
                Wallet: {activeWallet.address}
              </p>
            )}
            <div className="mb-6">
              <label className="label">Wallet Password</label>
              <input
                type="password"
                value={exportPassword}
                onChange={(e) => setExportPassword(e.target.value)}
                placeholder="Wallet password"
                className={inputClass}
                required
              />
            </div>
            <button
              type="submit"
              disabled={exporting || !activeWallet}
              className="btn btn-primary w-full"
            >
              {exporting ? 'Exporting...' : 'Export Keystore'}
            </button>
          </form>

          {exportMessage && (
            <div className={`card mb-6 ${theme === 'dark' ? 'bg-green-900/30' : 'bg-green-50'}`}>
              <p className="text-sm text-green-500">{exportMessage}</p>
            </div>
          )}

          {exportedKeystore && (
            <div className={cardClass}>
              <div className="flex justify-between items-center mb-3">
                <h3 className="font-semibold">Keystore JSON</h3>
                <div className="flex gap-2">
                  <button onClick={handleCopy} className="btn btn-secondary text-sm">
                    {copied ? 'Copied!' : 'Copy'}
                  </button>
                  <button onClick={handleDownload} className="btn btn-secondary text-sm">
                    Download
                  </button>
                </div>
              </div>
              <pre
                className={`text-xs font-mono overflow-auto p-3 rounded max-h-80 ${
                  theme === 'dark' ? 'bg-slate-900 text-gray-300' : 'bg-gray-100 text-gray-800'
                }`}
              >
{exportedKeystore}
              </pre>
            </div>
          )}
        </>
      )}

      {activeTab === 'import' && (
        <>
          {importMessage && (
            <div className={`card mb-6 ${theme === 'dark' ? 'bg-green-900/30' : 'bg-green-50'}`}>
              <p className="text-sm text-green-500">{importMessage}</p>
            </div>
          )}

          <form onSubmit={handleImport} className={cardClass}>
            <h3 className="font-semibold mb-4">Import Keystore</h3>

            <div className="mb-4">
              <label className="label">Keystore JSON</label>
              <textarea
                value={keystoreJson}
                onChange={(e) => setKeystoreJson(e.target.value)}
                placeholder='{"crypto": {...}, "address": "..."}'
                rows={8}
                className={`${inputClass} font-mono text-xs`}
                required
              />
            </div>

            <div className="mb-4">
              <label className="label">Password</label>
              <input
                type="password"
                value={importPassword}
                onChange={(e) => setImportPassword(e.target.value)}
                placeholder="Keystore password"
                className={inputClass}
                required
              />
            </div>

            <div className="mb-6">
              <label className="label">Label (optional)</label>
              <input
                type="text"
                value={importLabel}
                onChange={(e) => setImportLabel(e.target.value)}
                placeholder="Imported wallet"
                className={inputClass}
              />
            </div>

            <button type="submit" disabled={importing} className="btn btn-primary w-full">
              {importing ? 'Importing...' : 'Import Keystore'}
            </button>
          </form>
        </>
      )}
    </div>
  );
}

export default KeystorePage;
