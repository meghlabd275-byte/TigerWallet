/**
 * TigerWallet Super Admin - Settings Page
 */

import React, { useState, useEffect } from 'react';
import { superAdminApi } from '../services/api';

export default function Settings() {
  const [config, setConfig] = useState<Record<string, any>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [configKey, setConfigKey] = useState('');
  const [configValue, setConfigValue] = useState('');
  const [twoFa, setTwoFa] = useState({ user_id: '', method: 'totp', phone: '', email: '' });
  const [twoFaSetup, setTwoFaSetup] = useState<any | null>(null);
  const [verifyCode, setVerifyCode] = useState('');
  const [backupUserId, setBackupUserId] = useState('');
  const [backupCode, setBackupCode] = useState('');
  const [backupResult, setBackupResult] = useState<any | null>(null);

  const load = async () => {
    try {
      setLoading(true);
      setError(null);
      const c: any = await superAdminApi.getConfig();
      setConfig(c || {});
    } catch (err: any) {
      setError(err?.message || 'Failed to load settings');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const handleConfigUpdate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!configKey) return;
    setActionLoading(true);
    try {
      let value: any = configValue;
      try { value = JSON.parse(configValue); } catch { /* keep string */ }
      await superAdminApi.updateConfig({ [configKey]: value });
      setConfigKey('');
      setConfigValue('');
      load();
    } catch (err: any) {
      alert(err?.message || 'Failed to update config');
    } finally {
      setActionLoading(false);
    }
  };

  const handleSetup2FA = async (e: React.FormEvent) => {
    e.preventDefault();
    setActionLoading(true);
    try {
      const r = await superAdminApi.setup2FA(twoFa.user_id, twoFa.method, twoFa.phone || undefined, twoFa.email || undefined);
      setTwoFaSetup(r);
    } catch (err: any) {
      alert(err?.message || '2FA setup failed');
    } finally {
      setActionLoading(false);
    }
  };

  const handleVerify2FA = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!twoFaSetup) return;
    setActionLoading(true);
    try {
      const r = await superAdminApi.verify2FASetup(twoFa.user_id, twoFaSetup.secret, verifyCode);
      alert(`2FA verified: ${r.status}`);
      setTwoFaSetup(null);
      setVerifyCode('');
    } catch (err: any) {
      alert(err?.message || '2FA verification failed');
    } finally {
      setActionLoading(false);
    }
  };

  const handleRegenerateBackup = async (e: React.FormEvent) => {
    e.preventDefault();
    setActionLoading(true);
    try {
      const r = await superAdminApi.regenerateBackupCodes(backupUserId, backupCode);
      setBackupResult(r);
    } catch (err: any) {
      alert(err?.message || 'Failed to regenerate backup codes');
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold text-primary mb-6">Settings</h1>

      {error ? (
        <div className="alert alert-error mb-4"><p className="text-error">{error}</p><button className="btn btn-secondary mt-2" onClick={load}>Retry</button></div>
      ) : loading ? (
        <div className="flex items-center justify-center p-8"><div className="loader"></div></div>
      ) : (
        <>
          <h2 className="text-xl font-bold text-primary mb-3">System Configuration</h2>
          <div className="card mb-6"><div className="card-body">
            <form onSubmit={handleConfigUpdate} className="flex gap-3 mb-4">
              <div className="form-group flex-1"><label className="text-secondary">Key</label><input className="input w-full" value={configKey} onChange={(e) => setConfigKey(e.target.value)} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">Value (string or JSON)</label><input className="input w-full" value={configValue} onChange={(e) => setConfigValue(e.target.value)} required /></div>
              <div className="form-group" style={{ alignSelf: 'flex-end' }}><button className="btn btn-primary" disabled={actionLoading} type="submit">Update</button></div>
            </form>
            {Object.keys(config).length === 0 ? (
              <p className="text-secondary">No configuration values.</p>
            ) : (
              <pre className="text-secondary" style={{ whiteSpace: 'pre-wrap' }}>{JSON.stringify(config, null, 2)}</pre>
            )}
          </div></div>

          <h2 className="text-xl font-bold text-primary mb-3">Two-Factor Authentication</h2>
          <div className="card mb-6"><div className="card-body">
            <form onSubmit={handleSetup2FA} className="flex flex-col gap-3 mb-4">
              <div className="flex gap-3">
                <div className="form-group flex-1"><label className="text-secondary">User ID</label><input className="input w-full" value={twoFa.user_id} onChange={(e) => setTwoFa({ ...twoFa, user_id: e.target.value })} required /></div>
                <div className="form-group flex-1"><label className="text-secondary">Method</label><select className="input w-full" value={twoFa.method} onChange={(e) => setTwoFa({ ...twoFa, method: e.target.value })}><option>totp</option><option>sms</option><option>email</option></select></div>
                <div className="form-group flex-1"><label className="text-secondary">Phone</label><input className="input w-full" value={twoFa.phone} onChange={(e) => setTwoFa({ ...twoFa, phone: e.target.value })} /></div>
                <div className="form-group flex-1"><label className="text-secondary">Email</label><input className="input w-full" value={twoFa.email} onChange={(e) => setTwoFa({ ...twoFa, email: e.target.value })} /></div>
              </div>
              <button className="btn btn-primary" disabled={actionLoading} type="submit">Setup 2FA</button>
            </form>

            {twoFaSetup && (
              <div className="alert alert-info mb-4">
                <p className="text-secondary">Secret: <strong>{twoFaSetup.secret}</strong></p>
                {twoFaSetup.qr_code_url && <p className="text-secondary">QR: {twoFaSetup.qr_code_url}</p>}
                <form onSubmit={handleVerify2FA} className="flex gap-3 mt-3">
                  <div className="form-group flex-1"><label className="text-secondary">Verification Code</label><input className="input w-full" value={verifyCode} onChange={(e) => setVerifyCode(e.target.value)} required /></div>
                  <div className="form-group" style={{ alignSelf: 'flex-end' }}><button className="btn btn-primary" disabled={actionLoading} type="submit">Verify</button></div>
                </form>
              </div>
            )}

            <h3 className="text-primary mb-2">Regenerate Backup Codes</h3>
            <form onSubmit={handleRegenerateBackup} className="flex gap-3">
              <div className="form-group flex-1"><label className="text-secondary">User ID</label><input className="input w-full" value={backupUserId} onChange={(e) => setBackupUserId(e.target.value)} required /></div>
              <div className="form-group flex-1"><label className="text-secondary">2FA Code</label><input className="input w-full" value={backupCode} onChange={(e) => setBackupCode(e.target.value)} required /></div>
              <div className="form-group" style={{ alignSelf: 'flex-end' }}><button className="btn btn-secondary" disabled={actionLoading} type="submit">Regenerate</button></div>
            </form>
            {backupResult && (
              <div className="alert alert-info mt-3"><pre className="text-info" style={{ whiteSpace: 'pre-wrap' }}>{JSON.stringify(backupResult, null, 2)}</pre></div>
            )}
          </div></div>
        </>
      )}
    </div>
  );
}
